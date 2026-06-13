package projectdoc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ProjectStatus is the lifecycle state of a project (its cwd). A project with
// no project_lifecycle row is StatusActive — the table records only explicit
// pause/archive decisions, so a fresh project needs no migration backfill.
//
// Semantics (P-D):
//   - active   — normal: injected at spawn, distilled into Knowledge.
//   - paused   — temporarily shelved: frozen like archived, but signals intent
//     to resume; surfaced separately in the UI.
//   - archived — permanently shelved: frozen, excluded from spawn injection and
//     cross-project distillation, surfaced read-only.
type ProjectStatus string

const (
	StatusActive   ProjectStatus = "active"
	StatusPaused   ProjectStatus = "paused"
	StatusArchived ProjectStatus = "archived"
)

// ValidStatus reports whether s is a known lifecycle status.
func ValidStatus(s ProjectStatus) bool {
	switch s {
	case StatusActive, StatusPaused, StatusArchived:
		return true
	}
	return false
}

// IsFrozen reports whether a project in this status is excluded from spawn
// injection and cross-project distillation. paused + archived are frozen.
func (s ProjectStatus) IsFrozen() bool {
	return s == StatusPaused || s == StatusArchived
}

// ErrInvalidStatus is returned by SetStatus for an unknown status value.
var ErrInvalidStatus = errors.New("projectdoc: invalid project status")

// ProjectSummary is one project's lifecycle row joined with its last activity,
// for the operator's project-list view + idle auto-suggest.
type ProjectSummary struct {
	Cwd            string        `json:"cwd"`
	Status         ProjectStatus `json:"status"`
	UpdatedBy      Author        `json:"updated_by"`
	LastActivityAt *time.Time    `json:"last_activity_at,omitempty"`
	IdleDays       int           `json:"idle_days"`
	// SuggestArchive is true when an active project has been idle long enough
	// that the operator should consider archiving it (Decision 3: idle
	// auto-suggest). Computed against the caller's threshold.
	SuggestArchive bool `json:"suggest_archive"`
}

// GetStatus returns the lifecycle status for cwd, defaulting to StatusActive
// when there is no row. Never returns ErrNotFound — absence means active.
func (s *Service) GetStatus(ctx context.Context, cwd string) (ProjectStatus, error) {
	if strings.TrimSpace(cwd) == "" {
		return StatusActive, ErrEmptyCwd
	}
	var status string
	err := s.pool.QueryRow(ctx,
		`SELECT status FROM project_lifecycle WHERE cwd = $1`, cwd).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return StatusActive, nil
	}
	if err != nil {
		return StatusActive, fmt.Errorf("projectdoc: get status: %w", err)
	}
	return ProjectStatus(status), nil
}

// WithStatusChangeHook installs a callback invoked after every
// successful SetStatus that actually changed the status. The app
// wires the memory bridge here (project archived → soft-archive its
// memories; unarchived → restore them) without projectdoc importing
// internal/memory. Best-effort: the hook owns its own error handling.
func (s *Service) WithStatusChangeHook(fn func(ctx context.Context, cwd string, old, new ProjectStatus)) *Service {
	s.onStatusChange = fn
	return s
}

// SetStatus upserts the lifecycle row for cwd. Setting StatusActive is kept as
// an explicit row (rather than deleting) so the audit of "operator re-activated
// on <date>" survives; callers treat a missing row and an active row alike.
func (s *Service) SetStatus(ctx context.Context, cwd string, status ProjectStatus, author Author) error {
	if strings.TrimSpace(cwd) == "" {
		return ErrEmptyCwd
	}
	if !ValidStatus(status) {
		return ErrInvalidStatus
	}
	if author == "" {
		author = AuthorOperator
	}
	// Prior status feeds the change hook. A read failure must not block
	// the write — fall back to the no-row default (active).
	old, gerr := s.GetStatus(ctx, cwd)
	if gerr != nil {
		old = StatusActive
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO project_lifecycle (cwd, status, updated_by, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (cwd) DO UPDATE
		   SET status     = EXCLUDED.status,
		       updated_by = EXCLUDED.updated_by,
		       updated_at = NOW()`,
		cwd, string(status), string(author))
	if err != nil {
		return fmt.Errorf("projectdoc: set status: %w", err)
	}
	if s.onStatusChange != nil && old != status {
		s.onStatusChange(ctx, cwd, old, status)
	}
	return nil
}

// ListProjects returns every cwd opendray knows about (from project_docs)
// joined with its lifecycle status + last journal activity. idleSuggestDays
// drives the SuggestArchive flag: an active project whose newest journal entry
// is older than that many days is flagged (≤0 disables the suggestion).
func (s *Service) ListProjects(ctx context.Context, idleSuggestDays int) ([]ProjectSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.cwd,
		       COALESCE(l.status, 'active')      AS status,
		       COALESCE(l.updated_by, 'operator') AS updated_by,
		       la.last_activity_at
		  FROM (
		       SELECT cwd FROM project_docs   WHERE cwd <> '' AND cwd <> $1
		       UNION
		       SELECT cwd FROM session_logs   WHERE cwd <> '' AND cwd <> $1
		  ) d
		  LEFT JOIN project_lifecycle l ON l.cwd = d.cwd
		  LEFT JOIN (
		       SELECT cwd, MAX(created_at) AS last_activity_at
		         FROM session_logs GROUP BY cwd
		  ) la ON la.cwd = d.cwd
		 ORDER BY d.cwd`, GlobalCwd)
	if err != nil {
		return nil, fmt.Errorf("projectdoc: list projects: %w", err)
	}
	defer rows.Close()

	now := time.Now().UTC()
	var out []ProjectSummary
	for rows.Next() {
		var (
			p          ProjectSummary
			statusStr  string
			byStr      string
			lastActive *time.Time
		)
		if err := rows.Scan(&p.Cwd, &statusStr, &byStr, &lastActive); err != nil {
			return nil, err
		}
		// Throwaway /tmp-style cwds (third-party consumers, tests) are
		// not projects — never list them. New footprint for them is
		// blocked at the source; this also hides any legacy residue.
		if IsEphemeralCwd(p.Cwd) {
			continue
		}
		p.Status = ProjectStatus(statusStr)
		p.UpdatedBy = Author(byStr)
		p.LastActivityAt = lastActive
		if lastActive != nil {
			p.IdleDays = int(now.Sub(*lastActive).Hours() / 24)
		}
		if idleSuggestDays > 0 && p.Status == StatusActive &&
			lastActive != nil && p.IdleDays >= idleSuggestDays {
			p.SuggestArchive = true
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
