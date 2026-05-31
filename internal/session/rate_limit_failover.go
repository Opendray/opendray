package session

import (
	"context"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// tryRateLimitFailover scans the session's rolling output window for
// the Claude rate-limit banner and, on a match, marks the current
// account throttled and switches the session to the next available
// account. Called from pumpStdout via the rate-limit-scan tick (at
// most once per rateLimitScanCooldown per session).
//
// Failure modes (all non-fatal — we log and let the session keep
// running on the throttled account):
//
//   - resolver nil OR autoFailover disabled → never called (gated upstream)
//   - banner not found → silent return
//   - already-throttled current account → silent return (don't re-fire)
//   - no failover target available → log warn, publish "no_target" event
//   - SwitchClaudeAccount itself fails → log warn, publish "failover_failed" event
//
// MVP scope: only sessions pinned to a NAMED account get failover.
// Sessions running on the empty-id default (~/.claude) are skipped
// here because mapping the default to a real account row would
// require an extra round-trip the resolver doesn't expose yet. The
// majority of sessions on a multi-account install end up pinned at
// spawn time via PickAutoAssignClaudeAccount, so this gap shrinks to
// zero in practice once the operator has ≥2 enabled accounts.
func (m *Manager) tryRateLimitFailover(rs *runningSession, now time.Time) {
	window := rs.rateLimitWindow()
	reset, ok := ScanForRateLimitBanner(window, now)
	if !ok {
		return
	}

	rs.sessMu.RLock()
	sessID := rs.sess.ID
	currentAccountID := rs.sess.ClaudeAccountID
	rs.sessMu.RUnlock()

	// MVP: don't try to fail over from the empty-id default. The
	// scanner has no clean way to resolve it without another DB hop
	// the resolver interface deliberately doesn't expose.
	if currentAccountID == "" {
		return
	}

	// Don't re-fire on the same throttle window — if the banner is
	// still in the rolling buffer after a successful failover, the
	// new account's session shouldn't get switched away again.
	if m.claudeAccounts.IsClaudeAccountThrottled(currentAccountID) {
		return
	}

	m.claudeAccounts.MarkClaudeAccountThrottled(currentAccountID, reset)
	m.log.Info("claude rate-limit detected, attempting failover",
		"session", sessID,
		"current_account", currentAccountID,
		"reset", reset.Format(time.RFC3339),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	target, err := m.claudeAccounts.PickFailoverClaudeAccount(ctx, currentAccountID)
	if err != nil {
		m.log.Warn("failover account-pick failed", "session", sessID, "err", err)
		return
	}
	if target == "" {
		m.log.Warn("rate-limit hit but no failover target available",
			"session", sessID,
			"current_account", currentAccountID,
		)
		if m.bus != nil {
			m.bus.Publish(eventbus.Event{
				Topic: "session.auto_failover_no_target",
				Data: map[string]any{
					"session_id": sessID,
					"account_id": currentAccountID,
					"reset":      reset.UTC().Format(time.RFC3339),
				},
			})
		}
		return
	}

	if _, err := m.SwitchClaudeAccount(ctx, sessID, target); err != nil {
		m.log.Warn("auto-failover switch failed",
			"session", sessID,
			"from_account", currentAccountID,
			"to_account", target,
			"err", err,
		)
		if m.bus != nil {
			m.bus.Publish(eventbus.Event{
				Topic: "session.auto_failover_failed",
				Data: map[string]any{
					"session_id":   sessID,
					"from_account": currentAccountID,
					"to_account":   target,
					"err":          err.Error(),
				},
			})
		}
		return
	}

	rs.clearRateLimitWindow()
	m.log.Info("auto-failover succeeded",
		"session", sessID,
		"from_account", currentAccountID,
		"to_account", target,
	)
	if m.bus != nil {
		m.bus.Publish(eventbus.Event{
			Topic: "session.auto_switched",
			Data: map[string]any{
				"session_id":   sessID,
				"from_account": currentAccountID,
				"to_account":   target,
				"reason":       "rate_limit",
				"reset":        reset.UTC().Format(time.RFC3339),
			},
		})
	}
}
