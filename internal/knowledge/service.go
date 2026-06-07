package knowledge

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service is the thin application layer over Store. Phase 0 only delegates;
// later phases hang the consolidation / graduation engine off this type
// (and give it a read-only handle to internal/memory as its feedstock).
type Service struct {
	store *Store
	log   *slog.Logger
}

// NewService matches the projectdoc.NewService(pool, log) shape used at the
// app composition root.
func NewService(pool *pgxpool.Pool, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{store: NewStore(pool), log: log.With("component", "knowledge")}
}

// CreateNode validates and persists a node.
func (s *Service) CreateNode(ctx context.Context, n Node) (Node, error) {
	return s.store.CreateNode(ctx, n)
}

// GetNode returns a node by id (ErrNotFound when missing).
func (s *Service) GetNode(ctx context.Context, id string) (Node, error) {
	return s.store.GetNode(ctx, id)
}

// ListNodes returns live nodes matching the filter.
func (s *Service) ListNodes(ctx context.Context, f NodeFilter) ([]Node, error) {
	return s.store.ListNodes(ctx, f)
}

// CreateEdge persists a typed edge between two nodes.
func (s *Service) CreateEdge(ctx context.Context, e Edge) error {
	return s.store.CreateEdge(ctx, e)
}

// ListEdges returns all edges incident to nodeID.
func (s *Service) ListEdges(ctx context.Context, nodeID string) ([]Edge, error) {
	return s.store.ListEdges(ctx, nodeID)
}

// Neighborhood returns a node and its 1-hop neighbors.
func (s *Service) Neighborhood(ctx context.Context, id string) (Node, []Neighbor, error) {
	return s.store.Neighborhood(ctx, id)
}

// BrainView is a synthesised snapshot of one project's knowledge, assembled
// from the graph (Phase 2). It grows richer as later phases add playbooks /
// skills / related entities.
type BrainView struct {
	Project *Node  `json:"project"` // nil when the project has no entity yet
	Facts   []Node `json:"facts"`
}

// ProjectBrain assembles the project entity + its anchored facts for a cwd.
// An absent project entity yields an empty view (not an error) so a freshly
// enabled install returns 200 with nothing rather than 404.
func (s *Service) ProjectBrain(ctx context.Context, cwd string) (BrainView, error) {
	center, neighbors, err := s.store.Neighborhood(ctx, ProjectEntityID(cwd))
	if errors.Is(err, ErrNotFound) {
		return BrainView{}, nil
	}
	if err != nil {
		return BrainView{}, err
	}
	view := BrainView{Project: &center}
	for _, nb := range neighbors {
		if nb.Node.Kind == KindFact {
			view.Facts = append(view.Facts, nb.Node)
		}
	}
	return view, nil
}
