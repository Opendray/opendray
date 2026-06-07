// Package memquery composes layer-5 (memory facts) and
// layer-2-4 (projectdoc journal + goal/plan) into a single
// search surface. Agents talk to it through the new
// project_search MCP tool; operators get the same endpoint over
// REST for the UI's cross-layer search panel.
//
// Why a separate package: memory and projectdoc are deliberately
// kept one-way (projectdoc must not import internal/memory).
// memquery sits above both and is consumed by the MCP / HTTP
// layer, paralleling the memhealth split.
package memquery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// SourceLayer names which memory layer a hit came from. Surfaced
// in API responses so callers can render layer badges (e.g.
// "memory" vs "journal") and filter post hoc.
type SourceLayer string

const (
	SourceFact    SourceLayer = "fact"    // memories table
	SourceJournal SourceLayer = "journal" // session_logs table
	SourceGoal    SourceLayer = "goal"    // project_docs.kind='goal'
	SourcePlan    SourceLayer = "plan"    // project_docs.kind='plan'
)

// Hit is one merged search result. Raw cosine Similarity is run
// through the shared memory ranker into EffectiveScore (the final
// ordering key); Similarity is preserved for debugging / UI display.
// HitCount / Confidence feed the ranker: fact rows carry the real
// values so a popular fact outranks an equal-cosine journal line;
// journal / goal / plan rows leave them zero/nil so they score on
// similarity × recency alone.
type Hit struct {
	Source         SourceLayer `json:"source"`
	ID             string      `json:"id"`
	Text           string      `json:"text"`
	Title          string      `json:"title,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	Similarity     float32     `json:"similarity"`
	EffectiveScore float32     `json:"effective_score"`
	HitCount       int64       `json:"hit_count,omitempty"`
	Confidence     *float32    `json:"confidence,omitempty"`
}

// SearchRequest scopes one cross-layer query.
type SearchRequest struct {
	Cwd   string
	Query string
	TopK  int
}

// Service runs cross-layer queries. Constructed once per process.
// All three deps are required: memory.Service for fact search
// (and the shared Embedder), projectdoc.Service for the journal
// vector column + goal/plan reads, pgxpool because we issue raw
// vector SQL against session_logs (projectdoc doesn't expose a
// typed Search method yet).
type Service struct {
	mem  *memory.Service
	docs *projectdoc.Service
	pool *pgxpool.Pool
}

// New wires a Service. Returns an error when any required
// dependency is nil so the composition root surfaces the problem
// at boot instead of producing empty searches at runtime.
func New(mem *memory.Service, docs *projectdoc.Service, pool *pgxpool.Pool) (*Service, error) {
	if mem == nil {
		return nil, errors.New("memquery: memory service required")
	}
	if docs == nil {
		return nil, errors.New("memquery: projectdoc service required")
	}
	if pool == nil {
		return nil, errors.New("memquery: pool required")
	}
	return &Service{mem: mem, docs: docs, pool: pool}, nil
}

// Search runs the three sub-searches concurrently, merges the results
// through the one shared ranker, and returns the top-K by effective
// score. Per-layer failures degrade gracefully — a broken journal index
// doesn't stop fact + plan hits from surfacing. The layers hit
// independent tables (memories / session_logs / project_docs), so
// fanning them out makes the wall-clock the slowest single layer rather
// than the sum.
func (s *Service) Search(ctx context.Context, req SearchRequest) ([]Hit, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return nil, errors.New("memquery: cwd required")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("memquery: query required")
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 100 {
		topK = 100
	}

	var (
		facts, journal, docs []Hit
		wg                   sync.WaitGroup
	)
	wg.Add(3)
	go func() { defer wg.Done(); facts = s.searchFacts(ctx, req, topK) }()
	go func() { defer wg.Done(); journal = s.searchJournal(ctx, req, topK) }()
	go func() { defer wg.Done(); docs = s.searchDocs(ctx, req, topK) }()
	wg.Wait()

	all := make([]Hit, 0, len(facts)+len(journal)+len(docs))
	all = append(all, facts...)
	all = append(all, journal...)
	all = append(all, docs...)

	// Score every layer through the one shared ranker (the same
	// memory.RankingScoreFields the single-layer memory.Search uses), so
	// a fact ranks identically here and in memory_search. Journal / goal
	// / plan rows pass hitCount=0 / confidence=nil, reducing their score
	// to similarity × age — no second formula to drift.
	now := time.Now().UTC()
	for i := range all {
		all[i].EffectiveScore = memory.RankingScoreFields(
			all[i].Similarity, all[i].CreatedAt, all[i].HitCount, all[i].Confidence, now,
		)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].EffectiveScore > all[j].EffectiveScore
	})
	if len(all) > topK {
		all = all[:topK]
	}
	return all, nil
}

// searchFacts runs the existing memory.Search and converts hits.
// Failures are logged into the returned slice as zero hits — the
// caller continues with whatever the other layers produced.
func (s *Service) searchFacts(ctx context.Context, req SearchRequest, topK int) []Hit {
	hits, err := s.mem.Search(ctx, memory.SearchRequest{
		Query:    req.Query,
		Scope:    memory.ScopeProject,
		ScopeKey: req.Cwd,
		TopK:     topK,
	})
	if err != nil {
		return nil
	}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		out = append(out, Hit{
			Source:     SourceFact,
			ID:         h.Memory.ID,
			Text:       h.Memory.Text,
			CreatedAt:  h.Memory.CreatedAt,
			Similarity: h.Similarity,
			HitCount:   h.Memory.HitCount,
			Confidence: h.Memory.Confidence,
		})
	}
	return out
}

// searchJournal embeds the query and runs a raw vector query
// against session_logs. We bypass projectdoc.Service to keep
// projectdoc free of a vector-search API surface — the package's
// public API is doc CRUD + journal append + spawn render.
//
// Filters: scope by cwd; only rows with a matching embedder so
// cosine comparisons stay sound; LIMIT by topK.
func (s *Service) searchJournal(ctx context.Context, req SearchRequest, topK int) []Hit {
	embedder := s.docs.Embedder()
	if embedder == nil {
		return nil
	}
	vecs, err := embedder.Embed(ctx, []string{req.Query})
	if err != nil || len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, content, created_at,
		       1 - (embedding <=> $1::vector) AS similarity
		  FROM session_logs
		 WHERE cwd = $2
		   AND embedder = $3
		   AND embedding IS NOT NULL
		 ORDER BY embedding <=> $1::vector
		 LIMIT $4`,
		pgvecLiteral(vecs[0]), req.Cwd, embedder.Name(), topK)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Hit
	for rows.Next() {
		var (
			id, title, content string
			createdAt          time.Time
			similarity         float64
		)
		if err := rows.Scan(&id, &title, &content, &createdAt, &similarity); err != nil {
			return out
		}
		out = append(out, Hit{
			Source:     SourceJournal,
			ID:         id,
			Title:      title,
			Text:       content,
			CreatedAt:  createdAt,
			Similarity: float32(similarity),
		})
	}
	return out
}

