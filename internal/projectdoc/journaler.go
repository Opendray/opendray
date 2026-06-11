package projectdoc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// SessionInfo is the minimal slice of session metadata the
// journaler needs. Defined here (rather than depending on
// internal/session.Session directly) so the import graph stays
// one-way — internal/session keeps a clean surface and projectdoc
// stays test-friendly with a fake adapter.
type SessionInfo struct {
	ID         string
	ProviderID string
	Cwd        string
	StartedAt  time.Time
	EndedAt    *time.Time
	ExitCode   *int
}

// HistoryEntry mirrors session.ProjectInput. Same decoupling rationale.
type HistoryEntry struct {
	Ts   time.Time
	Text string
}

// SessionLookup is the contract the journaler needs from the
// session manager.
//
// Get returns the persisted session row by id (works for ended
// sessions too — that's load-bearing for the journaler since events
// fire AFTER the row is marked terminal).
//
// History returns the operator's recent inputs in the same cwd; we
// use the last N entries as raw material for the auto-journal body.
// Empty / unsupported provider → empty slice + nil error, journaler
// still writes a metadata-only entry.
//
// TranscriptText (M18) returns the full conversation log
// (user + assistant turns) formatted as plain markdown. The
// journaler feeds this to an LLM to produce a real "what did the
// agent do" summary across Claude / Codex / Gemini. Returns empty
// string for providers without a transcript reader yet — the
// journaler falls back to metadata-only journaling in that case.
type SessionLookup interface {
	Get(ctx context.Context, id string) (SessionInfo, error)
	History(ctx context.Context, id string, limit int) ([]HistoryEntry, error)
	TranscriptText(ctx context.Context, id string, maxBytes int) (string, error)
}

// TranscriptSummariser is the optional LLM hook that turns a raw
// transcript into a 1-3 paragraph "what the agent did" narrative.
// The journaler degrades to metadata-only when this is nil, when
// it errors, or when the transcript is empty — never blocks the
// journal write on LLM availability.
type TranscriptSummariser interface {
	SummariseTranscript(ctx context.Context, transcript string) (string, error)
}

// Journaler subscribes to session-end events on the eventbus and
// appends a session_logs row (kind=session_summary) per session.
// This is the M8 "auto-journal" — without it, the journal only gets
// entries when the operator or an agent explicitly appends one,
// which is rare in practice.
//
// Current implementation is summarizer-LLM-free on purpose: it
// builds a deterministic markdown body from session metadata + the
// last few operator inputs. This means it works on a fresh install
// without any LLM provider configured; a richer LLM-backed
// summarizer can layer on top later by swapping the body assembly
// inside writeEntry.
type Journaler struct {
	docs   *Service
	bus    *eventbus.Hub
	lookup SessionLookup
	log    *slog.Logger

	// inputsLimit caps how many operator inputs the journal entry
	// quotes back. The journal is meant to be a glanceable record,
	// not a transcript replay; 5 is plenty.
	inputsLimit int

	// summariser is the M18 LLM hook. Nil disables the
	// transcript-aware path; metadata-only journaling still works.
	summariser TranscriptSummariser

	// planDetector is the M-PA LLM hook. When set, every successful
	// transcript summary kicks off a plan-drift check; if the
	// detector says the plan needs updating, a proposal is filed
	// into project_doc_proposals (operator approves in the inbox).
	// Nil disables — journaling works exactly as before.
	planDetector PlanDriftDetector

	// driftJournalLookback is how many recent journal entries the
	// drift detector sees as context. 5 is enough to give the model
	// a sense of project momentum without exploding the prompt.
	driftJournalLookback int

	// skillUsage, when set, receives the raw session transcript after
	// each session ends so skill use_count / last_used_at can be
	// bumped (Cortex distillation workbench). Best-effort.
	skillUsage func(ctx context.Context, transcript string)

	// transcriptMaxBytes caps how much transcript we feed the LLM.
	// 16 KiB ≈ 4k tokens — enough context for a meaningful summary
	// without paying tokens we don't need. Older content is
	// trimmed by the session reader before this layer sees it.
	transcriptMaxBytes int
}

// NewJournaler builds a Journaler. docs / bus / lookup must be
// non-nil at app startup. The optional TranscriptSummariser is
// installed separately via WithSummariser so the LLM dependency
// stays isolated from journaler core wiring.
func NewJournaler(docs *Service, bus *eventbus.Hub, lookup SessionLookup, log *slog.Logger) *Journaler {
	if log == nil {
		log = slog.Default()
	}
	return &Journaler{
		docs:                 docs,
		bus:                  bus,
		lookup:               lookup,
		log:                  log.With("component", "projectdoc.journaler"),
		inputsLimit:          5,
		driftJournalLookback: 5,
		transcriptMaxBytes:   16 * 1024,
	}
}

