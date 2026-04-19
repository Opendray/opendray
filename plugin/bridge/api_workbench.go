package bridge

// WorkbenchAPI implements opendray.workbench.* server-side.
// No capability required (workbench is pure UX).
//
// Every method is best-effort fire-and-forget from the plugin's
// perspective — the bridge returns as fast as the host can acknowledge,
// and the visible effect (SnackBar, view transition, status-bar repaint)
// propagates to Flutter out-of-band via ShowMessageSink / OpenViewSink /
// StatusBarSink. Those sinks are implemented in gateway (T14/T15 SSE
// stream) — this file accepts them as interfaces so tests can inject
// fakes.

import (
	"context"
	"encoding/json"
	"fmt"
)

// ─────────────────────────────────────────────
// Sink and source interfaces
// ─────────────────────────────────────────────

// ShowMessageSink receives a request to surface a transient message to
// the user. In production this is the gateway's SSE hub, scoped by
// userID from the bridge handshake (currently always "default" in
// single-user OpenDray; future-proof).
type ShowMessageSink interface {
	ShowMessage(userID, plugin string, opts ShowMessageOpts) error
}

// ShowMessageOpts carries the parameters for a showMessage call.
type ShowMessageOpts struct {
	Text  string `json:"text"`
	Kind  string `json:"kind,omitempty"`  // "info" (default) | "warn" | "error"
	TTLMs int    `json:"ttlMs,omitempty"` // 0 = host default
}

// OpenViewSink requests the Flutter workbench focus a given view id.
type OpenViewSink interface {
	OpenView(userID, plugin, viewID string) error
}

// StatusBarSink receives a plugin's status-bar override. The plugin
// supplies a full replacement for the items it contributes (matched
// by id); items it doesn't own are untouched on the client side.
type StatusBarSink interface {
	UpdateStatusBar(userID, plugin string, items []StatusBarOverride) error
}

// StatusBarOverride is a single status-bar item contributed by a plugin.
type StatusBarOverride struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Tooltip string `json:"tooltip,omitempty"`
	Command string `json:"command,omitempty"`
}

// CommandInvoker is the minimum surface runCommand needs from M1's
// dispatcher. *commands.Dispatcher satisfies it at runtime wiring.
type CommandInvoker interface {
	Invoke(ctx context.Context, plugin, commandID string, args map[string]any) (any, error)
}

// ThemeSource lets the host report the active theme (light/dark/etc).
type ThemeSource interface {
	ThemeID() string
}

// ─────────────────────────────────────────────
// Config + constructor
// ─────────────────────────────────────────────

// WorkbenchConfig collects optional dependencies for WorkbenchAPI.
type WorkbenchConfig struct {
	Message   ShowMessageSink
	OpenView  OpenViewSink
	StatusBar StatusBarSink
	Command   CommandInvoker
	Theme     ThemeSource
}

// WorkbenchAPI implements opendray.workbench.* server-side.
// All sinks are optional; missing sinks return EUNAVAIL for their
// corresponding methods. This lets main.go wire namespaces incrementally.
type WorkbenchAPI struct {
	msg       ShowMessageSink
	openView  OpenViewSink
	statusBar StatusBarSink
	cmd       CommandInvoker
	theme     ThemeSource
}

// NewWorkbenchAPI returns a workbench dispatcher. Every sink is
// optional — missing sinks cause their matching methods to return
// EUNAVAIL.
func NewWorkbenchAPI(cfg WorkbenchConfig) *WorkbenchAPI {
	return &WorkbenchAPI{
		msg:       cfg.Message,
		openView:  cfg.OpenView,
		statusBar: cfg.StatusBar,
		cmd:       cfg.Command,
		theme:     cfg.Theme,
	}
}

// ─────────────────────────────────────────────
// userID source
// ─────────────────────────────────────────────

// defaultUserID is the hard-coded single-user ID used for all sink calls.
// TODO(M3): thread userID from JWT via Conn.
const defaultUserID = "default"

// ─────────────────────────────────────────────
// Dispatch
// ─────────────────────────────────────────────

