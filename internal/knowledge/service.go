package knowledge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service is the thin application layer over Store. Phase 0 only delegates;
// later phases hang the consolidation / graduation engine off this type
// (and give it a read-only handle to internal/memory as its feedstock).
type Service struct {
	store       *Store
	emb         Embedder                    // optional; semantic search + backfill
	skillSink   SkillSink                   // optional; render promoted skills
	taskSink    TaskSink                    // optional; register compiled skills as custom tasks
	skillifyLLM LLM                         // optional; LLM-authored SKILL.md on promotion
	reanchor    func(context.Context) error // optional; re-derive the graph after reset
	kbDrafter   *KBDrafter                  // optional; M-KB curated page drafting
	overview    *OverviewDrafter            // optional; per-project Overview drafting
	log         *slog.Logger
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

// Neighborhood returns a node and its 1-hop neighbors.
func (s *Service) Neighborhood(ctx context.Context, id string) (Node, []Neighbor, error) {
	return s.store.Neighborhood(ctx, id)
}

// BrainView is a synthesised snapshot of one project's knowledge, assembled
// from the graph (Phase 2). It grows richer as later phases add playbooks /
// skills / related entities.
type BrainView struct {
	Project *Node  `json:"project"` // nil when the project has no entity yet
	Facts   []Node `json:"facts"`
}

// ProjectBrain assembles the project entity + the entities it touches for a
// cwd. An absent project entity yields an empty view (not an error) so a
// freshly enabled install returns 200 with nothing rather than 404.
//
// P-G: fact nodes are retired (Memory is the fact store), so the view's Facts
// field now carries the project's linked ENTITIES instead of a fact mirror —
// the demoted graph tab shows "what this project touches"; the declarative
// facts themselves live in Memory and the curated KB pages.
func (s *Service) ProjectBrain(ctx context.Context, cwd string) (BrainView, error) {
	center, neighbors, err := s.store.Neighborhood(ctx, ProjectEntityID(cwd))
	if errors.Is(err, ErrNotFound) {
		return BrainView{}, nil
	}
	if err != nil {
		return BrainView{}, err
	}
	view := BrainView{Project: &center}
	for _, nb := range neighbors {
		if nb.Node.Kind == KindEntity && nb.Node.ID != center.ID {
			view.Facts = append(view.Facts, nb.Node)
		}
	}
	return view, nil
}

// WithEmbedder enables semantic search + the embed backfill (Phase 6).
func (s *Service) WithEmbedder(emb Embedder) *Service {
	s.emb = emb
	return s
}

// SearchNodes embeds the query and returns the top-K most similar nodes.
func (s *Service) SearchNodes(ctx context.Context, query, scopeKey string, topK int) ([]SearchHit, error) {
	if s.emb == nil {
		return nil, errors.New("knowledge: semantic search requires an embedder")
	}
	vecs, err := s.emb.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, errors.New("knowledge: embedder returned no vector")
	}
	return s.store.SearchNodes(ctx, s.emb.Name(), vecs[0], scopeKey, topK)
}

// PromoteNode lifts a node to a wider scope (project -> domain/global) so its
// knowledge transfers to other projects' searches.
func (s *Service) PromoteNode(ctx context.Context, id string, scope Scope, scopeKey string) error {
	return s.store.PromoteNode(ctx, id, scope, scopeKey)
}

// WithSkillSink enables Phase 4 skill rendering (playbook -> skill -> SKILL.md).
func (s *Service) WithSkillSink(sink SkillSink) *Service {
	s.skillSink = sink
	return s
}

// WithTaskSink enables the compiled-skill path: when a promoted candidate
// carries an executable script, promotion also registers an opendray custom
// task pointing at the rendered run.sh.
func (s *Service) WithTaskSink(sink TaskSink) *Service {
	s.taskSink = sink
	return s
}

// WithSkillifyLLM enables LLM-authored skills: instead of copying the
// playbook body verbatim into SKILL.md, promotion drafts a complete,
// structured skill document (overview / when-to-use / procedure with
// real commands / pitfalls / verification). Without it, promotion
// falls back to the mechanical render.
func (s *Service) WithSkillifyLLM(llm LLM) *Service {
	s.skillifyLLM = llm
	return s
}

