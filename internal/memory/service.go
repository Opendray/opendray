package memory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
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

	// dedupThreshold (M11): cosine similarity above which Store
	// merges into an existing row instead of inserting a new one.
	// 0 disables dedup. Sensible defaults are ~0.85 for dense
	// embedders (bge-m3 / ada-002), ~0.4 for BM25.
	dedupThreshold float32

	// gatekeeper (M12): optional pre-write LLM that judges whether
	// a memory is durable. nil → no gatekeeping (agent's own
	// judgement only).
	gatekeeper Gatekeeper

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

// Gatekeeper is the contract a pre-write LLM judge satisfies (M12).
// Returns (durable, category, reason). When durable=false the Store
// path short-circuits and rejects the write, surfacing reason in the
// returned error so the agent can adjust.
//
// Implementations should be fast (≤ 2s) and cheap; a remote 30B-class
// model is overkill — Haiku / LM Studio with a 3B model is the
// expected backend.
type Gatekeeper interface {
	Judge(ctx context.Context, text string) (durable bool, category string, reason string, err error)
}

// ErrNotDurable is returned by Store when the gatekeeper rejects a
// write. Surfaced as a "tool error" by the MCP wrapper so the agent
// sees the rejection reason but the session is otherwise unaffected.
var ErrNotDurable = errors.New("memory: gatekeeper rejected — not a durable fact")

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

	// DedupThreshold (M11) — cosine similarity above which Store
	// merges into an existing row instead of inserting a new one.
	// 0 (zero value) disables dedup; that's the historical behaviour
	// and stays the default for fresh installs until the operator
	// opts in via config.
	DedupThreshold float32

	// Gatekeeper (M12) — optional pre-write LLM judge. nil disables
	// gatekeeping. See Gatekeeper interface for the contract.
	Gatekeeper Gatekeeper

	Logger *slog.Logger
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
	if opts.DedupThreshold < 0 {
		opts.DedupThreshold = 0
	}
	return &Service{
		emb:            opts.Embedder,
		store:          opts.Store,
		threshold:      opts.SimilarityThreshold,
		topK:           opts.DefaultTopK,
		scope:          opts.Scope,
		dedupThreshold: opts.DedupThreshold,
		gatekeeper:     opts.Gatekeeper,
		log:            opts.Logger.With("component", "memory"),
	}, nil
}

// SetGatekeeper installs (or removes) the pre-write LLM judge after
// Service construction. Used by the app when the summarizer registry
// finishes booting after memory.New runs. Pass nil to disable.
func (s *Service) SetGatekeeper(g Gatekeeper) { s.gatekeeper = g }

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

// Embedder returns the active Embedder. Exposed so cross-package
// callers (M-PB journal indexing in projectdoc, the cross-layer
// search service) can use the same vector space the memory store
// already speaks — comparing vectors produced by different
// embedders is meaningless, so sharing the instance is load-
// bearing.
func (s *Service) Embedder() Embedder { return s.emb }

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

