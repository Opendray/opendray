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
	store     *Store
	emb       Embedder                    // optional; semantic search + backfill
	skillSink SkillSink                   // optional; render promoted skills
	reanchor  func(context.Context) error // optional; re-derive the graph after reset
	log       *slog.Logger
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

// ProjectBrain assembles the project entity + its anchored facts for a cwd.
// An absent project entity yields an empty view (not an error) so a freshly
// enabled install returns 200 with nothing rather than 404.
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
		if nb.Node.Kind == KindFact {
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

// Skillify promotes a playbook to a skill: it creates a skill node (the final
// rung of the fact->playbook->skill maturity axis), links it to the source
// playbook, and renders a SKILL.md the skills loader can pick up. Conservative
// by design — promotion is explicit (operator/agent-triggered), not automatic;
// evidence-based auto-promotion needs the runtime usage-feedback loop.
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
	skill, err := s.store.CreateNode(ctx, Node{
		Kind:       KindSkill,
		Title:      pb.Title,
		Body:       pb.Body,
		Scope:      pb.Scope,
		ScopeKey:   pb.ScopeKey,
		Maturity:   MaturitySkill,
		Provenance: map[string]any{"source": "skillify", "from_playbook": pb.ID, "skill_id": slug},
	})
	if err != nil {
		return Node{}, err
	}
	if err := s.store.CreateEdge(ctx, Edge{SrcID: skill.ID, EdgeType: EdgeDerivedFrom, DstID: pb.ID}); err != nil {
		s.log.Warn("skill derived_from edge failed", "err", err)
	}
	md := renderSkillMarkdown(slug, skillDescription(pb.Title, pb.Body), pb.Body)
	if err := s.skillSink.WriteSkill(ctx, slug, md); err != nil {
		return skill, fmt.Errorf("knowledge: write SKILL.md: %w", err)
	}
	return skill, nil
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

func (s *Service) gatherForSpawn(ctx context.Context, kind NodeKind, cwd string) []Node {
	proj, _ := s.store.ListNodes(ctx, NodeFilter{Kind: kind, Scope: ScopeProject, ScopeKey: cwd, Limit: 50})
	global, _ := s.store.ListNodes(ctx, NodeFilter{Kind: kind, Scope: ScopeGlobal, Limit: 50})
	return append(proj, global...)
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
