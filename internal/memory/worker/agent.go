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
	"strings"
	"time"

	"github.com/google/uuid"
)

// AgentWorker spawns a headless Claude, Codex, or Antigravity CLI in
// --print mode to perform one LLM judgement / summary call.
//
// Why this exists: the existing SummarizerWorker calls a generic
// OpenAI-compatible endpoint (typically LM Studio with a 9-13B
// local model). For high-frequency low-quality work that's fine,
// but for the narrative summary tasks (gitactivity, transcript)
// the operator may want frontier-model quality, and they already
// pay for a Claude subscription that opendray manages.
// M25 lets them flip those touchpoints to "use one of my Claude
// accounts as a one-shot worker" without standing up a separate
// LLM service.
//
// Implementation contract:
//   - Spawns `claude --print --append-system-prompt <prompt>
//     --session-id <fresh-uuid> --bare` (or `agy --print ...`).
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
	cfg         Config
	accounts    AccountReader
	agyAccounts AgyAccountReader
	log         *slog.Logger
}

// AccountReader is the subset of cliacct.Service the AgentWorker
// needs. Kept minimal so the worker package doesn't pull the full
// service surface — easier to mock in tests.
type AccountReader interface {
	ResolveSpawnCreds(ctx context.Context, id string) (configDir, token string, err error)
}

// AgyAccountReader is the subset of agyacct.Service the AgentWorker needs to
// bind an antigravity headless call to a specific account. agy keys its whole
// credential state off $HOME, so an account resolves to a dedicated HOME dir
// (mirrors catalog/adapter.go's interactive spawn path).
type AgyAccountReader interface {
	ResolveSpawnHome(ctx context.Context, id string) (home string, err error)
}

// NewAgentWorker constructs a worker that will spawn the agent CLI
// named by cfg.ProviderID. cfg.AccountID is consulted for multi-account
// auth: Claude accounts resolve to a config dir + OAuth token, antigravity
// accounts to a dedicated HOME. Empty means "use the default account"
// (whatever the CLI resolves on its own from the host config). accounts /
// agyAccounts may be nil, in which case account pinning is skipped.
func NewAgentWorker(accounts AccountReader, agyAccounts AgyAccountReader, cfg Config, log *slog.Logger) *AgentWorker {
	if log == nil {
		log = slog.Default()
	}
	return &AgentWorker{cfg: cfg, accounts: accounts, agyAccounts: agyAccounts, log: log.With(
		"component", "memory.worker.agent",
		"provider", cfg.ProviderID,
		"task", string(cfg.Task))}
}

func (w *AgentWorker) Kind() WorkerKind { return WorkerAgent }

