package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned by GetNode when no live node has the given id.
var ErrNotFound = errors.New("knowledge: node not found")

// Store persists the knowledge graph in opendray's existing PostgreSQL
// (pgvector). It reuses the same pool as memory but writes only to the
// knowledge_* tables; memory's tables are never touched from here.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs a Store over an existing pgx pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// NodeFilter narrows ListNodes. The zero value lists all live (non-archived)
// nodes, newest first.
type NodeFilter struct {
	Kind     NodeKind
	Scope    Scope
	ScopeKey string
	Limit    int
}

// CreateNode validates and inserts a node, minting an id and defaulting
// maturity when the caller omits them.
func (s *Store) CreateNode(ctx context.Context, n Node) (Node, error) {
	if err := n.Validate(); err != nil {
		return Node{}, err
	}
	if n.ID == "" {
		n.ID = NewID(string(n.Kind))
	}
	if n.Maturity == "" {
		n.Maturity = MaturityCandidate
	}
	if n.Provenance == nil {
		n.Provenance = map[string]any{}
	}
	provJSON, err := json.Marshal(n.Provenance)
	if err != nil {
		return Node{}, fmt.Errorf("knowledge: marshal provenance: %w", err)
	}
	var entityType any // NULL for non-entity nodes
	if n.EntityType != "" {
		entityType = string(n.EntityType)
	}
	var confidence any // NULL when unset
	if n.Confidence != nil {
		confidence = *n.Confidence
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO knowledge_nodes
			(id, kind, entity_type, title, body, scope, scope_key,
			 maturity, confidence, provenance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
		RETURNING created_at, updated_at`,
		n.ID, string(n.Kind), entityType, n.Title, n.Body, string(n.Scope),
		n.ScopeKey, string(n.Maturity), confidence, provJSON)
	if err := row.Scan(&n.CreatedAt, &n.UpdatedAt); err != nil {
		return Node{}, fmt.Errorf("knowledge: insert node: %w", err)
	}
	return n, nil
}

// GetNode returns a single node by id, or ErrNotFound.
func (s *Store) GetNode(ctx context.Context, id string) (Node, error) {
	n, err := scanNode(s.pool.QueryRow(ctx, selectNodeSQL+` WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Node{}, ErrNotFound
	}
	return n, err
}

// ListNodes returns live nodes matching the filter, newest first.
func (s *Store) ListNodes(ctx context.Context, f NodeFilter) ([]Node, error) {
	q := selectNodeSQL + ` WHERE archived_at IS NULL`
	args := []any{}
	i := 1
	if f.Kind != "" {
		q += fmt.Sprintf(" AND kind = $%d", i)
		args = append(args, string(f.Kind))
		i++
	}
	if f.Scope != "" {
		q += fmt.Sprintf(" AND scope = $%d", i)
		args = append(args, string(f.Scope))
		i++
	}
	if f.ScopeKey != "" {
		q += fmt.Sprintf(" AND scope_key = $%d", i)
		args = append(args, f.ScopeKey)
		i++
	}
	q += " ORDER BY updated_at DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, f.Limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("knowledge: list nodes: %w", err)
	}
	defer rows.Close()
	out := []Node{}
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// CreateEdge inserts a typed edge, ignoring exact duplicates.
func (s *Store) CreateEdge(ctx context.Context, e Edge) error {
	if err := e.Validate(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO knowledge_edges (src_id, edge_type, dst_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (src_id, edge_type, dst_id) DO NOTHING`,
		e.SrcID, string(e.EdgeType), e.DstID)
	if err != nil {
		return fmt.Errorf("knowledge: insert edge: %w", err)
	}
	return nil
}

// ListEdges returns every edge where nodeID is the source or destination.
func (s *Store) ListEdges(ctx context.Context, nodeID string) ([]Edge, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT src_id, edge_type, dst_id, created_at
		FROM knowledge_edges
		WHERE src_id = $1 OR dst_id = $1
		ORDER BY created_at`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("knowledge: list edges: %w", err)
	}
	defer rows.Close()
	out := []Edge{}
	for rows.Next() {
		var e Edge
		var et string
		if err := rows.Scan(&e.SrcID, &et, &e.DstID, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.EdgeType = EdgeType(et)
		out = append(out, e)
	}
	return out, rows.Err()
}

const selectNodeSQL = `
	SELECT id, kind, COALESCE(entity_type, ''), title, body, scope,
	       scope_key, maturity, confidence, provenance,
	       created_at, updated_at, archived_at
	FROM knowledge_nodes`

// rowScanner is satisfied by both pgx.Row (QueryRow) and pgx.Rows (Query).
type rowScanner interface {
	Scan(dest ...any) error
}

func scanNode(r rowScanner) (Node, error) {
	var n Node
	var kind, entityType, scope, maturity string
	var confidence *float64
	var provJSON []byte
	var archivedAt *time.Time
	if err := r.Scan(&n.ID, &kind, &entityType, &n.Title, &n.Body, &scope,
		&n.ScopeKey, &maturity, &confidence, &provJSON,
		&n.CreatedAt, &n.UpdatedAt, &archivedAt); err != nil {
		return Node{}, err
	}
	n.Kind = NodeKind(kind)
	n.EntityType = EntityType(entityType)
	n.Scope = Scope(scope)
	n.Maturity = Maturity(maturity)
	n.Confidence = confidence
	n.ArchivedAt = archivedAt
	if len(provJSON) > 0 {
		_ = json.Unmarshal(provJSON, &n.Provenance)
	}
	return n, nil
}
