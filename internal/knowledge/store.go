package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// EnsureEntity inserts a node when its (deterministic) id is new, and leaves
// any existing row untouched; it returns the resulting node either way. Used
// for idempotent canonical entities such as per-cwd project entities.
func (s *Store) EnsureEntity(ctx context.Context, n Node) (Node, error) {
	if err := n.Validate(); err != nil {
		return Node{}, err
	}
	if n.ID == "" {
		return Node{}, errors.New("knowledge: EnsureEntity requires a deterministic id")
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
	var entityType any
	if n.EntityType != "" {
		entityType = string(n.EntityType)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO knowledge_nodes
			(id, kind, entity_type, title, body, scope, scope_key, maturity, provenance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
		ON CONFLICT (id) DO NOTHING`,
		n.ID, string(n.Kind), entityType, n.Title, n.Body, string(n.Scope),
		n.ScopeKey, string(n.Maturity), provJSON)
	if err != nil {
		return Node{}, fmt.Errorf("knowledge: ensure entity: %w", err)
	}
	return s.GetNode(ctx, n.ID)
}

// LinkFactSource records that a fact node is backed by a memory row. The
// memory_id is a soft reference (no FK to memories — keeps memory decoupled).
func (s *Store) LinkFactSource(ctx context.Context, nodeID, memoryID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO knowledge_fact_sources (node_id, memory_id)
		VALUES ($1, $2) ON CONFLICT (node_id, memory_id) DO NOTHING`,
		nodeID, memoryID)
	if err != nil {
		return fmt.Errorf("knowledge: link fact source: %w", err)
	}
	return nil
}

// AnchoredMemoryIDs returns the set of memory ids already lifted into the
// graph for a given project scope key, so the sweep skips them.
func (s *Store) AnchoredMemoryIDs(ctx context.Context, scopeKey string) (map[string]struct{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT fs.memory_id
		FROM knowledge_fact_sources fs
		JOIN knowledge_nodes n ON n.id = fs.node_id
		WHERE n.scope = 'project' AND n.scope_key = $1`, scopeKey)
	if err != nil {
		return nil, fmt.Errorf("knowledge: anchored memory ids: %w", err)
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

// Neighbor is a node reached from a center node via one edge.
type Neighbor struct {
	Node      Node     `json:"node"`
	EdgeType  EdgeType `json:"edge_type"`
	Direction string   `json:"direction"` // "out" = center->neighbor, "in" = neighbor->center
}

// Neighborhood returns a node plus its live 1-hop neighbors (both directions).
// Powers the project-brain / graph views (Phase 2).
func (s *Store) Neighborhood(ctx context.Context, id string) (Node, []Neighbor, error) {
	center, err := s.GetNode(ctx, id)
	if err != nil {
		return Node{}, nil, err
	}
	edges, err := s.ListEdges(ctx, id)
	if err != nil {
		return Node{}, nil, err
	}
	ids := make([]string, 0, len(edges))
	seen := map[string]struct{}{}
	for _, e := range edges {
		other := e.DstID
		if e.DstID == id {
			other = e.SrcID
		}
		if _, ok := seen[other]; !ok {
			seen[other] = struct{}{}
			ids = append(ids, other)
		}
	}
	byID := map[string]Node{}
	if len(ids) > 0 {
		rows, err := s.pool.Query(ctx, selectNodeSQL+` WHERE archived_at IS NULL AND id = ANY($1)`, ids)
		if err != nil {
			return Node{}, nil, fmt.Errorf("knowledge: neighborhood nodes: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			n, err := scanNode(rows)
			if err != nil {
				return Node{}, nil, err
			}
			byID[n.ID] = n
		}
		if err := rows.Err(); err != nil {
			return Node{}, nil, err
		}
	}
	neighbors := make([]Neighbor, 0, len(edges))
	for _, e := range edges {
		other, dir := e.DstID, "out"
		if e.DstID == id {
			other, dir = e.SrcID, "in"
		}
		n, ok := byID[other]
		if !ok {
			continue // neighbor archived or missing
		}
		neighbors = append(neighbors, Neighbor{Node: n, EdgeType: e.EdgeType, Direction: dir})
	}
	return center, neighbors, nil
}

// FindEntityByName returns a live entity node matching (entityType, title)
// case-insensitively within a project scope, or ErrNotFound. Phase 1B
// canonicalization is exact / case-insensitive within the project; fuzzier
// resolution (embedding-NN, LLM adjudication) and cross-project merge are
// later refinements best tuned against live data.
func (s *Store) FindEntityByName(ctx context.Context, entityType EntityType, name, scopeKey string) (Node, error) {
	n, err := scanNode(s.pool.QueryRow(ctx, selectNodeSQL+`
		WHERE archived_at IS NULL AND kind = 'entity'
		  AND entity_type = $1 AND lower(title) = lower($2) AND scope_key = $3
		ORDER BY created_at LIMIT 1`, string(entityType), name, scopeKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return Node{}, ErrNotFound
	}
	return n, err
}

// ListProjectScopeKeys returns the distinct project scope keys (cwds) present
// in the graph. Drives the reflect sweep over projects.
func (s *Store) ListProjectScopeKeys(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT scope_key FROM knowledge_nodes
		WHERE scope = 'project' AND scope_key <> '' AND archived_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("knowledge: list project scope keys: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// SetEmbedding stores a node's vector + the embedder that produced it.
func (s *Store) SetEmbedding(ctx context.Context, id, embedder string, vec []float32) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE knowledge_nodes
		SET embedding = $1::vector, embedder = $2, embedding_at = now()
		WHERE id = $3`, vectorLiteral(vec), embedder, id)
	if err != nil {
		return fmt.Errorf("knowledge: set embedding: %w", err)
	}
	return nil
}

// ListNodesNeedingEmbedding returns live nodes not yet embedded by embedder
// (NULL vector, or embedded by a different embedder — drives convergence).
func (s *Store) ListNodesNeedingEmbedding(ctx context.Context, embedder string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, selectNodeSQL+`
		WHERE archived_at IS NULL AND (embedding IS NULL OR embedder IS DISTINCT FROM $1)
		ORDER BY updated_at DESC LIMIT $2`, embedder, limit)
	if err != nil {
		return nil, fmt.Errorf("knowledge: list nodes needing embedding: %w", err)
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

// SearchHit is a node plus its cosine similarity to a query vector.
type SearchHit struct {
	Node       Node    `json:"node"`
	Similarity float64 `json:"similarity"`
}

// SearchNodes runs a cosine search over nodes embedded by the given embedder.
func (s *Store) SearchNodes(ctx context.Context, embedder string, vec []float32, scopeKey string, topK int) ([]SearchHit, error) {
	if topK <= 0 {
		topK = 10
	}
	// Scope-aware: a project ($3 set) sees its own nodes + all global nodes,
	// and never other projects' nodes (isolation). Empty $3 = search all.
	rows, err := s.pool.Query(ctx, `
		SELECT id, kind, COALESCE(entity_type, ''), title, body, scope, scope_key,
		       maturity, confidence, provenance, created_at, updated_at, archived_at,
		       1 - (embedding <=> $1::vector) AS similarity
		FROM knowledge_nodes
		WHERE archived_at IS NULL AND embedding IS NOT NULL AND embedder = $2
		  AND ($3 = '' OR scope = 'global' OR scope_key = $3)
		ORDER BY embedding <=> $1::vector ASC
		LIMIT $4`, vectorLiteral(vec), embedder, scopeKey, topK)
	if err != nil {
		return nil, fmt.Errorf("knowledge: search nodes: %w", err)
	}
	defer rows.Close()
	out := []SearchHit{}
	for rows.Next() {
		var n Node
		var kind, entityType, scope, maturity string
		var confidence *float64
		var provJSON []byte
		var archivedAt *time.Time
		var sim float64
		if err := rows.Scan(&n.ID, &kind, &entityType, &n.Title, &n.Body, &scope,
			&n.ScopeKey, &maturity, &confidence, &provJSON,
			&n.CreatedAt, &n.UpdatedAt, &archivedAt, &sim); err != nil {
			return nil, err
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
		out = append(out, SearchHit{Node: n, Similarity: sim})
	}
	return out, rows.Err()
}

// PromoteNode lifts a node to a wider scope (e.g. project -> global) so its
// knowledge transfers to other projects' searches. scopeKey is the new key
// (” for global).
func (s *Store) PromoteNode(ctx context.Context, id string, scope Scope, scopeKey string) error {
	if !scope.Valid() {
		return fmt.Errorf("knowledge: invalid scope %q", scope)
	}
	ct, err := s.pool.Exec(ctx, `
		UPDATE knowledge_nodes SET scope = $1, scope_key = $2, updated_at = now()
		WHERE id = $3 AND archived_at IS NULL`, string(scope), scopeKey, id)
	if err != nil {
		return fmt.Errorf("knowledge: promote node: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MergeDuplicateGlobalEntities collapses entities that denote the same thing
// into one canonical node, re-pointing edges onto it. Entities are grouped by
// normalised title (NOT entity_type): the extractor assigns inconsistent types
// to the same named entity — "opendray" as tool / service / project — so the
// name is the identity. A group with duplicates (cross-project) collapses to a
// single global node; a lone tech/tool entity is promoted to global (inherently
// cross-project); other lone entities keep their project scope. Cwd anchor
// entities are titled by path, hence unique → never merged. Idempotent (a clean
// graph is a no-op). Returns the number of duplicate nodes merged away.
func (s *Store) MergeDuplicateGlobalEntities(ctx context.Context) (int, error) {
	type ent struct{ id, etype, title, scope string }
	rows, err := s.pool.Query(ctx, `
		SELECT id, entity_type, title, scope
		FROM knowledge_nodes
		WHERE kind = 'entity' AND archived_at IS NULL
		ORDER BY created_at`)
	if err != nil {
		return 0, fmt.Errorf("knowledge: scan entities: %w", err)
	}
	var all []ent
	for rows.Next() {
		var e ent
		if err := rows.Scan(&e.id, &e.etype, &e.title, &e.scope); err != nil {
			rows.Close()
			return 0, err
		}
		all = append(all, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	groups := map[string][]ent{}
	for _, e := range all {
		k := strings.ToLower(strings.TrimSpace(e.title))
		if k == "" {
			continue
		}
		groups[k] = append(groups[k], e)
	}

	merged := 0
	for _, g := range groups {
		lone := len(g) == 1
		// A lone non-tech/tool entity keeps its project scope (it may be
		// project-specific); only cross-project duplicates or inherently-global
		// tech/tool entities collapse to global.
		if lone && g[0].etype != "tech" && g[0].etype != "tool" {
			continue
		}
		canonical := ""
		for _, e := range g {
			if e.scope == "global" {
				canonical = e.id
				break
			}
		}
		if canonical != "" && lone {
			continue // already a lone global entity — nothing to do
		}
		if canonical == "" {
			canonical = g[0].id
			if _, err := s.pool.Exec(ctx, `
				UPDATE knowledge_nodes SET scope = 'global', scope_key = '', updated_at = now()
				WHERE id = $1`, canonical); err != nil {
				return merged, fmt.Errorf("knowledge: promote canonical entity: %w", err)
			}
			if lone {
				continue // single entity, now global — done
			}
		}
		for _, e := range g {
			if e.id == canonical {
				continue
			}
			edges, err := s.ListEdges(ctx, e.id)
			if err != nil {
				return merged, err
			}
			for _, ed := range edges {
				src, dst := ed.SrcID, ed.DstID
				if src == e.id {
					src = canonical
				}
				if dst == e.id {
					dst = canonical
				}
				// Re-point onto the canonical; CreateEdge validates (skips
				// self-edges) + ON CONFLICT DO NOTHING (skips collisions).
				_ = s.CreateEdge(ctx, Edge{SrcID: src, EdgeType: ed.EdgeType, DstID: dst})
			}
			if _, err := s.pool.Exec(ctx, `DELETE FROM knowledge_nodes WHERE id = $1`, e.id); err != nil {
				return merged, fmt.Errorf("knowledge: delete duplicate entity: %w", err)
			}
			merged++
		}
	}
	return merged, nil
}

// ReflectSig returns the stored reflection signature for a project entity
// (empty when none / node missing) — used to skip re-reflecting unchanged input.
func (s *Store) ReflectSig(ctx context.Context, projID string) (string, error) {
	var sig string
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(provenance->>'reflect_sig','') FROM knowledge_nodes WHERE id = $1`,
		projID).Scan(&sig)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("knowledge: reflect sig: %w", err)
	}
	return sig, nil
}

// SetReflectSig records the reflection feedstock signature on a project entity.
func (s *Store) SetReflectSig(ctx context.Context, projID, sig string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE knowledge_nodes
		   SET provenance = jsonb_set(COALESCE(provenance, '{}'::jsonb), '{reflect_sig}', to_jsonb($2::text)),
		       updated_at = now()
		 WHERE id = $1`, projID, sig)
	if err != nil {
		return fmt.Errorf("knowledge: set reflect sig: %w", err)
	}
	return nil
}

// DeleteNode removes a node (and, via FK cascade, its edges + fact sources).
// Returns ErrNotFound when nothing matched.
func (s *Store) DeleteNode(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM knowledge_nodes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("knowledge: delete node: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Reset wipes the entire knowledge graph. The graph is fully derived from
// episodic memory, so the next anchor sweep rebuilds it from scratch with the
// current logic. Destructive — exposed only on the admin/dual-auth surface.
func (s *Store) Reset(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`TRUNCATE knowledge_edges, knowledge_fact_sources, knowledge_nodes CASCADE`)
	if err != nil {
		return fmt.Errorf("knowledge: reset: %w", err)
	}
	return nil
}

const selectNodeSQL = `
	SELECT id, kind, COALESCE(entity_type, ''), title, body, scope,
	       scope_key, maturity, confidence, provenance,
	       created_at, updated_at, archived_at,
	       COALESCE(use_count, 0), last_used_at, COALESCE(enabled, TRUE)
	FROM knowledge_nodes`

// RecordSkillUsage bumps use_count / last_used_at for every active
// skill whose title appears (case-insensitively) in the given session
// transcript text. One SQL statement, no LLM cost — a deliberate
// heuristic: a skill the agent talked about / followed shows up by
// name; skills that never do are retirement candidates.
func (s *Store) RecordSkillUsage(ctx context.Context, transcript string) (int64, error) {
	if strings.TrimSpace(transcript) == "" {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE knowledge_nodes
		   SET use_count = use_count + 1, last_used_at = NOW()
		 WHERE kind = 'skill'
		   AND archived_at IS NULL
		   AND title <> ''
		   AND POSITION(lower(title) IN lower($1)) > 0`, transcript)
	if err != nil {
		return 0, fmt.Errorf("knowledge: record skill usage: %w", err)
	}
	return tag.RowsAffected(), nil
}

// SetNodeEnabled flips a node's enabled flag (skills: controls whether
// its SKILL.md exists in the vault). Returns the updated node.
func (s *Store) SetNodeEnabled(ctx context.Context, id string, enabled bool) (Node, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE knowledge_nodes SET enabled = $1, updated_at = NOW()
		 WHERE id = $2 AND archived_at IS NULL`, enabled, id)
	if err != nil {
		return Node{}, fmt.Errorf("knowledge: set enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Node{}, ErrNotFound
	}
	return s.GetNode(ctx, id)
}

// UpdateNodeBody replaces a node's body (used when the skillify LLM
// produces the full SKILL.md so the stored node matches the file).
func (s *Store) UpdateNodeBody(ctx context.Context, id, body string) (Node, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE knowledge_nodes SET body = $1, updated_at = NOW()
		 WHERE id = $2 AND archived_at IS NULL`, body, id)
	if err != nil {
		return Node{}, fmt.Errorf("knowledge: update node body: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Node{}, ErrNotFound
	}
	return s.GetNode(ctx, id)
}

// ImpactEntity is one row of the impact view: an entity plus how many
// live nodes connect to it (its blast radius).
type ImpactEntity struct {
	Node   Node `json:"node"`
	Degree int  `json:"degree"`
}

// ListImpactEntities returns live entities ordered by connection
// degree — the production face of the graph: pick an entity (a
// database, a host, a tool) and see everything that depends on it
// before you touch it.
func (s *Store) ListImpactEntities(ctx context.Context, limit int) ([]ImpactEntity, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT n.id, COUNT(e.*) AS degree
		  FROM knowledge_nodes n
		  LEFT JOIN knowledge_edges e
		         ON (e.src_id = n.id OR e.dst_id = n.id)
		 WHERE n.kind = 'entity' AND n.archived_at IS NULL
		 GROUP BY n.id
		 ORDER BY degree DESC, n.updated_at DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("knowledge: impact entities: %w", err)
	}
	type row struct {
		id     string
		degree int
	}
	var ids []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.degree); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]ImpactEntity, 0, len(ids))
	for _, r := range ids {
		n, err := s.GetNode(ctx, r.id)
		if err != nil {
			continue
		}
		out = append(out, ImpactEntity{Node: n, Degree: r.degree})
	}
	return out, nil
}

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
		&n.CreatedAt, &n.UpdatedAt, &archivedAt,
		&n.UseCount, &n.LastUsedAt, &n.Enabled); err != nil {
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