func (w *AgentWorker) Run(ctx context.Context, req Request) (Response, error) {
	switch w.cfg.ProviderID {
	case "claude", "codex", "antigravity", "grok", "opencode":
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
	// from the host filesystem layout. Claude / Antigravity both read
	// surrounding CLAUDE.md / AGENTS.md when invoked; an empty
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

	// Optional MCP attach: render the provider's MCP config, apply the
	// returned args + env, and add the per-provider flags that let the CLI
	// execute the tools headless. runCwd is where the CLI runs — the scratch
	// dir for every provider except antigravity, which derives the memory
	// scope from its own cwd and so must run in the project dir.
	runCwd := scratch
	if req.MCP != nil && req.MCP.Provision != nil {
		home := w.effectiveHome(runCtx)
		if w.cfg.ProviderID == "antigravity" && strings.TrimSpace(req.MCP.Cwd) != "" {
			runCwd = req.MCP.Cwd
		}
		mcpArgs, mcpEnv, perr := req.MCP.Provision(w.cfg.ProviderID, scratch, runCwd, req.MCP.Cwd, home)
		if perr != nil {
			return Response{}, fmt.Errorf("agent worker: provision mcp: %w", perr)
		}
		args = append(args, mcpArgs...)
		for k, v := range mcpEnv {
			env = append(env, k+"="+v)
		}
		args = append(args, mcpToolApprovalArgs(w.cfg.ProviderID)...)
	}

	binary := agentBinary(w.cfg.ProviderID)
	if p, err := exec.LookPath(binary); err == nil {
		binary = p
	}

	cmd := exec.CommandContext(runCtx, binary, args...)
	cmd.Dir = runCwd
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
	switch w.cfg.ProviderID {
	case "codex":
		// codex exec has no system-prompt flag, so the system block
		// (+ JSON-schema instruction) is folded into stdin ahead of the
		// user input.
		//
		// codex's stdin reader rejects the whole prompt if it contains any
		// invalid UTF-8 ("input is not valid UTF-8 (invalid byte at offset
		// N)"). Scrub any stray invalid sequences (e.g. a multi-byte char
		// severed by an upstream byte-slice truncation) rather than fail the
		// call. arg-based CLIs (agy/grok/opencode) don't hit this path.
		input = strings.ToValidUTF8(combinedPrompt(req), "�")
	case "antigravity", "grok", "opencode":
		// These CLIs take the prompt as a command-line ARG, not from stdin
		// (agy via --print, grok via -p, opencode as `run`'s positional
		// message — see buildCommand). Feeding stdin too is ignored, so
		// send nothing.
		input = ""
	}
	go func() {
		defer stdin.Close()
		_, _ = stdin.Write([]byte(input))
	}()

	if err := cmd.Wait(); err != nil {
		// Claude / Antigravity CLIs print auth + 4xx errors to stdout
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

// effectiveHome resolves the HOME the MCP renderer should target. For an
// account-bound antigravity spawn that's the account's dedicated HOME (agy
// keys its whole state — and its mcp_config.json — off HOME); everything
// else uses the gateway user's real HOME. Mirrors buildCommand's agy account
// resolution so the mcp_config.json lands where the CLI will read it.
func (w *AgentWorker) effectiveHome(ctx context.Context) string {
	if w.cfg.ProviderID == "antigravity" && w.cfg.AccountID != "" && w.agyAccounts != nil {
		hctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if home, err := w.agyAccounts.ResolveSpawnHome(hctx, w.cfg.AccountID); err == nil && home != "" {
			return home
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

// mcpToolApprovalArgs returns the extra CLI flags a headless spawn needs to
// actually EXECUTE MCP tools (not merely have them configured). Each CLI
// gates tool use differently in non-interactive mode:
//
//   - claude allow-lists the memory server's tools so --print runs them with
//     no prompt while every other tool (file writes, bash) stays unlisted and
//     is auto-denied — this is both the tool gate and the file-safety boundary.
//   - grok / opencode auto-DENY tool approvals in headless mode unless told to
//     approve; the read-only memory server is the only tool source and the CLI
//     runs in a throwaway scratch dir, so blanket approval is safe here.
//   - antigravity already carries --dangerously-skip-permissions from
//     buildCommand; codex exec auto-runs configured MCP tools while its
//     read-only sandbox still blocks its own file/shell writes.
func mcpToolApprovalArgs(providerID string) []string {
	switch providerID {
	case "claude":
		return []string{"--allowedTools", "mcp__opendray-memory"}
	case "grok":
		return []string{"--always-approve"}
	case "opencode":
		return []string{"--dangerously-skip-permissions"}
	default:
		return nil
	}
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
	case "antigravity":
		// agy takes the prompt as the VALUE of --print, NOT from stdin.
		// The old stdin approach made agy read its own
		// `--dangerously-skip-permissions` flag as the prompt (it echoed a
		// guide about that flag and never answered the real question) — a
		// bug that broke every antigravity headless call. Pass the folded
		// system+user block as the --print value; keep
		// --dangerously-skip-permissions FIRST so it isn't swallowed as the
		// prompt. No system-prompt flag on agy, so the system block (+ the
		// JSON-schema instruction) is folded into the prompt value.
		args := []string{"--dangerously-skip-permissions", "--print", combinedPrompt(req)}
		if w.cfg.Model != "" {
			args = append(args, "--model", w.cfg.Model)
		}
		// Multi-account: agy keys its entire credential state off $HOME, so
		// binding to an account = pointing HOME at the account's dedicated dir
		// (mirrors catalog/adapter.go's interactive spawn). Empty AccountID or
		// no reader → the host-global agy login is used.
		var env []string
		if w.cfg.AccountID != "" && w.agyAccounts != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			home, err := w.agyAccounts.ResolveSpawnHome(ctx, w.cfg.AccountID)
			if err != nil {
				return nil, nil, fmt.Errorf("agent worker: read antigravity account %s: %w",
					w.cfg.AccountID, err)
			}
			if home != "" {
				env = append(env, "HOME="+home)
			}
		}
		return args, env, nil
	case "grok":
		// grok's headless flag is `-p <PROMPT>` (the analog of claude -p /
		// codex exec), with the prompt as a direct string arg — NOT stdin.
		// No system-prompt flag, so the system block (+ any JSON-schema
		// instruction) is folded into the prompt value via combinedPrompt.
		// --output-format plain keeps stdout to the final answer (json /
		// streaming-json would wrap it in event envelopes we'd have to
		// parse). We do NOT pass --always-approve: a discussion reply must
		// never write, and headless auto-denies tool approvals anyway.
		args := []string{"-p", combinedPrompt(req), "--output-format", "plain"}
		if w.cfg.Model != "" {
			args = append(args, "-m", w.cfg.Model)
		}
		return args, nil, nil
	case "opencode":
		// `opencode run <message>` is the non-interactive path: the prompt is
		// a positional arg (folded system+user, since opencode has no
		// system-prompt flag), and cmd.Dir is the isolated scratch dir. Model
		// is provider/model form (e.g. anthropic/claude-sonnet-4-6). We do NOT
		// pass --dangerously-skip-permissions: a worker reply must never write,
		// and without it opencode auto-denies tool approvals in headless mode.
		args := []string{"run", combinedPrompt(req)}
		if w.cfg.Model != "" {
			args = append(args, "--model", w.cfg.Model)
		}
		return args, nil, nil
	}
	return nil, nil, ErrAgentUnsupported
}

// combinedPrompt folds the system block (+ optional JSON-schema
// instruction) ahead of the user input, for CLIs with no system-prompt
// flag: codex reads it from stdin, antigravity takes it as the --print
// value.
func combinedPrompt(req Request) string {
	sys := req.SystemPrompt
	if req.ResponseFormatJSONSchema != "" {
		sys = sys + "\n\nReturn a single JSON object conforming to this schema:\n```json\n" +
			req.ResponseFormatJSONSchema + "\n```\nOutput nothing else."
	}
	if sys != "" {
		return sys + "\n\n---\n\n" + req.UserInput
	}
	return req.UserInput
}

// agentBinary maps a worker provider id to its executable. Identity for
// claude/codex; antigravity's CLI is `agy`.
func agentBinary(providerID string) string {
	if providerID == "antigravity" {
		return "agy"
	}
	return providerID
}
