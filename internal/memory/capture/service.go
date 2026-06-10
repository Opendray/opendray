package capture

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/memory/summarizer"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// isEphemeralCwd delegates to the canonical predicate so capture and
// the Notes layer agree on what counts as a throwaway temp dir.
func isEphemeralCwd(cwd string) bool { return projectdoc.IsEphemeralCwd(cwd) }

// MemoryWriter is the slice of memory.Service the capture pipeline
// needs. Defined locally so tests can pass a mock.
type MemoryWriter interface {
	Search(ctx context.Context, req memory.SearchRequest) ([]memory.SearchHit, error)
	Store(ctx context.Context, req memory.StoreRequest) (string, error)
}

// SessionInfo is the minimal session shape capture needs — id +
// provider id + cwd + provenance. Real impl wraps session.Manager.
type SessionInfo struct {
	ID         string
	ProviderID string
	Cwd        string
	State      string // "running" / "idle" / etc.
	// Origin is who created the session ("operator"|"integration"|
	// "cli"; empty = operator). IntegrationID is set for
	// origin=integration. Cortex Phase 2: the runner routes captured
	// facts by the integration's memory_policy.
	Origin        string
	IntegrationID string
}

// PolicyResolver answers "what memory policy does this integration
// declare?" — backed by integration.Service. Returned values are
// "none" | "quarantine" | "full"; errors degrade to quarantine
// (safe-by-default).
type PolicyResolver interface {
	MemoryPolicy(ctx context.Context, integrationID string) (string, error)
}

// DefaultQuarantineTTL bounds how long un-reviewed quarantined facts
// live before the cleaner purges them.
const DefaultQuarantineTTL = 30 * 24 * time.Hour

// SessionLister is what capture needs from session.Manager.
type SessionLister interface {
	List(ctx context.Context) ([]SessionInfo, error)
}

// HistoryReader fetches a session's user-prompt transcript. Returns
// chronological order (oldest first). Empty slice = no transcript yet.
type HistoryReader interface {
	History(ctx context.Context, sessionID string, limit int) ([]TranscriptEntry, error)
}

// TranscriptEntry is one user message in the transcript.
// Independent of internal/session.ProjectInput so we can mock it.
type TranscriptEntry struct {
	Ts   time.Time
	Text string
}

// SummarizerCallLogger is the slice of summarizer.Store this package
// uses to record invocations + token usage.
type SummarizerCallLogger interface {
	LogCall(ctx context.Context, row summarizer.CallLogRow) error
}

// runCapture is the per-(rule, session) capture step:
//
//  1. read transcript history
//  2. evaluate trigger; bail if not ready
//  3. resolve summarizer provider (rule's pin OR registry default)
//  4. call provider.Summarize on the new messages
//  5. for each fact: dedup search; store if novel
//  6. mark cursor at the last index processed
//  7. write a memory_summarizer_calls row recording the outcome
//
// Errors at step 3-5 don't abort the engine — they're logged into
// the call log + bumped on the failure streak. Engine keeps ticking.
type runner struct {
	rules    *RuleStore
	registry *summarizer.Registry
	// workerProvider is the M-PE worker-fabric-backed default. When
	// non-nil and a rule doesn't pin SummarizerProviderID, the
	// capture engine resolves its provider through the worker
	// registry (so operators can pick Agent (CLI --print) for
	// capture the same way they do for gitactivity / transcript /
	// cleaner). Nil falls back to the pre-M-PE behaviour:
	// summarizer.Registry.Default() returns an HTTP provider.
	workerProvider summarizer.Provider
	memory         MemoryWriter
	history        HistoryReader
	callLog        SummarizerCallLogger
	state          *stateMap
	historyLimit   int
	log            *slog.Logger
	// policy + quarantineTTL route integration-created sessions'
	// facts into the right memory tier. Nil policy treats every
	// integration session as quarantine (safe-by-default).
	policy        PolicyResolver
	quarantineTTL time.Duration
}

// routeForSession resolves the memory tier for a session's captured
// facts from its origin + the integration's declared policy.
// skip=true means the session must produce no memory at all.
func (r *runner) routeForSession(ctx context.Context, sess SessionInfo) (tier string, expiry *time.Time, skip bool) {
	// Sessions running in throwaway temp dirs (third-party consumers,
	// tests) are not project work — they leave no memory at all,
	// regardless of origin. Mirrors projectdoc.IsEphemeralCwd.
	if isEphemeralCwd(sess.Cwd) {
		return "", nil, true
	}
	if sess.Origin != "integration" {
		return memory.TierDurable, nil, false
	}
	policy := "quarantine"
	if r.policy != nil {
		p, err := r.policy.MemoryPolicy(ctx, sess.IntegrationID)
		if err != nil {
			r.log.Warn("capture: memory policy lookup failed — quarantining",
				"session_id", sess.ID, "integration_id", sess.IntegrationID, "err", err)
		} else if p != "" {
			policy = p
		}
	}
	switch policy {
	case "none":
		return "", nil, true
	case "full":
		return memory.TierDurable, nil, false
	default: // quarantine
		ttl := r.quarantineTTL
		if ttl <= 0 {
			ttl = DefaultQuarantineTTL
		}
		t := time.Now().UTC().Add(ttl)
		return memory.TierQuarantine, &t, false
	}
}

