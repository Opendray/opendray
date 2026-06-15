package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgvectorStore persists memories in opendray's existing PostgreSQL
// using the pgvector extension. The schema (see migration 0011) is
// dimension-agnostic at the column level so different embedders can
// coexist; we issue per-(embedder,dim) HNSW indexes lazily on first
// insert.
//
// The cosine-similarity operator pgvector exposes is `<=>`; values
// are 0 (identical) → 2 (opposite). We invert to "similarity"
// (1 - distance/2 → 0..1) when returning hits so callers see the
// same scale as our in-memory CosineSimilarity helper.
type PgvectorStore struct {
	pool *pgxpool.Pool

	// indexedMu guards the in-memory mirror of memory_index_state.
	// Pre-loaded on Open so we avoid a SELECT per insert.
	indexedMu sync.Mutex
	indexed   map[string]int // embedder → dim
}

// OpenPgvectorStore constructs the store and pre-loads the
// "(embedder, dim)" combinations we've already indexed so subsequent
// inserts can short-circuit the lazy-index check.
func OpenPgvectorStore(ctx context.Context, pool *pgxpool.Pool) (*PgvectorStore, error) {
	if pool == nil {
		return nil, errors.New("memory: pgvector store requires a *pgxpool.Pool")
	}
	s := &PgvectorStore{pool: pool, indexed: make(map[string]int)}
	if err := s.loadIndexed(ctx); err != nil {
		return nil, fmt.Errorf("memory: load indexed state: %w", err)
	}
	return s, nil
}

func (s *PgvectorStore) loadIndexed(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT embedder, dim FROM memory_index_state`)
	if err != nil {
		// Tolerate "relation does not exist" — that just means the
		// migration hasn't run yet (we get called on every app.New
		// including the one inside `opendray migrate`). The next
		// startup after the migration will populate the cache.
		if isRelationDoesNotExist(err) {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var dim int
		if err := rows.Scan(&name, &dim); err != nil {
			return err
		}
		s.indexed[name] = dim
	}
	return rows.Err()
}

// isRelationDoesNotExist returns true for pg error 42P01.
func isRelationDoesNotExist(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01"
	}
	return false
}

func (s *PgvectorStore) Close() error { return nil }

// Insert writes a memory and lazily creates an HNSW index for the
// (embedder, dim) pair on first observation. Index creation is best-
// effort: a CREATE INDEX failure is logged-and-swallowed because
// pgvector falls back to seq scan automatically — we'd rather take
// the perf hit than reject the insert.
func (s *PgvectorStore) Insert(ctx context.Context, req InsertRequest) (string, error) {
	if err := req.Scope.Validate(); err != nil {
		return "", err
	}
	if len(req.Embedding) == 0 {
		return "", errors.New("memory: empty embedding")
	}
	if strings.TrimSpace(req.Text) == "" {
		return "", errors.New("memory: empty text")
	}

	id := NewID()
	meta := req.Metadata
	if meta == nil {
		meta = map[string]interface{}{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("memory: marshal metadata: %w", err)
	}

	vec := vectorLiteral(req.Embedding)
	// Provenance: empty SourceKind defers to DB default ('manual').
	// nil Confidence stays NULL.
	sourceKind := nullableStr(req.SourceKind)
	sourceRef := nullableStr(req.SourceRef)
	summSession := nullableStr(req.SummarizerSession)
	var confidence any
	if req.Confidence != nil {
		confidence = *req.Confidence
	}

	// Tier: empty defers to the DB default ('durable'). Quarantined
	// rows must carry their TTL deadline.
	tier := req.Tier
	if tier == "" {
		tier = TierDurable
	}
	if tier == TierQuarantine && req.QuarantineExpiresAt == nil {
		return "", errors.New("memory: quarantine insert needs an expiry")
	}
	var quarantineExpiry any
	if req.QuarantineExpiresAt != nil {
		quarantineExpiry = *req.QuarantineExpiresAt
	}

	if req.SourceKind != "" {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO memories
				(id, scope, scope_key, text, embedding, embedder, metadata,
				 source_kind, source_ref, summarizer_session, confidence,
				 tier, quarantine_expires_at)
			VALUES ($1, $2, $3, $4, $5::vector, $6, $7::jsonb,
			        $8, $9, $10, $11, $12, $13)
		`, id, string(req.Scope), req.ScopeKey, req.Text, vec, req.Embedder, metaJSON,
			sourceKind, sourceRef, summSession, confidence,
			tier, quarantineExpiry)
	} else {
		// Skip provenance columns entirely so DB CHECK + DEFAULT apply
		// — equivalent to legacy callers' behaviour pre-Phase-A.
		_, err = s.pool.Exec(ctx, `
			INSERT INTO memories (id, scope, scope_key, text, embedding, embedder, metadata,
			                      tier, quarantine_expires_at)
			VALUES ($1, $2, $3, $4, $5::vector, $6, $7::jsonb, $8, $9)
		`, id, string(req.Scope), req.ScopeKey, req.Text, vec, req.Embedder, metaJSON,
			tier, quarantineExpiry)
	}
	if err != nil {
		return "", fmt.Errorf("memory: insert: %w", err)
	}

	s.ensureIndex(ctx, req.Embedder, len(req.Embedding))
	return id, nil
}

