// Package bridge — Events API namespace + HookBus bridge (M2 T11).
//
// # Design rationale: why EventsAPI holds its own ConsentReader
//
// The existing Gate.Check / evaluate() switch in capabilities.go (T5) handles
// exec / http / fs / session / storage / secret.  The `events` capability is
// deliberately absent from that switch — its grant format is a []string of
// glob patterns, not the simpler bool/string/globs-against-cmdline shapes the
// T5 matchers cover.  Adding events there would require editing capabilities.go
// while T7/T10 have it open in parallel branches.
//
// Instead EventsAPI performs its own consent check:
//
//  1. Load raw perms JSON via ConsentReader.
//  2. Unmarshal into eventsPermsWire (events:[]string).
//  3. Call MatchEventPattern(patterns, requestedName).
//
// The Gate field is retained for future audit integration (T12 will wire audit
// events through the Gate for all caps including events).  A follow-up task can
// migrate to a native Gate.Check once capabilities.go gains an events case.
//
// # Adapter note
//
// The production plugin.HookBus (plugin/hooks.go) does NOT implement
// HookBusLike directly: it exposes SubscribeLocal (typed HookEvent
// callbacks) and Dispatch (typed HookEvent), not the generic
// SubscribeByName / Publish surface required by the bridge events API.
//
// Rather than modifying hooks.go (frozen by T7/T10 parallel work), a thin
// adapter — HookBusAdapter — should be provided by production wiring code
// that wraps *plugin.HookBus. Tests pass a fakeBus directly.
//
// If a future task lands a native SubscribeByName on HookBus, the adapter can
// be deleted and *plugin.HookBus passed as HookBusLike directly.
//
// # Capability rules
//
//   - events.subscribe(name): consent check required (MatchEventPattern).
//   - events.unsubscribe(subId): no cap (always allowed to release own sub).
//   - events.publish(name, data): no cap check; name is always rewritten to
//     plugin.<pluginName>.<name> so plugins cannot masquerade as host events.
//
// # Subscription lifecycle
//
// subscribe → random subId → conn.Subscribe(subId,"events") → returns done ch
//
//	→ SubscribeByName on bus → pump goroutine:
//	     select { done ch closed → busUnsub(); cleanup map; return
//	              event arrives  → conn.WriteEnvelope(chunk) }
//
// unsubscribe → busUnsub() → conn.Unsubscribe(subId) → delete map entry.
//
// conn.Close() closes all done channels via conn.subs.drain(); the pump
// goroutines observe that and call busUnsub before exiting — no goroutine
// leak.
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────
// HookBusLike — minimum surface EventsAPI needs
// ─────────────────────────────────────────────

// HookBusLike is the minimum surface EventsAPI needs from a publish/subscribe
// bus. The production HookBus is wrapped by an adapter in production wiring so
// that this file does not depend on the plugin package (avoiding circular
// imports). Tests pass a fakeBus directly.
//
// Method semantics:
//
//   - SubscribeByName: registers a handler for events whose name matches
//     pattern (glob with * within dotted segments). Returns an unsubscribe
//     func that the caller MUST invoke on cleanup to avoid goroutine / memory
//     leaks.
//   - Publish: broadcasts an event. data must be JSON-serialisable.
type HookBusLike interface {
	SubscribeByName(pattern string, handler func(name string, data any)) (unsubscribe func(), err error)
	Publish(name string, data any)
}

// ─────────────────────────────────────────────
// eventsPermsWire — internal type for parsing events perms JSON
// ─────────────────────────────────────────────

// eventsPermsWire is the subset of PermissionsV1 JSON that EventsAPI cares
// about.  We parse only the `events` field to avoid coupling this file to the
// full permissionsV1Wire type in capabilities.go.
type eventsPermsWire struct {
	Events []string `json:"events,omitempty"`
}

// ─────────────────────────────────────────────
// eventSub — one active subscription
// ─────────────────────────────────────────────

type eventSub struct {
	busUnsub func()
	conn     *Conn
}

// ─────────────────────────────────────────────
// EventsAPI
// ─────────────────────────────────────────────

// EventsAPI adapts HookBusLike to the v1 events.* bridge namespace.
//
// It is safe for concurrent use. All public methods are goroutine-safe.
type EventsAPI struct {
	bus      HookBusLike
	gate     *Gate       // retained for future audit integration
	consents ConsentReader // for events-specific cap check

	subMu sync.Mutex
	subs  map[string]eventSub // subId → active sub
}

// NewEventsAPI constructs an EventsAPI.
//
// gate is retained for audit and future events cap migration; the events
// capability check itself is performed by loading perms via gate's internal
// consent reader. To avoid coupling, we require the caller to supply the same
// ConsentReader that the Gate uses. Pass nil to skip capability gating (tests
// that want unrestricted access).
func NewEventsAPI(bus HookBusLike, gate *Gate) *EventsAPI {
	return &EventsAPI{
		bus:      bus,
		gate:     gate,
		consents: gate.consents,
		subs:     make(map[string]eventSub),
	}
}

