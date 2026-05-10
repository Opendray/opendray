package memory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Service wires one Embedder + one Store together. Handlers and the
// MCP server hold a *Service; they don't reach into the underlying
// pieces directly.
//
// Lifecycle: built once at app startup, lives until shutdown. Safe
// for concurrent use.
type Service struct {
	emb       Embedder
	store     Store
	threshold float32
	topK      int
	scope     ScopeDefaults
	log       *slog.Logger

	// AutoDetected captures the embedding services opendray noticed
	// at startup (ollama / LM Studio on their default ports). Pure
	// metadata — never auto-switches the active embedder. The UI
	// uses this to surface "we see ollama running, click here to
	// switch your backend".
	autoDetected []ProbeResult

	// mirror is the optional ingestor for Claude's local <cwd>/.claude/
	// memory/*.md files. Wired by the app at startup; nil means the
	// HTTP "Sync now" endpoint returns 503.
	mirror *Mirror
}

// SetAutoDetected stores the results of a startup probe sweep so
// the UI can surface them. Idempotent.
func (s *Service) SetAutoDetected(hits []ProbeResult) { s.autoDetected = hits }

// AutoDetected returns the captured probe results. Empty when no
// service responded (BM25 fallback is the default).
func (s *Service) AutoDetected() []ProbeResult { return s.autoDetected }

// SetMirror wires a Mirror so the HTTP "Sync .md files now" button
// can trigger an on-demand ingest from outside the spawn-time path.
// Idempotent. Pass nil to disable.
func (s *Service) SetMirror(m *Mirror) { s.mirror = m }

// MirrorCwd runs an idempotent re-sync of Claude's <cwd>/.claude/
// memory/*.md files into the project-scoped pgvector store.
// Returns the number of new memories ingested in this call (0 when
// nothing changed). Returns ErrMirrorUnavailable when no mirror is
// wired (e.g. memory subsystem is in BM25-only mode).
func (s *Service) MirrorCwd(ctx context.Context, cwd string) (int, error) {
	if s.mirror == nil {
		return 0, ErrMirrorUnavailable
	}
	return s.mirror.SyncCwd(ctx, cwd)
}

// ScopeDefaults captures the operator's per-scope policy. These
// only set defaults — every API call can override.
type ScopeDefaults struct {
	Default Scope
}

// Options is the constructor argument bag. Everything is optional
// except Embedder, Store and Logger.
type Options struct {
	Embedder            Embedder
	Store               Store
	SimilarityThreshold float32
	DefaultTopK         int
	Scope               ScopeDefaults
	Logger              *slog.Logger
}

