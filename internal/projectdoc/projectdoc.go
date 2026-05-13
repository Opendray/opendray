// Package projectdoc owns memory layers 2-4 of the unified
// cross-agent memory architecture:
//
//	layer 2: project goal      — single markdown doc per cwd
//	layer 3: project plan      — single markdown doc per cwd
//	layer 4: session journal   — append-only chronological log
//
// Layer 1 (project rules / CLAUDE.md) is in git, owned by the
// operator outside this package. Layer 5 (discrete facts) lives in
// internal/memory.
//
// Why this package is separate from internal/memory: the data
// shapes are fundamentally different. memories are short discrete
// claims that get top-K-relevant ranked; project_docs are
// replace-in-place document bodies; session_logs are append-only
// timeline rows. Forcing them into one schema made the API confusing
// (do you Search a goal? Top-K a session log?), so we keep them
// distinct but compose them at injection time (catalog/adapter.go).
//
// All persistence lives behind a Service so callers (HTTP handlers,
// MCP tools, spawn-time injector) don't talk to pgxpool directly.
package projectdoc

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Kind enumerates the project_docs.kind column. Currently goal /
// plan; future kinds (rationale, decision_index) extend by
// loosening the DB CHECK + adding constants here.
type Kind string

const (
	KindGoal           Kind = "goal"
	KindPlan           Kind = "plan"
	KindTechStack      Kind = "tech_stack"      // M16b — scanner-managed
	KindRecentActivity Kind = "recent_activity" // M16c — git-summary-managed
)

// ValidKind returns true for the supported document kinds.
// Callers should use this rather than a hardcoded switch so the
// list stays in one place.
func ValidKind(k Kind) bool {
	switch k {
	case KindGoal, KindPlan, KindTechStack, KindRecentActivity:
		return true
	}
	return false
}

// LogKind enumerates session_logs.kind. session_summary covers the
// M8 auto-generated case; manual is operator-typed via UI; decision
// is the ADR-style entry the M7 decision_record MCP tool writes.
type LogKind string

const (
	LogKindSessionSummary LogKind = "session_summary"
	LogKindManual         LogKind = "manual"
	LogKindDecision       LogKind = "decision"
)

func ValidLogKind(k LogKind) bool {
	switch k {
	case LogKindSessionSummary, LogKindManual, LogKindDecision:
		return true
	}
	return false
}

// Author classifies the writer of a row. Surfaced in the UI so the
// operator can tell apart "agent proposed and I approved" vs
// "I wrote this myself" at a glance.
type Author string

const (
	AuthorOperator   Author = "operator"
	AuthorAgent      Author = "agent"
	AuthorSummarizer Author = "summarizer"
	AuthorManual     Author = "manual"
	AuthorScanner    Author = "scanner" // M16 — project scanner
)

