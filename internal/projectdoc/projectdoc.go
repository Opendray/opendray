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
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// LogEmbedder is the minimal embedding surface projectdoc needs
// to vector-index session_logs at append time + during the M-PB
// backfill loop. The memory subsystem's own Embedder implements
// this; defined here as a narrow local interface so projectdoc
// doesn't import internal/memory and risk circular dependencies.
type LogEmbedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Name() string
}

// Service is the CRUD-plus-policy surface for project docs +
// proposals + session logs. Constructed once per process and shared
// across HTTP handlers / MCP tools / spawn-time injector.
type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger

	// embedder is the optional M-PB hook for journal vector
	// indexing. When non-nil, AppendLog also embeds the entry; the
	// backfill goroutine uses the same one for catching up legacy
	// rows. Nil disables — append-time path stays unchanged.
	embedder LogEmbedder

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

// WithEmbedder installs the M-PB journal embedding hook. Returns
// the receiver for chained setup at composition-root time. Passing
// nil clears any previously-installed embedder.
func (s *Service) WithEmbedder(emb LogEmbedder) *Service {
	s.embedder = emb
	return s
}

// Embedder returns the currently-installed embedder, or nil.
// Exposed for the backfill goroutine + cross-layer search service
// which need to embed the user's query the same way.
func (s *Service) Embedder() LogEmbedder { return s.embedder }

// Pool exposes the underlying pgxpool for callers that need raw
// SQL access — specifically the backfill worker (which writes
// embedding columns) and the cross-layer search service (which
// runs vector queries against session_logs).
func (s *Service) Pool() *pgxpool.Pool { return s.pool }

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
		// M-PB — embed the new entry synchronously when an embedder
		// is wired. Failure here is non-fatal: the row is already
		// committed, and the backfill goroutine will catch up the
		// missing vector on its next sweep. Embedding logged so
		// operators can spot a misconfigured embedder.
		s.embedLogBestEffort(ctx, out)
	}
	return out, err
}

// embedLogBestEffort computes + persists an embedding for one
// freshly-appended journal entry. No-op when no embedder is
// configured. Soft-fails: a logged warning is the worst outcome.
func (s *Service) embedLogBestEffort(ctx context.Context, e LogEntry) {
	if s.embedder == nil {
		return
	}
	text := embedTextForLog(e)
	if text == "" {
		return
	}
	vecs, err := s.embedder.Embed(ctx, []string{text})
	if err != nil || len(vecs) == 0 || len(vecs[0]) == 0 {
		s.log.Debug("projectdoc: log embed at append-time failed (will retry via backfill)",
			"log_id", e.ID, "err", err)
		return
	}
	name := s.embedder.Name()
	if _, err := s.pool.Exec(ctx, `
		UPDATE session_logs
		   SET embedding = $1,
		       embedder = $2,
		       embedding_at = NOW()
		 WHERE id = $3`, pgvecString(vecs[0]), name, e.ID); err != nil {
		s.log.Debug("projectdoc: log embed write-back failed",
			"log_id", e.ID, "err", err)
	}
}

// embedTextForLog assembles the string passed to the embedder.
// "title — content" mirrors how the spawn-time banner renders the
// entry, so query semantics match what the agent will eventually
// see in its system prompt.
func embedTextForLog(e LogEntry) string {
	title := strings.TrimSpace(e.Title)
	content := strings.TrimSpace(e.Content)
	if title == "" && content == "" {
		return ""
	}
	if title == "" {
		return content
	}
	if content == "" {
		return title
	}
	return title + " — " + content
}