// New builds a Service. Sensible defaults fill in zero-valued
// options so the caller can pass a near-empty struct in tests.
func New(opts Options) (*Service, error) {
	if opts.Embedder == nil {
		return nil, errors.New("memory: Service requires an Embedder")
	}
	if opts.Store == nil {
		return nil, errors.New("memory: Service requires a Store")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.SimilarityThreshold <= 0 {
		// 0.1 is a permissive default — BM25 hash-bucket vectors
		// rarely score above ~0.3 even for clearly related text,
		// so we lean on Top-K for filtering and only use the
		// threshold to cut hits with literally zero overlap.
		// When operators wire in a dense embedder (HTTP backend
		// or LocalONNX bge-m3) they can tighten this in [memory]
		// to 0.6+ for stricter recall.
		opts.SimilarityThreshold = 0.1
	}
	if opts.DefaultTopK <= 0 {
		opts.DefaultTopK = 5
	}
	if opts.Scope.Default == "" {
		opts.Scope.Default = ScopeProject
	}
	return &Service{
		emb:       opts.Embedder,
		store:     opts.Store,
		threshold: opts.SimilarityThreshold,
		topK:      opts.DefaultTopK,
		scope:     opts.Scope,
		log:       opts.Logger.With("component", "memory"),
	}, nil
}

// Close releases the store's resources.
func (s *Service) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

// EmbedderName + StoreName are exposed for the Settings UI's
// "what's currently active?" status pane.
func (s *Service) EmbedderName() string { return s.emb.Name() }
func (s *Service) Dimensions() int      { return s.emb.Dimensions() }

// StoreRequest is the public shape callers (MCP, HTTP debug API)
// pass to Store. It mirrors InsertRequest minus the embedding
// (we compute that here).
//
// Provenance fields (Phase A) — all optional. SourceKind defaults
// to "manual" via DB CHECK constraint when left empty so existing
// callers (MCP tool, mirror, HTTP UI) need no changes.
type StoreRequest struct {
	Text     string                 `json:"text"`
	Scope    Scope                  `json:"scope"`
	ScopeKey string                 `json:"scope_key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Provenance — set by ambient memory writers (summarizer, mirror,
	// importer). Empty values cause DB defaults to apply.
	SourceKind        string   `json:"source_kind,omitempty"`        // 'manual'|'mcp_call'|'summarizer'|'mirror_claude_md'|'imported'
	SourceRef         string   `json:"source_ref,omitempty"`         // summarizer call id, mirror file path, etc.
	SummarizerSession string   `json:"summarizer_session,omitempty"` // session id when source_kind='summarizer'
	Confidence        *float32 `json:"confidence,omitempty"`         // summarizer self-reported 0..1
}

// SearchRequest mirrors a /memory.search tool call — text query
// plus the scope filter, no vector (we embed here).
//
// MinSimilarity overrides the service-level threshold. 0 = use
// default; explicit -1 keeps every hit regardless of score (handy
// for debugging). UI's "show all matches" toggle sends -1.
type SearchRequest struct {
	Query         string  `json:"query"`
	Scope         Scope   `json:"scope,omitempty"`
	ScopeKey      string  `json:"scope_key,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	MinSimilarity float32 `json:"min_similarity,omitempty"`
}

// Store embeds + persists a fact. Always inserts a new row; we
// don't dedupe / merge similar facts — operators trim with the
// inspector if a particular scope's memories pile up.
func (s *Service) Store(ctx context.Context, req StoreRequest) (string, error) {
	if strings.TrimSpace(req.Text) == "" {
		return "", errors.New("memory: empty text")
	}
	if req.Scope == "" {
		req.Scope = s.scope.Default
	}
	if err := req.Scope.Validate(); err != nil {
		return "", err
	}
	if req.Scope != ScopeGlobal && strings.TrimSpace(req.ScopeKey) == "" {
		return "", fmt.Errorf("memory: scope %q requires a scope_key", req.Scope)
	}

	emb, err := s.emb.Embed(ctx, []string{req.Text})
	if err != nil {
		return "", fmt.Errorf("memory: embed for store: %w", err)
	}
	if len(emb) != 1 {
		return "", fmt.Errorf("memory: embedder returned %d vectors", len(emb))
	}

	id, err := s.store.Insert(ctx, InsertRequest{
		Scope:             req.Scope,
		ScopeKey:          req.ScopeKey,
		Text:              req.Text,
		Embedder:          s.emb.Name(),
		Embedding:         emb[0],
		Metadata:          req.Metadata,
		SourceKind:        req.SourceKind,
		SourceRef:         req.SourceRef,
		SummarizerSession: req.SummarizerSession,
		Confidence:        req.Confidence,
	})
	if err != nil {
		return "", err
	}
	s.log.Info("memory.store", "id", id, "scope", req.Scope, "scope_key", req.ScopeKey, "len", len(req.Text))
	return id, nil
}

// Search embeds the query and asks the store for top-K similar
// memories, then drops anything below threshold so callers see only
// likely matches.
func (s *Service) Search(ctx context.Context, req SearchRequest) ([]SearchHit, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("memory: empty query")
	}
	if req.Scope == "" {
		req.Scope = s.scope.Default
	}
	if err := req.Scope.Validate(); err != nil {
		return nil, err
	}
	if req.Scope != ScopeGlobal && strings.TrimSpace(req.ScopeKey) == "" {
		return nil, fmt.Errorf("memory: scope %q requires a scope_key", req.Scope)
	}
	topK := req.TopK
	if topK <= 0 {
		topK = s.topK
	}

	t0 := time.Now()
	emb, err := s.emb.Embed(ctx, []string{req.Query})
	if err != nil {
		return nil, fmt.Errorf("memory: embed for search: %w", err)
	}

	hits, err := s.store.Search(ctx, SearchQuery{
		Vector:   emb[0],
		Embedder: s.emb.Name(),
		Scope:    req.Scope,
		ScopeKey: req.ScopeKey,
		TopK:     topK,
	})
	if err != nil {
		return nil, err
	}

	threshold := s.threshold
	switch {
	case req.MinSimilarity == -1:
		threshold = -2 // never filter
	case req.MinSimilarity > 0:
		threshold = req.MinSimilarity
	}
	out := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		if h.Similarity >= threshold {
			out = append(out, h)
		}
	}
	// Fire-and-forget: bump hit_count for every memory we're about to
	// return so the inspector can show "this fact has been used N times".
	// Detach from the request context so the bump survives even if the
	// caller hangs up after receiving the response. Best-effort by design.
	if len(out) > 0 {
		ids := make([]string, len(out))
		for i, h := range out {
			ids[i] = h.Memory.ID
		}
		go func(ids []string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.store.RecordHits(bgCtx, ids); err != nil {
				s.log.Debug("memory.record_hits_failed", "err", err, "n", len(ids))
			}
		}(ids)
	}
	s.log.Debug("memory.search", "query_len", len(req.Query), "hits", len(out), "kept_of", len(hits), "dur", time.Since(t0))
	return out, nil
}