// Dispatch routes bridge requests. Conn is unused (workbench is not
// stream-capable), accepted to match the Namespace interface shape.
//
// Methods implemented in M2:
//
//	showMessage(text, opts?)        — fires sink; returns null
//	openView(viewId)                — fires sink; returns null
//	updateStatusBar(items)          — fires sink; returns null
//	runCommand(id, args?)           — pass through to CommandInvoker
//	theme()                         — returns { id: string }
//
// Methods returning EUNAVAIL in M2:
//
//	confirm(text, opts?)            — SnackBar-based confirm is M6 polish
//	prompt(text, opts?)             — M6
//	onThemeChange()                 — should be an events.subscribe; M3
func (w *WorkbenchAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, conn *Conn) (any, error) {
	switch method {
	case "showMessage":
		return w.handleShowMessage(plugin, args)
	case "openView":
		return w.handleOpenView(plugin, args)
	case "updateStatusBar":
		return w.handleUpdateStatusBar(plugin, args)
	case "runCommand":
		return w.handleRunCommand(ctx, plugin, args)
	case "theme":
		return w.handleTheme()
	case "confirm":
		we := &WireError{Code: "EUNAVAIL", Message: "workbench.confirm: SnackBar-based confirm is planned for M6"}
		return nil, fmt.Errorf("workbench confirm: %w", we)
	case "prompt":
		we := &WireError{Code: "EUNAVAIL", Message: "workbench.prompt: interactive prompt is planned for M6"}
		return nil, fmt.Errorf("workbench prompt: %w", we)
	case "onThemeChange":
		we := &WireError{Code: "EUNAVAIL", Message: "workbench.onThemeChange: use events.subscribe in M3"}
		return nil, fmt.Errorf("workbench onThemeChange: %w", we)
	default:
		we := &WireError{Code: "EUNAVAIL", Message: fmt.Sprintf("workbench.%s: method not available", method)}
		return nil, fmt.Errorf("workbench %s: %w", method, we)
	}
}

// ─────────────────────────────────────────────
// showMessage
// ─────────────────────────────────────────────

// handleShowMessage implements: showMessage(text string, opts *ShowMessageOpts?) → null
//
// args is a JSON array: [text] or [text, opts].
// Malformed args → EINVAL. Missing sink → EUNAVAIL.
func (w *WorkbenchAPI) handleShowMessage(plugin string, args json.RawMessage) (any, error) {
	if w.msg == nil {
		we := &WireError{Code: "EUNAVAIL", Message: "workbench.showMessage: ShowMessageSink not configured"}
		return nil, fmt.Errorf("workbench showMessage: %w", we)
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: "showMessage: args must be [text] or [text, opts]"}
		return nil, fmt.Errorf("workbench showMessage: %w", we)
	}

	var text string
	if err := json.Unmarshal(raw[0], &text); err != nil {
		we := &WireError{Code: "EINVAL", Message: "showMessage: text must be a string"}
		return nil, fmt.Errorf("workbench showMessage: %w", we)
	}

	opts := ShowMessageOpts{
		Text: text,
		Kind: "info", // default
	}

	if len(raw) >= 2 {
		// Second element is optional opts object. We decode only the fields we care about.
		var provided ShowMessageOpts
		if err := json.Unmarshal(raw[1], &provided); err != nil {
			we := &WireError{Code: "EINVAL", Message: "showMessage: opts must be an object"}
			return nil, fmt.Errorf("workbench showMessage: %w", we)
		}
		if provided.Kind != "" {
			opts.Kind = provided.Kind
		}
		opts.TTLMs = provided.TTLMs
	}

	if err := w.msg.ShowMessage(defaultUserID, plugin, opts); err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("workbench showMessage: %w", we)
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// openView
// ─────────────────────────────────────────────

