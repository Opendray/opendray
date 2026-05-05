package capture

import (
	"errors"
	"fmt"
)

// Trigger decides whether a rule should fire given (a) its
// configuration and (b) the current and last-seen message counts.
//
// Phase A only ships AfterMessagesTrigger. Adding new kinds
// (on_idle, k_chars, manual) means a new switch arm in
// triggerFromRule + a new struct here — no schema changes needed.
type Trigger interface {
	// Evaluate returns true when the rule should fire NOW given the
	// state in (lastSeenIndex, currentMessageCount). lastSeenIndex
	// is the index of the last message we already summarised
	// (-1 = none seen yet). The trigger doesn't mutate state — that's
	// runCapture's job after a successful summarizer call.
	Evaluate(lastSeenIndex, currentMessageCount int) bool
	// Description is shown in the UI.
	Description() string
}

// AfterMessagesTrigger fires when at least N new messages have
// arrived since the last capture. The DB JSONB shape is:
//
//	{"n": 6}
//
// Default n = 6 if missing or non-positive.
type AfterMessagesTrigger struct {
	N int
}

// Evaluate fires when current_count - last_seen_index - 1 >= N.
//   - lastSeenIndex = -1 (nothing seen yet, current_count=N) → fire on first N msgs
//   - last_seen_index = 5, current = 11 → 5 new messages, n=6 wait
//   - last_seen_index = 5, current = 12 → 6 new messages, fire
func (t AfterMessagesTrigger) Evaluate(lastSeenIndex, currentMessageCount int) bool {
	n := t.N
	if n <= 0 {
		n = 6
	}
	newCount := currentMessageCount - lastSeenIndex - 1
	return newCount >= n
}

func (t AfterMessagesTrigger) Description() string {
	n := t.N
	if n <= 0 {
		n = 6
	}
	return fmt.Sprintf("after_messages: every %d new user messages", n)
}

// triggerFromRule materialises a Trigger from the rule's
// trigger_kind + trigger_config JSONB. Returns an error when the
// kind isn't recognised — caller logs + skips the rule.
func triggerFromRule(r Rule) (Trigger, error) {
	switch r.TriggerKind {
	case "after_messages":
		n := 6
		if v, ok := r.TriggerConfig["n"]; ok {
			switch x := v.(type) {
			case float64:
				n = int(x)
			case int:
				n = x
			case int64:
				n = int(x)
			}
		}
		if n <= 0 {
			n = 6
		}
		return AfterMessagesTrigger{N: n}, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedTriggerKind, r.TriggerKind)
	}
}

var errUnsupportedTriggerKind = errors.New("capture: unsupported trigger kind")
