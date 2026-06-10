// Package cortex is the unified module governing the three-layer
// experience flywheel: Memory (raw episodic facts) → Notes (the
// project's official doc) → Knowledge (cross-project, iterable
// expertise). See docs/cortex-architecture.md.
//
// Cortex is a facade/orchestration layer, NOT a physical merge: the
// memory, projectdoc, and knowledge packages keep owning their
// plumbing (capture, consolidation, proposals, mirroring, embeddings).
// Cortex owns only what is genuinely cross-layer:
//
//   - the unified /api/v1/cortex HTTP namespace (handler.go)
//   - flywheel status aggregation across the three rungs (Status)
//   - quarantine review + promotion (Phase 2)
//   - the doc blueprint proposer (Phase 3)
//   - curation conversations (Phase 4)
//
// Dependency direction is strictly one-way: cortex depends on the
// three layer packages; they never depend on cortex.
package cortex

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// DocsSource is the slice of *projectdoc.Service the Status
// aggregation needs. Declared here (where it is consumed) so the
// facade stays testable without a live pgx pool.
type DocsSource interface {
	ListProjects(ctx context.Context, idleSuggestDays int) ([]projectdoc.ProjectSummary, error)
	ListPendingProposals(ctx context.Context, cwd string) ([]projectdoc.Proposal, error)
}

// Service aggregates the three layers behind one module. Layer
// enablement mirrors app wiring: memory and knowledge are optional
// subsystems (nil when disabled), notes/projectdoc always exists.
type Service struct {
	docs             DocsSource
	memoryEnabled    bool
	knowledgeEnabled bool
	log              *slog.Logger

	// quarantineCount is wired in Phase 2 (memories.tier). Until then
	// Status reports zero — the field exists so the web home can ship
	// against a stable payload shape.
	quarantineCount func(ctx context.Context) (int, error)
}

// Option configures the Service.
type Option func(*Service)

// WithMemoryEnabled records whether the memory layer is live (the
// memory service is nil when disabled in config).
func WithMemoryEnabled(on bool) Option {
	return func(s *Service) { s.memoryEnabled = on }
}

// WithKnowledgeEnabled records whether the knowledge layer is live.
func WithKnowledgeEnabled(on bool) Option {
	return func(s *Service) { s.knowledgeEnabled = on }
}

// WithQuarantineCounter wires the quarantined-memory counter (Phase 2).
func WithQuarantineCounter(fn func(ctx context.Context) (int, error)) Option {
	return func(s *Service) { s.quarantineCount = fn }
}

// NewService builds the Cortex facade. docs must be non-nil — the
// Notes rung ships with every install (migration 0025).
func NewService(docs DocsSource, log *slog.Logger, opts ...Option) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{docs: docs, log: log.With("component", "cortex")}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Status is the flywheel home payload: where each rung stands and
// what is waiting on the operator. Shapes are additive-only until
// mobile parity ships (Phase 6).
type Status struct {
	Notes     NotesStatus     `json:"notes"`
	Memory    MemoryStatus    `json:"memory"`
	Knowledge KnowledgeStatus `json:"knowledge"`
}

// NotesStatus describes the per-project doc rung.
type NotesStatus struct {
	Projects         int `json:"projects"`
	ActiveProjects   int `json:"active_projects"`
	FrozenProjects   int `json:"frozen_projects"`
	PendingProposals int `json:"pending_proposals"`
}

// MemoryStatus describes the raw episodic rung.
type MemoryStatus struct {
	Enabled         bool `json:"enabled"`
	QuarantineCount int  `json:"quarantine_count"`
}

// KnowledgeStatus describes the cross-project rung. KB pages live in
// projectdoc under the __global__ cwd, so their pending proposals are
// counted from the same proposal store as Notes — split by cwd.
type KnowledgeStatus struct {
	Enabled          bool `json:"enabled"`
	PendingProposals int  `json:"pending_proposals"`
}

// idleSuggestDays matches the web project list default; Status only
// needs counts, not the suggestion flag itself.
const idleSuggestDays = 30

// Status aggregates cross-layer state in one round trip for the
// Cortex home screen.
func (s *Service) Status(ctx context.Context) (Status, error) {
	var out Status
	out.Memory.Enabled = s.memoryEnabled
	out.Knowledge.Enabled = s.knowledgeEnabled

	projects, err := s.docs.ListProjects(ctx, idleSuggestDays)
	if err != nil {
		return Status{}, fmt.Errorf("cortex status: list projects: %w", err)
	}
	out.Notes.Projects = len(projects)
	for _, p := range projects {
		if p.Status.IsFrozen() {
			out.Notes.FrozenProjects++
		} else {
			out.Notes.ActiveProjects++
		}
	}

	proposals, err := s.docs.ListPendingProposals(ctx, "")
	if err != nil {
		return Status{}, fmt.Errorf("cortex status: list proposals: %w", err)
	}
	for _, p := range proposals {
		if p.Cwd == projectdoc.GlobalCwd {
			out.Knowledge.PendingProposals++
		} else {
			out.Notes.PendingProposals++
		}
	}

	if s.quarantineCount != nil {
		n, err := s.quarantineCount(ctx)
		if err != nil {
			// Soft-fail: quarantine is advisory on the home screen;
			// a counting error must not blank the whole status.
			s.log.Warn("cortex status: quarantine count failed", "err", err)
		} else {
			out.Memory.QuarantineCount = n
		}
	}
	return out, nil
}
