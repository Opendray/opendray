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

// Hit is one merged search result. Cosine similarity gets time-
// decayed into EffectiveScore which is what the final ordering
// uses; raw Similarity is preserved for debugging / UI display.
type Hit struct {
	Source         SourceLayer `json:"source"`
	ID             string      `json:"id"`
	Text           string      `json:"text"`
	Title          string      `json:"title,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	Similarity     float32     `json:"similarity"`
	EffectiveScore float32     `json:"effective_score"`
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

// Search runs the three sub-searches sequentially, merges the
// results with time decay applied, and returns the top-K by
// effective score. Per-layer failures degrade gracefully — a
// broken journal index doesn't stop fact + plan hits from
// surfacing.
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

	var all []Hit
	all = append(all, s.searchFacts(ctx, req, topK)...)
	all = append(all, s.searchJournal(ctx, req, topK)...)
	all = append(all, s.searchDocs(ctx, req)...)

	// Apply time decay then sort by effective score descending.
	now := time.Now().UTC()
	for i := range all {
		all[i].EffectiveScore = decayScore(all[i].Similarity, now.Sub(all[i].CreatedAt))
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

// searchDocs lexically matches the query against goal + plan
// content. project_docs is a tiny set per cwd (one row per kind)
// so the cost of scanning + Lower-comparing both is negligible —
// no embedding needed. If a match exists we attach a fixed
// similarity of 0.6 so the doc shows up alongside vector hits
// without dominating them.
func (s *Service) searchDocs(ctx context.Context, req SearchRequest) []Hit {
	docs, err := s.docs.ListDocsForCwd(ctx, req.Cwd)
	if err != nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(req.Query))
	if q == "" {
		return nil
	}
	var out []Hit
	for _, d := range docs {
		if d.Kind != projectdoc.KindGoal && d.Kind != projectdoc.KindPlan {
			continue
		}
		if !strings.Contains(strings.ToLower(d.Content), q) {
			continue
		}
		layer := SourceGoal
		if d.Kind == projectdoc.KindPlan {
			layer = SourcePlan
		}
		out = append(out, Hit{
			Source:     layer,
			ID:         d.ID,
			Text:       d.Content,
			CreatedAt:  d.UpdatedAt,
			Similarity: 0.6,
		})
	}
	return out
}

// decayScore applies a gentle linear decay so a 6-month-old
// memory ranks ~50% of a brand-new one at the same cosine, but a
// brand-new mediocre match doesn't crowd out a year-old gem. The
// 0.5 floor means recency can never outweigh relevance entirely.
func decayScore(similarity float32, age time.Duration) float32 {
	days := float32(age.Hours()) / 24
	if days < 0 {
		days = 0
	}
	decay := 1 - days/180
	if decay < 0.5 {
		decay = 0.5
	}
	return similarity * decay
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