// runForceForSession bypasses trigger evaluation and pause state,
// running the capture pipeline immediately. Used by the /run-now
// endpoint and Phase C UI buttons. Equivalent to runForSession
// minus the gate at the top.
func (r *runner) runForceForSession(ctx context.Context, rule Rule, sess SessionInfo) {
	r.runForSessionWithForce(ctx, rule, sess, true)
}

func (r *runner) runForSession(ctx context.Context, rule Rule, sess SessionInfo) {
	r.runForSessionWithForce(ctx, rule, sess, false)
}

func (r *runner) runForSessionWithForce(ctx context.Context, rule Rule, sess SessionInfo, force bool) {
	if rule.SessionID != "" && rule.SessionID != sess.ID {
		return // shouldn't happen but cheap to defend
	}
	if !force && r.state.IsPaused(rule.ID, sess.ID) {
		return
	}

	// Provenance routing (Cortex Phase 2) — decided up front so a
	// policy=none session skips the transcript read + summarizer call
	// entirely, not just the store.
	tier, quarantineExpiry, skip := r.routeForSession(ctx, sess)
	if skip {
		return
	}

	transcript, err := r.history.History(ctx, sess.ID, r.historyLimit)
	if err != nil {
		r.log.Warn("capture: history read failed",
			"rule_id", rule.ID, "session_id", sess.ID, "err", err)
		return
	}
	if len(transcript) == 0 {
		return
	}

	st := r.state.Get(rule.ID, sess.ID)
	currentIndex := len(transcript) - 1
	trig, err := triggerFromRule(rule)
	if err != nil {
		r.log.Warn("capture: bad trigger config", "rule_id", rule.ID, "err", err)
		return
	}
	inputs := EvaluationInputs{
		LastSeenIndex:       st.LastSeenIndex,
		CurrentMessageCount: len(transcript),
		Now:                 time.Now().UTC(),
	}
	if len(transcript) > 0 {
		inputs.LastMessageAt = transcript[len(transcript)-1].Ts
	}
	startForChars := st.LastSeenIndex + 1
	if startForChars < 0 || startForChars >= len(transcript) {
		startForChars = 0
	}
	for _, e := range transcript[startForChars:] {
		inputs.CharsSinceLastFire += len(e.Text)
	}
	if !force && !trig.Evaluate(inputs) {
		return
	}

	// Slice off the new messages. lastSeenIndex=-1 → take from 0.
	startIdx := st.LastSeenIndex + 1
	if startIdx < 0 || startIdx >= len(transcript) {
		startIdx = 0
	}
	new := transcript[startIdx:]
	if len(new) == 0 {
		return
	}

	// Convert to summarizer.Message — Phase A user-only.
	msgs := make([]summarizer.Message, 0, len(new))
	for _, e := range new {
		txt := strings.TrimSpace(e.Text)
		if txt == "" {
			continue
		}
		msgs = append(msgs, summarizer.Message{
			Role:      summarizer.RoleUser,
			Text:      txt,
			Timestamp: e.Ts,
		})
	}
	if len(msgs) == 0 {
		return
	}

	// Pick provider.
	prov, perr := r.pickProvider(ctx, rule)
	if perr != nil {
		r.log.Info("capture: no provider available",
			"rule_id", rule.ID, "session_id", sess.ID, "err", perr)
		r.recordCall(ctx, summarizer.CallLogRow{
			RuleID:    rule.ID,
			SessionID: sess.ID,
			StartedAt: time.Now().UTC(),
			Status:    "provider_unavailable",
			Error:     perr.Error(),
		})
		r.state.MarkFailure(rule.ID, sess.ID)
		return
	}

	// Call summarizer.
	startedAt := time.Now().UTC()
	res, sumErr := prov.Summarize(ctx, msgs)
	finishedAt := time.Now().UTC()

	if sumErr != nil {
		r.log.Warn("capture: summarizer failed",
			"rule_id", rule.ID, "session_id", sess.ID, "err", sumErr)
		r.recordCall(ctx, summarizer.CallLogRow{
			RuleID:               rule.ID,
			ProviderID:           rule.SummarizerProviderID,
			SessionID:            sess.ID,
			StartedAt:            startedAt,
			FinishedAt:           finishedAt,
			DurationMs:           int(finishedAt.Sub(startedAt).Milliseconds()),
			InputTokens:          res.InputTokens,
			OutputTokens:         res.OutputTokens,
			EstimatedUSD:         res.EstimatedUSD,
			Status:               classifyError(sumErr),
			Error:                sumErr.Error(),
			RawResponseTruncated: res.RawResponse,
		})
		r.state.MarkFailure(rule.ID, sess.ID)
		return
	}

	// Dedup + store.
	stored, skipped := 0, 0
	scopeKey := scopeKeyForRule(rule, sess)
	for _, fact := range res.Facts {
		if r.isDuplicate(ctx, fact, rule, scopeKey) {
			skipped++
			continue
		}
		conf := fact.Confidence // copy local for pointer
		meta := map[string]any{
			"summarizer_category": string(fact.Category),
			"provider_kind":       prov.Kind(),
			"provider_name":       prov.Name(),
		}
		if sess.Origin == "integration" && sess.IntegrationID != "" {
			meta["integration_id"] = sess.IntegrationID
		}
		_, err := r.memory.Store(ctx, memory.StoreRequest{
			Text:                fact.Text,
			Scope:               memory.Scope(rule.TargetScope),
			ScopeKey:            scopeKey,
			SourceKind:          "summarizer",
			SourceRef:           rule.ID,
			SummarizerSession:   sess.ID,
			Confidence:          &conf,
			Tier:                tier,
			QuarantineExpiresAt: quarantineExpiry,
			Metadata:            meta,
		})
		if err != nil {
			r.log.Warn("capture: memory store failed",
				"rule_id", rule.ID, "fact", fact.Text, "err", err)
			continue
		}
		stored++
	}

	// Record success in call log.
	r.recordCall(ctx, summarizer.CallLogRow{
		RuleID:               rule.ID,
		ProviderID:           rule.SummarizerProviderID,
		SessionID:            sess.ID,
		StartedAt:            startedAt,
		FinishedAt:           finishedAt,
		DurationMs:           int(finishedAt.Sub(startedAt).Milliseconds()),
		InputTokens:          res.InputTokens,
		OutputTokens:         res.OutputTokens,
		EstimatedUSD:         res.EstimatedUSD,
		FactsExtracted:       len(res.Facts),
		FactsStored:          stored,
		FactsSkippedDedup:    skipped,
		Status:               "succeeded",
		RawResponseTruncated: res.RawResponse,
	})

	// Advance cursor.
	r.state.MarkFired(rule.ID, sess.ID, currentIndex)
	r.log.Info("capture: rule fired",
		"rule_id", rule.ID, "session_id", sess.ID,
		"new_messages", len(msgs), "facts_stored", stored, "facts_skipped", skipped,
		"input_tokens", res.InputTokens, "output_tokens", res.OutputTokens)
}