// Dispatch routes a bridge method call to the appropriate handler.
//
// Supported methods: subscribe, unsubscribe, publish.
// Unknown methods → *WireError{Code:"EUNAVAIL"}.
// Malformed args  → *WireError{Code:"EINVAL"}.
func (e *EventsAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, conn *Conn) (any, error) {
	switch method {
	case "subscribe":
		return e.subscribe(ctx, plugin, args, conn)
	case "unsubscribe":
		return e.unsubscribe(args)
	case "publish":
		return e.publish(plugin, args)
	default:
		return nil, &WireError{
			Code:    "EUNAVAIL",
			Message: fmt.Sprintf("events.%s: unknown method", method),
		}
	}
}

// ─────────────────────────────────────────────
// subscribe
// ─────────────────────────────────────────────

type subscribeArgs struct {
	Name string `json:"name"`
}

func (e *EventsAPI) subscribe(ctx context.Context, plugin string, raw json.RawMessage, conn *Conn) (any, error) {
	var a subscribeArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &WireError{Code: "EINVAL", Message: fmt.Sprintf("events.subscribe: bad args: %v", err)}
	}
	if a.Name == "" {
		return nil, &WireError{Code: "EINVAL", Message: "events.subscribe: name is required"}
	}

	// Events capability check: load consent and verify via MatchEventPattern.
	if err := e.checkEventsCap(ctx, plugin, a.Name); err != nil {
		return nil, err
	}

	// Generate subscription ID.
	subId := newSubID()

	// Register with conn (returns done channel; closed on revoke or Unsubscribe).
	done, err := conn.Subscribe(subId, "events")
	if err != nil {
		return nil, &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("events.subscribe: register conn: %v", err)}
	}

	// Register with bus.
	busUnsub, err := e.bus.SubscribeByName(a.Name, func(name string, data any) {
		env, buildErr := NewStreamChunk(subId, map[string]any{
			"name": name,
			"data": data,
		})
		if buildErr != nil {
			return
		}
		// Best-effort write; conn.WriteEnvelope returns an error on close.
		_ = conn.WriteEnvelope(env)
	})
	if err != nil {
		// Roll back conn subscription.
		conn.Unsubscribe(subId)
		return nil, &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("events.subscribe: bus subscribe: %v", err)}
	}

	// Store the sub so unsubscribe can clean it up.
	e.subMu.Lock()
	e.subs[subId] = eventSub{busUnsub: busUnsub, conn: conn}
	e.subMu.Unlock()

	// Pump goroutine: exits when done is closed (conn close, hot-revoke, or
	// explicit unsubscribe). Calls busUnsub before returning so the bus handler
	// is cleaned up even when the conn is closed from the outside.
	go func() {
		<-done
		busUnsub()

		// Remove from our map if still present (may have been removed by
		// unsubscribe already — idempotent).
		e.subMu.Lock()
		delete(e.subs, subId)
		e.subMu.Unlock()
	}()

	return map[string]string{"subId": subId}, nil
}