// List proxies straight through. Used by the admin debug page —
// agents shouldn't need raw listing.
func (s *Service) List(ctx context.Context, scope Scope, scopeKey string, limit int) ([]Memory, error) {
	if scope == "" {
		scope = s.scope.Default
	}
	return s.store.List(ctx, scope, scopeKey, limit)
}

// Delete proxies straight through.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// DeleteByScope wipes every memory under (scope, scopeKey).
// Refuses to fire on non-global scopes with an empty scope_key —
// otherwise a typo would clear every project / session at once.
// Global scope explicitly accepts an empty scope_key (that's the
// only valid value there) so callers must pass scope=ScopeGlobal
// AND scopeKey="" together to wipe global memories. Returns the
// number of rows deleted.
func (s *Service) DeleteByScope(
	ctx context.Context,
	scope Scope,
	scopeKey string,
) (int64, error) {
	if err := scope.Validate(); err != nil {
		return 0, err
	}
	if scope != ScopeGlobal && strings.TrimSpace(scopeKey) == "" {
		return 0, fmt.Errorf(
			"memory: scope %q requires a scope_key for bulk delete", scope,
		)
	}
	if scope == ScopeGlobal && scopeKey != "" {
		return 0, fmt.Errorf(
			"memory: global scope must have empty scope_key (got %q)", scopeKey,
		)
	}
	n, err := s.store.DeleteByScope(ctx, scope, scopeKey)
	if err == nil && n > 0 {
		s.log.Info("memory.delete_by_scope",
			"scope", scope, "scope_key", scopeKey, "deleted", n)
	}
	return n, err
}

// Get returns one memory by id, including provenance fields.
// Used by the GET /memory/{id} admin endpoint and the
// memory_get_provenance MCP tool.
func (s *Service) Get(ctx context.Context, id string) (Memory, error) {
	return s.store.Get(ctx, id)
}

