package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// AgentWorker spawns a headless Claude or Gemini CLI in --print
// mode to perform one LLM judgement / summary call.
//
// Why this exists: the existing SummarizerWorker calls a generic
// OpenAI-compatible endpoint (typically LM Studio with a 9-13B
// local model). For high-frequency low-quality work that's fine,
// but for the narrative summary tasks (gitactivity, transcript)
// the operator may want frontier-model quality, and they already
// pay for a Claude / Gemini subscription that opendray manages.
// M25 lets them flip those touchpoints to "use one of my Claude
// accounts as a one-shot worker" without standing up a separate
// LLM service.
//
// Implementation contract:
//   - Spawns `claude --print --append-system-prompt <prompt>
//     --session-id <fresh-uuid> --bare` (or `gemini --print ...`).
//   - Feeds Request.UserInput on stdin.
//   - Captures stdout until EOF; that's the response Content.
//   - Process gets killed if Request.Timeout elapses.
//   - NO session row is written: these are out-of-band agent
//     invocations, deliberately invisible to the journaler /
//     session manager. The fresh UUID still gives Claude its
//     own jsonl file (so transcript readers in OTHER spawns
//     can't accidentally pick up the worker's content), but
//     opendray doesn't index it.
//   - Working directory is a scratch dir to keep project
//     context (CLAUDE.md, .opendray banner) from polluting the
//     worker's prompt.
type AgentWorker struct {
	cfg      Config
	accounts AccountReader
	log      *slog.Logger
}

// AccountReader is the subset of cliacct.Service the AgentWorker
// needs. Kept minimal so the worker package doesn't pull the full
// service surface — easier to mock in tests.
type AccountReader interface {
	ResolveSpawnCreds(ctx context.Context, id string) (configDir, token string, err error)
}

// NewAgentWorker constructs a worker that will spawn the agent CLI
// named by cfg.ProviderID. cfg.AccountID is consulted for
// Claude multi-account auth; empty means "use the default account"
// (whatever Claude resolves on its own with the host's
// ~/.claude/.claude.json — usually the only authed account).
func NewAgentWorker(accounts AccountReader, cfg Config, log *slog.Logger) *AgentWorker {
	if log == nil {
		log = slog.Default()
	}
	return &AgentWorker{cfg: cfg, accounts: accounts, log: log.With(
		"component", "memory.worker.agent",
		"provider", cfg.ProviderID,
		"task", string(cfg.Task))}
}

func (w *AgentWorker) Kind() WorkerKind { return WorkerAgent }

