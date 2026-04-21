package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// LLMEndpoint is the metadata shape exposed to plugins via
// opendray.llm.list. This is a **metadata-only** view — API keys
// and bearer tokens stay in the kernel and never cross the bridge.
// A plugin that needs to make an outbound call should either:
//
//  1. Let the user paste their own key into its configSchema (fine
//     for plugin-specific secrets), or
//  2. Use a future opendray.llm.proxy method that has the gateway
//     attach credentials server-side (not implemented in this
//     phase — tracked for Phase 2.2).
type LLMEndpoint struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DisplayName  string `json:"displayName,omitempty"`
	ProviderType string `json:"providerType"`
	BaseURL      string `json:"baseUrl"`
	Description  string `json:"description,omitempty"`
	Enabled      bool   `json:"enabled"`
}

// LLMSource is the minimum surface LLMAPI needs from kernel/store.
// Implementations return every row (enabled + disabled); the bridge
// filter optionally trims to enabled-only. Declared locally so this
// package doesn't import kernel/store.
type LLMSource interface {
	ListLLMEndpoints(ctx context.Context) ([]LLMEndpoint, error)
}

// LLMAPI implements opendray.llm.* over the bridge.
//
// Only method today: list.
//
//	list(enabledOnly bool?) → []LLMEndpoint
//
// Requires `"permissions": {"llm": true}` in the caller's manifest.
// Gate.Check runs before the DB query so an unauthorised caller
// can't learn how many endpoints exist via timing.
type LLMAPI struct {
	src  LLMSource
	gate *Gate
	// now is injected for tests; production uses time.Now.
	now func() time.Time
}

// LLMConfig wires an LLMAPI's dependencies.
type LLMConfig struct {
	Source LLMSource
	Gate   *Gate
	// Now is optional — defaults to time.Now. Tests inject a stable
	// clock; production leaves it nil.
	Now func() time.Time
}

// NewLLMAPI constructs an LLMAPI. The src argument must be non-nil
// in production; nil is permitted in tests that don't exercise list.
func NewLLMAPI(cfg LLMConfig) *LLMAPI {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &LLMAPI{src: cfg.Source, gate: cfg.Gate, now: now}
}

// Dispatch routes an inbound bridge envelope to the matching method.
// Signature matches [Namespace] — envID and conn are ignored (list
// is not stream-capable).
func (a *LLMAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	_ = envID
	_ = conn
	if err := a.gate.Check(ctx, plugin, Need{Cap: "llm"}); err != nil {
		return nil, err
	}
	switch method {
	case "list":
		return a.handleList(ctx, args)
	default:
		we := &WireError{Code: "EUNAVAIL", Message: fmt.Sprintf("llm: method %q not available", method)}
		return nil, fmt.Errorf("llm %s: %w", method, we)
	}
}

// listArgs is the optional { "enabledOnly": bool } payload.
// Missing / null / unspecified keys default to {enabledOnly: true}
// so a naïve caller doesn't accidentally pick up a disabled endpoint
// the user hid deliberately.
type listArgs struct {
	EnabledOnly *bool `json:"enabledOnly,omitempty"`
}

func (a *LLMAPI) handleList(ctx context.Context, raw json.RawMessage) (any, error) {
	var args listArgs
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, &WireError{Code: "EINVAL", Message: "llm.list: invalid args"}
		}
	}
	enabledOnly := true
	if args.EnabledOnly != nil {
		enabledOnly = *args.EnabledOnly
	}
	if a.src == nil {
		return []LLMEndpoint{}, nil
	}
	all, err := a.src.ListLLMEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("llm.list: %w", err)
	}
	if !enabledOnly {
		// Never return nil — plugins prefer [] over null when no
		// endpoints are configured.
		if all == nil {
			return []LLMEndpoint{}, nil
		}
		return all, nil
	}
	out := make([]LLMEndpoint, 0, len(all))
	for _, e := range all {
		if e.Enabled {
			out = append(out, e)
		}
	}
	return out, nil
}