// WithSummariser installs the optional LLM hook for M18 — when set,
// every session.ended event also kicks off a transcript-based
// "what did the agent do" summary that gets appended to the
// metadata-only body. Pass nil to disable. Returns the receiver
// for chained setup.
func (j *Journaler) WithSummariser(s TranscriptSummariser) *Journaler {
	j.summariser = s
	return j
}

// WithSkillUsage installs the optional skill-usage recorder: after a
// session ends its transcript is handed over so skills it referenced
// get their use counters bumped. Pass nil to disable.
func (j *Journaler) WithSkillUsage(fn func(ctx context.Context, transcript string)) *Journaler {
	j.skillUsage = fn
	return j
}

// WithPlanDetector installs the optional M-PA plan-drift hook.
// After the transcript summary lands, the journaler asks the
// detector whether the project's plan document needs updating
// based on this session's work; if so, it files a proposal that
// the operator approves in the inbox (same flow as a manual
// `project_plan_set` MCP call). Pass nil to disable.
func (j *Journaler) WithPlanDetector(d PlanDriftDetector) *Journaler {
	j.planDetector = d
	return j
}

// Run subscribes to session.ended + session.stopped topics and
// processes each event until ctx is cancelled. Errors per-event are
// logged and dropped — failing to journal one session must not
// break journaling for the next.
//
// Topics are exact-match (no wildcard) to keep the dependency on
// eventbus.Hub small. If new terminal topics get added we add them
// here.
func (j *Journaler) Run(ctx context.Context) {
	ended, cancelEnded := j.bus.Subscribe("session.ended", 16)
	defer cancelEnded()
	stopped, cancelStopped := j.bus.Subscribe("session.stopped", 16)
	defer cancelStopped()

	j.log.Info("session journaler running")
	for {
		select {
		case <-ctx.Done():
			j.log.Info("session journaler stopping")
			return
		case ev := <-ended:
			j.process(ctx, ev, "ended")
		case ev := <-stopped:
			j.process(ctx, ev, "stopped")
		}
	}
}

// process turns one eventbus.Event into a session_logs row.
func (j *Journaler) process(ctx context.Context, ev eventbus.Event, state string) {
	data, ok := ev.Data.(map[string]any)
	if !ok {
		j.log.Warn("journaler: event data is not a map", "topic", ev.Topic)
		return
	}
	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		j.log.Warn("journaler: event missing session_id", "topic", ev.Topic)
		return
	}
	sess, err := j.lookup.Get(ctx, sessionID)
	if err != nil {
		j.log.Warn("journaler: lookup session failed", "session_id", sessionID, "err", err)
		return
	}
	if sess.Cwd == "" {
		// Without cwd we can't anchor the row to a project.
		// Shouldn't happen for normal spawns; log and skip rather
		// than write an orphan journal entry.
		j.log.Warn("journaler: session has no cwd", "session_id", sessionID)
		return
	}
	if IsEphemeralCwd(sess.Cwd) {
		// Temp dirs (third-party consumers, tests) are not projects:
		// no journal entry, no transcript-summary LLM spend, no drift.
		j.log.Debug("journaler: ephemeral cwd — skipped", "session_id", sessionID, "cwd", sess.Cwd)
		return
	}
	inputs, err := j.lookup.History(ctx, sessionID, j.inputsLimit)
	if err != nil {
		// History errors are non-fatal — we just emit a metadata-only
		// entry. Common in tests + when the provider has no history
		// reader configured.
		j.log.Debug("journaler: history fetch failed", "session_id", sessionID, "err", err)
		inputs = nil
	}
	title, body := buildJournalBody(sess, state, inputs)

	// M18 — append an LLM-generated narrative when we have both a
	// transcript reader and a summariser configured. Both calls
	// are best-effort: failure logs + we ship the metadata-only
	// body so the journal never goes silent. transcriptSummary is
	// reused by the M-PA plan-drift detector below so we don't
	// re-summarise.
	var transcriptSummary string
	if j.summariser != nil {
		transcript, terr := j.lookup.TranscriptText(ctx, sessionID, j.transcriptMaxBytes)
		if terr != nil {
			j.log.Debug("journaler: transcript fetch failed", "session_id", sessionID, "err", terr)
		} else if strings.TrimSpace(transcript) != "" {
			// Skill usage tracking — cheap title grep, no LLM. Done
			// before the summary call so a summariser failure can't
			// starve the usage counters.
			if j.skillUsage != nil {
				j.skillUsage(ctx, transcript)
			}
			// LLM gets its own background context — the eventbus
			// goroutine that delivered session.ended is short-lived;
			// don't block it on a 60-180s reasoning model call.
			llmCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			summary, lerr := j.summariser.SummariseTranscript(llmCtx, transcript)
			cancel()
			if lerr != nil {
				j.log.Warn("journaler: llm summarise failed; metadata-only entry",
					"session_id", sessionID, "err", lerr)
			} else if s := strings.TrimSpace(summary); s != "" {
				transcriptSummary = s
				body = body + "\n**Agent activity summary**\n\n" + s + "\n"
			}
		}
	}

	entry := LogEntry{
		Cwd:       sess.Cwd,
		SessionID: sess.ID,
		Kind:      LogKindSessionSummary,
		Title:     title,
		Content:   body,
		UpdatedBy: AuthorSummarizer,
	}
	if _, err := j.docs.AppendLog(ctx, entry); err != nil {
		j.log.Warn("journaler: append failed", "session_id", sessionID, "cwd", sess.Cwd, "err", err)
		return
	}
	j.log.Info("journaler: appended session summary",
		"session_id", sessionID, "cwd", sess.Cwd, "title", title)

	// M-PA / Cortex Phase 3 — once the journal entry is safely
	// persisted, walk the project's blueprint and keep each
	// drift-eligible section current. Done AFTER the journal write so
	// a failure here can never block the basic journal flow.
	j.driftBlueprintSections(sess, transcriptSummary)
}