func (w *AgentWorker) Run(ctx context.Context, req Request) (Response, error) {
	switch w.cfg.ProviderID {
	case "claude", "gemini", "codex", "antigravity":
	default:
		return Response{}, ErrAgentUnsupported
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Scratch CWD — a per-call temp dir keeps the spawn isolated
	// from the host filesystem layout. Claude / Gemini both read
	// surrounding CLAUDE.md / GEMINI.md when invoked; an empty
	// scratch dir avoids accidentally pulling in unrelated
	// project context.
	scratch, err := os.MkdirTemp("", "opd-memory-worker-*")
	if err != nil {
		return Response{}, fmt.Errorf("agent worker: scratch dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()

	sessionID := uuid.NewString()
	args, env, err := w.buildCommand(req, sessionID, scratch)
	if err != nil {
		return Response{}, err
	}

	binary := agentBinary(w.cfg.ProviderID)
	if p, err := exec.LookPath(binary); err == nil {
		binary = p
	}

	cmd := exec.CommandContext(runCtx, binary, args...)
	cmd.Dir = scratch
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Response{}, fmt.Errorf("agent worker: stdin pipe: %w", err)
	}

	t0 := time.Now()
	if err := cmd.Start(); err != nil {
		return Response{}, fmt.Errorf("agent worker: start: %w", err)
	}

	// Feed the user input then close stdin so the agent knows
	// the prompt is complete. `claude --print` reads until EOF
	// before generating, mirroring most non-interactive CLIs.
	// Codex has no system-prompt flag, so its system block (plus the
	// JSON-schema instruction) is folded into stdin ahead of the
	// user input.
	input := req.UserInput
	// Gemini receives the prompt via the --prompt arg (see buildCommand),
	// not stdin, so leave its stdin empty to avoid duplicating the prompt.
	if w.cfg.ProviderID == "gemini" {
		input = ""
	}
	// codex and antigravity (agy) have no system-prompt CLI flag, so the
	// system block (+ JSON-schema instruction) is folded into stdin ahead
	// of the user input.
	if w.cfg.ProviderID == "codex" || w.cfg.ProviderID == "antigravity" {
		sys := req.SystemPrompt
		if req.ResponseFormatJSONSchema != "" {
			sys = sys + "\n\nReturn a single JSON object conforming to this schema:\n```json\n" +
				req.ResponseFormatJSONSchema + "\n```\nOutput nothing else."
		}
		if sys != "" {
			input = sys + "\n\n---\n\n" + input
		}
	}
	go func() {
		defer stdin.Close()
		_, _ = stdin.Write([]byte(input))
	}()

	if err := cmd.Wait(); err != nil {
		// Claude / Gemini CLIs print auth + 4xx errors to stdout
		// (not stderr), so include both streams in the error
		// message — operators can't debug "exit status 1 (stderr: )"
		// blind.
		stderrTrunc := truncate(stderr.String(), 200)
		stdoutTrunc := truncate(stdout.String(), 400)
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return Response{}, fmt.Errorf("agent worker: timeout after %s (stdout: %s, stderr: %s)",
				timeout, stdoutTrunc, stderrTrunc)
		}
		return Response{}, fmt.Errorf("agent worker: %s headless run: %w (stdout: %s, stderr: %s)",
			w.cfg.ProviderID, err, stdoutTrunc, stderrTrunc)
	}
	dur := time.Since(t0).Milliseconds()

	out := stdout.String()
	// Codex prints a progress transcript to stdout; the clean final
	// message lands in the --output-last-message file.
	if w.cfg.ProviderID == "codex" {
		if data, rerr := os.ReadFile(filepath.Join(scratch, "last-message.txt")); rerr == nil {
			if s := string(bytes.TrimSpace(data)); s != "" {
				out = s
			}
		}
	}
	return Response{
		Content:    out,
		DurationMS: dur,
		WorkerKind: WorkerAgent,
		ProviderID: w.cfg.ProviderID,
		AccountID:  w.cfg.AccountID,
		// Token counts unknown — agent CLIs don't expose them
		// reliably. The metrics table records 0; cost UI will
		// estimate from byte counts as a stopgap.
	}, nil
}

func (w *AgentWorker) buildCommand(req Request, sessionID, scratch string) ([]string, []string, error) {
	switch w.cfg.ProviderID {
	case "claude":
		args := []string{
			"--print",
			"--session-id", sessionID,
		}
		// NOTE: --bare is tempting (it skips hooks / plugin
		// sync / CLAUDE.md auto-discovery), but it forces
		// auth via ANTHROPIC_API_KEY only — our multi-account
		// OAuth tokens (CLAUDE_CODE_OAUTH_TOKEN) get ignored
		// and the call fails with exit 1 "Not logged in".
		// We rely on the scratch CWD to isolate from project
		// CLAUDE.md, and --print already skips tool use so
		// PostToolUse hooks won't fire.
		if w.cfg.Model != "" {
			// Per-task model pin: cheap chores on cheap models.
			args = append(args, "--model", w.cfg.Model)
		}
		sys := req.SystemPrompt
		if req.ResponseFormatJSONSchema != "" {
			sys = sys + "\n\nReturn a single JSON object conforming to this schema:\n```json\n" +
				req.ResponseFormatJSONSchema + "\n```\nOutput nothing else."
		}
		if sys != "" {
			args = append(args, "--append-system-prompt", sys)
		}
		env := []string{}
		if w.cfg.AccountID != "" && w.accounts != nil {
			// Multi-account auth — point Claude at the right
			// config dir + OAuth token. Same plumbing the
			// session manager uses in catalog/adapter.go.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			configDir, token, err := w.accounts.ResolveSpawnCreds(ctx, w.cfg.AccountID)
			if err != nil {
				return nil, nil, fmt.Errorf("agent worker: read claude account %s: %w",
					w.cfg.AccountID, err)
			}
			if token != "" {
				env = append(env, "CLAUDE_CODE_OAUTH_TOKEN="+token)
			}
			if configDir != "" {
				env = append(env, "CLAUDE_CONFIG_DIR="+configDir)
			}
		}
		return args, env, nil
	case "codex":
		// `codex exec` is the non-interactive mode: prompt from stdin
		// ("-"), read-only sandbox (a worker must never write), no git
		// requirement in the scratch dir, and the clean final message
		// written to a file Run reads back (stdout carries a progress
		// transcript). System prompt is folded into stdin by Run.
		args := []string{
			"exec",
			"--skip-git-repo-check",
			"--sandbox", "read-only",
			"--output-last-message", filepath.Join(scratch, "last-message.txt"),
		}
		if w.cfg.Model != "" {
			args = append(args, "--model", w.cfg.Model)
		}
		args = append(args, "-")
		return args, nil, nil
	case "gemini":
		// Gemini's headless flag is -p/--prompt (NOT --print, which is
		// Claude-only — gemini errors "Unknown argument: print"). The
		// prompt is passed as the flag value; the system block is read
		// from GEMINI.md via --include-directories below. Run does not
		// also pipe the prompt to stdin for gemini (it is here in args).
		args := []string{
			"--prompt", req.UserInput,
			"--session-id", sessionID,
		}
		if w.cfg.Model != "" {
			args = append(args, "--model", w.cfg.Model)
		}
		sys := req.SystemPrompt
		if req.ResponseFormatJSONSchema != "" {
			sys = sys + "\n\nReturn a single JSON object conforming to this schema:\n```json\n" +
				req.ResponseFormatJSONSchema + "\n```\nOutput nothing else."
		}
		if sys != "" {
			// Gemini ingests system instructions via GEMINI.md
			// in workspace. Write a scratch one alongside the
			// run dir; --include-directories pulls it in.
			path := filepath.Join(os.TempDir(),
				"opd-memory-worker-gemini-"+sessionID+".md")
			if err := os.WriteFile(path, []byte(sys), 0o600); err != nil {
				return nil, nil, fmt.Errorf("agent worker: write GEMINI.md: %w", err)
			}
			args = append(args, "--include-directories", filepath.Dir(path))
		}
		return args, nil, nil
	case "antigravity":
		// agy --print reads the prompt from stdin and prints the
		// response to stdout (verified). No system-prompt flag — the
		// system block is folded into stdin by Run (see above).
		// --dangerously-skip-permissions keeps a worker non-interactive
		// (no tool-permission prompts that would hang the pipe).
		args := []string{"--print", "--dangerously-skip-permissions"}
		if w.cfg.Model != "" {
			args = append(args, "--model", w.cfg.Model)
		}
		return args, nil, nil
	}
	return nil, nil, ErrAgentUnsupported
}

// agentBinary maps a worker provider id to its executable. Identity for
// claude/gemini/codex; antigravity's CLI is `agy`.
func agentBinary(providerID string) string {
	if providerID == "antigravity" {
		return "agy"
	}
	return providerID
}
