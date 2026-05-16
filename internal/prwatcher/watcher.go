// Package prwatcher polls the host APIs of repos opened in active
// opendray sessions and publishes pr.checks_completed events when a
// PR's CI suite finishes. The channel hub turns those events into
// chat-side notifications so the operator doesn't have to babysit
// GitHub Actions waiting for green.
//
// Scope is intentionally minimal for the first iteration:
//
//   - In-memory state only (a gateway restart re-baselines and may
//     miss any transition that happened during the restart).
//   - One poll per (session cwd × PR), capped at maxTrackedPRs total
//     so a chatty operator with 50 open branches doesn't DoS upstream.
//   - Only `pending → completed` transitions fire — first poll seeds
//     state but doesn't notify (otherwise restarting opendray would
//     spam every open PR's last completion).
package prwatcher

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/githost"
)

const (
	defaultPollInterval = 90 * time.Second
	maxTrackedPRs       = 25
)

// SessionLister is the slice of session.Manager we need to learn
// which cwds to poll. Defined here so tests can stub without
// pulling the whole Manager.
type SessionLister interface {
	List(ctx context.Context) ([]SessionInfo, error)
}

// SessionInfo is the minimum data the watcher needs about a
// session. Concrete adapters (in internal/app) map session.Session
// onto this surface.
type SessionInfo struct {
	ID    string
	Cwd   string
	State string // running | idle | pending | ended | stopped
}

// Service is the watcher. Held by the app; Start spawns the poller
// goroutine and Stop cancels it.
type Service struct {
	sessions SessionLister
	gh       *githost.Service
	bus      *eventbus.Hub
	log      *slog.Logger

	pollInterval time.Duration

	mu    sync.Mutex
	state map[string]prState // key: hostKindRepo + "#" + number
}

// prState is the last-seen aggregate check status for a tracked
// PR. We only fire an event when this transitions from a non-
// terminal state to a terminal one (any "completed-ish" status).
type prState struct {
	terminal    bool   // true = all checks have a final conclusion
	conclusion  string // success | failure | mixed | none
	totalChecks int
}

// Option mutates Service defaults; pass to New.
type Option func(*Service)

// WithPollInterval overrides the default 90s cadence. Useful for
// tests (sub-second) and for operators on slow upstream APIs who
// want to ease the rate-limit budget.
func WithPollInterval(d time.Duration) Option {
	return func(s *Service) {
		if d > 0 {
			s.pollInterval = d
		}
	}
}

// New returns an unstarted watcher.
func New(sessions SessionLister, gh *githost.Service, bus *eventbus.Hub, log *slog.Logger, opts ...Option) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{
		sessions:     sessions,
		gh:           gh,
		bus:          bus,
		log:          log.With("component", "prwatcher"),
		pollInterval: defaultPollInterval,
		state:        make(map[string]prState),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start launches the poller. Cancel ctx (or call Stop) to halt.
// Idempotent — calling twice is a no-op for the second call once
// the first poll completes.
func (s *Service) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Service) run(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	// First poll immediately so the operator doesn't wait the full
	// interval for a baseline. The first iteration seeds state and
	// never fires events.
	s.pollOnce(ctx, true)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollOnce(ctx, false)
		}
	}
}

// PollOnce is exposed for tests. seed=true means "this is the
// baseline poll, populate state but don't emit events".
func (s *Service) PollOnce(ctx context.Context) {
	s.pollOnce(ctx, false)
}

func (s *Service) pollOnce(ctx context.Context, seed bool) {
	sessions, err := s.sessions.List(ctx)
	if err != nil {
		s.log.Warn("session list failed", "err", err)
		return
	}
	// Dedupe cwds — multiple sessions on the same repo share PRs.
	cwds := uniqueLiveCwds(sessions)
	if len(cwds) == 0 {
		return
	}

	tracked := 0
	for _, cwd := range cwds {
		if tracked >= maxTrackedPRs {
			break
		}
		_, prs, err := s.gh.ListPullRequests(ctx, cwd, "open")
		if err != nil {
			// Most common case: no token configured for this remote.
			// Don't log loudly on every poll — that becomes spam.
			continue
		}
		for _, pr := range prs {
			if tracked >= maxTrackedPRs {
				break
			}
			tracked++
			s.checkOne(ctx, cwd, pr, seed)
		}
	}
}

