// Package commands implements the M1 command dispatcher for the OpenDray
// plugin platform. It resolves commands declared in a plugin's manifest,
// performs capability checks through the bridge gate, and executes the
// matching run-kind handler.
//
// # Design decisions vs. M1-PLAN
//
// Invoke returns (*Result, error) rather than (any, error) as written in the
// M1-PLAN spec. The concrete type is required for T11's HTTP handler to
// marshal the response without a type assertion; using any was acknowledged as
// vagueness in the spec. T11 imports this package and marshals *Result directly.
//
// No audit from Dispatcher: capability-gated kinds (exec, runTask) are audited
// by bridge.Gate.Check via its existing hook. Non-gated kinds (notify, openUrl)
// are observable through HTTP access logs. Adding a second audit path here
// would create duplicate rows and tightly couple this package to store types,
// which violates the decoupling rule already established in bridge/capabilities.go.
package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os/exec"
	"time"

	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/contributions"
)

// ─── Sentinel errors ──────────────────────────────────────────────────────────

var (
	// ErrCommandNotFound is returned when the plugin is not registered or
	// the commandID is absent from its contributions.
	ErrCommandNotFound = errors.New("command not found")

	// ErrRunKindUnimpl is returned for run kinds that are not implemented in M1
	// (host, openView) or for unknown kinds (defensive future-proofing).
	ErrRunKindUnimpl = errors.New("run kind requires M2/M3")

	// ErrMissingRunSpec is returned when a command has a nil Run field or the
	// required run-kind fields are empty (e.g. exec with no Method).
	ErrMissingRunSpec = errors.New("command has no run spec")
)

// output truncation limit (bytes).
const maxOutput = 16384

// execTimeout is the hard timeout applied to every exec run-kind invocation.
const execTimeout = 10 * time.Second

// ─── Interfaces ───────────────────────────────────────────────────────────────

// CapabilityChecker is the minimum surface Dispatcher needs from bridge.Gate.
// Accepting an interface here keeps tests free from booting a real Gate+DB;
// bridge.*Gate satisfies it automatically.
type CapabilityChecker interface {
	Check(ctx context.Context, plugin string, need bridge.Need) error
}

// TaskRunner is the minimum surface Dispatcher needs from the task runner for
// the runTask kind. Nil disables the kind.
type TaskRunner interface {
	Run(ctx context.Context, id string, stdout, stderr io.Writer) (exitCode int, err error)
}

// ─── Config + Dispatcher ─────────────────────────────────────────────────────

// Config bundles Dispatcher dependencies. Using a struct instead of positional
// args lets us add future deps (LLM client, telegram bridge) without breaking
// the constructor signature.
type Config struct {
	Registry *contributions.Registry // required — looks up commands
	Gate     CapabilityChecker       // required — capability enforcement
	Tasks    TaskRunner              // optional — nil disables runTask kind
	Log      *slog.Logger            // optional — defaults to slog.Default()
}

// Dispatcher resolves and executes commands declared by installed plugins.
// It is thread-safe; one instance per Runtime.
type Dispatcher struct {
	registry *contributions.Registry
	gate     CapabilityChecker
	tasks    TaskRunner
	log      *slog.Logger
}

// NewDispatcher builds a Dispatcher. Returns error iff required deps are nil.
func NewDispatcher(cfg Config) (*Dispatcher, error) {
	if cfg.Registry == nil {
		return nil, errors.New("commands: Registry is required")
	}
	if cfg.Gate == nil {
		return nil, errors.New("commands: Gate is required")
	}
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	return &Dispatcher{
		registry: cfg.Registry,
		gate:     cfg.Gate,
		tasks:    cfg.Tasks,
		log:      log,
	}, nil
}

// ─── Result ───────────────────────────────────────────────────────────────────

// Result is the normalised response shape returned on successful invocation.
// Kind is always set; other fields are populated by the matching run kind.
// T11 marshals this directly to JSON.
type Result struct {
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
	URL     string `json:"url,omitempty"`
	TaskID  string `json:"taskId,omitempty"`
	Output  string `json:"output,omitempty"`
	Exit    int    `json:"exit,omitempty"`
}

// ─── resolvedCmd is the internal flat representation after registry lookup ────

type resolvedCmd struct {
	pluginName string
	id         string
	kind       string
	message    string
	rawURL     string
	method     string
	taskID     string
}

// ─── Invoke ───────────────────────────────────────────────────────────────────

// Invoke looks up commandID in the named plugin's contributions, checks
// capabilities, and executes the declared run. args is a freeform map
// the caller may populate with runtime parameters (currently unused by M1
// run kinds — reserved for M2 when webview invocations pass data).
//
// Errors:
//
//	ErrCommandNotFound   — plugin not registered OR commandID absent
//	ErrRunKindUnimpl     — kind is "host" or "openView" (M2/M3 territory)
//	*bridge.PermError    — capability denied (pass through unchanged)
//	other errors         — execution failures
func (d *Dispatcher) Invoke(ctx context.Context, pluginName, commandID string, _ map[string]any) (*Result, error) {
	// ── 1. Resolve the command from the registry ──────────────────────────────
	rc, err := d.resolve(pluginName, commandID)
	if err != nil {
		return nil, err
	}

	// ── 2. Dispatch by kind ───────────────────────────────────────────────────
	switch rc.kind {
	case "notify":
		return d.runNotify(rc)
	case "openUrl":
		return d.runOpenURL(rc)
	case "exec":
		return d.runExec(ctx, rc)
	case "runTask":
		return d.runTask(ctx, rc)
	case "host", "openView":
		return nil, fmt.Errorf("%w: kind=%q requires M2/M3", ErrRunKindUnimpl, rc.kind)
	default:
		return nil, fmt.Errorf("%w: unknown kind=%q", ErrRunKindUnimpl, rc.kind)
	}
}

