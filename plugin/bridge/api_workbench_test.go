package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// ─────────────────────────────────────────────
// Fake sinks and invokers
// ─────────────────────────────────────────────

type fakeMsgSink struct {
	calls []struct {
		UserID string
		Plugin string
		Opts   ShowMessageOpts
	}
	err error
}

func (f *fakeMsgSink) ShowMessage(userID, plugin string, opts ShowMessageOpts) error {
	f.calls = append(f.calls, struct {
		UserID string
		Plugin string
		Opts   ShowMessageOpts
	}{userID, plugin, opts})
	return f.err
}

type fakeViewSink struct {
	calls []struct {
		UserID string
		Plugin string
		View   string
	}
	err error
}

func (f *fakeViewSink) OpenView(userID, plugin, viewID string) error {
	f.calls = append(f.calls, struct {
		UserID string
		Plugin string
		View   string
	}{userID, plugin, viewID})
	return f.err
}

type fakeStatusSink struct {
	calls []struct {
		UserID string
		Plugin string
		Items  []StatusBarOverride
	}
	err error
}

func (f *fakeStatusSink) UpdateStatusBar(userID, plugin string, items []StatusBarOverride) error {
	f.calls = append(f.calls, struct {
		UserID string
		Plugin string
		Items  []StatusBarOverride
	}{userID, plugin, items})
	return f.err
}

type fakeCmdInvoker struct {
	lastPlugin string
	lastCmd    string
	lastArgs   map[string]any
	ret        any
	err        error
}

func (f *fakeCmdInvoker) Invoke(_ context.Context, plugin, commandID string, args map[string]any) (any, error) {
	f.lastPlugin = plugin
	f.lastCmd = commandID
	f.lastArgs = args
	return f.ret, f.err
}

type fakeTheme struct {
	id string
}

func (f *fakeTheme) ThemeID() string { return f.id }

// ─────────────────────────────────────────────
// Helper: marshal args as JSON array for Dispatch calls
// ─────────────────────────────────────────────

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// dispatchWorkbench is a test helper that calls Dispatch on a WorkbenchAPI.
func dispatchWorkbench(t *testing.T, w *WorkbenchAPI, method string, args json.RawMessage) (any, error) {
	t.Helper()
	return w.Dispatch(context.Background(), "test-plugin", method, args, "", nil)
}

// ─────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────