const skillifySystem = `You turn a distilled PLAYBOOK into a production-grade agent SKILL file (Claude Code SKILL.md format).

Output the COMPLETE file and nothing else:
1. YAML frontmatter: name (the given slug, verbatim), description (ONE sentence, <200 chars: what this skill does AND when to reach for it — this line is all an agent sees before deciding to load the skill, make it count).
2. Body sections:
   ## Overview — 2-3 sentences: the problem this solves and the outcome.
   ## When to use — concrete triggers; also when NOT to use it.
   ## Procedure — numbered, executable steps reusing the playbook's REAL commands, paths, hostnames, file names. An agent must be able to follow this without guessing.
   ## Pitfalls — the actual failure modes and how to avoid them.
   ## Verification — how to confirm it worked.

Rules: never invent commands or values absent from the playbook; keep every concrete detail it has; NEVER include secrets (passwords, tokens, keys) — name where a credential lives, never its value; no markdown fences around the file.`

// generateSkillMarkdown asks the skillify LLM for a full SKILL.md.
// Empty string on any failure — callers fall back to the mechanical
// render rather than blocking promotion.
func (s *Service) generateSkillMarkdown(ctx context.Context, slug string, pb Node) string {
	if s.skillifyLLM == nil {
		return ""
	}
	input := "Slug: " + slug + "\n\nPlaybook title: " + pb.Title + "\n\nPlaybook body:\n" + pb.Body
	if provenanceString(pb, "script") != "" {
		input += "\n\nNote: this skill ships an executable `run.sh` (same directory as SKILL.md) that performs the whole procedure including its validation step. The Procedure section should lead with running it and keep the manual steps as the fallback."
	}
	out, err := s.skillifyLLM.Complete(ctx, skillifySystem, input)
	if err != nil {
		s.log.Warn("skillify llm failed — using mechanical render", "err", err)
		return ""
	}
	out = strings.TrimSpace(out)
	if !strings.HasPrefix(out, "---") {
		s.log.Warn("skillify llm returned non-frontmatter output — using mechanical render")
		return ""
	}
	return out
}

// Skillify promotes a playbook to a skill: it creates a skill node (the final
// rung of the maturity axis), links it to the source candidate, and renders a
// SKILL.md the skills loader can pick up. When the candidate was compiled by
// the experience compiler with an executable form, promotion also materialises
// run.sh next to SKILL.md and registers it as an opendray custom task — a
// skill is a tested artifact, not prose, wherever possible. Promotion stays
// explicit (operator/agent-triggered), not automatic.
func (s *Service) Skillify(ctx context.Context, playbookID string) (Node, error) {
	if s.skillSink == nil {
		return Node{}, errors.New("knowledge: skill rendering is not configured")
	}
	pb, err := s.store.GetNode(ctx, playbookID)
	if err != nil {
		return Node{}, err
	}
	if pb.Kind != KindPlaybook {
		return Node{}, fmt.Errorf("knowledge: can only skillify a playbook, got %q", pb.Kind)
	}
	slug := skillSlug(pb.Title)
	prov := map[string]any{"source": "skillify", "from_playbook": pb.ID, "skill_id": slug}
	// Carry the compiled form (and its provenance trail) onto the skill node
	// so disable/enable can re-materialise run.sh without the playbook.
	for _, k := range []string{"script", "validation", "sessions", "projects", "recurrence", "est_minutes", "score", "cluster_sig"} {
		if v, ok := pb.Provenance[k]; ok {
			prov[k] = v
		}
	}
	skill, err := s.store.CreateNode(ctx, Node{
		Kind:       KindSkill,
		Title:      pb.Title,
		Body:       pb.Body,
		Scope:      pb.Scope,
		ScopeKey:   pb.ScopeKey,
		Maturity:   MaturitySkill,
		Provenance: prov,
	})
	if err != nil {
		return Node{}, err
	}
	if err := s.store.CreateEdge(ctx, Edge{SrcID: skill.ID, EdgeType: EdgeDerivedFrom, DstID: pb.ID}); err != nil {
		s.log.Warn("skill derived_from edge failed", "err", err)
	}
	md := s.generateSkillMarkdown(ctx, slug, pb)
	if md == "" {
		md = renderSkillMarkdown(slug, skillDescription(pb.Title, pb.Body), pb.Body)
	} else if _, uerr := s.store.UpdateNodeBody(ctx, skill.ID, md); uerr != nil {
		s.log.Warn("skillify: store generated body failed", "err", uerr)
	}
	if err := s.skillSink.WriteSkill(ctx, slug, md); err != nil {
		return skill, fmt.Errorf("knowledge: write SKILL.md: %w", err)
	}
	s.materialiseCompiledForm(ctx, slug, skill)
	return skill, nil
}

