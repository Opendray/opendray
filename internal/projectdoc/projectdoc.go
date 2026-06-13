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

// Kind is the project_docs.kind column. Since the blueprint system
// (Cortex Phase 3, migration 0046) it is a per-project SECTION SLUG —
// the constants below are the default blueprint's slugs plus the
// reserved global kb_* pages, not a closed enum. Validation is
// syntax-level (ValidKind) plus a service-layer blueprint check on
// writes.
type Kind string

const (
	KindGoal           Kind = "goal"
	KindPlan           Kind = "plan"
	KindTechStack      Kind = "tech_stack"      // M16b — scanner-managed
	KindRecentActivity Kind = "recent_activity" // M16c — git-summary-managed
	// KindOverview is the project's rich, AI-maintained OFFICIAL document — the
	// comprehensive page a developer reads to understand the whole project.
	// Per-project (Notes), AI-drafted, human-lockable.
	KindOverview Kind = "overview"
	// Knowledge pages (AI-drafted, human-lockable) live under GlobalCwd.
	// They are CROSS-PROJECT only (Experience Flywheel) — there is no
	// per-project handbook; per-project docs are goal/plan/tech/journal above.
	KindInfrastructure Kind = "kb_infrastructure"
	KindConventions    Kind = "kb_conventions"
	KindLessons        Kind = "kb_lessons"
	KindReusable       Kind = "kb_reusable" // reusable features/components to lift into new projects
	// KindHandbook is RETIRED — kept only so the DB CHECK + legacy rows stay
	// valid until migration 0042 clears them. Not created or injected.
	KindHandbook Kind = "kb_handbook"
)

// GlobalCwd is the sentinel cwd under which cross-project (global) KB pages are
// stored, so they reuse the existing (cwd, kind) document model unchanged.
const GlobalCwd = "__global__"

// IsGlobalKBKind reports whether k is one of the fixed cross-project
// Knowledge pages stored under GlobalCwd. The retired kb_handbook is
// intentionally excluded.
func IsGlobalKBKind(k Kind) bool {
	switch k {
	case KindInfrastructure, KindConventions, KindLessons, KindReusable:
		return true
	}
	return false
}

// ValidKind is the syntax-level check: either a global knowledge page
// slug (kb_*, extensible since the knowledge blueprint) or a
// well-formed per-project section slug. Whether a slug actually exists
// in its blueprint is checked on the write paths (PutDoc / ProposeDoc)
// — reads stay permissive so content of a removed-then-re-added
// section is never unreachable.
func ValidKind(k Kind) bool {
	return ValidGlobalKBSlug(string(k)) || ValidSectionSlug(string(k))
}

// validateKindForCwd enforces the kind↔cwd pairing: kb_* pages live
// only under GlobalCwd, and per-project slugs never under it.
func validateKindForCwd(cwd string, k Kind) error {
	if !ValidKind(k) {
		return ErrInvalidKind
	}
	isKB := ValidGlobalKBSlug(string(k))
	if isKB != (cwd == GlobalCwd) {
		if isKB {
			return fmt.Errorf("%w: %s is a global knowledge page (cwd must be %s)", ErrInvalidKind, k, GlobalCwd)
		}
		return fmt.Errorf("%w: %s is not a global knowledge page", ErrInvalidKind, k)
	}
	return nil
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

	// spawnMode resolves the operator's spawn injection mode at render
	// time ("full"|"lean", Cortex settings). Nil = full (legacy).
	spawnMode SpawnModeSource

	// onStatusChange fires after a successful lifecycle transition
	// (see WithStatusChangeHook). The app bridges project archive /
	// unarchive to the memory store through this. Nil = no bridge.
	onStatusChange func(ctx context.Context, cwd string, old, new ProjectStatus)
}

// SpawnModeSource resolves the spawn injection mode per render —
// "full" injects everything inject-flagged, "lean" injects guardrails
// plus a compact index and lets agents fetch the rest on demand.
type SpawnModeSource func(ctx context.Context) string