// searchDocs matches the query against goal + plan docs. Embedded docs
// (the common case after Phase 2) are scored semantically by cosine,
// exactly like facts and journal, so a goal can rank by meaning rather
// than by literal substring. Docs not yet embedded (embedder outage, or
// the brief backfill window) fall back to a lexical substring match at a
// fixed 0.6 similarity so a fresh edit is still findable. project_docs
// is tiny per cwd (one row per kind), so doing both passes is cheap.
func (s *Service) searchDocs(ctx context.Context, req SearchRequest, topK int) []Hit {
	docLayer := func(kind projectdoc.Kind) SourceLayer {
		if kind == projectdoc.KindPlan {
			return SourcePlan
		}
		return SourceGoal
	}

	var out []Hit
	embedded := map[string]bool{} // doc ids covered by the semantic pass

	// Semantic pass over embedded goal/plan docs.
	if embedder := s.docs.Embedder(); embedder != nil {
		if vecs, err := embedder.Embed(ctx, []string{req.Query}); err == nil && len(vecs) > 0 && len(vecs[0]) > 0 {
			rows, err := s.pool.Query(ctx, `
				SELECT id, kind, content, updated_at,
				       1 - (embedding <=> $1::vector) AS similarity
				  FROM project_docs
				 WHERE cwd = $2
				   AND kind IN ('goal', 'plan')
				   AND embedder = $3
				   AND embedding IS NOT NULL
				 ORDER BY embedding <=> $1::vector
				 LIMIT $4`,
				pgvecLiteral(vecs[0]), req.Cwd, embedder.Name(), topK)
			if err == nil {
				for rows.Next() {
					var (
						id, kind, content string
						updatedAt         time.Time
						similarity        float64
					)
					if err := rows.Scan(&id, &kind, &content, &updatedAt, &similarity); err != nil {
						break
					}
					embedded[id] = true
					out = append(out, Hit{
						Source:     docLayer(projectdoc.Kind(kind)),
						ID:         id,
						Text:       content,
						CreatedAt:  updatedAt,
						Similarity: float32(similarity),
					})
				}
				rows.Close()
			}
		}
	}

	// Lexical fallback for goal/plan docs the semantic pass didn't cover.
	docs, err := s.docs.ListDocsForCwd(ctx, req.Cwd)
	if err != nil {
		return out
	}
	q := strings.ToLower(strings.TrimSpace(req.Query))
	if q == "" {
		return out
	}
	for _, d := range docs {
		if d.Kind != projectdoc.KindGoal && d.Kind != projectdoc.KindPlan {
			continue
		}
		if embedded[d.ID] {
			continue
		}
		if !strings.Contains(strings.ToLower(d.Content), q) {
			continue
		}
		out = append(out, Hit{
			Source:     docLayer(d.Kind),
			ID:         d.ID,
			Text:       d.Content,
			CreatedAt:  d.UpdatedAt,
			Similarity: 0.6,
		})
	}
	return out
}

// pgvecLiteral mirrors projectdoc.pgvecString. Duplicated rather
// than exported to keep the projectdoc API surface minimal — this
// is an implementation detail of one query.
func pgvecLiteral(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.Grow(2 + len(v)*8)
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}

// silence "imported and not used" when pgx itself is only needed
// transitively. Some Go versions complain otherwise.
var _ = pgx.ErrNoRows
