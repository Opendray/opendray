package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MemorySource is the read-only view of the episodic memory system that the
// anchorer needs. The app wires a memory-backed adapter, so knowledge depends
// on an interface it owns — never on internal/memory directly (one-way rule).
type MemorySource interface {
	// ListProjectKeys returns every project scope key (cwd) that has memory.
	ListProjectKeys(ctx context.Context) ([]string, error)
	// ListProjectMemories returns episodic fact rows for one project cwd,
	// newest first, capped at limit.
	ListProjectMemories(ctx context.Context, scopeKey string, limit int) ([]MemoryRow, error)
}

// MemoryRow is one episodic memory fact, as the anchorer sees it.
type MemoryRow struct {
	ID        string
	Text      string
	ScopeKey  string
	CreatedAt time.Time
}

// Anchorer lifts episodic memory facts into the knowledge graph (Phase 1):
// it ensures a canonical project entity per cwd and anchors each not-yet-seen
// memory fact as a fact node that REFERENCES the memory row (no text copy —
// the node carries a short title; the body lives in memory), linked by an
// `about` edge to its project entity so nothing is an orphan.
//
// Phase 1A is deterministic (project-level, no LLM). Phase 1B adds LLM entity
// extraction + canonicalization for finer entities (services / hosts / …).
type Anchorer struct {
	store *Store
	mem   MemorySource
	llm   LLM // optional; nil = no fine-entity extraction (deterministic 1A)
	log   *slog.Logger
}

// NewAnchorer builds an Anchorer over the shared pool and a memory source.
func NewAnchorer(pool *pgxpool.Pool, mem MemorySource, log *slog.Logger) *Anchorer {
	if log == nil {
		log = slog.Default()
	}
	return &Anchorer{store: NewStore(pool), mem: mem, log: log.With("component", "knowledge.anchor")}
}

// WithLLM enables Phase 1B fine-entity extraction. Optional — without it the
// anchorer stays deterministic (project-level anchoring only, the 1A path).
func (a *Anchorer) WithLLM(llm LLM) *Anchorer {
	a.llm = llm
	return a
}

const projectEntityIDPrefix = "ent-project-"

// ProjectEntityID is a deterministic, idempotent id for a cwd's project
// entity, so EnsureEntity is a no-op on repeat sweeps. The full cwd is kept
// in scope_key + title; the id only needs to be stable and collision-free.
func ProjectEntityID(cwd string) string {
	sum := sha256.Sum256([]byte(cwd))
	return projectEntityIDPrefix + hex.EncodeToString(sum[:8])
}

// EnsureProjectEntity idempotently creates the canonical project entity for a
// cwd and returns its node id.
func (a *Anchorer) EnsureProjectEntity(ctx context.Context, cwd string) (string, error) {
	id := ProjectEntityID(cwd)
	if _, err := a.store.EnsureEntity(ctx, Node{
		ID:         id,
		Kind:       KindEntity,
		EntityType: EntityProject,
		Title:      cwd,
		Scope:      ScopeProject,
		ScopeKey:   cwd,
		Maturity:   MaturityFact, // existence of the project is a confirmed fact
		Provenance: map[string]any{"source": "anchorer", "derived": "cwd"},
	}); err != nil {
		return "", err
	}
	return id, nil
}

// AnchorProject lifts not-yet-anchored memory facts for one cwd into the
// graph. Returns the number of facts newly anchored this call.
func (a *Anchorer) AnchorProject(ctx context.Context, cwd string, limit int) (int, error) {
	projID, err := a.EnsureProjectEntity(ctx, cwd)
	if err != nil {
		return 0, err
	}
	rows, err := a.mem.ListProjectMemories(ctx, cwd, limit)
	if err != nil {
		return 0, err
	}
	anchored, err := a.store.AnchoredMemoryIDs(ctx, cwd)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, row := range rows {
		if _, ok := anchored[row.ID]; ok {
			continue
		}
		if err := a.anchorOne(ctx, projID, row); err != nil {
			a.log.Warn("anchor memory failed", "memory_id", row.ID, "err", err)
			continue
		}
		n++
	}
	return n, nil
}