// materialiseCompiledForm writes the executable run.sh next to SKILL.md and
// registers the custom task, when the node carries a compiled script. Both
// are best-effort: the prose skill stands on its own.
func (s *Service) materialiseCompiledForm(ctx context.Context, slug string, n Node) {
	script := provenanceString(n, "script")
	if script == "" || s.skillSink == nil {
		return
	}
	if err := s.skillSink.WriteSkillAsset(ctx, slug, "run.sh", script); err != nil {
		s.log.Warn("skillify: write run.sh failed", "slug", slug, "err", err)
		return
	}
	if s.taskSink == nil {
		return
	}
	cwd := ""
	if n.Scope == ScopeProject {
		cwd = n.ScopeKey
	}
	desc := "Compiled skill (validated procedure): " + n.Title
	if err := s.taskSink.EnsureSkillTask(ctx, slug, n.Title, desc, cwd); err != nil {
		s.log.Warn("skillify: custom task registration failed", "slug", slug, "err", err)
	}
}

// provenanceString reads a string field off a node's provenance.
func provenanceString(n Node, key string) string {
	v, _ := n.Provenance[key].(string)
	return strings.TrimSpace(v)
}

// SetSkillEnabled flips a skill on/off: disabled removes its SKILL.md
// from the vault (no session loads it; the node and its history stay),
// enabled re-renders the file. With hundreds of distilled skills a
// project needs 1-2 — this is the per-skill switch.
func (s *Service) SetSkillEnabled(ctx context.Context, id string, enabled bool) (Node, error) {
	n, err := s.store.SetNodeEnabled(ctx, id, enabled)
	if err != nil {
		return Node{}, err
	}
	if n.Kind != KindSkill || s.skillSink == nil {
		return n, nil
	}
	slug, _ := n.Provenance["skill_id"].(string)
	if slug == "" {
		slug = skillSlug(n.Title)
	}
	if enabled {
		md := n.Body
		if !strings.HasPrefix(strings.TrimSpace(md), "---") {
			md = renderSkillMarkdown(slug, skillDescription(n.Title, n.Body), n.Body)
		}
		if werr := s.skillSink.WriteSkill(ctx, slug, md); werr != nil {
			return n, fmt.Errorf("knowledge: re-render skill: %w", werr)
		}
		// Compiled skills get their executable form back too.
		s.materialiseCompiledForm(ctx, slug, n)
	} else if derr := s.skillSink.DeleteSkill(ctx, slug); derr != nil {
		return n, fmt.Errorf("knowledge: remove skill file: %w", derr)
	}
	return n, nil
}

// RetirementCandidate is one skill the closed feedback loop proposes to
// retire, with the machine-readable reason.
type RetirementCandidate struct {
	Node Node `json:"node"`
	// Reason: never_used | low_success | dormant
	Reason string `json:"reason"`
}

// Retirement thresholds. A skill is proposed for retirement when it is
// enabled and (a) was never referenced 14+ days after creation, (b) keeps
// getting loaded into sessions that then FAIL, or (c) used to be referenced
// but has gone quiet. The loop only proposes — the operator retires.
const (
	retireNeverUsedAfter = 14 * 24 * time.Hour
	retireDormantAfter   = 45 * 24 * time.Hour
	retireMinOutcomes    = 3
	retireMaxSuccessRate = 0.4
)