// nullableStr converts "" to a SQL NULL so DB defaults apply, and
// passes non-empty strings through. Used for the provenance columns
// added in migration 0018 — each is nullable except source_kind
// which has a DEFAULT 'manual'.
func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ensureIndex creates an HNSW index for (embedder, dim) once. Errors
// are non-fatal: pgvector still serves queries via sequential scan,
// just slower. Locking via indexedMu prevents concurrent inserts
// from racing on the same DDL.
func (s *PgvectorStore) ensureIndex(ctx context.Context, embedder string, dim int) {
	s.indexedMu.Lock()
	defer s.indexedMu.Unlock()
	if existing, ok := s.indexed[embedder]; ok && existing == dim {
		return
	}
	idxName := fmt.Sprintf("memories_emb_%s_idx", sqlSafe(embedder))
	// HNSW with vector_cosine_ops is what pgvector recommends for
	// cosine-similarity workloads; defaults (m=16, ef_construction=64)
	// are fine for our scale.
	_, err := s.pool.Exec(ctx, fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON memories USING hnsw ((embedding::vector(%d)) vector_cosine_ops) WHERE embedder = $1`,
		idxName, dim,
	), embedder)
	if err != nil {
		// Don't surface this — silently degrade. The caller already
		// successfully inserted; failing here would be misleading.
		// In tests we log this; in prod the operator notices via slow
		// queries + the call log.
		return
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO memory_index_state (embedder, dim) VALUES ($1, $2)
		ON CONFLICT (embedder) DO UPDATE SET dim = EXCLUDED.dim
	`, embedder, dim)
	if err != nil {
		return
	}
	s.indexed[embedder] = dim
}

// Search returns the top-K hits for q.Vector, filtered by embedder
// (so cosine comparisons stay honest across multiple embedders) and
// by scope. Empty TopK defaults to 5.
func (s *PgvectorStore) Search(ctx context.Context, q SearchQuery) ([]SearchHit, error) {
	if err := q.Scope.Validate(); err != nil {
		return nil, err
	}
	if len(q.Vector) == 0 {
		return nil, errors.New("memory: empty query vector")
	}
	if q.TopK <= 0 {
		q.TopK = 5
	}

	vec := vectorLiteral(q.Vector)
	args := []interface{}{vec, q.Embedder, string(q.Scope), q.ScopeKey, q.TopK}
	// For global scope, ignore scope_key entirely.
	whereScope := `scope = $3 AND scope_key = $4`
	if q.Scope == ScopeGlobal {
		whereScope = `scope = $3`
		args = []interface{}{vec, q.Embedder, string(q.Scope), q.TopK}
	}

	// pgvector's <=> returns cosine *distance* (1 - cosine_similarity),
	// so similarity = 1 - distance. Range is [-1, 1]; the service
	// layer threshold filter discards anything below the configured
	// minimum (default 0.5 since the BM25 fallback rarely scores high).
	sql := fmt.Sprintf(`
		SELECT id, scope, scope_key, text, embedder, metadata,
		       created_at, updated_at, hit_count, last_hit_at,
		       1 - (embedding <=> $1::vector) AS similarity
		FROM memories
		WHERE embedder = $2 AND archived_at IS NULL AND tier = 'durable' AND %s
		ORDER BY embedding <=> $1::vector ASC
		LIMIT $%d
	`, whereScope, len(args))

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: search: %w", err)
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var (
			m    Memory
			meta []byte
			sim  float32
		)
		if err := rows.Scan(
			&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta,
			&m.CreatedAt, &m.UpdatedAt, &m.HitCount, &m.LastHitAt, &sim,
		); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &m.Metadata)
		}
		hits = append(hits, SearchHit{Memory: m, Similarity: sim})
	}
	return hits, rows.Err()
}