// Store embeds + persists a fact. Two filters run before the insert:
//
//  1. (M12) Gatekeeper — if installed, an LLM judges "is this a
//     durable cross-session fact?" Rejection returns ErrNotDurable
//     so the MCP wrapper can surface the reason to the agent.
//  2. (M11) Dedup-on-store — if dedupThreshold > 0, search the
//     same scope for a near-duplicate. If top-1 similarity ≥
//     threshold, merge into that row (overwrite text, re-embed)
//     instead of inserting a new one. The returned id is the
//     existing row, with metadata.deduped_count++ so audit views
//     can see how often this kicks in.
//
// Both filters are best-effort: a gatekeeper error degrades to
// "allow" rather than blocking the write (we'd rather let noise
// through than silently drop signal during an outage). A dedup
// search error degrades to "insert a new row" — same fallback.
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

	// (M12) Gatekeeper — pre-write LLM judgement.
	if s.gatekeeper != nil {
		durable, category, reason, gerr := s.gatekeeper.Judge(ctx, req.Text)
		if gerr != nil {
			// Outage / timeout — fall through to insert. Log so the
			// operator notices the gate is mis-configured.
			s.log.Warn("memory.gatekeeper_error", "err", gerr, "text_len", len(req.Text))
		} else if !durable {
			s.log.Info("memory.store_rejected",
				"reason", reason, "scope", req.Scope, "scope_key", req.ScopeKey, "len", len(req.Text))
			return "", fmt.Errorf("%w: %s", ErrNotDurable, reason)
		} else if category != "" && req.Metadata != nil {
			// Auto-tag if gatekeeper assigned a category and caller
			// didn't provide one. Useful for agents that forgot to
			// set metadata.type.
			if _, has := req.Metadata["type"]; !has {
				req.Metadata["type"] = category
			}
		} else if category != "" && req.Metadata == nil {
			req.Metadata = map[string]any{"type": category}
		}
	}

	emb, err := s.emb.Embed(ctx, []string{req.Text})
	if err != nil {
		return "", fmt.Errorf("memory: embed for store: %w", err)
	}
	if len(emb) != 1 {
		return "", fmt.Errorf("memory: embedder returned %d vectors", len(emb))
	}

	// (M11) Dedup-on-store. Reuses the embedding we just computed so
	// we don't pay a second embed call for the same text.
	if s.dedupThreshold > 0 {
		hits, sErr := s.store.Search(ctx, SearchQuery{
			Vector:   emb[0],
			Embedder: s.emb.Name(),
			Scope:    req.Scope,
			ScopeKey: req.ScopeKey,
			TopK:     1,
		})
		if sErr != nil {
			s.log.Warn("memory.dedup_search_failed", "err", sErr)
		} else if len(hits) > 0 && hits[0].Similarity >= s.dedupThreshold {
			existing := hits[0].Memory
			merged := mergeMetadata(existing.Metadata, req.Metadata)
			merged["deduped_count"] = dedupedCount(existing.Metadata) + 1
			merged["deduped_last_at"] = time.Now().UTC().Format(time.RFC3339)
			if uErr := s.store.Update(ctx, UpdateRequest{
				ID:        existing.ID,
				Text:      req.Text,
				Embedder:  s.emb.Name(),
				Embedding: emb[0],
				Metadata:  merged,
			}); uErr != nil {
				s.log.Warn("memory.dedup_update_failed", "err", uErr, "merged_into", existing.ID)
				// Fall through to insert — better duplicate than lost signal.
			} else {
				s.log.Info("memory.store_deduped",
					"id", existing.ID,
					"similarity", hits[0].Similarity,
					"threshold", s.dedupThreshold,
					"scope", req.Scope, "scope_key", req.ScopeKey)
				return existing.ID, nil
			}
		}
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

// mergeMetadata combines an existing memory's metadata with the
// incoming write's metadata. Incoming values win on key collision so
// agent corrections ("oh actually it's pnpm not yarn") replace stale
// values. Always returns a non-nil map so callers can safely set keys
// without nil checks.
func mergeMetadata(existing, incoming map[string]any) map[string]any {
	out := make(map[string]any, len(existing)+len(incoming)+2)
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range incoming {
		out[k] = v
	}
	return out
}

// dedupedCount returns the previous deduped_count value from
// metadata, or 0 if missing / malformed. Defensive against the JSON
// round-tripping that turns ints into float64s.
func dedupedCount(meta map[string]any) int {
	if meta == nil {
		return 0
	}
	switch v := meta["deduped_count"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
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
	// M-PC — compute the effective score for each hit (similarity
	// dampened by age, lifted by hit count + stored confidence) and
	// re-sort by it. Threshold filtering keeps using the raw
	// similarity so an explicit MinSimilarity from the caller behaves
	// like it always did; the new score affects ordering only.
	out := make([]SearchHit, 0, len(hits))
	now := time.Now().UTC()
	for _, h := range hits {
		if h.Similarity < threshold {
			continue
		}
		h.EffectiveScore = RankingScore(h.Similarity, h.Memory, now)
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].EffectiveScore > out[j].EffectiveScore
	})
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