// RetirementCandidates closes the feedback loop: skills carry use counters
// and per-session outcome counters (did the session that referenced the
// skill end in success?); this surfaces the ones the evidence says to drop.
func (s *Service) RetirementCandidates(ctx context.Context) ([]RetirementCandidate, error) {
	skills, err := s.store.ListNodes(ctx, NodeFilter{Kind: KindSkill, Limit: 500})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := []RetirementCandidate{}
	for _, n := range skills {
		if !n.Enabled {
			continue
		}
		switch {
		case n.UseCount == 0 && now.Sub(n.CreatedAt) > retireNeverUsedAfter:
			out = append(out, RetirementCandidate{Node: n, Reason: "never_used"})
		case n.SuccessCount+n.FailureCount >= retireMinOutcomes &&
			float64(n.SuccessCount)/float64(n.SuccessCount+n.FailureCount) < retireMaxSuccessRate:
			// Loaded but abandoned: sessions reference it and still fail.
			out = append(out, RetirementCandidate{Node: n, Reason: "low_success"})
		case n.UseCount > 0 && n.LastUsedAt != nil && now.Sub(*n.LastUsedAt) > retireDormantAfter:
			out = append(out, RetirementCandidate{Node: n, Reason: "dormant"})
		}
	}
	return out, nil
}

// RenderForSpawn builds a compact "Project knowledge" banner (the project's +
// global skills and playbooks) to prepend to a spawning agent's system prompt.
// Budget-capped; empty when there's nothing yet. This is the payoff — the brain
// feeds accumulated cross-session/cross-project expertise into every session.
func (s *Service) RenderForSpawn(ctx context.Context, cwd string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	skills := s.gatherForSpawn(ctx, KindSkill, cwd)
	playbooks := s.gatherForSpawn(ctx, KindPlaybook, cwd)
	if len(skills) == 0 && len(playbooks) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("## Project knowledge (opendray)\n")
	writeSection := func(header string, nodes []Node) {
		if len(nodes) == 0 || b.Len()+len(header) >= maxBytes {
			return
		}
		b.WriteString(header)
		for _, n := range nodes {
			line := "- " + strings.TrimSpace(n.Title) + "\n"
			if b.Len()+len(line) > maxBytes {
				break
			}
			b.WriteString(line)
		}
	}
	writeSection("\n### Skills\n", skills)
	writeSection("\n### Playbooks\n", playbooks)
	return b.String(), nil
}

// ImpactEntities returns entities ordered by blast radius.
func (s *Service) ImpactEntities(ctx context.Context, limit int) ([]ImpactEntity, error) {
	return s.store.ListImpactEntities(ctx, limit)
}

// WithReanchor wires a re-derive trigger used after Reset so the graph rebuilds
// immediately rather than waiting for the next scheduled sweep.
func (s *Service) WithReanchor(fn func(context.Context) error) *Service {
	s.reanchor = fn
	return s
}

// Reset wipes the knowledge graph and kicks off a background re-derive from
// episodic memory (with the current logic). The re-derive runs detached so the
// HTTP response returns immediately.
func (s *Service) Reset(ctx context.Context) error {
	if err := s.store.Reset(ctx); err != nil {
		return err
	}
	if s.reanchor != nil {
		go func() {
			if err := s.reanchor(context.Background()); err != nil {
				s.log.Warn("re-anchor after reset failed", "err", err)
			}
		}()
	}
	return nil
}

// WithKBDrafter wires the KB-page drafter so the manual draft endpoint can
// regenerate the curated knowledge-base pages on demand.
func (s *Service) WithKBDrafter(d *KBDrafter) *Service {
	s.kbDrafter = d
	return s
}

// WithOverviewDrafter wires the per-project Overview drafter so the manual
// draft endpoint refreshes the official project documents too.
func (s *Service) WithOverviewDrafter(d *OverviewDrafter) *Service {
	s.overview = d
	return s
}