// pgvecString encodes a float32 slice into pgvector's bracketed
// literal form. We feed it as a parameter rather than building a
// SQL fragment so pgx still treats it as a typed argument (no
// injection risk). Same encoding pattern lives in
// memory/store_pgvector.go for the memories table.
func pgvecString(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.Grow(2 + len(v)*8)
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// StaleJournalEntries returns journal entries that are candidates
// for cleanup: older than `olderThan`, kind=session_summary, and
// NOT referenced by any pending memory_conflicts row. The result
// is sorted oldest first so callers can present a prune list.
//
// The intent is the M-PC "cleaner extension to layer 4" — give
// operators a one-shot view of accumulated noise without forcing
// a destructive default. UI bulk-delete + a dedicated cleanup
// tab can layer on top later; for now this is just the query.
func (s *Service) StaleJournalEntries(ctx context.Context, cwd string, olderThan time.Duration) ([]LogEntry, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, ErrEmptyCwd
	}
	if olderThan <= 0 {
		olderThan = 90 * 24 * time.Hour
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	rows, err := s.pool.Query(ctx, `
		SELECT sl.id, sl.cwd, COALESCE(sl.session_id, ''), sl.kind,
		       sl.title, sl.content, sl.updated_by, sl.created_at
		  FROM session_logs sl
		 WHERE sl.cwd = $1
		   AND sl.kind = 'session_summary'
		   AND sl.created_at < $2
		   AND NOT EXISTS (
		       SELECT 1 FROM memory_conflicts mc
		        WHERE mc.cwd = sl.cwd
		          AND mc.status = 'pending'
		          AND ((mc.layer_a = 'journal' AND mc.ref_a = sl.id)
		            OR (mc.layer_b = 'journal' AND mc.ref_b = sl.id))
		   )
		 ORDER BY sl.created_at ASC
		 LIMIT 200`, cwd, cutoff)
	if err != nil {
		return nil, fmt.Errorf("projectdoc: stale journal: %w", err)
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

// ResetCwdOptions controls which tables/kinds the reset wipes.
type ResetCwdOptions struct {
	// IncludeScannerDocs deletes tech_stack + recent_activity rows
	// too. Default false: scanner doc kinds auto-rebuild on the
	// next spawn anyway, and operators usually want to keep them
	// (they're objective project facts, not user content).
	IncludeScannerDocs bool

	// IncludeCleanupDecisions deletes memory_cleanup_decisions rows
	// keyed to this cwd. Defaults to true since they're tightly
	// scoped to the cwd's memories — kept as an option in case a
	// future caller wants to preserve the audit trail.
	IncludeCleanupDecisions bool
}

// ResetCounts is what ResetCwd returns: counts of rows deleted per
// table so the UI can show "deleted X docs, Y journal entries, …".
type ResetCounts struct {
	ProjectDocs      int64 `json:"project_docs"`
	Proposals        int64 `json:"project_doc_proposals"`
	SessionLogs      int64 `json:"session_logs"`
	CleanupDecisions int64 `json:"memory_cleanup_decisions"`
}

// ResetCwd wipes per-cwd project memory state in a single
// transaction. Always deletes session_logs + project_doc_proposals
// (no use without their parent project_docs anyway) + operator-
// editable docs (goal/plan). Optionally also wipes scanner-managed
// docs (tech_stack/recent_activity — auto-rebuild on next spawn)
// and the M13 cleanup decisions queue.
//
// memories rows are NOT deleted here — they live in the memory
// subsystem with its own scope_key indexing. Callers (the
// `/project-docs/reset` HTTP handler) chain memory.DeleteByScope
// when the operator opts in.
func (s *Service) ResetCwd(ctx context.Context, cwd string, opts ResetCwdOptions) (ResetCounts, error) {
	if cwd == "" {
		return ResetCounts{}, fmt.Errorf("projectdoc: reset: cwd required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ResetCounts{}, fmt.Errorf("projectdoc: reset begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var counts ResetCounts

	// project_docs — always wipe goal+plan; scanner docs gated by opt.
	var docCmd pgconn.CommandTag
	if opts.IncludeScannerDocs {
		docCmd, err = tx.Exec(ctx, `DELETE FROM project_docs WHERE cwd = $1`, cwd)
	} else {
		docCmd, err = tx.Exec(ctx,
			`DELETE FROM project_docs WHERE cwd = $1 AND kind IN ('goal','plan')`, cwd)
	}
	if err != nil {
		return ResetCounts{}, fmt.Errorf("projectdoc: reset docs: %w", err)
	}
	counts.ProjectDocs = docCmd.RowsAffected()

	propCmd, err := tx.Exec(ctx,
		`DELETE FROM project_doc_proposals WHERE cwd = $1`, cwd)
	if err != nil {
		return ResetCounts{}, fmt.Errorf("projectdoc: reset proposals: %w", err)
	}
	counts.Proposals = propCmd.RowsAffected()

	logCmd, err := tx.Exec(ctx,
		`DELETE FROM session_logs WHERE cwd = $1`, cwd)
	if err != nil {
		return ResetCounts{}, fmt.Errorf("projectdoc: reset logs: %w", err)
	}
	counts.SessionLogs = logCmd.RowsAffected()

	if opts.IncludeCleanupDecisions {
		cdCmd, err := tx.Exec(ctx,
			`DELETE FROM memory_cleanup_decisions
			 WHERE memory_scope = 'project' AND memory_scope_key = $1`, cwd)
		if err != nil {
			return ResetCounts{}, fmt.Errorf("projectdoc: reset cleanup decisions: %w", err)
		}
		counts.CleanupDecisions = cdCmd.RowsAffected()
	}

	if err := tx.Commit(ctx); err != nil {
		return ResetCounts{}, fmt.Errorf("projectdoc: reset commit: %w", err)
	}
	return counts, nil
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
//
// This is the legacy entry point — equivalent to
// RenderForSpawnWithBudget(ctx, cwd, recentLogs, 0). Kept stable
// so existing callers don't need an immediate update.
func (s *Service) RenderForSpawn(ctx context.Context, cwd string, recentLogs int) (string, error) {
	return s.RenderForSpawnWithBudget(ctx, cwd, recentLogs, 0)
}

// RenderForSpawnWithBudget is the M-PB token-budgeted renderer.
// maxBytes <= 0 disables the cap (matches legacy RenderForSpawn).
// When set, sections are appended in priority order and rendering
// halts once the budget is exceeded, with a "truncated" notice
// added so the agent knows the prompt is incomplete.
//
// Priority order is fixed to favour the things an agent acts on:
//  1. plan          — most useful for picking up where work left off
//  2. tech_stack    — orients the agent in the codebase
//  3. goal          — long-term direction (rare changes; smaller body)
//  4. recent_activity — git narrative (large; lower priority by design)
//  5. journal       — episodic detail; takes whatever budget is left
//
// 4 KiB ≈ 1k tokens is a sensible default for most operators; the
// catalog adapter can pass its own value once we expose it.
func (s *Service) RenderForSpawnWithBudget(ctx context.Context, cwd string, recentLogs, maxBytes int) (string, error) {
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
	header := "## Project context (cross-agent shared, read-only)\n\n"
	footer := "If your work changes the goal or plan, do **not** silently overwrite them. Use the `project_goal_set` / `project_plan_set` MCP tools — they file a proposal that the operator approves before the live doc updates.\n"
	b.WriteString(header)

	// Reserve room for the footer when a budget is active; without
	// the footer the agent loses the proposal-flow nudge.
	footerReserve := 0
	truncated := false
	if maxBytes > 0 {
		footerReserve = len(footer)
	}

	appendSection := func(title, body string) {
		if body == "" || truncated {
			return
		}
		section := "### " + title + "\n\n" + body + "\n\n"
		if maxBytes > 0 && b.Len()+len(section)+footerReserve > maxBytes {
			truncated = true
			return
		}
		b.WriteString(section)
	}

	appendSection("Project plan", plan)
	appendSection("Tech stack & structure", techStack)
	appendSection("Project goal", goal)
	appendSection("Recent activity", recentActivity)

	if len(logs) > 0 && !truncated {
		jb, jTrunc := renderJournalSection(logs, maxBytes-b.Len()-footerReserve)
		if jb != "" {
			b.WriteString(jb)
		}
		if jTrunc {
			truncated = true
		}
	}

	if truncated {
		b.WriteString("_(banner truncated to fit spawn-prompt budget — visit /memory/project for the full set)_\n\n")
	}

	b.WriteString(footer)
	return b.String(), nil
}

// renderJournalSection assembles the journal block, stopping when
// it would exceed remaining bytes. Returns the section text + a
// "we hit the limit" flag so the caller can append a truncation
// note. remaining <=0 disables the cap.
func renderJournalSection(logs []LogEntry, remaining int) (string, bool) {
	var b strings.Builder
	b.WriteString("### Recent journal\n\n")
	truncated := false
	// logs are newest-first; render oldest-first so chronology reads
	// top-to-bottom.
	for i := len(logs) - 1; i >= 0; i-- {
		e := logs[i]
		body := strings.TrimSpace(e.Content)
		if len(body) > 600 {
			body = body[:600] + "…"
		}
		var line strings.Builder
		line.WriteString("- ")
		if e.Title != "" {
			line.WriteString("**")
			line.WriteString(e.Title)
			line.WriteString("** — ")
		}
		line.WriteString(body)
		line.WriteString("\n")
		if remaining > 0 && b.Len()+line.Len() > remaining {
			truncated = true
			break
		}
		b.WriteString(line.String())
	}
	b.WriteString("\n")
	// If we wrote nothing past the heading, drop the section entirely.
	if strings.Count(b.String(), "\n") <= 3 {
		return "", truncated
	}
	return b.String(), truncated
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
