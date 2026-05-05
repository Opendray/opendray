package capture

import (
	"errors"
	"fmt"
	"time"
)

// Trigger decides whether a rule should fire given (a) its
// configuration and (b) per-tick session state in EvaluationInputs.
//
// Phase A: AfterMessagesTrigger.
// Phase B: OnIdleTrigger, KCharsTrigger, ManualTrigger.
type Trigger interface {
	// Evaluate returns true when the rule should fire NOW.
	// Implementations don't mutate state.
	Evaluate(in EvaluationInputs) bool
	// Description is shown in the UI / logs.
	Description() string
}

// EvaluationInputs bundles every signal a Trigger may need.
// Adding a new field is non-breaking — old trigger kinds ignore it.
type EvaluationInputs struct {
	LastSeenIndex       int
	CurrentMessageCount int
	LastMessageAt       time.Time
	// CharsSinceLastFire is the cumulative character count across
	// new (post-LastSeenIndex) user messages — drives k_chars.
	CharsSinceLastFire int
	// Now is the wall-clock at evaluation time, threaded in so
	// tests are deterministic.
	Now time.Time
}

// AfterMessagesTrigger fires when at least N new user messages
// have arrived since the last capture. trigger_config: {"n": 6}.
type AfterMessagesTrigger struct {
	N int
}

func (t AfterMessagesTrigger) Evaluate(in EvaluationInputs) bool {
	n := t.N
	if n <= 0 {
		n = 6
	}
	newCount := in.CurrentMessageCount - in.LastSeenIndex - 1
	return newCount >= n
}
func (t AfterMessagesTrigger) Description() string {
	n := t.N
	if n <= 0 {
		n = 6
	}
	return fmt.Sprintf("after_messages: every %d new user messages", n)
}

// OnIdleTrigger fires when the session has been idle (no new user
// messages) for at least Seconds since the last user message AND
// at least one new message exists since the last fire.
// trigger_config: {"seconds": 60}.
type OnIdleTrigger struct {
	Seconds int
}

func (t OnIdleTrigger) Evaluate(in EvaluationInputs) bool {
	sec := t.Seconds
	if sec <= 0 {
		sec = 60
	}
	if in.LastMessageAt.IsZero() {
		return false
	}
	if in.CurrentMessageCount-in.LastSeenIndex-1 <= 0 {
		return false
	}
	idle := in.Now.Sub(in.LastMessageAt)
	return idle >= time.Duration(sec)*time.Second
}
func (t OnIdleTrigger) Description() string {
	sec := t.Seconds
	if sec <= 0 {
		sec = 60
	}
	return fmt.Sprintf("on_idle: when session idle ≥ %ds", sec)
}

// KCharsTrigger fires when the cumulative character count across
// new user messages crosses K. trigger_config: {"k": 4000}.
type KCharsTrigger struct {
	K int
}

func (t KCharsTrigger) Evaluate(in EvaluationInputs) bool {
	k := t.K
	if k <= 0 {
		k = 4000
	}
	return in.CharsSinceLastFire >= k
}
func (t KCharsTrigger) Description() string {
	k := t.K
	if k <= 0 {
		k = 4000
	}
	return fmt.Sprintf("k_chars: every %d new chars in user messages", k)
}

// ManualTrigger never auto-fires. Operators trigger it via
// POST /memory-capture-rules/{id}/run-now (Phase C UI provides a
// button).
type ManualTrigger struct{}

func (t ManualTrigger) Evaluate(in EvaluationInputs) bool { return false }
func (t ManualTrigger) Description() string {
	return "manual: only fires via POST .../run-now"
}

// triggerFromRule materialises a Trigger from rule.trigger_kind +
// trigger_config. Returns an error for unsupported kinds.
func triggerFromRule(r Rule) (Trigger, error) {
	switch r.TriggerKind {
	case "after_messages":
		return AfterMessagesTrigger{N: cfgInt(r.TriggerConfig, "n")}, nil
	case "on_idle":
		return OnIdleTrigger{Seconds: cfgInt(r.TriggerConfig, "seconds")}, nil
	case "k_chars":
		return KCharsTrigger{K: cfgInt(r.TriggerConfig, "k")}, nil
	case "manual":
		return ManualTrigger{}, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedTriggerKind, r.TriggerKind)
	}
}

func cfgInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

var errUnsupportedTriggerKind = errors.New("capture: unsupported trigger kind")