// EditRequest is the API-facing shape for an in-place memory edit.
// The Service re-embeds the new text before calling Store.Update —
// callers don't compute or pass vectors.
type EditRequest struct {
	ID       string                 `json:"-"`
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Update re-embeds the new text and overwrites the row. ID identity
// is preserved; scope/scope_key/embedder stay whatever was on the
// original row (Store.Update doesn't touch those columns).
func (s *Service) Update(ctx context.Context, req EditRequest) error {
	if strings.TrimSpace(req.ID) == "" {
		return errors.New("memory: update missing id")
	}
	if strings.TrimSpace(req.Text) == "" {
		return errors.New("memory: update empty text")
	}
	emb, err := s.emb.Embed(ctx, []string{req.Text})
	if err != nil {
		return fmt.Errorf("memory: embed for update: %w", err)
	}
	if len(emb) != 1 {
		return fmt.Errorf("memory: embedder returned %d vectors", len(emb))
	}
	if err := s.store.Update(ctx, UpdateRequest{
		ID:        req.ID,
		Text:      req.Text,
		Embedding: emb[0],
		Metadata:  req.Metadata,
	}); err != nil {
		return err
	}
	s.log.Info("memory.update", "id", req.ID, "len", len(req.Text))
	return nil
}

// ListScopeKeys returns distinct scope_key values currently used
// under the given scope, ordered alphabetically. Powers the UI's
// "browse used scope keys" picker.
func (s *Service) ListScopeKeys(ctx context.Context, scope Scope) ([]string, error) {
	if scope == "" {
		scope = s.scope.Default
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	return s.store.ListScopeKeys(ctx, scope)
}

// EmbedderStats reports how many memories live under each embedder
// name. Used by the Settings UI's "Migrate" panel to warn that
// older memories are silently invisible to the current embedder.
type EmbedderStats struct {
	Current string         `json:"current"`
	Counts  map[string]int `json:"counts"`
}

// EmbedderStats returns the current embedder name plus a
// map[embedder]→count over every stored memory. Cheap (one
// COUNT(*) GROUP BY).
func (s *Service) EmbedderStats(ctx context.Context) (EmbedderStats, error) {
	counts, err := s.store.CountByEmbedder(ctx)
	if err != nil {
		return EmbedderStats{}, err
	}
	return EmbedderStats{Current: s.emb.Name(), Counts: counts}, nil
}

// ReembedReport summarises one Reembed pass. All counts are
// across every scope; we don't filter.
type ReembedReport struct {
	Examined  int      `json:"examined"`
	Reembed   int      `json:"reembed"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
	StartedAt string   `json:"started_at"`
	EndedAt   string   `json:"ended_at"`
	From      []string `json:"from"`
	To        string   `json:"to"`
}

// Reembed walks every memory whose `embedder` column differs from
// the current embedder, recomputes its vector, and writes it back
// in place — id stays the same. This is the migration tool for
// when an operator switches embedder backends mid-flight (e.g.
// BM25 → bge-m3); without it the older memories silently drop out
// of search because pgvector's similarity index is partitioned
// by (embedder, dim).
//
// Synchronous + sequential — fine for the kilo-row scale we expect
// from a single operator's gateway. Batches of `batchSize` keep
// memory usage flat. ctx cancellation aborts cleanly between
// batches.
func (s *Service) Reembed(ctx context.Context, batchSize int) (ReembedReport, error) {
	if batchSize <= 0 {
		batchSize = 32
	}
	current := s.emb.Name()
	report := ReembedReport{
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		To:        current,
	}
	fromSet := map[string]struct{}{}

	cursor := ""
	for {
		if err := ctx.Err(); err != nil {
			report.EndedAt = time.Now().UTC().Format(time.RFC3339)
			report.From = setKeys(fromSet)
			return report, err
		}
		batch, err := s.store.ListNeedingReembed(ctx, current, batchSize, cursor)
		if err != nil {
			report.EndedAt = time.Now().UTC().Format(time.RFC3339)
			report.From = setKeys(fromSet)
			return report, fmt.Errorf("memory: list needing reembed: %w", err)
		}
		if len(batch) == 0 {
			break
		}
		report.Examined += len(batch)

		texts := make([]string, len(batch))
		for i, m := range batch {
			texts[i] = m.Text
			fromSet[m.Embedder] = struct{}{}
		}
		vecs, err := s.emb.Embed(ctx, texts)
		if err != nil {
			// Whole batch fails — record one error and advance the
			// cursor past the batch so we don't loop forever.
			report.Failed += len(batch)
			report.Errors = appendCapped(report.Errors,
				fmt.Sprintf("embed batch starting %s: %v", batch[0].ID, err), 20)
			cursor = batch[len(batch)-1].ID
			continue
		}
		for i, m := range batch {
			if err := s.store.Update(ctx, UpdateRequest{
				ID:        m.ID,
				Text:      m.Text,
				Embedding: vecs[i],
				Embedder:  current,
				Metadata:  m.Metadata,
			}); err != nil {
				report.Failed++
				report.Errors = appendCapped(report.Errors,
					fmt.Sprintf("update %s: %v", m.ID, err), 20)
				continue
			}
			report.Reembed++
		}
		cursor = batch[len(batch)-1].ID
		// If we got a partial batch, we know we've drained the table.
		if len(batch) < batchSize {
			break
		}
	}

	report.EndedAt = time.Now().UTC().Format(time.RFC3339)
	report.From = setKeys(fromSet)
	s.log.Info("memory.reembed",
		"examined", report.Examined,
		"reembed", report.Reembed,
		"failed", report.Failed,
		"from", report.From, "to", report.To,
	)
	return report, nil
}

func setKeys(s map[string]struct{}) []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	return out
}

func appendCapped(s []string, x string, cap int) []string {
	if len(s) >= cap {
		return s
	}
	return append(s, x)
}