// DraftKB regenerates all curated KB pages + per-project Overviews now,
// returning per-page results.
func (s *Service) DraftKB(ctx context.Context) ([]KBDraftResult, error) {
	if s.kbDrafter == nil && s.overview == nil {
		return nil, errors.New("knowledge: KB drafter not configured")
	}
	var out []KBDraftResult
	if s.kbDrafter != nil {
		res, err := s.kbDrafter.DraftAll(ctx)
		out = append(out, res...)
		if err != nil {
			return out, err
		}
	}
	if s.overview != nil {
		res, err := s.overview.DraftAll(ctx)
		out = append(out, res...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// DeleteNode removes a node — used to undo a mis-click (an accidental skillify
// or promote). For skills it also deletes the rendered SKILL.md. Note:
// auto-derived facts/entities re-appear on the next anchor sweep; skills
// (explicitly created) stay deleted.
func (s *Service) DeleteNode(ctx context.Context, id string) error {
	if n, err := s.store.GetNode(ctx, id); err == nil && n.Kind == KindSkill && s.skillSink != nil {
		if slug, ok := n.Provenance["skill_id"].(string); ok && slug != "" {
			_ = s.skillSink.DeleteSkill(ctx, slug)
		}
	}
	return s.store.DeleteNode(ctx, id)
}

func (s *Service) gatherForSpawn(ctx context.Context, kind NodeKind, cwd string) []Node {
	proj, _ := s.store.ListNodes(ctx, NodeFilter{Kind: kind, Scope: ScopeProject, ScopeKey: cwd, Limit: 50})
	global, _ := s.store.ListNodes(ctx, NodeFilter{Kind: kind, Scope: ScopeGlobal, Limit: 50})
	all := append(proj, global...)
	out := all[:0]
	for _, n := range all {
		if n.Enabled {
			out = append(out, n)
		}
	}
	return out
}

// EmbedBackfillConfig tunes the background node-embedding loop.
type EmbedBackfillConfig struct {
	Interval     time.Duration
	InitialDelay time.Duration
	Batch        int
}

func (c EmbedBackfillConfig) withDefaults() EmbedBackfillConfig {
	if c.Interval <= 0 {
		c.Interval = 5 * time.Minute
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = 90 * time.Second
	}
	if c.Batch <= 0 {
		c.Batch = 64
	}
	return c
}

// RunEmbedBackfill blocks until ctx is cancelled, embedding nodes that lack a
// vector for the active embedder. No-op without an embedder. Mirrors the
// projectdoc embed-backfill loop.
func (s *Service) RunEmbedBackfill(ctx context.Context, cfg EmbedBackfillConfig) {
	if s.emb == nil {
		return
	}
	cfg = cfg.withDefaults()
	s.log.Info("knowledge embed backfill running", "embedder", s.emb.Name())
	timer := time.NewTimer(cfg.InitialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if err := s.embedOnce(ctx, cfg.Batch); err != nil && !errors.Is(err, context.Canceled) {
			s.log.Warn("embed backfill cycle failed", "err", err)
		}
		timer.Reset(cfg.Interval)
	}
}

func (s *Service) embedOnce(ctx context.Context, batch int) error {
	nodes, err := s.store.ListNodesNeedingEmbedding(ctx, s.emb.Name(), batch)
	if err != nil || len(nodes) == 0 {
		return err
	}
	texts := make([]string, len(nodes))
	for i, n := range nodes {
		texts[i] = embedText(n)
	}
	vecs, err := s.emb.Embed(ctx, texts)
	if err != nil {
		return err
	}
	if len(vecs) != len(nodes) {
		return nil
	}
	for i, n := range nodes {
		if err := s.store.SetEmbedding(ctx, n.ID, s.emb.Name(), vecs[i]); err != nil {
			s.log.Warn("set embedding failed", "id", n.ID, "err", err)
		}
	}
	return nil
}

func embedText(n Node) string {
	if n.Body == "" {
		return n.Title
	}
	return n.Title + "\n" + n.Body
}