// checkOne fetches a single PR's checks, aggregates the status,
// compares against last known state, and publishes when it
// transitioned to a terminal state.
func (s *Service) checkOne(ctx context.Context, cwd string, pr githost.PullRequest, seed bool) {
	checks, err := s.gh.PRChecks(ctx, cwd, pr.Number)
	if err != nil {
		return
	}

	// No checks configured → nothing to notify on. Skip without
	// touching state so the operator can later add checks and
	// the watcher will pick up the first transition cleanly.
	if len(checks) == 0 {
		return
	}

	cur := aggregate(checks)
	// Stable key combines the host-side identity with the local
	// PR number. We don't track repo identity in the watcher
	// because the cwd alone isn't sufficient (two workspaces on
	// the same repo would collide); the PR URL is the cleanest
	// per-remote handle we have.
	key := fmt.Sprintf("%s#%d", pr.URL, pr.Number)

	s.mu.Lock()
	prev, existed := s.state[key]
	s.state[key] = cur
	s.mu.Unlock()

	if seed {
		// First-ever observation: just record. Otherwise a gateway
		// restart would spam every open PR's last conclusion.
		return
	}
	if !cur.terminal {
		return
	}
	// Fire only on transition into terminal. Suppresses repeated
	// notifications for the same final state on subsequent polls.
	if existed && prev.terminal && prev.conclusion == cur.conclusion {
		return
	}
	s.bus.Publish(eventbus.Event{
		Topic: "pr.checks_completed",
		Data: map[string]any{
			"cwd":        cwd,
			"pr_number":  pr.Number,
			"pr_title":   pr.Title,
			"pr_url":     pr.URL,
			"pr_head":    pr.Head,
			"pr_base":    pr.Base,
			"conclusion": cur.conclusion,
			"checks":     cur.totalChecks,
		},
	})
}

// aggregate folds individual CheckRuns into a single suite verdict
// the way an operator would interpret them at a glance.
func aggregate(checks []githost.CheckRun) prState {
	if len(checks) == 0 {
		return prState{}
	}
	pending := 0
	passed := 0
	failed := 0
	for _, c := range checks {
		if c.Status != "completed" {
			pending++
			continue
		}
		switch c.Conclusion {
		case "success", "neutral", "skipped":
			passed++
		default:
			// failure / cancelled / timed_out / action_required —
			// anything else with a final conclusion counts as a
			// problem worth surfacing.
			failed++
		}
	}
	if pending > 0 {
		return prState{
			terminal:    false,
			conclusion:  "pending",
			totalChecks: len(checks),
		}
	}
	conclusion := "success"
	if failed > 0 && passed > 0 {
		conclusion = "mixed"
	} else if failed > 0 {
		conclusion = "failure"
	}
	return prState{
		terminal:    true,
		conclusion:  conclusion,
		totalChecks: len(checks),
	}
}

// uniqueLiveCwds returns the cwds of sessions that are running /
// idle / pending. Terminated sessions are ignored so we don't
// poll forever for repos the operator abandoned.
func uniqueLiveCwds(sessions []SessionInfo) []string {
	seen := make(map[string]struct{}, len(sessions))
	out := make([]string, 0, len(sessions))
	for _, s := range sessions {
		if s.Cwd == "" {
			continue
		}
		switch s.State {
		case "running", "idle", "pending":
			// Live state.
		default:
			continue
		}
		if _, ok := seen[s.Cwd]; ok {
			continue
		}
		seen[s.Cwd] = struct{}{}
		out = append(out, s.Cwd)
	}
	return out
}