// pickProvider resolves the runtime provider:
//  1. If rule pins SummarizerProviderID, build that — explicit
//     overrides win, so existing operator configurations don't
//     change behaviour after M-PE.
//  2. Else, if a worker-fabric provider is wired (M-PE default),
//     dispatch through it. The worker registry decides
//     summarizer-HTTP vs agent-CLI per task config.
//  3. Else, fall back to summarizer.Registry.Default() — the
//     pre-M-PE behaviour, preserved for installs that haven't
//     wired the worker registry yet.
func (r *runner) pickProvider(ctx context.Context, rule Rule) (summarizer.Provider, error) {
	if rule.SummarizerProviderID != "" {
		return r.registry.Build(ctx, rule.SummarizerProviderID)
	}
	if r.workerProvider != nil {
		return r.workerProvider, nil
	}
	return r.registry.Default(ctx)
}

// isDuplicate runs memory.Search with the fact text against the
// target scope; if any hit comes back with similarity >= rule's
// dedup_threshold, the fact is treated as already-known.
func (r *runner) isDuplicate(ctx context.Context, fact summarizer.Fact, rule Rule, scopeKey string) bool {
	hits, err := r.memory.Search(ctx, memory.SearchRequest{
		Query:    fact.Text,
		Scope:    memory.Scope(rule.TargetScope),
		ScopeKey: scopeKey,
		TopK:     1,
	})
	if err != nil {
		// On search error, prefer to write (a duplicate is better
		// than losing a novel fact). Log + continue.
		r.log.Warn("capture: dedup search failed", "rule_id", rule.ID, "err", err)
		return false
	}
	if len(hits) == 0 {
		return false
	}
	return hits[0].Similarity >= rule.DedupThreshold
}

// scopeKeyForRule decides the scope_key written into memory rows.
// 'project' targets (and the retired 'session', which now folds to
// project) use the session's cwd; 'global' uses an operator-style
// placeholder.
func scopeKeyForRule(rule Rule, sess SessionInfo) string {
	switch rule.TargetScope {
	case "global":
		return "operator"
	default:
		return sess.Cwd
	}
}

// classifyError maps a summarizer error into the call log's
// status enum.
func classifyError(err error) string {
	switch {
	case errors.Is(err, summarizer.ErrUnreachable),
		errors.Is(err, summarizer.ErrAuthFailed),
		errors.Is(err, summarizer.ErrRateLimited),
		errors.Is(err, summarizer.ErrModelNotFound):
		return "provider_unavailable"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	default:
		return "failed"
	}
}

// recordCall is a thin wrapper that swallows + logs LogCall errors
// — capture's main path can't do anything useful with a failed
// audit write.
func (r *runner) recordCall(ctx context.Context, row summarizer.CallLogRow) {
	if err := r.callLog.LogCall(ctx, row); err != nil {
		r.log.Warn("capture: log call failed", "err", err)
	}
}

// silence unused import flag
var _ = fmt.Sprintf