// TestWorkbench_ShowMessageFiresSink verifies that a bare showMessage("hi")
// call (no opts) hits the sink with Kind="info" and the plugin name set.
func TestWorkbench_ShowMessageFiresSink(t *testing.T) {
	sink := &fakeMsgSink{}
	w := NewWorkbenchAPI(WorkbenchConfig{Message: sink})

	args := mustMarshal([]any{"hi"})
	result, err := dispatchWorkbench(t, w, "showMessage", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if len(sink.calls) != 1 {
		t.Fatalf("expected 1 sink call, got %d", len(sink.calls))
	}
	call := sink.calls[0]
	if call.Plugin != "test-plugin" {
		t.Errorf("plugin: want %q, got %q", "test-plugin", call.Plugin)
	}
	if call.Opts.Text != "hi" {
		t.Errorf("text: want %q, got %q", "hi", call.Opts.Text)
	}
	if call.Opts.Kind != "info" {
		t.Errorf("kind: want %q, got %q", "info", call.Opts.Kind)
	}
	if call.Opts.TTLMs != 0 {
		t.Errorf("ttlMs: want 0, got %d", call.Opts.TTLMs)
	}
}

// TestWorkbench_ShowMessageWithOpts verifies that opts override the defaults.
func TestWorkbench_ShowMessageWithOpts(t *testing.T) {
	sink := &fakeMsgSink{}
	w := NewWorkbenchAPI(WorkbenchConfig{Message: sink})

	opts := ShowMessageOpts{Kind: "warn", TTLMs: 3000}
	args := mustMarshal([]any{"bye", opts})
	_, err := dispatchWorkbench(t, w, "showMessage", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.calls) != 1 {
		t.Fatalf("expected 1 sink call, got %d", len(sink.calls))
	}
	call := sink.calls[0]
	if call.Opts.Text != "bye" {
		t.Errorf("text: want %q, got %q", "bye", call.Opts.Text)
	}
	if call.Opts.Kind != "warn" {
		t.Errorf("kind: want %q, got %q", "warn", call.Opts.Kind)
	}
	if call.Opts.TTLMs != 3000 {
		t.Errorf("ttlMs: want 3000, got %d", call.Opts.TTLMs)
	}
}

// TestWorkbench_ShowMessageMalformedArgs verifies that a non-array arg triggers EINVAL.
func TestWorkbench_ShowMessageMalformedArgs(t *testing.T) {
	sink := &fakeMsgSink{}
	w := NewWorkbenchAPI(WorkbenchConfig{Message: sink})

	// Object instead of array.
	_, err := dispatchWorkbench(t, w, "showMessage", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("code: want EINVAL, got %q", we.Code)
	}
}

// TestWorkbench_ShowMessageSinkNil_ReturnsEUNAVAIL verifies that a nil msg sink
// causes showMessage to return EUNAVAIL.
func TestWorkbench_ShowMessageSinkNil_ReturnsEUNAVAIL(t *testing.T) {
	w := NewWorkbenchAPI(WorkbenchConfig{}) // no sinks

	args := mustMarshal([]any{"hello"})
	_, err := dispatchWorkbench(t, w, "showMessage", args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Errorf("code: want EUNAVAIL, got %q", we.Code)
	}
}

// TestWorkbench_OpenView_FiresSink verifies that openView passes the viewID to the sink.
func TestWorkbench_OpenView_FiresSink(t *testing.T) {
	sink := &fakeViewSink{}
	w := NewWorkbenchAPI(WorkbenchConfig{OpenView: sink})

	args := mustMarshal([]any{"kanban.board"})
	result, err := dispatchWorkbench(t, w, "openView", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if len(sink.calls) != 1 {
		t.Fatalf("expected 1 sink call, got %d", len(sink.calls))
	}
	if sink.calls[0].View != "kanban.board" {
		t.Errorf("viewID: want %q, got %q", "kanban.board", sink.calls[0].View)
	}
	if sink.calls[0].Plugin != "test-plugin" {
		t.Errorf("plugin: want %q, got %q", "test-plugin", sink.calls[0].Plugin)
	}
}

// TestWorkbench_OpenView_MissingArg verifies that missing viewID triggers EINVAL.
func TestWorkbench_OpenView_MissingArg(t *testing.T) {
	sink := &fakeViewSink{}
	w := NewWorkbenchAPI(WorkbenchConfig{OpenView: sink})

	// Empty args array — no viewID.
	_, err := dispatchWorkbench(t, w, "openView", json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("code: want EINVAL, got %q", we.Code)
	}
}

// TestWorkbench_UpdateStatusBar_FiresSink verifies that updateStatusBar forwards
// a full item list to the sink.
func TestWorkbench_UpdateStatusBar_FiresSink(t *testing.T) {
	sink := &fakeStatusSink{}
	w := NewWorkbenchAPI(WorkbenchConfig{StatusBar: sink})

	items := []StatusBarOverride{
		{ID: "ext.status", Text: "Ready", Tooltip: "All good", Command: "ext.refresh"},
	}
	// args = [items] — single element array where element is the items array.
	args := mustMarshal([]any{items})
	result, err := dispatchWorkbench(t, w, "updateStatusBar", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if len(sink.calls) != 1 {
		t.Fatalf("expected 1 sink call, got %d", len(sink.calls))
	}
	got := sink.calls[0].Items
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].ID != "ext.status" {
		t.Errorf("id: want %q, got %q", "ext.status", got[0].ID)
	}
	if got[0].Text != "Ready" {
		t.Errorf("text: want %q, got %q", "Ready", got[0].Text)
	}
	if got[0].Tooltip != "All good" {
		t.Errorf("tooltip: want %q, got %q", "All good", got[0].Tooltip)
	}
	if got[0].Command != "ext.refresh" {
		t.Errorf("command: want %q, got %q", "ext.refresh", got[0].Command)
	}
}

// TestWorkbench_UpdateStatusBar_InvalidItemShape verifies that an item missing
// the required "id" field returns EINVAL.
func TestWorkbench_UpdateStatusBar_InvalidItemShape(t *testing.T) {
	sink := &fakeStatusSink{}
	w := NewWorkbenchAPI(WorkbenchConfig{StatusBar: sink})

	// items array where first element has no "id" field.
	args := json.RawMessage(`[[{"text":"no id here"}]]`)
	_, err := dispatchWorkbench(t, w, "updateStatusBar", args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("code: want EINVAL, got %q", we.Code)
	}
}

// TestWorkbench_RunCommand_DelegatesToInvoker verifies that runCommand passes
// the command ID and args to the CommandInvoker and returns its result as-is.
func TestWorkbench_RunCommand_DelegatesToInvoker(t *testing.T) {
	invoker := &fakeCmdInvoker{
		ret: map[string]any{"kind": "notify", "message": "x"},
	}
	w := NewWorkbenchAPI(WorkbenchConfig{Command: invoker})

	args := mustMarshal([]any{"ext.doSomething", map[string]any{"key": "val"}})
	result, err := dispatchWorkbench(t, w, "runCommand", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Result must be the invoker's return value as-is.
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if m["kind"] != "notify" {
		t.Errorf("kind: want %q, got %v", "notify", m["kind"])
	}
	if invoker.lastCmd != "ext.doSomething" {
		t.Errorf("commandID: want %q, got %q", "ext.doSomething", invoker.lastCmd)
	}
	if invoker.lastPlugin != "test-plugin" {
		t.Errorf("plugin: want %q, got %q", "test-plugin", invoker.lastPlugin)
	}
}

// TestWorkbench_RunCommand_InvokerNil_ReturnsEUNAVAIL verifies that a nil
// CommandInvoker causes runCommand to return EUNAVAIL.
func TestWorkbench_RunCommand_InvokerNil_ReturnsEUNAVAIL(t *testing.T) {
	w := NewWorkbenchAPI(WorkbenchConfig{}) // no command invoker

	args := mustMarshal([]any{"ext.doSomething"})
	_, err := dispatchWorkbench(t, w, "runCommand", args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Errorf("code: want EUNAVAIL, got %q", we.Code)
	}
}

// TestWorkbench_RunCommand_PassesErrorUp verifies that an invoker error is
// passed through without wrapping in EUNAVAIL (T7's classify logic handles it).
func TestWorkbench_RunCommand_PassesErrorUp(t *testing.T) {
	invokerErr := errors.New("command not found")
	invoker := &fakeCmdInvoker{err: invokerErr}
	w := NewWorkbenchAPI(WorkbenchConfig{Command: invoker})

	args := mustMarshal([]any{"ext.missing"})
	_, err := dispatchWorkbench(t, w, "runCommand", args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Must NOT be a WireError with EUNAVAIL (the invoker's error propagates as-is).
	var we *WireError
	if errors.As(err, &we) && we.Code == "EUNAVAIL" {
		t.Errorf("expected raw invoker error to pass through, got EUNAVAIL")
	}
	// The error message should contain the original error.
	if !containsSubstr(err.Error(), "command not found") {
		t.Errorf("error %q should mention %q", err.Error(), "command not found")
	}
}

// TestWorkbench_Theme_ReturnsFromThemeSource verifies that theme() returns
// the ThemeSource's reported ID.
func TestWorkbench_Theme_ReturnsFromThemeSource(t *testing.T) {
	theme := &fakeTheme{id: "dark"}
	w := NewWorkbenchAPI(WorkbenchConfig{Theme: theme})

	result, err := dispatchWorkbench(t, w, "theme", json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T: %v", result, result)
	}
	if m["id"] != "dark" {
		t.Errorf("id: want %q, got %q", "dark", m["id"])
	}
}

// TestWorkbench_Theme_NilSource_ReturnsDefaultDark verifies that a nil
// ThemeSource returns the documented default "dark".
func TestWorkbench_Theme_NilSource_ReturnsDefaultDark(t *testing.T) {
	w := NewWorkbenchAPI(WorkbenchConfig{}) // no theme source

	result, err := dispatchWorkbench(t, w, "theme", json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T: %v", result, result)
	}
	if m["id"] != "dark" {
		t.Errorf("id: want %q (default), got %q", "dark", m["id"])
	}
}

// TestWorkbench_Confirm_ReturnsEUNAVAIL verifies that confirm is stubbed out
// with a clear M6 message.
func TestWorkbench_Confirm_ReturnsEUNAVAIL(t *testing.T) {
	w := NewWorkbenchAPI(WorkbenchConfig{})

	_, err := dispatchWorkbench(t, w, "confirm", json.RawMessage(`["Are you sure?"]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Errorf("code: want EUNAVAIL, got %q", we.Code)
	}
	if !containsSubstr(we.Message, "M6") {
		t.Errorf("message %q should mention M6", we.Message)
	}
}

// TestWorkbench_Prompt_ReturnsEUNAVAIL verifies that prompt is stubbed as M6.
func TestWorkbench_Prompt_ReturnsEUNAVAIL(t *testing.T) {
	w := NewWorkbenchAPI(WorkbenchConfig{})

	_, err := dispatchWorkbench(t, w, "prompt", json.RawMessage(`["Enter value:"]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Errorf("code: want EUNAVAIL, got %q", we.Code)
	}
	if !containsSubstr(we.Message, "M6") {
		t.Errorf("message %q should mention M6", we.Message)
	}
}

// TestWorkbench_OnThemeChange_ReturnsEUNAVAIL verifies that onThemeChange is
// stubbed with a clear M3 message (should be events.subscribe in M3).
func TestWorkbench_OnThemeChange_ReturnsEUNAVAIL(t *testing.T) {
	w := NewWorkbenchAPI(WorkbenchConfig{})

	_, err := dispatchWorkbench(t, w, "onThemeChange", json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Errorf("code: want EUNAVAIL, got %q", we.Code)
	}
}

// TestWorkbench_UnknownMethod_ReturnsEUNAVAIL verifies that unrecognised method
// names are rejected with EUNAVAIL.
func TestWorkbench_UnknownMethod_ReturnsEUNAVAIL(t *testing.T) {
	w := NewWorkbenchAPI(WorkbenchConfig{})

	_, err := dispatchWorkbench(t, w, "frobnicate", json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Errorf("code: want EUNAVAIL, got %q", we.Code)
	}
}