// handleOpenView implements: openView(viewID string) → null
//
// args is a JSON array: [viewID].
// Missing viewID → EINVAL. Missing sink → EUNAVAIL.
func (w *WorkbenchAPI) handleOpenView(plugin string, args json.RawMessage) (any, error) {
	if w.openView == nil {
		we := &WireError{Code: "EUNAVAIL", Message: "workbench.openView: OpenViewSink not configured"}
		return nil, fmt.Errorf("workbench openView: %w", we)
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: "openView: args must be [viewID]"}
		return nil, fmt.Errorf("workbench openView: %w", we)
	}

	var viewID string
	if err := json.Unmarshal(raw[0], &viewID); err != nil || viewID == "" {
		we := &WireError{Code: "EINVAL", Message: "openView: viewID must be a non-empty string"}
		return nil, fmt.Errorf("workbench openView: %w", we)
	}

	if err := w.openView.OpenView(defaultUserID, plugin, viewID); err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("workbench openView: %w", we)
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// updateStatusBar
// ─────────────────────────────────────────────

// handleUpdateStatusBar implements: updateStatusBar(items []StatusBarOverride) → null
//
// args is a JSON array: [items] where items is an array of StatusBarOverride.
// Items missing "id" field → EINVAL. Missing sink → EUNAVAIL.
func (w *WorkbenchAPI) handleUpdateStatusBar(plugin string, args json.RawMessage) (any, error) {
	if w.statusBar == nil {
		we := &WireError{Code: "EUNAVAIL", Message: "workbench.updateStatusBar: StatusBarSink not configured"}
		return nil, fmt.Errorf("workbench updateStatusBar: %w", we)
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: "updateStatusBar: args must be [items]"}
		return nil, fmt.Errorf("workbench updateStatusBar: %w", we)
	}

	var items []StatusBarOverride
	if err := json.Unmarshal(raw[0], &items); err != nil {
		we := &WireError{Code: "EINVAL", Message: "updateStatusBar: items must be an array of StatusBarOverride"}
		return nil, fmt.Errorf("workbench updateStatusBar: %w", we)
	}

	// Validate that every item has a non-empty id.
	for i, item := range items {
		if item.ID == "" {
			we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("updateStatusBar: item[%d]: id is required", i)}
			return nil, fmt.Errorf("workbench updateStatusBar: %w", we)
		}
	}

	if err := w.statusBar.UpdateStatusBar(defaultUserID, plugin, items); err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("workbench updateStatusBar: %w", we)
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// runCommand
// ─────────────────────────────────────────────

// handleRunCommand implements: runCommand(id string, args? map[string]any) → any
//
// args is a JSON array: [commandID] or [commandID, argsMap].
// Missing invoker → EUNAVAIL. Invoker errors are passed through as-is
// (not wrapped in EUNAVAIL) so T7's classify logic can map them properly.
func (w *WorkbenchAPI) handleRunCommand(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	if w.cmd == nil {
		we := &WireError{Code: "EUNAVAIL", Message: "workbench.runCommand: CommandInvoker not configured"}
		return nil, fmt.Errorf("workbench runCommand: %w", we)
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: "runCommand: args must be [commandID] or [commandID, argsMap]"}
		return nil, fmt.Errorf("workbench runCommand: %w", we)
	}

	var commandID string
	if err := json.Unmarshal(raw[0], &commandID); err != nil || commandID == "" {
		we := &WireError{Code: "EINVAL", Message: "runCommand: commandID must be a non-empty string"}
		return nil, fmt.Errorf("workbench runCommand: %w", we)
	}

	var cmdArgs map[string]any
	if len(raw) >= 2 {
		if err := json.Unmarshal(raw[1], &cmdArgs); err != nil {
			we := &WireError{Code: "EINVAL", Message: "runCommand: args must be an object"}
			return nil, fmt.Errorf("workbench runCommand: %w", we)
		}
	}

	// Pass the error through as-is — do NOT wrap in WireError or EUNAVAIL.
	// T7's dispatchInvoke classify logic maps errors to the appropriate envelope.
	return w.cmd.Invoke(ctx, plugin, commandID, cmdArgs)
}

// ─────────────────────────────────────────────
// theme
// ─────────────────────────────────────────────

// handleTheme implements: theme() → { id: string }
//
// Returns the active theme ID from ThemeSource. If ThemeSource is nil,
// the documented default "dark" is returned.
func (w *WorkbenchAPI) handleTheme() (any, error) {
	id := "dark" // documented default when ThemeSource is nil
	if w.theme != nil {
		id = w.theme.ThemeID()
	}
	return map[string]string{"id": id}, nil
}