// Doc represents one row from project_docs — the live state of a
// goal/plan document for a single project.
type Doc struct {
	ID        string    `json:"id"`
	Cwd       string    `json:"cwd"`
	Kind      Kind      `json:"kind"`
	Content   string    `json:"content"`
	UpdatedBy Author    `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Proposal represents a pending or decided change request from an
// agent. The agent-side MCP tool always creates a Proposal; only the
// approve/reject API path mutates the live Doc.
type Proposal struct {
	ID                string     `json:"id"`
	Cwd               string     `json:"cwd"`
	Kind              Kind       `json:"kind"`
	ProposedContent   string     `json:"proposed_content"`
	ProposedBySession string     `json:"proposed_by_session,omitempty"`
	Reason            string     `json:"reason"`
	Decision          string     `json:"decision,omitempty"` // "approved" | "rejected" | ""
	DecidedAt         *time.Time `json:"decided_at,omitempty"`
	PriorContent      string     `json:"prior_content,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// LogEntry represents one row from session_logs.
type LogEntry struct {
	ID        string    `json:"id"`
	Cwd       string    `json:"cwd"`
	SessionID string    `json:"session_id,omitempty"`
	Kind      LogKind   `json:"kind"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	UpdatedBy Author    `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Sentinel errors.
var (
	ErrNotFound       = errors.New("projectdoc: not found")
	ErrAlreadyDecided = errors.New("projectdoc: proposal already decided")
	ErrInvalidKind    = errors.New("projectdoc: invalid kind")
	ErrInvalidLogKind = errors.New("projectdoc: invalid log kind")
	ErrEmptyCwd       = errors.New("projectdoc: cwd is required")
)

// Service is the CRUD-plus-policy surface for project docs +
// proposals + session logs. Constructed once per process and shared
// across HTTP handlers / MCP tools / spawn-time injector.
type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger

	// mirrorDisabled turns off the on-write `.opendray/*.md` mirror.
	// Tests flip this on via DisableMirror() so they don't dirty
	// arbitrary directories on the host.
	mirrorDisabled bool
}

// NewService wires a Service against an existing pgx pool.
func NewService(pool *pgxpool.Pool, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{pool: pool, log: log.With("component", "projectdoc")}
}

// ─── docs (goal / plan) ────────────────────────────────────────

// GetDoc returns the current document for (cwd, kind). Returns
// ErrNotFound when there's no row yet — caller can treat that as
// "empty doc" rather than a hard error.
func (s *Service) GetDoc(ctx context.Context, cwd string, kind Kind) (Doc, error) {
	if !ValidKind(kind) {
		return Doc{}, ErrInvalidKind
	}
	if strings.TrimSpace(cwd) == "" {
		return Doc{}, ErrEmptyCwd
	}
	row := s.pool.QueryRow(ctx, `
		SELECT id, cwd, kind, content, updated_by, created_at, updated_at
		  FROM project_docs
		 WHERE cwd = $1 AND kind = $2`, cwd, string(kind))
	d, err := scanDoc(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Doc{}, ErrNotFound
	}
	return d, err
}

// ListDocsForCwd returns all docs (goal + plan, if present) for one
// cwd in a single query. UI uses this to render the project page.
func (s *Service) ListDocsForCwd(ctx context.Context, cwd string) ([]Doc, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, ErrEmptyCwd
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, cwd, kind, content, updated_by, created_at, updated_at
		  FROM project_docs
		 WHERE cwd = $1
		 ORDER BY kind ASC`, cwd)
	if err != nil {
		return nil, fmt.Errorf("projectdoc: list docs: %w", err)
	}
	defer rows.Close()
	var out []Doc
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// PutDoc upserts the (cwd, kind) document. The whole `content`
// string replaces what was there — there is no incremental patch
// surface. Operator UI binds the field to a markdown textarea.
func (s *Service) PutDoc(ctx context.Context, cwd string, kind Kind, content string, author Author) (Doc, error) {
	if !ValidKind(kind) {
		return Doc{}, ErrInvalidKind
	}
	if strings.TrimSpace(cwd) == "" {
		return Doc{}, ErrEmptyCwd
	}
	if author == "" {
		author = AuthorOperator
	}
	id := newID("pd_")
	row := s.pool.QueryRow(ctx, `
		INSERT INTO project_docs (id, cwd, kind, content, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (cwd, kind) DO UPDATE
		   SET content    = EXCLUDED.content,
		       updated_by = EXCLUDED.updated_by,
		       updated_at = NOW()
		RETURNING id, cwd, kind, content, updated_by, created_at, updated_at`,
		id, cwd, string(kind), content, string(author))
	d, err := scanDoc(row)
	if err == nil {
		s.mirrorBestEffort(ctx, cwd)
	}
	return d, err
}

// ─── proposals ─────────────────────────────────────────────────

// ProposeDoc records an agent's proposed change. Decision 3 from
// the design discussion: agents cannot directly overwrite goal /
// plan; the change lands here in 'pending' state until the
// operator approves via ApproveProposal.
func (s *Service) ProposeDoc(ctx context.Context, cwd string, kind Kind, proposedContent, reason, sessionID string) (Proposal, error) {
	if !ValidKind(kind) {
		return Proposal{}, ErrInvalidKind
	}
	if strings.TrimSpace(cwd) == "" {
		return Proposal{}, ErrEmptyCwd
	}
	id := newID("pdp_")
	var byID any
	if sessionID != "" {
		byID = sessionID
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO project_doc_proposals (id, cwd, kind, proposed_content, proposed_by_session, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, cwd, kind, proposed_content, COALESCE(proposed_by_session, ''),
		          reason, COALESCE(decision, ''), decided_at, COALESCE(prior_content, ''), created_at`,
		id, cwd, string(kind), proposedContent, byID, reason)
	return scanProposal(row)
}

// ListPendingProposals returns un-decided proposals for one cwd,
// newest first. Used by the operator inbox in the UI. If cwd is
// empty, returns every pending proposal across all projects (admin
// "everything I owe a decision on" view).
func (s *Service) ListPendingProposals(ctx context.Context, cwd string) ([]Proposal, error) {
	var rows pgx.Rows
	var err error
	if cwd == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, cwd, kind, proposed_content, COALESCE(proposed_by_session, ''),
			       reason, COALESCE(decision, ''), decided_at, COALESCE(prior_content, ''), created_at
			  FROM project_doc_proposals
			 WHERE decided_at IS NULL
			 ORDER BY created_at DESC`)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, cwd, kind, proposed_content, COALESCE(proposed_by_session, ''),
			       reason, COALESCE(decision, ''), decided_at, COALESCE(prior_content, ''), created_at
			  FROM project_doc_proposals
			 WHERE cwd = $1 AND decided_at IS NULL
			 ORDER BY created_at DESC`, cwd)
	}
	if err != nil {
		return nil, fmt.Errorf("projectdoc: list proposals: %w", err)
	}
	defer rows.Close()
	var out []Proposal
	for rows.Next() {
		p, err := scanProposal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ApproveProposal merges the proposed content into project_docs
// and stamps the proposal row 'approved' with the prior content
// captured for audit. Idempotent in the sense that re-approving an
// already-decided proposal returns ErrAlreadyDecided.
//
// The write happens in a single transaction so we never end up
// with an approved proposal pointing at a doc that didn't get the
// update.
func (s *Service) ApproveProposal(ctx context.Context, id string) (Doc, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Doc{}, fmt.Errorf("projectdoc: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, cwd, kind, proposed_content, COALESCE(proposed_by_session, ''),
		       reason, COALESCE(decision, ''), decided_at, COALESCE(prior_content, ''), created_at
		  FROM project_doc_proposals
		 WHERE id = $1`, id)
	p, err := scanProposal(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Doc{}, ErrNotFound
	}
	if err != nil {
		return Doc{}, err
	}
	if p.DecidedAt != nil {
		return Doc{}, ErrAlreadyDecided
	}

	// Capture the prior live content so the proposal row preserves a
	// "before" snapshot for audit. Missing row = empty prior.
	var prior string
	_ = tx.QueryRow(ctx,
		`SELECT content FROM project_docs WHERE cwd=$1 AND kind=$2`,
		p.Cwd, string(p.Kind)).Scan(&prior)

	newDocID := newID("pd_")
	docRow := tx.QueryRow(ctx, `
		INSERT INTO project_docs (id, cwd, kind, content, updated_by)
		VALUES ($1, $2, $3, $4, 'agent')
		ON CONFLICT (cwd, kind) DO UPDATE
		   SET content    = EXCLUDED.content,
		       updated_by = 'agent',
		       updated_at = NOW()
		RETURNING id, cwd, kind, content, updated_by, created_at, updated_at`,
		newDocID, p.Cwd, string(p.Kind), p.ProposedContent)
	d, err := scanDoc(docRow)
	if err != nil {
		return Doc{}, fmt.Errorf("projectdoc: upsert doc: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE project_doc_proposals
		   SET decision = 'approved',
		       decided_at = NOW(),
		       prior_content = $1
		 WHERE id = $2`, prior, id); err != nil {
		return Doc{}, fmt.Errorf("projectdoc: mark approved: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Doc{}, fmt.Errorf("projectdoc: commit: %w", err)
	}
	s.mirrorBestEffort(ctx, d.Cwd)
	return d, nil
}

// RejectProposal stamps the proposal 'rejected' without touching
// the live doc. The proposal row stays around for audit history.
func (s *Service) RejectProposal(ctx context.Context, id string) error {
	cmd, err := s.pool.Exec(ctx, `
		UPDATE project_doc_proposals
		   SET decision = 'rejected', decided_at = NOW()
		 WHERE id = $1 AND decided_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("projectdoc: reject: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		// Either the row doesn't exist or it was already decided.
		// Distinguish so the UI can show the right error.
		var probe int
		err := s.pool.QueryRow(ctx,
			`SELECT 1 FROM project_doc_proposals WHERE id = $1`, id).Scan(&probe)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return ErrAlreadyDecided
	}
	return nil
}

// ─── session_logs ──────────────────────────────────────────────

// AppendLog adds one row. Returns the persisted entry so the UI
// can show it without a refetch. cwd + kind + content required;
// session_id and title optional.
func (s *Service) AppendLog(ctx context.Context, e LogEntry) (LogEntry, error) {
	if strings.TrimSpace(e.Cwd) == "" {
		return LogEntry{}, ErrEmptyCwd
	}
	if e.Kind == "" {
		e.Kind = LogKindManual
	}
	if !ValidLogKind(e.Kind) {
		return LogEntry{}, ErrInvalidLogKind
	}
	if e.UpdatedBy == "" {
		switch e.Kind {
		case LogKindSessionSummary:
			e.UpdatedBy = AuthorSummarizer
		case LogKindManual:
			e.UpdatedBy = AuthorOperator
		case LogKindDecision:
			e.UpdatedBy = AuthorAgent
		default:
			e.UpdatedBy = AuthorOperator
		}
	}
	id := newID("sl_")
	var sessID any
	if e.SessionID != "" {
		sessID = e.SessionID
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO session_logs (id, cwd, session_id, kind, title, content, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, cwd, COALESCE(session_id, ''), kind, title, content, updated_by, created_at`,
		id, e.Cwd, sessID, string(e.Kind), e.Title, e.Content, string(e.UpdatedBy))
	out, err := scanLog(row)
	if err == nil {
		s.mirrorBestEffort(ctx, out.Cwd)
	}
	return out, err
}

// ListLogs returns chronological journal entries newest first.
// limit ≤ 0 falls back to 50; values >200 are clamped.
func (s *Service) ListLogs(ctx context.Context, cwd string, limit int) ([]LogEntry, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, ErrEmptyCwd
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, cwd, COALESCE(session_id, ''), kind, title, content, updated_by, created_at
		  FROM session_logs
		 WHERE cwd = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, cwd, limit)
	if err != nil {
		return nil, fmt.Errorf("projectdoc: list logs: %w", err)
	}
	defer rows.Close()
	var out []LogEntry
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// DeleteLog removes a log entry. Used by the UI's "delete" action.
// Returns ErrNotFound if the id doesn't exist.
func (s *Service) DeleteLog(ctx context.Context, id string) error {
	cmd, err := s.pool.Exec(ctx, `DELETE FROM session_logs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("projectdoc: delete log: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── spawn-time injection ──────────────────────────────────────

// RenderForSpawn produces a single markdown banner combining the
// project goal, plan, and the most recent session-log entries.
// Designed to be prepended to the agent's system prompt alongside
// the ambient memory banner (top-K facts) so the agent boots into a
// session already aware of the long-term arc (goal), the current WIP
// (plan), and the last few sessions' decisions (journal).
//
// Returns "" when there is nothing to inject — no goal, no plan, no
// logs. The caller treats empty string as "skip injection" so a
// fresh project does not get spammed with empty headers.
//
// recentLogs caps the number of session-log entries rendered;
// values ≤ 0 fall back to 5. Each entry shows title + content; the
// banner stops growing past ~6KB even at high recentLogs, so the
// spawn cost is bounded.
func (s *Service) RenderForSpawn(ctx context.Context, cwd string, recentLogs int) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		return "", nil
	}
	if recentLogs <= 0 {
		recentLogs = 5
	}

	docs, err := s.ListDocsForCwd(ctx, cwd)
	if err != nil {
		return "", fmt.Errorf("projectdoc: render spawn docs: %w", err)
	}
	logs, err := s.ListLogs(ctx, cwd, recentLogs)
	if err != nil {
		return "", fmt.Errorf("projectdoc: render spawn logs: %w", err)
	}

	var goal, plan, techStack, recentActivity string
	for _, d := range docs {
		switch d.Kind {
		case KindGoal:
			goal = strings.TrimSpace(d.Content)
		case KindPlan:
			plan = strings.TrimSpace(d.Content)
		case KindTechStack:
			techStack = strings.TrimSpace(d.Content)
		case KindRecentActivity:
			recentActivity = strings.TrimSpace(d.Content)
		}
	}

	if goal == "" && plan == "" && techStack == "" && recentActivity == "" && len(logs) == 0 {
		return "", nil
	}

	// M23 — banner is consumed by the agent, not the operator. Keep
	// structural headers (the LLM uses them to separate sections),
	// drop human-courtesy framing (intro essays, "auto-generated"
	// markers, last-scanned timestamps). Saves ~15-25% spawn tokens.
	var b strings.Builder
	b.WriteString("## Project context (cross-agent shared, read-only)\n\n")

	if techStack != "" {
		b.WriteString("### Tech stack & structure\n\n")
		b.WriteString(techStack)
		b.WriteString("\n\n")
	}
	if recentActivity != "" {
		b.WriteString("### Recent activity\n\n")
		b.WriteString(recentActivity)
		b.WriteString("\n\n")
	}
	if goal != "" {
		b.WriteString("### Project goal\n\n")
		b.WriteString(goal)
		b.WriteString("\n\n")
	}
	if plan != "" {
		b.WriteString("### Project plan\n\n")
		b.WriteString(plan)
		b.WriteString("\n\n")
	}
	if len(logs) > 0 {
		b.WriteString("### Recent journal\n\n")
		// logs are newest-first; render oldest-first inside the banner
		// so the chronology reads naturally top-to-bottom.
		for i := len(logs) - 1; i >= 0; i-- {
			e := logs[i]
			b.WriteString("- ")
			if e.Title != "" {
				b.WriteString("**")
				b.WriteString(e.Title)
				b.WriteString("** — ")
			}
			body := strings.TrimSpace(e.Content)
			// Keep each journal line compact — the goal here is "remind
			// the agent" not "replay full session summaries". 600 chars
			// is roughly two paragraphs; longer entries stay readable
			// when the operator drills into the journal page.
			if len(body) > 600 {
				body = body[:600] + "…"
			}
			b.WriteString(body)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("If your work changes the goal or plan, do **not** silently overwrite them. Use the `project_goal_set` / `project_plan_set` MCP tools — they file a proposal that the operator approves before the live doc updates.\n")

	return b.String(), nil
}

// ─── scanners ──────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDoc(row rowScanner) (Doc, error) {
	var d Doc
	var kindStr, byStr string
	if err := row.Scan(&d.ID, &d.Cwd, &kindStr, &d.Content, &byStr, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return Doc{}, err
	}
	d.Kind = Kind(kindStr)
	d.UpdatedBy = Author(byStr)
	return d, nil
}

func scanProposal(row rowScanner) (Proposal, error) {
	var (
		p        Proposal
		kindStr  string
		decision string
		dec      *time.Time
	)
	if err := row.Scan(
		&p.ID, &p.Cwd, &kindStr, &p.ProposedContent,
		&p.ProposedBySession, &p.Reason,
		&decision, &dec, &p.PriorContent, &p.CreatedAt,
	); err != nil {
		return Proposal{}, err
	}
	p.Kind = Kind(kindStr)
	p.Decision = decision
	p.DecidedAt = dec
	return p, nil
}

func scanLog(row rowScanner) (LogEntry, error) {
	var l LogEntry
	var kindStr, byStr string
	if err := row.Scan(&l.ID, &l.Cwd, &l.SessionID, &kindStr, &l.Title, &l.Content, &byStr, &l.CreatedAt); err != nil {
		return LogEntry{}, err
	}
	l.Kind = LogKind(kindStr)
	l.UpdatedBy = Author(byStr)
	return l, nil
}

// mirrorBestEffort calls Mirror but never returns an error to the
// caller — failures here are logged and swallowed because the DB
// write that triggered the mirror already succeeded; rolling back
// because the operator's filesystem refused a write would lose
// real data. Gated by mirrorDisabled so tests + integration suites
// can opt out cleanly.
func (s *Service) mirrorBestEffort(ctx context.Context, cwd string) {
	if s.mirrorDisabled || cwd == "" {
		return
	}
	if err := s.Mirror(ctx, cwd); err != nil {
		s.log.Debug("projectdoc mirror failed (non-fatal)", "cwd", cwd, "err", err)
	}
}

// DisableMirror turns off the on-write file mirror. Used by unit
// tests that don't want side effects on the host filesystem.
func (s *Service) DisableMirror() { s.mirrorDisabled = true }

// newID returns a short alphanumeric id with a typed prefix. Same
// 14-byte base32 entropy as the rest of the codebase (memory_capture
// rules, injection profiles, etc.) — operators can paste it into
// audit queries.
func newID(prefix string) string {
	var b [14]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("projectdoc: rand: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return prefix + strings.ToLower(enc)
}
