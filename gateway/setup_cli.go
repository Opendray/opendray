package gateway

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// cliPackages maps the short CLI name (as selected in the wizard) to the
// npm package that installs it. Hardcoded — the input is never
// interpolated into the shell command.
var cliPackages = map[string]string{
	"claude": "@anthropic-ai/claude-code",
	"codex":  "@openai/codex",
	"gemini": "@google/gemini-cli",
}

// installJob tracks one in-flight `npm install -g <pkg>` run. Output is
// fanned out to any number of WebSocket subscribers, so re-opening the
// stream after a browser reload replays the log from the beginning.
type installJob struct {
	id       string
	cli      string
	cmd      *exec.Cmd
	mu       sync.Mutex
	lines    []installLine // full history — kept so late subscribers see everything
	done     bool
	exitCode int
	subs     map[chan installLine]struct{}
}

// installLine wraps one chunk of output or a terminal "exit" signal.
// Marshaled directly as the WS payload.
type installLine struct {
	Type string `json:"type"` // "stdout" | "stderr" | "exit"
	Data string `json:"data,omitempty"`
	Code int    `json:"code,omitempty"` // present only on Type=="exit"
}

// installRegistry is the single source of truth for active install jobs.
// Lives on the setupHandlers so it's scoped to a setup-mode session; any
// jobs still running when finalize fires will be torn down by process
// shutdown.
type installRegistry struct {
	mu   sync.Mutex
	jobs map[string]*installJob
}

func newInstallRegistry() *installRegistry {
	return &installRegistry{jobs: make(map[string]*installJob)}
}

func (r *installRegistry) start(cli, pkg string) (*installJob, error) {
	id, err := newJobID()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("npm", "install", "-g", pkg)
	// Merge stderr into stdout would hide npm's progress pretty-printer —
	// keep them separate so the UI can style warnings differently.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	j := &installJob{
		id:   id,
		cli:  cli,
		cmd:  cmd,
		subs: make(map[chan installLine]struct{}),
	}

	// Scanners pump output into the job; when both reach EOF cmd.Wait
	// can collect the exit status.
	go j.pump("stdout", stdout)
	go j.pump("stderr", stderr)
	go func() {
		err := cmd.Wait()
		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				code = 1
			}
		}
		j.emit(installLine{Type: "exit", Code: code})
		j.mu.Lock()
		j.done = true
		j.exitCode = code
		subs := j.subs
		j.subs = nil
		j.mu.Unlock()
		for ch := range subs {
			close(ch)
		}
		// GC finished jobs after a grace window so users can't poll for
		// previous output forever.
		time.AfterFunc(90*time.Second, func() {
			r.mu.Lock()
			delete(r.jobs, id)
			r.mu.Unlock()
		})
	}()

	r.mu.Lock()
	r.jobs[id] = j
	r.mu.Unlock()
	return j, nil
}

func (r *installRegistry) get(id string) (*installJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	j, ok := r.jobs[id]
	return j, ok
}

// pump reads until EOF, emits each line.
func (j *installJob) pump(stream string, rc io.ReadCloser) {
	defer rc.Close()
	sc := bufio.NewScanner(rc)
	// npm can print long lines during install; bump the buffer.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1<<20)
	for sc.Scan() {
		j.emit(installLine{Type: stream, Data: sc.Text()})
	}
}

// emit appends to the history AND fans out to all live subscribers. The
// lock is held while doing the send; subscribers that drop slow are
// dropped — they were never going to catch up anyway.
func (j *installJob) emit(line installLine) {
	j.mu.Lock()
	j.lines = append(j.lines, line)
	for ch := range j.subs {
		select {
		case ch <- line:
		default:
			// Can't keep up — drop to avoid blocking pump.
		}
	}
	j.mu.Unlock()
}

// subscribe returns a replay of the history (snapshot) AND a channel
// that receives future emits. Closed when the job finishes.
func (j *installJob) subscribe() ([]installLine, <-chan installLine) {
	j.mu.Lock()
	defer j.mu.Unlock()
	history := make([]installLine, len(j.lines))
	copy(history, j.lines)
	if j.done {
		ch := make(chan installLine)
		close(ch)
		return history, ch
	}
	ch := make(chan installLine, 64)
	j.subs[ch] = struct{}{}
	return history, ch
}

// ──────────────────────────────────────────────────────────────────────
// HTTP handlers — wired into the setup-mode router.

// cliInstallStart — POST /api/setup/cli-install
// Body: {"cli":"claude"|"codex"|"gemini"}
// Returns: {"sessionId":"…"}
func (h *setupHandlers) cliInstallStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CLI string `json:"cli"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	pkg, ok := cliPackages[req.CLI]
	if !ok {
		respondError(w, http.StatusBadRequest, "unknown cli; must be one of claude, codex, gemini")
		return
	}
	// Check npm is on PATH — fail fast with a useful message instead of a
	// cryptic "exec: npm" on the other end.
	if _, err := exec.LookPath("npm"); err != nil {
		respondError(w, http.StatusBadRequest, "npm not found in PATH — install Node.js first")
		return
	}
	job, err := h.installs.start(req.CLI, pkg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "start: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"sessionId": job.id,
		"package":   pkg,
	})
}

// cliInstallStream — GET /api/setup/cli-install/{id}/ws?token=…
func (h *setupHandlers) cliInstallStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, ok := h.installs.get(id)
	if !ok {
		respondError(w, http.StatusNotFound, "install session not found")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	history, ch := job.subscribe()
	// Replay history first — a late-joining client (browser reload, slow
	// render) still sees every line.
	for _, line := range history {
		if err := conn.WriteJSON(line); err != nil {
			return
		}
	}
	// Then stream live updates until the channel closes (job done).
	for line := range ch {
		if err := conn.WriteJSON(line); err != nil {
			return
		}
	}
	// Give the client a beat to flush before the socket closes so the
	// "exit" line reliably lands in the UI.
	time.Sleep(100 * time.Millisecond)
}

func newJobID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