// WithSpawnMode installs the spawn-mode resolver (Cortex settings).
func (s *Service) WithSpawnMode(src SpawnModeSource) *Service {
	s.spawnMode = src
	return s
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
	if strings.TrimSpace(cwd) == "" {
		return Doc{}, ErrEmptyCwd
	}
	if err := s.validateWriteTarget(ctx, cwd, kind); err != nil {
		return Doc{}, err
	}
	if author == "" {
		author = AuthorOperator
	}
	id := newID("pd_")
	// Clear the embedding on a content change so a stale vector never
	// lingers: embedDocBestEffort repopulates it synchronously below, and
	// if that fails the NULL makes the backfill loop retry.
	row := s.pool.QueryRow(ctx, `
		INSERT INTO project_docs (id, cwd, kind, content, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (cwd, kind) DO UPDATE
		   SET content      = EXCLUDED.content,
		       updated_by   = EXCLUDED.updated_by,
		       updated_at   = NOW(),
		       embedding    = NULL,
		       embedder     = NULL,
		       embedding_at = NULL
		RETURNING id, cwd, kind, content, updated_by, created_at, updated_at`,
		id, cwd, string(kind), content, string(author))
	d, err := scanDoc(row)
	if err == nil {
		s.embedDocBestEffort(ctx, d)
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
	if strings.TrimSpace(cwd) == "" {
		return Proposal{}, ErrEmptyCwd
	}
	if err := s.validateWriteTarget(ctx, cwd, kind); err != nil {
		return Proposal{}, err
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
		   SET content      = EXCLUDED.content,
		       updated_by   = 'agent',
		       updated_at   = NOW(),
		       embedding    = NULL,
		       embedder     = NULL,
		       embedding_at = NULL
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
	s.embedDocBestEffort(ctx, d)
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
	if IsEphemeralCwd(e.Cwd) {
		return LogEntry{}, ErrEphemeralCwd
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

// embedDocBestEffort computes + persists an embedding for a doc so
// cross-layer search can match it semantically instead of by
// substring. Since the knowledge blueprint EVERY doc embeds — custom
// project sections and global kb_* pages included — so agents can
// retrieve exactly the section a task needs instead of swallowing
// whole injected pages. Soft-fails: on error the embedding stays NULL
// and the backfill loop retries. The vector lives in the same space
// as memories.embedding / session_logs.embedding (same embedder), so
// cosines are directly comparable across layers.
func (s *Service) embedDocBestEffort(ctx context.Context, d Doc) {
	if s.embedder == nil {
		return
	}
	text := strings.TrimSpace(d.Content)
	if text == "" {
		return
	}
	vecs, err := s.embedder.Embed(ctx, []string{text})
	if err != nil || len(vecs) == 0 || len(vecs[0]) == 0 {
		s.log.Debug("projectdoc: doc embed at write-time failed (will retry via backfill)",
			"doc_id", d.ID, "kind", d.Kind, "err", err)
		return
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE project_docs
		   SET embedding = $1,
		       embedder = $2,
		       embedding_at = NOW()
		 WHERE id = $3`, pgvecString(vecs[0]), s.embedder.Name(), d.ID); err != nil {
		s.log.Debug("projectdoc: doc embed write-back failed",
			"doc_id", d.ID, "err", err)
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

	// Lean mode (Cortex settings): guardrails + a compact index;
	// agents pull full sections/pages on demand instead of paying
	// for the whole corpus in every spawn.
	if s.spawnMode != nil && s.spawnMode(ctx) == "lean" {
		return s.renderLeanSpawn(ctx, cwd)
	}

	docs, err := s.ListDocsForCwd(ctx, cwd)
	if err != nil {
		return "", fmt.Errorf("projectdoc: render spawn docs: %w", err)
	}
	logs, err := s.ListLogs(ctx, cwd, recentLogs)
	if err != nil {
		return "", fmt.Errorf("projectdoc: render spawn logs: %w", err)
	}

	// The blueprint decides WHICH sections inject and in what order
	// (Cortex Phase 3). Lookup failure degrades to the default
	// blueprint so spawn never breaks on blueprint plumbing.
	sections, bErr := s.ListSections(ctx, cwd)
	if bErr != nil {
		s.log.Warn("projectdoc: render spawn blueprint failed — using defaults", "cwd", cwd, "err", bErr)
		sections = defaultSections(cwd)
	}
	content := make(map[string]string, len(docs))
	for _, d := range docs {
		content[string(d.Kind)] = strings.TrimSpace(d.Content)
	}

	// P-D — a frozen project (paused/archived) is shelved: drop its
	// project-specific Notes + journal so a session spawned in it isn't primed
	// with stale state. Cross-project Knowledge (global KB) still injects — it's
	// transferable expertise, useful regardless of this project's lifecycle.
	if status, _ := s.GetStatus(ctx, cwd); status.IsFrozen() {
		content = map[string]string{}
		logs = nil
	}

	// Knowledge (cross-project) splits into two natures (Experience
	// Flywheel), and since the knowledge blueprint the page set is
	// dynamic: foundational pages inject as binding guardrails,
	// emergent pages as reference — each only when its inject flag is
	// on. Pages with inject=false are reached on demand via
	// cross-layer search instead.
	kbSections := s.globalKBSections(ctx)
	foundational := s.foundationalRules(ctx, kbSections)
	type kbRef struct{ title, body string }
	var emergent []kbRef
	for _, sec := range kbSections {
		if sec.Nature != "emergent" || !sec.Inject {
			continue
		}
		if body := s.globalKBDoc(ctx, Kind(sec.Slug)); body != "" {
			emergent = append(emergent, kbRef{title: sec.Title + " (reference)", body: body})
		}
	}

	anySection := false
	for _, sec := range sections {
		if sec.Inject && content[sec.Slug] != "" {
			anySection = true
			break
		}
	}
	if !anySection && foundational == "" && len(emergent) == 0 && len(logs) == 0 {
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

	// Binding guardrails FIRST so a tight budget never truncates them away.
	appendSection("Foundational knowledge — RULES you MUST follow", foundational)
	// Blueprint sections in blueprint order (pinned first, then
	// position) — the operator controls spawn priority by reordering.
	for _, sec := range sections {
		if !sec.Inject {
			continue
		}
		appendSection(sec.Title, content[sec.Slug])
	}
	for _, ref := range emergent {
		appendSection(ref.title, ref.body)
	}

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

// renderLeanSpawn is the lean-mode banner: binding foundational rules
// in full (they are the guardrails — never indexed away), then a
// compact INDEX of the project's doc sections and the knowledge pages
// with instructions to fetch content on demand through the memory MCP.
// One screen instead of the whole corpus: spawn stays cheap and the
// agent's context window stays available for actual work.
func (s *Service) renderLeanSpawn(ctx context.Context, cwd string) (string, error) {
	frozen := false
	if status, _ := s.GetStatus(ctx, cwd); status.IsFrozen() {
		frozen = true
	}

	kbSections := s.globalKBSections(ctx)
	foundational := s.foundationalRules(ctx, kbSections)

	var b strings.Builder
	b.WriteString("## Project context (cross-agent shared, read-only)\n\n")
	if foundational != "" {
		b.WriteString("### Foundational knowledge — RULES you MUST follow\n\n")
		b.WriteString(foundational)
		b.WriteString("\n\n")
	}

	if !frozen {
		sections, err := s.ListSections(ctx, cwd)
		if err != nil {
			sections = defaultSections(cwd)
		}
		docs, _ := s.ListDocsForCwd(ctx, cwd)
		filled := make(map[string]Doc, len(docs))
		for _, d := range docs {
			filled[string(d.Kind)] = d
		}
		b.WriteString("### Project doc index\n\n")
		for _, sec := range sections {
			d, has := filled[sec.Slug]
			state := "empty"
			if has && strings.TrimSpace(d.Content) != "" {
				state = "updated " + d.UpdatedAt.Format("2006-01-02") + " by " + string(d.UpdatedBy)
			}
			fmt.Fprintf(&b, "- **%s** (`%s`, %s)", sec.Title, sec.Slug, state)
			if desc := strings.TrimSpace(sec.Description); desc != "" {
				b.WriteString(" — " + desc)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("### Knowledge index (cross-project)\n\n")
	for _, sec := range kbSections {
		if sec.Nature == "foundational" && sec.Inject {
			continue // already injected in full above
		}
		authority := "reference"
		if sec.Nature == "foundational" {
			authority = "binding"
		}
		fmt.Fprintf(&b, "- **%s** (`%s`, %s)", sec.Title, sec.Slug, authority)
		if desc := strings.TrimSpace(sec.Description); desc != "" {
			b.WriteString(" — " + desc)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("The indexes above are NOT loaded — fetch on demand via the opendray-memory MCP tools: " +
		"`doc_read` (pass a section slug or kb_* page slug) for a specific document, " +
		"`project_search` for cross-layer semantic search (facts, journal, docs, knowledge), " +
		"`memory_search` for episodic facts. Fetch ONLY what the task needs.\n\n" +
		"If your work changes the goal or plan, do **not** silently overwrite them. " +
		"Use the `project_goal_set` / `project_plan_set` MCP tools — they file a proposal " +
		"the operator approves before the live doc updates.\n")
	return b.String(), nil
}

// globalKBDoc fetches a global Knowledge page for spawn injection. Empty on
// absence or error — never blocks spawn.
func (s *Service) globalKBDoc(ctx context.Context, kind Kind) string {
	d, err := s.GetDoc(ctx, GlobalCwd, kind)
	if err != nil {
		return ""
	}
	return stripKBSig(d.Content)
}

// globalKBSections returns the knowledge blueprint (lazily seeded with
// the classic four). Empty on error — spawn must never block on it.
func (s *Service) globalKBSections(ctx context.Context) []Section {
	sections, err := s.ListSections(ctx, GlobalCwd)
	if err != nil {
		s.log.Warn("projectdoc: knowledge blueprint unavailable — using classics", "err", err)
		return kbDefaultSections()
	}
	return sections
}

// foundationalRules assembles the binding Foundational knowledge — every
// foundational-nature page with inject=true (classically infrastructure +
// conventions; extensible since the knowledge blueprint) — into one block
// injected as guardrails. An operator-locked page is ratified; an
// AI-drafted one is flagged so the agent treats it with care.
func (s *Service) foundationalRules(ctx context.Context, kbSections []Section) string {
	var b strings.Builder
	for _, sec := range kbSections {
		if sec.Nature != "foundational" || !sec.Inject {
			continue
		}
		d, err := s.GetDoc(ctx, GlobalCwd, Kind(sec.Slug))
		if err != nil {
			continue
		}
		body := stripKBSig(d.Content)
		if body == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(body)
		if d.UpdatedBy != AuthorOperator {
			b.WriteString("\n_(AI-drafted — verify before relying on it)_")
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return b.String() +
		"\n\nThese are the home-lab's standing rules. Follow them exactly; never deviate without the operator's say-so."
}

// stripKBSig removes the KB drafter's hidden signature marker line so it never
// leaks into a spawn prompt or a rendered page.
func stripKBSig(content string) string {
	var b strings.Builder
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "kb-sig:") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
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