// checkEventsCap loads the plugin's consent and verifies the requested event
// name matches at least one pattern in permissions.events.
func (e *EventsAPI) checkEventsCap(ctx context.Context, plugin, name string) error {
	rawPerms, found, err := e.consents.Load(ctx, plugin)
	if err != nil {
		return fmt.Errorf("events.subscribe: load consent for %q: %w", plugin, err)
	}
	if !found {
		return &PermError{
			Code: "EPERM",
			Msg:  fmt.Sprintf("no consent record for plugin %q; install the plugin first", plugin),
		}
	}

	var perms eventsPermsWire
	if len(rawPerms) > 0 {
		if err := json.Unmarshal(rawPerms, &perms); err != nil {
			return fmt.Errorf("events.subscribe: parse consent JSON for %q: %w", plugin, err)
		}
	}

	if !MatchEventPattern(perms.Events, name) {
		return &PermError{
			Code: "EPERM",
			Msg:  fmt.Sprintf("events not granted for %q (plugin %q)", name, plugin),
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// unsubscribe
// ─────────────────────────────────────────────

type unsubscribeArgs struct {
	SubID string `json:"subId"`
}

func (e *EventsAPI) unsubscribe(raw json.RawMessage) (any, error) {
	var a unsubscribeArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &WireError{Code: "EINVAL", Message: fmt.Sprintf("events.unsubscribe: bad args: %v", err)}
	}

	e.subMu.Lock()
	sub, ok := e.subs[a.SubID]
	if ok {
		delete(e.subs, a.SubID)
	}
	e.subMu.Unlock()

	if ok {
		// Stop the bus handler first, then close the conn sub (which triggers
		// the pump goroutine to exit).
		sub.busUnsub()
		sub.conn.Unsubscribe(a.SubID)
	}
	// Unknown subId → no-op (idempotent). Spec: "always allowed to release".
	return nil, nil
}

// ─────────────────────────────────────────────
// publish
// ─────────────────────────────────────────────

type publishArgs struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

func (e *EventsAPI) publish(plugin string, raw json.RawMessage) (any, error) {
	var a publishArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &WireError{Code: "EINVAL", Message: fmt.Sprintf("events.publish: bad args: %v", err)}
	}
	if a.Name == "" {
		return nil, &WireError{Code: "EINVAL", Message: "events.publish: name is required"}
	}

	// Always rewrite name to plugin.<pluginName>.<name> so plugins cannot
	// masquerade as host events (e.g. session.*).
	qualifiedName := "plugin." + plugin + "." + a.Name

	e.bus.Publish(qualifiedName, a.Data)
	return nil, nil
}

// ─────────────────────────────────────────────
// MatchEventPattern
// ─────────────────────────────────────────────

// MatchEventPattern reports whether name matches ANY pattern in granted.
//
// Glob semantics (dotted-segment model):
//
//	*   matches one or more characters within a SINGLE dotted segment.
//	    "session.*"  matches "session.idle"  but NOT "session.x.y".
//	**  matches across segments (zero or more additional segments including dots).
//	    "session.**" matches "session.idle" and "session.x.y".
//	*   alone matches any name that contains no dots.
//
// Examples (from spec):
//
//	MatchEventPattern(["session.*"],    "session.idle")   = true
//	MatchEventPattern(["session.*"],    "session.x.y")    = false
//	MatchEventPattern(["session.**"],   "session.x.y")    = true
//	MatchEventPattern(["*"],            "session.idle")   = false  (has a dot)
//	MatchEventPattern(["*"],            "session")        = true
//	MatchEventPattern(["session.idle"], "session.idle")   = true
//	MatchEventPattern([],              anything)          = false
func MatchEventPattern(granted []string, name string) bool {
	if len(granted) == 0 || name == "" {
		return false
	}
	for _, pattern := range granted {
		if matchEventSingle(pattern, name) {
			return true
		}
	}
	return false
}

// matchEventSingle tests one pattern against name.
func matchEventSingle(pattern, name string) bool {
	// Exact match — fast path.
	if pattern == name {
		return true
	}

	// Lone "*" — matches any single-segment name (no dots).
	if pattern == "*" {
		return !strings.Contains(name, ".")
	}

	// "prefix.**" — matches prefix.anything (prefix + "." + one-or-more-segments).
	if strings.HasSuffix(pattern, ".**") {
		prefix := pattern[:len(pattern)-3] // strip ".**"
		// name must start with prefix + "."
		return strings.HasPrefix(name, prefix+".")
	}

	// "prefix.*" — matches prefix.<single-segment> (exactly one more segment).
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2] // strip ".*"
		// name must start with prefix + "." and have no further dots after that.
		withDot := prefix + "."
		if !strings.HasPrefix(name, withDot) {
			return false
		}
		rest := name[len(withDot):]
		// rest must be non-empty and contain no dot.
		return rest != "" && !strings.Contains(rest, ".")
	}

	// General case: pattern may contain "*" within a segment.
	// Split both on "." and match segment by segment.
	patSegs := strings.Split(pattern, ".")
	nameSegs := strings.Split(name, ".")
	if len(patSegs) != len(nameSegs) {
		return false
	}
	for i, ps := range patSegs {
		if !matchSegment(ps, nameSegs[i]) {
			return false
		}
	}
	return true
}

// matchSegment performs simple * wildcard matching within a single segment.
// "*" within a segment matches any non-empty string in that position.
func matchSegment(pattern, seg string) bool {
	if pattern == seg {
		return true
	}
	if pattern == "*" {
		return true
	}

	// Left-anchored: "log*" matches "logger".
	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(seg, prefix)
	}
	// Right-anchored: "*log" matches "slog".
	if strings.HasPrefix(pattern, "*") && !strings.Contains(pattern[1:], "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(seg, suffix)
	}

	return false
}

// ─────────────────────────────────────────────
// newSubID — random subscription identifier
// ─────────────────────────────────────────────

const subIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func newSubID() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = subIDAlphabet[rand.Intn(len(subIDAlphabet))]
	}
	return string(b)
}

// ─────────────────────────────────────────────
// WireError implements the error interface so Dispatch callers can
// type-assert without importing a separate package.
// ─────────────────────────────────────────────

// Error implements the error interface.
func (w *WireError) Error() string {
	return fmt.Sprintf("%s: %s", w.Code, w.Message)
}
