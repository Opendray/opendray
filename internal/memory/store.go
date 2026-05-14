package memory

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

// Scope is the visibility band a memory belongs to. The MCP
// `memory.search` tool defaults to retrieving across the calling
// session's project but a caller can override per-query.
type Scope string

const (
	ScopeSession Scope = "session"
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

// Validate normalises and rejects unknown scopes. Empty input
// returns ErrInvalidScope so callers default explicitly.
func (s Scope) Validate() error {
	switch s {
	case ScopeSession, ScopeProject, ScopeGlobal:
		return nil
	}
	return ErrInvalidScope
}

var (
	// ErrInvalidScope means the caller passed something other than
	// session / project / global.
	ErrInvalidScope = errors.New("memory: invalid scope")
	// ErrNotFound means a Get or Delete by id missed.
	ErrNotFound = errors.New("memory: not found")
	// ErrDimensionMismatch means the embedder's Dimensions() doesn't
	// agree with the stored vector at insert time. Real-world cause:
	// operator changed embedder without flushing memories.
	ErrDimensionMismatch = errors.New("memory: vector dimension mismatch")
	// ErrMirrorUnavailable means the operator hit the manual mirror
	// sync endpoint but no Mirror was wired into the Service.
	ErrMirrorUnavailable = errors.New("memory: mirror not configured")
)

// Memory is a stored fact the agent decided was worth remembering.
// Public fields are JSON-tagged for the admin debug API.
type Memory struct {
	ID        string                 `json:"id"`
	Scope     Scope                  `json:"scope"`
	ScopeKey  string                 `json:"scope_key"`
	Text      string                 `json:"text"`
	Embedder  string                 `json:"embedder"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`

	// HitCount tracks how many times this memory has been returned
	// from a search (post-threshold filter). Bumped lazily by
	// Store.RecordHits — best-effort, never blocks the caller.
	HitCount int64 `json:"hit_count"`
	// LastHitAt is the timestamp of the most recent hit, or nil
	// when the memory has never been retrieved. Sortable in the
	// inspector to surface stale memories.
	LastHitAt *time.Time `json:"last_hit_at,omitempty"`

	// Embedding is omitted from JSON by default — vectors are noisy
	// and rarely useful in admin views. Inspector dumps include it
	// via a separate endpoint.
	Embedding []float32 `json:"-"`

	// Provenance — populated by Get() but absent from List() output
	// since it's not interesting for the inspector's row table.
	SourceKind        string   `json:"source_kind,omitempty"`
	SourceRef         string   `json:"source_ref,omitempty"`
	SummarizerSession string   `json:"summarizer_session,omitempty"`
	Confidence        *float32 `json:"confidence,omitempty"`
}

// SearchHit is one match returned by Store.Search, paired with its
// cosine similarity score for caller-side cutoff / display.
//
// EffectiveScore is the M-PC ranking signal — similarity adjusted
// for memory age, retrieval frequency, and stored confidence. The
// Store implementations leave it zero; Service.Search computes and
// sets it. The final ordering uses EffectiveScore but Similarity
// is preserved so the inspector can show the raw cosine number.
type SearchHit struct {
	Memory         Memory  `json:"memory"`
	Similarity     float32 `json:"similarity"`
	EffectiveScore float32 `json:"effective_score,omitempty"`
}

// SearchQuery describes a similarity search. ScopeKey is required
// for session and project scope; ignored for global.
type SearchQuery struct {
	Vector   []float32
	Embedder string
	Scope    Scope
	ScopeKey string
	TopK     int
}

// InsertRequest is the shape passed to Store.Insert.
//
// Provenance fields (Phase A): empty strings + nil Confidence are
// the "no opinion" signal — Store implementations let DB defaults
// kick in (source_kind='manual', other columns NULL).
type InsertRequest struct {
	Scope     Scope
	ScopeKey  string
	Text      string
	Embedder  string
	Embedding []float32
	Metadata  map[string]interface{}

	SourceKind        string
	SourceRef         string
	SummarizerSession string
	Confidence        *float32
}

// UpdateRequest carries the new text + (optional) metadata for an
// in-place edit. The caller is expected to re-embed the text
// before calling Update; Store implementations are dumb pipes for
// the writeback. scope / scope_key are identity for the row and
// never change.
//
// Embedder is normally identity too, but reembed migrations need
// to bump it together with the new vector — so when Embedder is
// non-empty, Store.Update overwrites that column too. Pass empty
// to keep the existing embedder.
type UpdateRequest struct {
	ID        string
	Text      string
	Embedding []float32
	Embedder  string
	Metadata  map[string]interface{}
}

// Store persists memories and answers similarity queries. Both
// pgvector and chromem-go implementations satisfy this interface;
// callers (the MCP server, the admin debug handler) work against
// it without knowing which is wired up.
type Store interface {
	// Insert persists a new memory and returns the assigned id.
	Insert(ctx context.Context, req InsertRequest) (string, error)
	// Update overwrites the text + embedding of an existing memory
	// in place. scope, scope_key, and embedder stay fixed.
	Update(ctx context.Context, req UpdateRequest) error
	// Search returns the top-K most similar memories, descending
	// by cosine similarity, filtered by scope.
	Search(ctx context.Context, q SearchQuery) ([]SearchHit, error)
	// List returns memories for a scope without similarity
	// ranking — used by the admin debug UI.
	List(ctx context.Context, scope Scope, scopeKey string, limit int) ([]Memory, error)
	// ListScopeKeys returns distinct scope_key values currently
	// stored under a given scope. Powers the UI's "browse used
	// scope keys" picker so operators don't have to type cwds.
	ListScopeKeys(ctx context.Context, scope Scope) ([]string, error)
	// CountByEmbedder returns a map[embedder]→row count across
	// every scope. Used by the reembed migration tool to show
	// "you have 1234 memories on bm25 that the current embedder
	// will silently miss".
	CountByEmbedder(ctx context.Context) (map[string]int, error)
	// ListNeedingReembed paginates memories whose embedder != the
	// supplied currentEmbedder, in id order. afterID="" starts at
	// the beginning; pass the last seen id to get the next page.
	// Used by the reembed migration loop.
	ListNeedingReembed(ctx context.Context, currentEmbedder string, limit int, afterID string) ([]Memory, error)
	// RecordHits bumps hit_count + last_hit_at for the given ids in
	// one statement. Best-effort: implementations should swallow
	// errors and log them — never propagate, since search results
	// have already been handed to the caller.
	RecordHits(ctx context.Context, ids []string) error
	// Get returns one Memory row by id, including provenance fields.
	// Returns ErrNotFound when the id is missing.
	Get(ctx context.Context, id string) (Memory, error)
	// Delete removes a memory by id; returns ErrNotFound when the
	// id wasn't there.
	Delete(ctx context.Context, id string) error
	// DeleteByScope wipes every memory under the given (scope,
	// scope_key) pair in a single SQL operation. Returns the row
	// count actually removed; zero is not an error.
	DeleteByScope(ctx context.Context, scope Scope, scopeKey string) (int64, error)
	// Close releases store-level resources (DB conns, files).
	// Safe to call multiple times.
	Close() error
}

// NewID returns a fresh memory id with a stable prefix. The 12-byte
// random suffix keeps collisions astronomically unlikely while
// staying short enough to surface in the admin UI.
func NewID() string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return "mem_" + base64.RawURLEncoding.EncodeToString(b[:])
}
