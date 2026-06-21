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
	// SourceDoc — any other per-project blueprint section.
	SourceDoc SourceLayer = "doc"
	// SourceKnowledge — a global kb_* knowledge page. Searching pulls
	// the one relevant page on demand instead of injecting everything.
	SourceKnowledge SourceLayer = "knowledge"
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
	// Slug + Section are set on global KB (SourceKnowledge) hits so the
	// caller can render an actionable `doc_read(slug, section=…)` pointer.
	// For KB hits Text is the matched SECTION, not the whole page — a
	// big page (e.g. 59K kb_integrations) never crosses the wire whole.
	Slug    string `json:"slug,omitempty"`
	Section string `json:"section,omitempty"`
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
	docLayer := func(cwd string, kind projectdoc.Kind) SourceLayer {
		switch {
		case cwd == projectdoc.GlobalCwd:
			return SourceKnowledge
		case kind == projectdoc.KindPlan:
			return SourcePlan
		case kind == projectdoc.KindGoal:
			return SourceGoal
		}
		return SourceDoc
	}

	var out []Hit
	embedded := map[string]bool{} // doc ids covered by the semantic pass

	// Semantic pass over embedded docs: every section of THIS project
	// plus the global knowledge pages — so an agent retrieves exactly
	// the section/page a task needs instead of swallowing the whole
	// injected corpus.
	if embedder := s.docs.Embedder(); embedder != nil {
		if vecs, err := embedder.Embed(ctx, []string{req.Query}); err == nil && len(vecs) > 0 && len(vecs[0]) > 0 {
			rows, err := s.pool.Query(ctx, `
				SELECT id, cwd, kind, content, updated_at,
				       1 - (embedding <=> $1::vector) AS similarity
				  FROM project_docs
				 WHERE cwd IN ($2, $3)
				   AND embedder = $4
				   AND embedding IS NOT NULL
				 ORDER BY embedding <=> $1::vector
				 LIMIT $5`,
				pgvecLiteral(vecs[0]), req.Cwd, projectdoc.GlobalCwd, embedder.Name(), topK)
			if err == nil {
				for rows.Next() {
					var (
						id, cwd, kind, content string
						updatedAt              time.Time
						similarity             float64
					)
					if err := rows.Scan(&id, &cwd, &kind, &content, &updatedAt, &similarity); err != nil {
						break
					}
					embedded[id] = true
					hit := Hit{
						Source:     docLayer(cwd, projectdoc.Kind(kind)),
						ID:         id,
						Title:      string(kind),
						Text:       content,
						CreatedAt:  updatedAt,
						Similarity: float32(similarity),
					}
					if cwd == projectdoc.GlobalCwd {
						hit.Text, hit.Section = kbSection(content, req.Query)
						hit.Slug = kind
					}
					out = append(out, hit)
				}
				rows.Close()
			}
		}
	}

	// Lexical fallback for docs the semantic pass didn't cover (fresh
	// edits inside the backfill window, embedder outages).
	q := strings.ToLower(strings.TrimSpace(req.Query))
	if q == "" {
		return out
	}
	for _, cwd := range []string{req.Cwd, projectdoc.GlobalCwd} {
		docs, err := s.docs.ListDocsForCwd(ctx, cwd)
		if err != nil {
			continue
		}
		for _, d := range docs {
			if embedded[d.ID] {
				continue
			}
			if !strings.Contains(strings.ToLower(d.Content), q) {
				continue
			}
			hit := Hit{
				Source:     docLayer(cwd, d.Kind),
				ID:         d.ID,
				Title:      string(d.Kind),
				Text:       d.Content,
				CreatedAt:  d.UpdatedAt,
				Similarity: 0.6,
			}
			if cwd == projectdoc.GlobalCwd {
				hit.Text, hit.Section = kbSection(d.Content, req.Query)
				hit.Slug = string(d.Kind)
			}
			out = append(out, hit)
		}
	}
	return out
}

// kbSection narrows a global KB page hit to the section most relevant to the
// query. Per-section embeddings don't exist in Phase 0, so intra-page
// selection is lexical: score each section by query-term frequency (heading
// hits weigh more). Returns the section text + its heading; falls back to the
// first section when there's no lexical signal, or the whole content when the
// page has no headings. Keeps a big page (e.g. 59K kb_integrations) from
// crossing the wire whole.
func kbSection(content, query string) (text, section string) {
	headings := projectdoc.ListSectionHeadings(content)
	if len(headings) == 0 {
		return cappedWhole(content), ""
	}
	terms := kbSectionTerms(query)
	// Score by DENSITY (body hits per ~1K chars) + a strong heading-term
	// bonus. Density is what stops a "container" section (e.g. the H1 whose
	// slice is the whole 60K page, which trivially contains every term) from
	// always winning over a focused subsection. A heading-term match is the
	// strongest signal, so it dominates.
	bestScore := 0.0
	bestText, bestHeading := "", ""
	for _, h := range headings {
		sec, ok := projectdoc.SliceSection(content, h)
		if !ok || len(sec) == len(content) {
			continue // skip the whole-page container heading entirely
		}
		lowerSec := strings.ToLower(sec)
		lowerH := strings.ToLower(h)
		bodyHits, headingHits := 0, 0
		for _, t := range terms {
			bodyHits += strings.Count(lowerSec, t)
			if strings.Contains(lowerH, t) {
				headingHits++
			}
		}
		density := float64(bodyHits) / (float64(len(sec))/1000.0 + 1.0)
		score := float64(headingHits)*10.0 + density
		if score > bestScore {
			bestScore, bestText, bestHeading = score, sec, h
		}
	}
	if bestScore == 0 {
		// No lexical signal: return the first NON-container section as a
		// teaser; the doc_read pointer guides the agent to the precise one.
		for _, h := range headings {
			if sec, ok := projectdoc.SliceSection(content, h); ok && len(sec) != len(content) {
				return sec, h
			}
		}
		return cappedWhole(content), ""
	}
	return bestText, bestHeading
}

// cappedWhole bounds a whole-page fallback so a big page (e.g. a single-H1
// 59K KB page with no subsections) never crosses the wire whole as a search
// hit. Truncates on a RUNE boundary (KB pages contain CJK) and points the
// caller at doc_read for the full page.
func cappedWhole(content string) string {
	const maxRunes = 1500
	r := []rune(content)
	if len(r) <= maxRunes {
		return content
	}
	return string(r[:maxRunes]) + "\n…(truncated — use doc_read(slug) for the full page)"
}

// kbSectionTerms reduces a free-text query to deduped 4-char stems, dropping
// short/stopword tokens. The crude prefix-stem lets morphological variants
// (authenticate/authentication, scope/scopes, session/sessions) share a match
// key, and the length floor kills single-char substring noise (e.g. "i").
func kbSectionTerms(query string) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range strings.Fields(strings.ToLower(query)) {
		w = strings.Trim(w, ".,:;!?\"'`()[]{}<>")
		if len(w) < 4 {
			continue
		}
		stem := w
		if len(stem) > 4 {
			stem = stem[:4]
		}
		if !seen[stem] {
			seen[stem] = true
			out = append(out, stem)
		}
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