// driftBlueprintSections runs drift detection for every blueprint
// section the AI is responsible for keeping current:
//
//   - mode=ai      → detector runs; an unlocked doc is auto-updated
//     (PutDoc as agent), a human-locked one gets a proposal.
//   - mode=human   → detector runs; always a proposal.
//   - mode=scanner → skipped (mechanically rebuilt elsewhere).
//   - overview     → skipped (its own consolidation drafter owns it).
//
// This is the "goal/plan/tech-stack actually track the work" fix: the
// old behaviour (everything piles up as proposals) is preserved only
// where a human has explicitly claimed a section.
func (j *Journaler) driftBlueprintSections(sess SessionInfo, transcriptSummary string) {
	if j.planDetector == nil || strings.TrimSpace(transcriptSummary) == "" {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	sections, err := j.docs.ListSections(bgCtx, sess.Cwd)
	cancel()
	if err != nil {
		j.log.Debug("journaler: blueprint list failed — falling back to goal/plan drift",
			"cwd", sess.Cwd, "err", err)
		sections = defaultSections(sess.Cwd)
	}
	for _, sec := range sections {
		if sec.MaintainerMode == "scanner" || sec.Slug == SlugOverview {
			continue
		}
		j.maybeProposeDocDrift(sess, transcriptSummary, sec)
	}
}

// maybeProposeDocDrift runs the drift detector for one blueprint
// section and applies or proposes the update per the section's
// maintainer mode + lock state. Soft-fail at every step — detector
// errors, write errors, and empty-doc short-circuits are all logged
// but never bubbled up. Runs on its own background context because
// the LLM call can take minutes and the event delivery goroutine
// must not block.
func (j *Journaler) maybeProposeDocDrift(sess SessionInfo, transcriptSummary string, sec Section) {
	kind := Kind(sec.Slug)

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// P-D — a frozen project (paused/archived) is shelved; don't spend an LLM
	// call proposing doc updates the operator has signalled they don't want.
	if status, serr := j.docs.GetStatus(bgCtx, sess.Cwd); serr == nil && status.IsFrozen() {
		return
	}

	currentDoc, err := j.docs.GetDoc(bgCtx, sess.Cwd, kind)
	if err != nil {
		// ErrNotFound is the "fresh project" case — leave it for the
		// operator to seed the first content. Other errors are logged.
		if err != ErrNotFound {
			j.log.Debug("journaler: drift get-doc failed", "cwd", sess.Cwd, "kind", kind, "err", err)
		}
		return
	}
	if strings.TrimSpace(currentDoc.Content) == "" {
		return
	}

	logs, err := j.docs.ListLogs(bgCtx, sess.Cwd, j.driftJournalLookback)
	if err != nil {
		j.log.Debug("journaler: drift list-logs failed", "cwd", sess.Cwd, "err", err)
		logs = nil
	}

	out, derr := j.planDetector.DetectDrift(bgCtx, DriftInput{
		Kind:               kind,
		Cwd:                sess.Cwd,
		CurrentPlan:        currentDoc.Content,
		TranscriptSummary:  transcriptSummary,
		RecentJournal:      logs,
		SectionTitle:       sec.Title,
		SectionDescription: sec.Description,
		SectionPromptHint:  sec.PromptHint,
	})
	if derr != nil {
		j.log.Warn("journaler: drift detector failed", "cwd", sess.Cwd, "kind", kind, "err", derr)
		return
	}
	if !out.ShouldPropose {
		j.log.Debug("journaler: drift detector saw no change needed",
			"cwd", sess.Cwd, "kind", kind, "session_id", sess.ID)
		return
	}

	reason := strings.TrimSpace(out.Reason)
	if reason == "" {
		reason = "Drift detector flagged this session as a likely " + string(kind) + " update."
	}

	// AI-maintained + unlocked → apply directly so the doc actually
	// tracks the work. A human edit (updated_by=operator) or an
	// explicit human maintainer mode keeps the proposal gate.
	locked := currentDoc.UpdatedBy == AuthorOperator
	if sec.MaintainerMode == "ai" && !locked {
		if _, uerr := j.docs.PutDoc(bgCtx, sess.Cwd, kind, out.NewPlan, AuthorAgent); uerr != nil {
			j.log.Warn("journaler: drift auto-apply failed",
				"cwd", sess.Cwd, "kind", kind, "session_id", sess.ID, "err", uerr)
			return
		}
		j.log.Info("journaler: drift auto-applied",
			"cwd", sess.Cwd, "kind", kind, "session_id", sess.ID, "reason", reason)
		return
	}

	proposal, perr := j.docs.ProposeDoc(bgCtx, sess.Cwd, kind, out.NewPlan, reason, sess.ID)
	if perr != nil {
		j.log.Warn("journaler: drift propose failed",
			"cwd", sess.Cwd, "kind", kind, "session_id", sess.ID, "err", perr)
		return
	}
	j.log.Info("journaler: drift proposal filed",
		"cwd", sess.Cwd, "kind", kind, "session_id", sess.ID, "proposal_id", proposal.ID)
}

// buildJournalBody assembles a deterministic markdown summary. Kept
// in a separate function so unit tests don't need an event bus or a
// pgx pool.
func buildJournalBody(sess SessionInfo, state string, inputs []HistoryEntry) (string, string) {
	// Short id suffix is more friendly to read in a list than the
	// full UUID. Falls back to full id when shorter than 8 chars.
	shortID := sess.ID
	if len(shortID) > 8 {
		shortID = shortID[len(shortID)-8:]
	}

	title := fmt.Sprintf("Session %s — %s — %s",
		shortID, providerLabel(sess.ProviderID), state)

	var b strings.Builder
	b.WriteString("**Session metadata**\n\n")
	fmt.Fprintf(&b, "- id: `%s`\n", sess.ID)
	fmt.Fprintf(&b, "- provider: %s\n", providerLabel(sess.ProviderID))
	fmt.Fprintf(&b, "- cwd: `%s`\n", sess.Cwd)
	fmt.Fprintf(&b, "- started: %s\n", sess.StartedAt.Format(time.RFC3339))
	if sess.EndedAt != nil {
		fmt.Fprintf(&b, "- ended: %s\n", sess.EndedAt.Format(time.RFC3339))
		dur := sess.EndedAt.Sub(sess.StartedAt).Round(time.Second)
		fmt.Fprintf(&b, "- duration: %s\n", dur)
	}
	if sess.ExitCode != nil {
		fmt.Fprintf(&b, "- exit_code: %d\n", *sess.ExitCode)
	}
	b.WriteString("\n")

	if len(inputs) > 0 {
		b.WriteString("**Recent operator inputs**\n\n")
		// Inputs come newest-first; render newest-first here too
		// (most-relevant-on-top is what you want when skimming a
		// journal list later).
		for _, in := range inputs {
			text := compactOneLine(in.Text, 200)
			if text == "" {
				continue
			}
			fmt.Fprintf(&b, "- %s\n", text)
		}
		b.WriteString("\n")
	}

	b.WriteString("_This entry is an auto-generated session summary. Replace with a richer LLM-based summary by configuring a summarizer provider._\n")
	return title, b.String()
}

// providerLabel returns a friendly display name for known provider
// ids and the id verbatim for anything else.
func providerLabel(id string) string {
	switch id {
	case "claude":
		return "Claude"
	case "codex":
		return "Codex"
	case "gemini":
		return "Gemini"
	case "shell":
		return "Shell"
	default:
		return id
	}
}

// compactOneLine collapses internal whitespace into single spaces
// and truncates with an ellipsis. Long operator inputs (multi-line
// prompts pasted from a doc) otherwise blow up the journal list
// preview.
func compactOneLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Collapse whitespace runs.
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		switch r {
		case '\n', '\r', '\t':
			r = ' '
		}
		if r == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		b.WriteRune(r)
	}
	out := b.String()
	if len(out) > max {
		out = out[:max] + "…"
	}
	return out
}
