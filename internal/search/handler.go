// Package search wraps `ripgrep` (rg) for the Inspector's Search tab.
// One endpoint: GET /api/v1/search?path=&q=&case=&include=&max=
//
// Why shell out instead of importing a Go regex engine: rg already
// handles file walking, .gitignore awareness, binary detection, glob
// filters, and parallelism. Reimplementing that surface for a panel
// query is wasted code. The cost is a hard runtime dependency on rg —
// startup probes for it once and the handler returns 501 otherwise.
package search

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	// rgTimeout caps a single search. ripgrep on a sane source tree
	// (~10k files) finishes in <100ms; this guard is for pathological
	// regexes (catastrophic backtracking) and huge monorepos.
	rgTimeout = 6 * time.Second

	// defaultMax / hardMax bound results returned to the client. The
	// panel renders a finite list; beyond a few hundred matches the
	// user wants to refine the query, not page through 10k hits.
	defaultMax = 200
	hardMax    = 1000

	// maxFileBytes mirrors what most editors index; skips lockfiles
	// and minified bundles that flood results without value.
	maxFileBytes = "1M"
)

type Handlers struct {
	log    *slog.Logger
	rgPath string // resolved at construction; "" if rg unavailable
}

func NewHandlers(log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	rgPath, _ := exec.LookPath("rg")
	if rgPath == "" {
		log.Warn("ripgrep not on PATH; /search will return 501",
			"hint", "brew install ripgrep")
	}
	return &Handlers{log: log.With("component", "search.http"), rgPath: rgPath}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Get("/search", h.search)
}

// Match is one search hit. Path is repo-relative to the search root
// when possible (rg gives relative paths when invoked with -C dir).
type Match struct {
	Path       string     `json:"path"`
	Line       int        `json:"line"`
	Text       string     `json:"text"`
	Submatches []Submatch `json:"submatches,omitempty"`
}

// Submatch is a {start,end} byte offset inside `Text` so the client
// can render highlighted spans. Comes straight from rg --json.
type Submatch struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Response struct {
	Matches   []Match `json:"matches"`
	Truncated bool    `json:"truncated,omitempty"`
	Elapsed   string  `json:"elapsed,omitempty"`
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	if h.rgPath == "" {
		writeError(w, http.StatusNotImplemented,
			errors.New("ripgrep (rg) not installed on the gateway host"))
		return
	}

	dir, err := dirParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, errors.New("q is required"))
		return
	}

	caseMode := r.URL.Query().Get("case")
	include := strings.TrimSpace(r.URL.Query().Get("include"))
	maxResults := defaultMax
	if v := r.URL.Query().Get("max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxResults = n
		}
	}
	if maxResults > hardMax {
		maxResults = hardMax
	}

	args := []string{
		"--json",
		"--no-heading",
		"--max-filesize", maxFileBytes,
		"--max-columns", "400", // truncate insane single-line files
		"--max-columns-preview",
	}
	switch caseMode {
	case "sensitive":
		args = append(args, "--case-sensitive")
	case "insensitive":
		args = append(args, "--ignore-case")
	default:
		// rg's smart-case is the right default — uppercase letters in
		// the query trigger sensitive mode, lowercase stays loose.
		args = append(args, "--smart-case")
	}
	if include != "" {
		// rg supports multiple -g globs; allow space-separated input
		// from the client without committing to a JSON array shape.
		for _, g := range strings.Fields(include) {
			args = append(args, "-g", g)
		}
	}
	args = append(args, "-e", q, ".")

	ctx, cancel := context.WithTimeout(r.Context(), rgTimeout)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(ctx, h.rgPath, args...)
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := cmd.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	matches, truncated := readMatches(stdout, maxResults)

	// rg exits 1 when there are zero matches — that's not an error
	// to the user, just an empty result. Anything else (2+) is real.
	werr := cmd.Wait()
	if werr != nil {
		if exitErr, ok := werr.(*exec.ExitError); ok && exitErr.ExitCode() <= 1 {
			// 0 = matches found, 1 = no matches; both fine.
		} else if ctx.Err() != nil {
			writeError(w, http.StatusGatewayTimeout,
				fmt.Errorf("search timed out after %s", rgTimeout))
			return
		} else {
			writeError(w, http.StatusInternalServerError, werr)
			return
		}
	}

	writeJSON(w, http.StatusOK, Response{
		Matches:   matches,
		Truncated: truncated,
		Elapsed:   time.Since(start).Round(time.Millisecond).String(),
	})
}

// readMatches consumes rg --json output: one JSON object per line,
// types we care about are "match". Stops once max is reached and
// drains the pipe so rg can finish cleanly.
func readMatches(rdr interface{ Read(p []byte) (int, error) }, max int) ([]Match, bool) {
	out := make([]Match, 0, max)
	truncated := false
	scanner := bufio.NewScanner(rdr)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		if len(out) >= max {
			truncated = true
			// Keep draining so rg's writes don't block on a full pipe.
			continue
		}
		var ev rgEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != "match" {
			continue
		}
		m := Match{
			Path: ev.Data.Path.Text,
			Line: ev.Data.LineNumber,
			Text: strings.TrimRight(ev.Data.Lines.Text, "\n"),
		}
		for _, sm := range ev.Data.Submatches {
			m.Submatches = append(m.Submatches, Submatch{
				Start: sm.Start,
				End:   sm.End,
			})
		}
		out = append(out, m)
	}
	return out, truncated
}

// rgEvent matches the slice of `rg --json` output we consume. See
// `man rg` for the full schema; we ignore begin/end/context/summary.
type rgEvent struct {
	Type string    `json:"type"`
	Data rgEvtData `json:"data"`
}

type rgEvtData struct {
	Path       rgText       `json:"path"`
	Lines      rgText       `json:"lines"`
	LineNumber int          `json:"line_number"`
	Submatches []rgSubmatch `json:"submatches"`
}

type rgText struct {
	Text string `json:"text"`
}

type rgSubmatch struct {
	Match rgText `json:"match"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func dirParam(r *http.Request) (string, error) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	if p == "" {
		return "", errors.New("path is required")
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("path must be absolute")
	}
	return filepath.Clean(p), nil
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