func (s *PgvectorStore) List(ctx context.Context, scope Scope, scopeKey string, limit int) ([]Memory, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	args := []interface{}{string(scope), scopeKey, limit}
	where := `scope = $1 AND scope_key = $2`
	if scope == ScopeGlobal {
		where = `scope = $1`
		args = []interface{}{string(scope), limit}
	}
	sql := fmt.Sprintf(`
		SELECT id, scope, scope_key, text, embedder, metadata,
		       created_at, updated_at, hit_count, last_hit_at
		FROM memories
		WHERE archived_at IS NULL AND tier = 'durable' AND %s
		ORDER BY created_at DESC
		LIMIT $%d
	`, where, len(args))

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: list: %w", err)
	}
	defer rows.Close()

	var out []Memory
	for rows.Next() {
		var (
			m    Memory
			meta []byte
		)
		if err := rows.Scan(&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta,
			&m.CreatedAt, &m.UpdatedAt, &m.HitCount, &m.LastHitAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &m.Metadata)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListArchived returns soft-archived memories for a scope, newest
// archived first, including archived_at / archived_reason so the
// restorable view can show when + why each row was archived.
func (s *PgvectorStore) ListArchived(ctx context.Context, scope Scope, scopeKey string, limit int) ([]Memory, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	// Empty scopeKey means "every key under this scope" — the cross-
	// project Archived view passes "" to list every project's archived
	// rows. Without this, an empty key filtered `scope_key = ''` and
	// matched nothing for project rows (whose key is the cwd), so the
	// Archived page was always blank. Global scope only has the empty key.
	where := `scope = $1 AND scope_key = $2`
	args := []interface{}{string(scope), scopeKey, limit}
	if scope == ScopeGlobal || scopeKey == "" {
		where = `scope = $1`
		args = []interface{}{string(scope), limit}
	}
	query := fmt.Sprintf(`
		SELECT id, scope, scope_key, text, embedder, metadata,
		       created_at, updated_at, hit_count, last_hit_at,
		       archived_at, archived_reason
		FROM memories
		WHERE archived_at IS NOT NULL AND %s
		ORDER BY archived_at DESC
		LIMIT $%d
	`, where, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: list archived: %w", err)
	}
	defer rows.Close()

	var out []Memory
	for rows.Next() {
		var (
			m      Memory
			meta   []byte
			reason sql.NullString
		)
		if err := rows.Scan(&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta,
			&m.CreatedAt, &m.UpdatedAt, &m.HitCount, &m.LastHitAt, &m.ArchivedAt, &reason); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &m.Metadata)
		}
		if reason.Valid {
			m.ArchivedReason = reason.String
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Get returns one Memory row by id, including provenance fields.
// Used by the memory_get_provenance MCP tool + future "show
// details" UI affordances.
func (s *PgvectorStore) Get(ctx context.Context, id string) (Memory, error) {
	var (
		m       Memory
		meta    []byte
		srcKind sql.NullString
		srcRef  sql.NullString
		summSes sql.NullString
		conf    sql.NullFloat64
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, scope, scope_key, text, embedder, metadata,
		       created_at, updated_at, hit_count, last_hit_at,
		       source_kind, source_ref, summarizer_session, confidence,
		       tier, quarantine_expires_at
		  FROM memories
		 WHERE id = $1 AND archived_at IS NULL`, id,
	).Scan(&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta,
		&m.CreatedAt, &m.UpdatedAt, &m.HitCount, &m.LastHitAt,
		&srcKind, &srcRef, &summSes, &conf,
		&m.Tier, &m.QuarantineExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Memory{}, ErrNotFound
		}
		return Memory{}, fmt.Errorf("memory: get: %w", err)
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &m.Metadata)
	}
	if srcKind.Valid {
		m.SourceKind = srcKind.String
	}
	if srcRef.Valid {
		m.SourceRef = srcRef.String
	}
	if summSes.Valid {
		m.SummarizerSession = summSes.String
	}
	if conf.Valid {
		v := float32(conf.Float64)
		m.Confidence = &v
	}
	return m, nil
}

// Update overwrites text + embedding + metadata for one row. scope,
// scope_key, embedder, created_at all stay as-is — only updated_at
// bumps to NOW(). Returns ErrNotFound when the id is missing.
func (s *PgvectorStore) Update(ctx context.Context, req UpdateRequest) error {
	if strings.TrimSpace(req.ID) == "" {
		return errors.New("memory: Update needs an id")
	}
	if strings.TrimSpace(req.Text) == "" {
		return errors.New("memory: empty text")
	}
	if len(req.Embedding) == 0 {
		return errors.New("memory: empty embedding")
	}
	meta := req.Metadata
	if meta == nil {
		meta = map[string]interface{}{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("memory: marshal metadata: %w", err)
	}
	var (
		tag pgconn.CommandTag
	)
	if strings.TrimSpace(req.Embedder) == "" {
		tag, err = s.pool.Exec(ctx, `
			UPDATE memories
			SET text = $1, embedding = $2::vector, metadata = $3::jsonb,
			    updated_at = NOW()
			WHERE id = $4 AND archived_at IS NULL
		`, req.Text, vectorLiteral(req.Embedding), metaJSON, req.ID)
	} else {
		// Reembed path: also overwrite the embedder column. The new
		// (embedder, dim) might warrant its own HNSW index — make sure
		// it exists.
		tag, err = s.pool.Exec(ctx, `
			UPDATE memories
			SET text = $1, embedding = $2::vector, metadata = $3::jsonb,
			    embedder = $4, updated_at = NOW()
			WHERE id = $5 AND archived_at IS NULL
		`, req.Text, vectorLiteral(req.Embedding), metaJSON, req.Embedder, req.ID)
		if err == nil && tag.RowsAffected() > 0 {
			s.ensureIndex(ctx, req.Embedder, len(req.Embedding))
		}
	}
	if err != nil {
		return fmt.Errorf("memory: update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountByEmbedder groups memories by their embedder column and
// returns the row counts. Used by the reembed inspector to show
// pre-migration stats.
func (s *PgvectorStore) CountByEmbedder(ctx context.Context) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT embedder, COUNT(*) FROM memories WHERE archived_at IS NULL GROUP BY embedder ORDER BY embedder
	`)
	if err != nil {
		// Tolerate "table doesn't exist yet" the same way loadIndexed
		// does — gives the caller a clean empty map at first boot.
		if isRelationDoesNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, fmt.Errorf("memory: count by embedder: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out[name] = n
	}
	return out, rows.Err()
}

// ListNeedingReembed returns up to limit memories whose embedder
// column differs from current, ordered by id ASC. afterID is a
// cursor (last id from the previous page) — pass "" to start at
// the beginning.
func (s *PgvectorStore) ListNeedingReembed(ctx context.Context, current string, limit int, afterID string) ([]Memory, error) {
	if limit <= 0 {
		limit = 50
	}
	args := []interface{}{current, limit}
	cursor := ""
	if afterID != "" {
		args = append(args, afterID)
		cursor = " AND id > $3"
	}
	sql := fmt.Sprintf(`
		SELECT id, scope, scope_key, text, embedder, metadata,
		       created_at, updated_at, hit_count, last_hit_at
		FROM memories
		WHERE embedder <> $1 AND archived_at IS NULL%s
		ORDER BY id ASC
		LIMIT $2
	`, cursor)
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: list needing reembed: %w", err)
	}
	defer rows.Close()
	var out []Memory
	for rows.Next() {
		var (
			m    Memory
			meta []byte
		)
		if err := rows.Scan(&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta,
			&m.CreatedAt, &m.UpdatedAt, &m.HitCount, &m.LastHitAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &m.Metadata)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RecordHits bumps hit_count + last_hit_at for the given ids in a
// single statement. We log-and-swallow errors: search results have
// already been returned to the caller, so failing here would be
// surprising and pointless. Empty input is a no-op.
func (s *PgvectorStore) RecordHits(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE memories
		SET hit_count = hit_count + 1, last_hit_at = NOW()
		WHERE id = ANY($1::text[]) AND archived_at IS NULL
	`, ids)
	if err != nil {
		return fmt.Errorf("memory: record hits: %w", err)
	}
	return nil
}

// ListScopeKeys returns distinct scope_key values seen under the
// given scope, alphabetically sorted. Empty scope_key entries are
// dropped. Used by the UI's scope-key picker.
func (s *PgvectorStore) ListScopeKeys(ctx context.Context, scope Scope) ([]string, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT scope_key FROM memories
		WHERE scope = $1 AND scope_key <> '' AND archived_at IS NULL AND tier = 'durable'
		ORDER BY scope_key
	`, string(scope))
	if err != nil {
		return nil, fmt.Errorf("memory: list scope keys: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *PgvectorStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM memories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("memory: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteByScope wipes every memory under the given scope/scope_key
// in a single SQL statement. Returns the row count actually
// removed. The empty scope_key is meaningful only for ScopeGlobal
// (caller must pass "" there) — for session/project an empty key
// is rejected upstream in the service layer to prevent the user
// from accidentally nuking everything.
func (s *PgvectorStore) DeleteByScope(
	ctx context.Context,
	scope Scope,
	scopeKey string,
) (int64, error) {
	tag, err := s.pool.Exec(
		ctx,
		`DELETE FROM memories WHERE scope = $1 AND scope_key = $2`,
		string(scope),
		scopeKey,
	)
	if err != nil {
		return 0, fmt.Errorf("memory: delete by scope: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Archive soft-deletes one memory. Idempotent: re-archiving an already
// archived row keeps the original archived_at (the WHERE guards it) so
// the grace window isn't reset.
func (s *PgvectorStore) Archive(ctx context.Context, id, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE memories
		   SET archived_at = NOW(), archived_reason = $2, updated_at = NOW()
		 WHERE id = $1 AND archived_at IS NULL`, id, reason)
	if err != nil {
		return fmt.Errorf("memory: archive: %w", err)
	}
	return nil
}

// ArchiveByScope soft-deletes every active memory under (scope, scopeKey).
func (s *PgvectorStore) ArchiveByScope(ctx context.Context, scope Scope, scopeKey, reason string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE memories
		   SET archived_at = NOW(), archived_reason = $3, updated_at = NOW()
		 WHERE scope = $1 AND scope_key = $2 AND archived_at IS NULL`,
		string(scope), scopeKey, reason)
	if err != nil {
		return 0, fmt.Errorf("memory: archive by scope: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ArchiveDormantStale archives never-hit, aged facts of a dormant
// project. The dormancy gate (the scope's newest activity predates
// dormantBefore) is evaluated in the same statement so the whole thing
// is atomic and there's no read-then-write race.
func (s *PgvectorStore) ArchiveDormantStale(ctx context.Context, scope Scope, scopeKey string, agedBefore, dormantBefore time.Time, reason string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		WITH activity AS (
		    SELECT MAX(GREATEST(created_at, COALESCE(last_hit_at, created_at))) AS last_active
		      FROM memories
		     WHERE scope = $1 AND scope_key = $2 AND archived_at IS NULL
		)
		UPDATE memories m
		   SET archived_at = NOW(), archived_reason = $5, updated_at = NOW()
		 WHERE m.scope = $1 AND m.scope_key = $2
		   AND m.archived_at IS NULL
		   AND m.hit_count = 0
		   AND m.created_at < $3
		   AND (SELECT last_active FROM activity) < $4`,
		string(scope), scopeKey, agedBefore, dormantBefore, reason)
	if err != nil {
		return 0, fmt.Errorf("memory: archive dormant stale: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Restore clears the archive flag on a previously archived memory.
func (s *PgvectorStore) Restore(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE memories
		   SET archived_at = NULL, archived_reason = NULL, updated_at = NOW()
		 WHERE id = $1 AND archived_at IS NOT NULL`, id)
	if err != nil {
		return fmt.Errorf("memory: restore: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RestoreByScope clears the archive flag on every row under (scope,
// scopeKey) that was archived with the given reason. Used by the
// project-unarchive bridge so reactivating a project brings back
// exactly what archiving it removed — cleaner-archived rows keep
// their verdicts. Returns the count restored; zero is not an error.
func (s *PgvectorStore) RestoreByScope(ctx context.Context, scope Scope, scopeKey, reason string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE memories
		   SET archived_at = NULL, archived_reason = NULL, updated_at = NOW()
		 WHERE scope = $1 AND scope_key = $2
		   AND archived_at IS NOT NULL AND archived_reason = $3`,
		string(scope), scopeKey, reason)
	if err != nil {
		return 0, fmt.Errorf("memory: restore by scope: %w", err)
	}
	return tag.RowsAffected(), nil
}

// PurgeArchived hard-deletes rows archived before cutoff (grace
// expired). Rows archived by the project-archive bridge are exempt:
// an archived project's memories stay restorable for as long as the
// project stays archived — unarchiving must always bring them back.
func (s *PgvectorStore) PurgeArchived(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM memories
		 WHERE archived_at IS NOT NULL AND archived_at < $1
		   AND COALESCE(archived_reason, '') <> $2`, cutoff, ReasonProjectArchived)
	if err != nil {
		return 0, fmt.Errorf("memory: purge archived: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Quarantine moves an active durable memory into the quarantine tier
// with the given TTL — the manual counterpart of the integration
// capture path, for facts the operator distrusts but isn't ready to
// delete. Returns ErrNotFound when id isn't an active durable row.
func (s *PgvectorStore) Quarantine(ctx context.Context, id string, expiresAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE memories
		   SET tier = 'quarantine', quarantine_expires_at = $2, updated_at = NOW()
		 WHERE id = $1 AND archived_at IS NULL AND tier = 'durable'`, id, expiresAt)
	if err != nil {
		return fmt.Errorf("memory: quarantine: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListQuarantined returns active quarantine-tier rows across every
// scope, newest first — the Cortex review queue. Provenance fields
// are included so the operator can see which session/integration
// produced each fact.
func (s *PgvectorStore) ListQuarantined(ctx context.Context, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, scope, scope_key, text, embedder, metadata,
		       created_at, updated_at, hit_count, last_hit_at,
		       source_kind, source_ref, summarizer_session, confidence,
		       tier, quarantine_expires_at
		FROM memories
		WHERE tier = 'quarantine' AND archived_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("memory: list quarantined: %w", err)
	}
	defer rows.Close()
	var out []Memory
	for rows.Next() {
		var (
			m       Memory
			meta    []byte
			srcKind sql.NullString
			srcRef  sql.NullString
			summSes sql.NullString
			conf    sql.NullFloat64
		)
		if err := rows.Scan(&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta,
			&m.CreatedAt, &m.UpdatedAt, &m.HitCount, &m.LastHitAt,
			&srcKind, &srcRef, &summSes, &conf,
			&m.Tier, &m.QuarantineExpiresAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &m.Metadata)
		}
		if srcKind.Valid {
			m.SourceKind = srcKind.String
		}
		if srcRef.Valid {
			m.SourceRef = srcRef.String
		}
		if summSes.Valid {
			m.SummarizerSession = summSes.String
		}
		if conf.Valid {
			v := float32(conf.Float64)
			m.Confidence = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountQuarantined returns the active quarantine-tier row count.
func (s *PgvectorStore) CountQuarantined(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM memories
		WHERE tier = 'quarantine' AND archived_at IS NULL`).Scan(&n)
	if err != nil {
		if isRelationDoesNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("memory: count quarantined: %w", err)
	}
	return n, nil
}

// Promote moves a quarantined memory into the durable tier.
func (s *PgvectorStore) Promote(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE memories
		   SET tier = 'durable', quarantine_expires_at = NULL, updated_at = NOW()
		 WHERE id = $1 AND tier = 'quarantine' AND archived_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("memory: promote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PurgeExpiredQuarantine hard-deletes quarantined rows past their TTL.
func (s *PgvectorStore) PurgeExpiredQuarantine(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM memories
		 WHERE tier = 'quarantine'
		   AND quarantine_expires_at IS NOT NULL
		   AND quarantine_expires_at < $1`, now)
	if err != nil {
		return 0, fmt.Errorf("memory: purge expired quarantine: %w", err)
	}
	return tag.RowsAffected(), nil
}

// vectorLiteral renders a []float32 as the pgvector text format
// "[v1,v2,...]" pgx-compat. We could use the pgvector-go driver's
// custom type, but a string literal keeps the dependency surface
// flat and works equally well with prepared statements.
func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", x)
	}
	b.WriteByte(']')
	return b.String()
}

// sqlSafe returns a slug usable inside an identifier without
// quoting. Used only for index names so the input set is small.
func sqlSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// Compile-time guarantee.
var _ Store = (*PgvectorStore)(nil)

// pgxPoolEnsureUsed avoids "imported and not used" if the file is
// compiled in a context where pgx is unused. Harmless at runtime.
var _ = pgx.ErrNoRows