func (a *Anchorer) anchorOne(ctx context.Context, projID string, row MemoryRow) error {
	scopeKey := row.ScopeKey
	node, err := a.store.CreateNode(ctx, Node{
		Kind:       KindFact,
		Title:      factTitle(row.Text),
		Scope:      ScopeProject,
		ScopeKey:   scopeKey,
		Maturity:   MaturityFact,
		Provenance: map[string]any{"source": "anchorer", "memory_id": row.ID},
	})
	if err != nil {
		return err
	}
	if err := a.store.LinkFactSource(ctx, node.ID, row.ID); err != nil {
		return err
	}
	if err := a.store.CreateEdge(ctx, Edge{SrcID: node.ID, EdgeType: EdgeAbout, DstID: projID}); err != nil {
		return err
	}
	// Phase 1B — optional fine-entity extraction. Best-effort: a failure here
	// never fails the project anchoring already committed above.
	if a.llm != nil {
		a.linkEntities(ctx, node.ID, row)
	}
	return nil
}

// linkEntities extracts fine entities from a fact and links the fact to each
// (canonicalised) entity with an `about` edge. Best-effort; logs and moves on.
func (a *Anchorer) linkEntities(ctx context.Context, factID string, row MemoryRow) {
	ents, err := ExtractEntities(ctx, a.llm, row.Text)
	if err != nil {
		a.log.Warn("entity extraction failed", "memory_id", row.ID, "err", err)
		return
	}
	for _, e := range ents {
		entID, err := a.findOrCreateEntity(ctx, e, row.ScopeKey)
		if err != nil {
			a.log.Warn("entity upsert failed", "name", e.Name, "err", err)
			continue
		}
		if err := a.store.CreateEdge(ctx, Edge{SrcID: factID, EdgeType: EdgeAbout, DstID: entID}); err != nil {
			a.log.Warn("entity edge failed", "name", e.Name, "err", err)
		}
	}
}

// findOrCreateEntity canonicalises an extracted entity within the project
// (exact, case-insensitive) and creates it when new.
func (a *Anchorer) findOrCreateEntity(ctx context.Context, e ExtractedEntity, scopeKey string) (string, error) {
	if existing, err := a.store.FindEntityByName(ctx, e.Type, e.Name, scopeKey); err == nil {
		return existing.ID, nil
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	node, err := a.store.CreateNode(ctx, Node{
		Kind:       KindEntity,
		EntityType: e.Type,
		Title:      e.Name,
		Scope:      ScopeProject,
		ScopeKey:   scopeKey,
		Maturity:   MaturityFact,
		Provenance: map[string]any{"source": "extractor"},
	})
	if err != nil {
		return "", err
	}
	return node.ID, nil
}

// factTitle makes a short display label from a memory fact. The full text is
// NOT copied — it stays in memory, reachable via knowledge_fact_sources.
func factTitle(text string) string {
	t := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if len(t) > 120 {
		t = strings.TrimSpace(t[:120]) + "…"
	}
	if t == "" {
		t = "(empty fact)"
	}
	return t
}

// AnchorSweepConfig tunes the background sweep.
type AnchorSweepConfig struct {
	Interval     time.Duration // between sweeps (default 10m)
	InitialDelay time.Duration // before the first sweep (default 1m)
	PerProject   int           // max memories pulled per project per sweep (default 500)
}

func (c AnchorSweepConfig) withDefaults() AnchorSweepConfig {
	if c.Interval <= 0 {
		c.Interval = 10 * time.Minute
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = time.Minute
	}
	if c.PerProject <= 0 {
		c.PerProject = 500
	}
	return c
}

// RunAnchorSweep blocks until ctx is cancelled, periodically anchoring new
// memory facts across all projects. Soft-fails every step. Mirrors the
// projectdoc embed-backfill loop shape; safe to launch in its own goroutine.
func (a *Anchorer) RunAnchorSweep(ctx context.Context, cfg AnchorSweepConfig) {
	cfg = cfg.withDefaults()
	a.log.Info("knowledge anchor sweep running", "interval", cfg.Interval)
	timer := time.NewTimer(cfg.InitialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if err := a.sweepOnce(ctx, cfg.PerProject); err != nil && !errors.Is(err, context.Canceled) {
			a.log.Warn("anchor sweep cycle failed", "err", err)
		}
		timer.Reset(cfg.Interval)
	}
}

func (a *Anchorer) sweepOnce(ctx context.Context, perProject int) error {
	cwds, err := a.mem.ListProjectKeys(ctx)
	if err != nil {
		return err
	}
	total := 0
	for _, cwd := range cwds {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, err := a.AnchorProject(ctx, cwd, perProject)
		if err != nil {
			a.log.Warn("anchor project failed", "cwd", cwd, "err", err)
			continue
		}
		total += n
	}
	if total > 0 {
		a.log.Info("anchor sweep done", "anchored", total, "projects", len(cwds))
	}
	return nil
}
