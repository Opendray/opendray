package cleaner

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status enumerates memory_cleanup_decisions.status.
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusExecuted Status = "executed"
	StatusExpired  Status = "expired"
)

// Decision is one row from memory_cleanup_decisions.
type Decision struct {
	ID                   string     `json:"id"`
	MemoryID             string     `json:"memory_id"`
	MemoryScope          string     `json:"memory_scope"`
	MemoryScopeKey       string     `json:"memory_scope_key"`
	MemoryTextSnapshot   string     `json:"memory_text_snapshot"`
	Verdict              Verdict    `json:"verdict"`
	Reason               string     `json:"reason"`
	MergeInto            string     `json:"merge_into,omitempty"`
	RunID                string     `json:"run_id"`
	Status               Status     `json:"status"`
	SummarizerProviderID string     `json:"summarizer_provider_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	DecidedAt            *time.Time `json:"decided_at,omitempty"`
	ExecutedAt           *time.Time `json:"executed_at,omitempty"`
}

// Sentinel errors.
var (
	ErrNotFound      = errors.New("cleaner: decision not found")
	ErrAlreadyClosed = errors.New("cleaner: decision is no longer pending")
)

// store wraps the pgx pool with decision-table operations. Kept
// unexported because all interaction goes through Service.
type store struct {
	pool *pgxpool.Pool
}

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

// Insert persists one fresh pending decision. id auto-generated.
func (s *store) Insert(ctx context.Context, d Decision) (Decision, error) {
	if d.ID == "" {
		d.ID = newID("mcd_")
	}
	if d.RunID == "" {
		return Decision{}, errors.New("cleaner: run_id required")
	}
	if !ValidVerdict(d.Verdict) {
		return Decision{}, fmt.Errorf("cleaner: invalid verdict %q", d.Verdict)
	}
	if d.Status == "" {
		d.Status = StatusPending
	}
	var mergeInto any
	if d.MergeInto != "" {
		mergeInto = d.MergeInto
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO memory_cleanup_decisions (
			id, memory_id, memory_scope, memory_scope_key,
			memory_text_snapshot, verdict, reason, merge_into,
			run_id, status, summarizer_provider_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, memory_id, memory_scope, memory_scope_key,
		          memory_text_snapshot, verdict, reason,
		          COALESCE(merge_into, ''), run_id, status,
		          summarizer_provider_id, created_at, decided_at, executed_at`,
		d.ID, d.MemoryID, d.MemoryScope, d.MemoryScopeKey,
		d.MemoryTextSnapshot, string(d.Verdict), d.Reason, mergeInto,
		d.RunID, string(d.Status), d.SummarizerProviderID,
	)
	return scanDecision(row)
}

// Get fetches one decision by id.
func (s *store) Get(ctx context.Context, id string) (Decision, error) {
	row := s.pool.QueryRow(ctx, decisionSelectStmt+` WHERE id = $1`, id)
	d, err := scanDecision(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Decision{}, ErrNotFound
	}
	return d, err
}

// List returns decisions filtered by status (or "" for all), scope,
// and scope_key. Limit caps the result; 0 → 50, >200 → 200.
func (s *store) List(ctx context.Context, status, scope, scopeKey string, limit int) ([]Decision, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := decisionSelectStmt + ` WHERE 1=1`
	args := make([]any, 0, 4)
	if status != "" {
		args = append(args, status)
		q += fmt.Sprintf(` AND status = $%d`, len(args))
	}
	if scope != "" {
		args = append(args, scope)
		q += fmt.Sprintf(` AND memory_scope = $%d`, len(args))
	}
	if scopeKey != "" {
		args = append(args, scopeKey)
		q += fmt.Sprintf(` AND memory_scope_key = $%d`, len(args))
	}
	args = append(args, limit)
	q += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d`, len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("cleaner: list decisions: %w", err)
	}
	defer rows.Close()
	var out []Decision
	for rows.Next() {
		d, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SetStatus moves a decision to the given target status iff its
// current status is pending (for approve/reject) or approved (for
// executed). Returns ErrAlreadyClosed when the row has progressed
// out from under the caller (idempotent retries are safe).
func (s *store) SetStatus(ctx context.Context, id string, target Status, allowedFrom ...Status) error {
	if len(allowedFrom) == 0 {
		return errors.New("cleaner: SetStatus needs at least one allowed source status")
	}
	allowed := make([]string, len(allowedFrom))
	for i, st := range allowedFrom {
		allowed[i] = string(st)
	}
	column := "decided_at"
	if target == StatusExecuted {
		column = "executed_at"
	}
	cmd, err := s.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE memory_cleanup_decisions
		   SET status = $1, %s = NOW()
		 WHERE id = $2 AND status = ANY($3)`, column),
		string(target), id, allowed,
	)
	if err != nil {
		return fmt.Errorf("cleaner: set status: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		// Distinguish missing-row from already-closed for callers.
		var probe int
		err := s.pool.QueryRow(ctx,
			`SELECT 1 FROM memory_cleanup_decisions WHERE id = $1`, id).Scan(&probe)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return ErrAlreadyClosed
	}
	return nil
}

const decisionSelectStmt = `
	SELECT id, memory_id, memory_scope, memory_scope_key,
	       memory_text_snapshot, verdict, reason,
	       COALESCE(merge_into, '') AS merge_into,
	       run_id, status, summarizer_provider_id,
	       created_at, decided_at, executed_at
	  FROM memory_cleanup_decisions`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDecision(row rowScanner) (Decision, error) {
	var (
		d              Decision
		verdict        string
		status         string
		decidedAt      *time.Time
		executedAt     *time.Time
		mergeInto      string
		summarizerID   string
		memoryScope    string
		memoryScopeKey string
	)
	if err := row.Scan(
		&d.ID, &d.MemoryID, &memoryScope, &memoryScopeKey,
		&d.MemoryTextSnapshot, &verdict, &d.Reason, &mergeInto,
		&d.RunID, &status, &summarizerID,
		&d.CreatedAt, &decidedAt, &executedAt,
	); err != nil {
		return Decision{}, err
	}
	d.MemoryScope = memoryScope
	d.MemoryScopeKey = memoryScopeKey
	d.Verdict = Verdict(verdict)
	d.Status = Status(status)
	d.MergeInto = mergeInto
	d.SummarizerProviderID = summarizerID
	d.DecidedAt = decidedAt
	d.ExecutedAt = executedAt
	return d, nil
}

func newID(prefix string) string {
	var b [14]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("cleaner: rand: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return prefix + strings.ToLower(enc)
}
