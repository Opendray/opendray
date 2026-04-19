package commands_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/commands"
	"github.com/opendray/opendray/plugin/contributions"
)

// ─── fakes ────────────────────────────────────────────────────────────────────

type fakeGate struct {
	allow bool
	calls []bridge.Need
}

func (f *fakeGate) Check(_ context.Context, _ string, need bridge.Need) error {
	f.calls = append(f.calls, need)
	if !f.allow {
		return &bridge.PermError{Code: "EPERM", Msg: "denied"}
	}
	return nil
}

type fakeTasks struct {
	out  string
	exit int
	err  error
}

func (f *fakeTasks) Run(_ context.Context, _ string, stdout, _ io.Writer) (int, error) {
	_, _ = io.WriteString(stdout, f.out)
	return f.exit, f.err
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// buildRegistry builds a Registry with a single plugin contributing one command.
func buildRegistry(pluginName string, cmd plugin.CommandV1) *contributions.Registry {
	reg := contributions.NewRegistry()
	reg.Set(pluginName, plugin.ContributesV1{
		Commands: []plugin.CommandV1{cmd},
	})
	return reg
}

func command(id string, run *plugin.CommandRunV1) plugin.CommandV1 {
	return plugin.CommandV1{ID: id, Title: id, Run: run}
}

func run(kind, msg, url, method, taskID string) *plugin.CommandRunV1 {
	return &plugin.CommandRunV1{
		Kind:    kind,
		Message: msg,
		URL:     url,
		Method:  method,
		TaskID:  taskID,
	}
}

func silentLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ─── TestNewDispatcher_RejectsMissingDeps ─────────────────────────────────────

func TestNewDispatcher_RejectsMissingDeps(t *testing.T) {
	gate := &fakeGate{allow: true}
	reg := contributions.NewRegistry()

	tests := []struct {
		name string
		cfg  commands.Config
	}{
		{
			name: "nil registry",
			cfg:  commands.Config{Registry: nil, Gate: gate},
		},
		{
			name: "nil gate",
			cfg:  commands.Config{Registry: reg, Gate: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := commands.NewDispatcher(tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestNewDispatcher_NilLogIsOK(t *testing.T) {
	gate := &fakeGate{allow: true}
	reg := contributions.NewRegistry()
	_, err := commands.NewDispatcher(commands.Config{Registry: reg, Gate: gate, Log: nil})
	if err != nil {
		t.Fatalf("nil log should be accepted, got: %v", err)
	}
}

// ─── TestDispatcher_CommandNotFound ───────────────────────────────────────────

func TestDispatcher_CommandNotFound_UnknownPlugin(t *testing.T) {
	reg := contributions.NewRegistry()
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})
	_, err := d.Invoke(context.Background(), "no-such-plugin", "cmd.foo", nil)
	if !errors.Is(err, commands.ErrCommandNotFound) {
		t.Fatalf("want ErrCommandNotFound, got %v", err)
	}
}

func TestDispatcher_CommandNotFound_UnknownCommandID(t *testing.T) {
	reg := buildRegistry("myplugin", command("cmd.real", run("notify", "hi", "", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})
	_, err := d.Invoke(context.Background(), "myplugin", "cmd.ghost", nil)
	if !errors.Is(err, commands.ErrCommandNotFound) {
		t.Fatalf("want ErrCommandNotFound, got %v", err)
	}
}

// ─── TestDispatcher_MissingRunSpec ────────────────────────────────────────────

func TestDispatcher_MissingRunSpec(t *testing.T) {
	reg := buildRegistry("myplugin", command("cmd.norun", nil))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})
	_, err := d.Invoke(context.Background(), "myplugin", "cmd.norun", nil)
	if !errors.Is(err, commands.ErrMissingRunSpec) {
		t.Fatalf("want ErrMissingRunSpec, got %v", err)
	}
}

// ─── TestDispatcher_NotifyKind ────────────────────────────────────────────────

func TestDispatcher_NotifyKind(t *testing.T) {
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.hello", run("notify", "hello", "", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	res, err := d.Invoke(context.Background(), "myplugin", "cmd.hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != "notify" {
		t.Errorf("want Kind=notify, got %q", res.Kind)
	}
	if res.Message != "hello" {
		t.Errorf("want Message=hello, got %q", res.Message)
	}
	// No gate check for notify
	if len(gate.calls) != 0 {
		t.Errorf("want 0 gate calls, got %d", len(gate.calls))
	}
}

// ─── TestDispatcher_OpenUrlKind ───────────────────────────────────────────────

func TestDispatcher_OpenUrlKind(t *testing.T) {
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.open", run("openUrl", "", "https://opendray.dev", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	res, err := d.Invoke(context.Background(), "myplugin", "cmd.open", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != "openUrl" {
		t.Errorf("want Kind=openUrl, got %q", res.Kind)
	}
	if res.URL != "https://opendray.dev" {
		t.Errorf("want URL=https://opendray.dev, got %q", res.URL)
	}
	// No gate check for openUrl
	if len(gate.calls) != 0 {
		t.Errorf("want 0 gate calls, got %d", len(gate.calls))
	}
}

// ─── TestDispatcher_OpenUrlInvalid ────────────────────────────────────────────

func TestDispatcher_OpenUrlInvalid(t *testing.T) {
	reg := buildRegistry("myplugin", command("cmd.bad", run("openUrl", "", "://not-valid", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})

	_, err := d.Invoke(context.Background(), "myplugin", "cmd.bad", nil)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "openUrl") {
		t.Errorf("error should mention openUrl, got: %v", err)
	}
}

// ─── TestDispatcher_ExecAllowedWithCapability ─────────────────────────────────

func TestDispatcher_ExecAllowedWithCapability(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.exec", run("exec", "", "", "echo hi", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	res, err := d.Invoke(context.Background(), "myplugin", "cmd.exec", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != "exec" {
		t.Errorf("want Kind=exec, got %q", res.Kind)
	}
	if !strings.Contains(res.Output, "hi") {
		t.Errorf("want output containing 'hi', got %q", res.Output)
	}
	if res.Exit != 0 {
		t.Errorf("want Exit=0, got %d", res.Exit)
	}
	// Gate should have been called exactly once with Cap="exec", Target="echo hi"
	if len(gate.calls) != 1 {
		t.Fatalf("want 1 gate call, got %d", len(gate.calls))
	}
	if gate.calls[0].Cap != "exec" {
		t.Errorf("want Cap=exec, got %q", gate.calls[0].Cap)
	}
	if gate.calls[0].Target != "echo hi" {
		t.Errorf("want Target='echo hi', got %q", gate.calls[0].Target)
	}
}

// ─── TestDispatcher_ExecDeniedWithoutCapability ───────────────────────────────

func TestDispatcher_ExecDeniedWithoutCapability(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}
	gate := &fakeGate{allow: false}
	reg := buildRegistry("myplugin", command("cmd.exec", run("exec", "", "", "echo hi", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	_, err := d.Invoke(context.Background(), "myplugin", "cmd.exec", nil)
	var permErr *bridge.PermError
	if !errors.As(err, &permErr) {
		t.Fatalf("want *bridge.PermError, got %T: %v", err, err)
	}
}

// ─── TestDispatcher_ExecTimeout ───────────────────────────────────────────────

func TestDispatcher_ExecTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.sleep", run("exec", "", "", "sleep 30", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err := d.Invoke(ctx, "myplugin", "cmd.sleep", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// Should complete well within the outer test timeout, not hang.
	if elapsed > 15*time.Second {
		t.Errorf("invoke hung too long: %v", elapsed)
	}
}

// ─── TestDispatcher_ExecOutputTruncation ─────────────────────────────────────

func TestDispatcher_ExecOutputTruncation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}
	gate := &fakeGate{allow: true}
	// printf repeats 'x' 20000 times via python to avoid shell portability issues.
	// We use dd from /dev/zero for reliable large output.
	cmd := `dd if=/dev/zero bs=1024 count=20 2>/dev/null | tr '\0' 'x'`
	reg := buildRegistry("myplugin", command("cmd.large", run("exec", "", "", cmd, "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	res, err := d.Invoke(context.Background(), "myplugin", "cmd.large", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const maxLen = 16384 + len("… [truncated]")
	if len(res.Output) > maxLen {
		t.Errorf("output not truncated: len=%d, want<=%d", len(res.Output), maxLen)
	}
	if !strings.HasSuffix(res.Output, "… [truncated]") {
		t.Errorf("expected truncation suffix, got tail: %q", res.Output[max(0, len(res.Output)-30):])
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ─── TestDispatcher_RunTaskAllowed ────────────────────────────────────────────

func TestDispatcher_RunTaskAllowed(t *testing.T) {
	gate := &fakeGate{allow: true}
	tasks := &fakeTasks{out: "done", exit: 0}
	reg := buildRegistry("myplugin", command("cmd.task", run("runTask", "", "", "", "foo")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Tasks:    tasks,
		Log:      silentLog(),
	})

	res, err := d.Invoke(context.Background(), "myplugin", "cmd.task", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != "runTask" {
		t.Errorf("want Kind=runTask, got %q", res.Kind)
	}
	if res.TaskID != "foo" {
		t.Errorf("want TaskID=foo, got %q", res.TaskID)
	}
	if res.Output != "done" {
		t.Errorf("want Output=done, got %q", res.Output)
	}
	if res.Exit != 0 {
		t.Errorf("want Exit=0, got %d", res.Exit)
	}
}

// ─── TestDispatcher_RunTaskNoRunner ───────────────────────────────────────────

func TestDispatcher_RunTaskNoRunner(t *testing.T) {
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.task", run("runTask", "", "", "", "foo")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Tasks:    nil, // no runner
		Log:      silentLog(),
	})

	_, err := d.Invoke(context.Background(), "myplugin", "cmd.task", nil)
	if !errors.Is(err, commands.ErrRunKindUnimpl) {
		t.Fatalf("want ErrRunKindUnimpl, got %v", err)
	}
}

// ─── TestDispatcher_HostKindReturnsEUNAVAIL ──────────────────────────────────

func TestDispatcher_HostKindReturnsEUNAVAIL(t *testing.T) {
	tests := []struct{ kind string }{
		{"host"},
		{"openView"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			reg := buildRegistry("myplugin", command("cmd.h", run(tt.kind, "", "", "", "")))
			d, _ := commands.NewDispatcher(commands.Config{
				Registry: reg,
				Gate:     &fakeGate{allow: true},
				Log:      silentLog(),
			})
			_, err := d.Invoke(context.Background(), "myplugin", "cmd.h", nil)
			if !errors.Is(err, commands.ErrRunKindUnimpl) {
				t.Errorf("want ErrRunKindUnimpl, got %v", err)
			}
		})
	}
}

// ─── TestDispatcher_UnknownKindReturnsEUNAVAIL ───────────────────────────────

func TestDispatcher_UnknownKindReturnsEUNAVAIL(t *testing.T) {
	reg := buildRegistry("myplugin", command("cmd.bogus", run("bogus", "", "", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})
	_, err := d.Invoke(context.Background(), "myplugin", "cmd.bogus", nil)
	if !errors.Is(err, commands.ErrRunKindUnimpl) {
		t.Fatalf("want ErrRunKindUnimpl, got %v", err)
	}
}

// ─── TestDispatcher_ContextCancellation ──────────────────────────────────────

func TestDispatcher_ContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.sleep", run("exec", "", "", "sleep 30", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel almost immediately to simulate caller cancellation mid-run.
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := d.Invoke(ctx, "myplugin", "cmd.sleep", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	// Must not hang — completes well within 5 seconds.
	if elapsed > 5*time.Second {
		t.Errorf("invoke hung: %v", elapsed)
	}
}

// ─── TestDispatcher_ExecMissingMethod ────────────────────────────────────────

func TestDispatcher_ExecMissingMethod(t *testing.T) {
	reg := buildRegistry("myplugin", command("cmd.exec", run("exec", "", "", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})
	_, err := d.Invoke(context.Background(), "myplugin", "cmd.exec", nil)
	if !errors.Is(err, commands.ErrMissingRunSpec) {
		t.Fatalf("want ErrMissingRunSpec, got %v", err)
	}
}

// ─── TestDispatcher_RunTaskMissingTaskID ─────────────────────────────────────

func TestDispatcher_RunTaskMissingTaskID(t *testing.T) {
	gate := &fakeGate{allow: true}
	tasks := &fakeTasks{out: "done", exit: 0}
	// taskID is empty
	reg := buildRegistry("myplugin", command("cmd.task", run("runTask", "", "", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Tasks:    tasks,
		Log:      silentLog(),
	})

	_, err := d.Invoke(context.Background(), "myplugin", "cmd.task", nil)
	if !errors.Is(err, commands.ErrMissingRunSpec) {
		t.Fatalf("want ErrMissingRunSpec, got %v", err)
	}
}

// ─── TestDispatcher_RunTask_GateDenied ───────────────────────────────────────

func TestDispatcher_RunTask_GateDenied(t *testing.T) {
	gate := &fakeGate{allow: false}
	tasks := &fakeTasks{out: "", exit: 0}
	reg := buildRegistry("myplugin", command("cmd.task", run("runTask", "", "", "", "foo")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Tasks:    tasks,
		Log:      silentLog(),
	})

	_, err := d.Invoke(context.Background(), "myplugin", "cmd.task", nil)
	var permErr *bridge.PermError
	if !errors.As(err, &permErr) {
		t.Fatalf("want *bridge.PermError, got %T: %v", err, err)
	}
}

// ─── TestDispatcher_ArgsPassedButUnused ──────────────────────────────────────

func TestDispatcher_ArgsPassedButUnused(t *testing.T) {
	// args map is reserved for M2; dispatcher should accept it without error.
	reg := buildRegistry("myplugin", command("cmd.notify", run("notify", "msg", "", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})
	args := map[string]any{"key": "value", "num": 42}
	res, err := d.Invoke(context.Background(), "myplugin", "cmd.notify", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != "notify" {
		t.Errorf("want Kind=notify, got %q", res.Kind)
	}
}

// ─── TestDispatcher_ExecNonZeroExit ──────────────────────────────────────────

func TestDispatcher_ExecNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.fail", run("exec", "", "", "exit 1", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	res, err := d.Invoke(context.Background(), "myplugin", "cmd.fail", nil)
	// Non-zero exit is NOT an error — exec ran, just returned non-zero.
	if err != nil {
		t.Fatalf("non-zero exit should not be an error, got: %v", err)
	}
	if res.Exit == 0 {
		t.Errorf("expected non-zero exit code, got 0")
	}
	if res.Kind != "exec" {
		t.Errorf("want Kind=exec, got %q", res.Kind)
	}
}

// ─── TestDispatcher_RunTask_OutputTruncation ──────────────────────────────────

func TestDispatcher_RunTask_OutputTruncation(t *testing.T) {
	gate := &fakeGate{allow: true}
	largeOut := strings.Repeat("x", 20000)
	tasks := &fakeTasks{out: largeOut, exit: 0}
	reg := buildRegistry("myplugin", command("cmd.task", run("runTask", "", "", "", "foo")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Tasks:    tasks,
		Log:      silentLog(),
	})

	res, err := d.Invoke(context.Background(), "myplugin", "cmd.task", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const maxLen = 16384 + len("… [truncated]")
	if len(res.Output) > maxLen {
		t.Errorf("output not truncated: len=%d", len(res.Output))
	}
	if !strings.HasSuffix(res.Output, "… [truncated]") {
		t.Errorf("expected truncation suffix")
	}
}

// ─── integration smoke: Invoke is safe on nil args ────────────────────────────

func TestDispatcher_NilArgs(t *testing.T) {
	reg := buildRegistry("myplugin", command("cmd.notify", run("notify", "hi", "", "", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     &fakeGate{allow: true},
		Log:      silentLog(),
	})
	res, err := d.Invoke(context.Background(), "myplugin", "cmd.notify", nil)
	if err != nil || res.Kind != "notify" {
		t.Fatalf("nil args should be fine: err=%v, kind=%q", err, res.Kind)
	}
}

// ─── TestDispatcher_ExecOutputDoesNotContainTruncSuffix_WhenSmall ────────────

func TestDispatcher_ExecOutputDoesNotTruncateSmallOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}
	gate := &fakeGate{allow: true}
	reg := buildRegistry("myplugin", command("cmd.small", run("exec", "", "", "echo small", "")))
	d, _ := commands.NewDispatcher(commands.Config{
		Registry: reg,
		Gate:     gate,
		Log:      silentLog(),
	})

	res, _ := d.Invoke(context.Background(), "myplugin", "cmd.small", nil)
	if strings.HasSuffix(res.Output, "… [truncated]") {
		t.Error("small output should not be truncated")
	}
	if !strings.Contains(res.Output, "small") {
		t.Errorf("expected 'small' in output, got: %q", res.Output)
	}
}

