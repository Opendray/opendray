package knowledge

import (
	"context"
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