// resolve looks up the command in the registry and returns a resolvedCmd.
// Returns ErrCommandNotFound if the plugin is not registered or the command id
// is absent. Returns ErrMissingRunSpec if the command has no Run field.
func (d *Dispatcher) resolve(pluginName, commandID string) (*resolvedCmd, error) {
	flat := d.registry.Flatten()

	for i := range flat.Commands {
		c := &flat.Commands[i]
		if c.PluginName != pluginName || c.ID != commandID {
			continue
		}
		// Found the command.
		if c.Run == nil {
			return nil, ErrMissingRunSpec
		}
		return &resolvedCmd{
			pluginName: c.PluginName,
			id:         c.ID,
			kind:       c.Run.Kind,
			message:    c.Run.Message,
			rawURL:     c.Run.URL,
			method:     c.Run.Method,
			taskID:     c.Run.TaskID,
		}, nil
	}
	return nil, ErrCommandNotFound
}

// ─── Run kind handlers ────────────────────────────────────────────────────────

// runNotify handles kind=notify. No capability check required.
// Returns Result{Kind:"notify", Message:...}.
func (d *Dispatcher) runNotify(rc *resolvedCmd) (*Result, error) {
	return &Result{Kind: "notify", Message: rc.message}, nil
}

// runOpenURL handles kind=openUrl. No capability check required;
// the client picks the scheme handler (url_launcher, etc.).
// Returns an error if the URL fails to parse.
func (d *Dispatcher) runOpenURL(rc *resolvedCmd) (*Result, error) {
	_, err := url.ParseRequestURI(rc.rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid openUrl: %w", err)
	}
	return &Result{Kind: "openUrl", URL: rc.rawURL}, nil
}

// runExec handles kind=exec.
// Requires exec capability via gate.Check.
// Spawns "sh -c <method>" with a 10-second hard timeout.
// Captures combined stdout+stderr; truncates at 16 KiB.
// Non-zero exit code is NOT an error — it is reflected in Result.Exit.
func (d *Dispatcher) runExec(ctx context.Context, rc *resolvedCmd) (*Result, error) {
	if rc.method == "" {
		return nil, ErrMissingRunSpec
	}

	// Capability check — gate audits this internally.
	if err := d.gate.Check(ctx, rc.pluginName, bridge.Need{Cap: "exec", Target: rc.method}); err != nil {
		return nil, err
	}

	// Hard 10-second timeout, independent of the caller's context lifetime but
	// still honouring cancellation of the caller's context.
	execCtx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	//nolint:gosec // method comes from plugin manifest, capability-checked above
	cmd := exec.CommandContext(execCtx, "sh", "-c", rc.method)
	setProcAttr(cmd)

	// Override the default cancel function so that when execCtx fires we kill
	// the entire process group (not just the sh process). This prevents zombie
	// children when the shell spawns sub-processes (e.g. "sleep 30").
	cmd.Cancel = func() error {
		killProcGroup(cmd)
		return nil
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()

	// Determine exit code from ProcessState when available.
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	// On timeout/cancellation: report the cause.
	// The process group is already killed by cmd.Cancel above.
	if execCtx.Err() != nil {
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			// Distinguish internal 10s timeout from caller-set deadline.
			if ctx.Err() == nil {
				// Only the internal 10s fired; caller ctx is still alive.
				return nil, fmt.Errorf("exec timed out after 10s")
			}
			// Both fired — report caller ctx.
			return nil, fmt.Errorf("exec cancelled: %w", ctx.Err())
		}
		// Caller cancelled before internal timeout.
		return nil, fmt.Errorf("exec cancelled: %w", ctx.Err())
	}

	// Non-zero exit is NOT an error — exec ran successfully.
	// cmd.Run() wraps *exec.ExitError for non-zero exits; we surface it via
	// Result.Exit instead of returning an error.
	var exitErr *exec.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		// A real spawn failure (binary not found, permission denied on /bin/sh…).
		return nil, fmt.Errorf("exec failed: %w", runErr)
	}

	return &Result{Kind: "exec", Output: truncate(buf.String()), Exit: exitCode}, nil
}

// runTask handles kind=runTask.
// Requires exec capability (tasks run shell).
// Returns ErrRunKindUnimpl if no TaskRunner is configured.
func (d *Dispatcher) runTask(ctx context.Context, rc *resolvedCmd) (*Result, error) {
	if d.tasks == nil {
		return nil, fmt.Errorf("%w: task runner not configured", ErrRunKindUnimpl)
	}
	if rc.taskID == "" {
		return nil, ErrMissingRunSpec
	}

	// Tasks run shell — require exec capability.
	if err := d.gate.Check(ctx, rc.pluginName, bridge.Need{Cap: "exec", Target: "task:" + rc.taskID}); err != nil {
		return nil, err
	}

	var outBuf bytes.Buffer
	exitCode, err := d.tasks.Run(ctx, rc.taskID, &outBuf, &outBuf)
	if err != nil {
		return nil, fmt.Errorf("runTask %q: %w", rc.taskID, err)
	}

	return &Result{
		Kind:   "runTask",
		TaskID: rc.taskID,
		Output: truncate(outBuf.String()),
		Exit:   exitCode,
	}, nil
}

// truncate caps s at maxOutput bytes, appending a suffix when truncated.
// The suffix is "… [truncated]" (3 bytes for the ellipsis + 13 ASCII chars = 16 total).
func truncate(s string) string {
	if len(s) <= maxOutput {
		return s
	}
	return s[:maxOutput] + "… [truncated]"
}
