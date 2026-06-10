package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"regexp"
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
	// ListAllMemories returns episodic fact rows across all project scope keys,
	// newest first, capped at limit. P-G: the cross-project KB pages distil
	// straight from Memory now that the fact-node layer is retired.
	ListAllMemories(ctx context.Context, limit int) ([]MemoryRow, error)
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
	store     *Store
	mem       MemorySource
	llm       LLM             // optional; nil = no fine-entity extraction (deterministic 1A)
	lifecycle LifecycleFilter // optional; skip frozen (paused/archived) projects
	log       *slog.Logger
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

// WithLifecycle installs the lifecycle filter so frozen (paused/archived)
// projects stop feeding the graph. Cortex Phase 2 closed this gap: the
// Reflector honoured P-D from the start, but the Anchor stage ran first
// in every consolidation cycle and kept lifting abandoned projects'
// facts regardless.
func (a *Anchorer) WithLifecycle(f LifecycleFilter) *Anchorer {
	a.lifecycle = f
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
	if isEphemeralCwd(cwd) {
		return 0, nil // skip throwaway /tmp-style cwds
	}
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

// anchorOne processes one not-yet-seen memory row. P-G: fact nodes are
// retired — Memory IS the fact store — so this no longer copies the fact into
// the graph. It extracts cross-project entities from the memory text and links
// each to the project entity, then records the memory id against the project
// entity (via knowledge_fact_sources) purely as a "processed" marker so the
// next sweep skips it. The fact body is never duplicated.
func (a *Anchorer) anchorOne(ctx context.Context, projID string, row MemoryRow) error {
	if looksLikeSecret(row.Text) {
		// Still mark processed so we don't re-scan a secret every sweep, but
		// never extract entities from it into the searchable graph.
		return a.store.LinkFactSource(ctx, projID, row.ID)
	}
	// Phase 1B — fine-entity extraction straight from the memory text. Best-
	// effort: a failure logs but we still mark the memory processed so the
	// sweep makes forward progress instead of retrying forever.
	if a.llm != nil {
		a.linkEntities(ctx, projID, row)
	}
	return a.store.LinkFactSource(ctx, projID, row.ID)
}

// linkEntities extracts entities from a memory fact and links each
// (canonicalised) entity to the project entity with an `about` edge, so the
// project's neighbourhood surfaces "what this project touches". Best-effort.
func (a *Anchorer) linkEntities(ctx context.Context, projID string, row MemoryRow) {
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
		if entID == projID {
			continue // don't self-link the project entity
		}
		if err := a.store.CreateEdge(ctx, Edge{SrcID: entID, EdgeType: EdgeAbout, DstID: projID}); err != nil {
			a.log.Warn("entity edge failed", "name", e.Name, "err", err)
		}
	}
}

// findOrCreateEntity canonicalises an extracted entity within the project
// (exact, case-insensitive) and creates it when new.
func (a *Anchorer) findOrCreateEntity(ctx context.Context, e ExtractedEntity, scopeKey string) (string, error) {
	scope, key := entityScopeFor(e.Type, scopeKey)
	if existing, err := a.store.FindEntityByName(ctx, e.Type, e.Name, key); err == nil {
		return existing.ID, nil
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	node, err := a.store.CreateNode(ctx, Node{
		Kind:       KindEntity,
		EntityType: e.Type,
		Title:      e.Name,
		Scope:      scope,
		ScopeKey:   key,
		Maturity:   MaturityFact,
		Provenance: map[string]any{"source": "extractor"},
	})
	if err != nil {
		return "", err
	}
	return node.ID, nil
}

// entityScopeFor decides where an extracted entity lives. tech + tool are
// inherently cross-project (npm, Go, PostgreSQL) → global singletons so they
// are not duplicated per project; everything else stays project-scoped.
func entityScopeFor(t EntityType, cwd string) (Scope, string) {
	switch t {
	case EntityTech, EntityTool:
		return ScopeGlobal, ""
	default:
		return ScopeProject, cwd
	}
}

// factTitle makes a short display label from a memory fact. The full text is
// NOT copied — it stays in memory, reachable via knowledge_fact_sources.
// Imported .md memories carry a leading YAML frontmatter block (--- … ---);
// we strip it so the title is the actual content, not "--- name: …".
func factTitle(text string) string {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "---") {
		if i := strings.Index(t[3:], "\n---"); i >= 0 {
			if body := strings.TrimSpace(t[3+i+4:]); body != "" {
				t = body
			}
		}
	}
	t = strings.TrimSpace(strings.ReplaceAll(t, "\n", " "))
	if len(t) > 120 {
		t = strings.TrimSpace(t[:120]) + "…"
	}
	if t == "" {
		t = "(empty fact)"
	}
	return t
}

// isEphemeralCwd reports whether a cwd is a throwaway/temp dir (sessions run
// from /tmp, /var/folders, etc.) — we don't want a project entity + facts for
// those polluting the graph.
func isEphemeralCwd(cwd string) bool {
	if cwd == "" {
		return true
	}
	c := strings.ToLower(cwd)
	return c == "/tmp" ||
		strings.HasPrefix(c, "/tmp/") ||
		strings.HasPrefix(c, "/private/tmp") ||
		strings.HasPrefix(c, "/var/folders/") ||
		strings.HasPrefix(c, "/private/var/folders/") ||
		strings.Contains(c, "/tmp.") ||
		strings.Contains(c, "/.cache/")
}

var (
	secretHintRe = regexp.MustCompile(`(?i)(api[\s_-]?key|secret|token|password|passwd|bearer|private[\s_-]?key|access[\s_-]?key|credential)`)
	longTokenRe  = regexp.MustCompile(`[A-Za-z0-9_\-/+]{20,}`)
	// "password: hunter2longish" / "app password Niv3!k8649" — a credential
	// keyword followed (within a few chars) by an 8+ char opaque value. Catches
	// the shorter passwords the long-token rule misses.
	secretAssignRe = regexp.MustCompile(`(?i)(password|passwd|secret|api[\s_-]?key|access[\s_-]?key|token|app[\s_-]?password|credential)\b[\s:=]*\S{8,}`)
)

// looksLikeSecret flags facts that carry a credential so we never lift them
// into the searchable/exportable knowledge graph (or the KB pages). Best-effort
// defence-in-depth — the secret should also be scrubbed from memory + rotated.
func looksLikeSecret(text string) bool {
	if secretAssignRe.MatchString(text) {
		return true
	}
	return secretHintRe.MatchString(text) && longTokenRe.MatchString(text)
}

// AnchorAll runs one full anchoring pass across all projects. Exported so the
// reset path can re-derive the graph immediately instead of waiting for the
// next scheduled sweep.
func (a *Anchorer) AnchorAll(ctx context.Context, perProject int) error {
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
		if a.lifecycle != nil && a.lifecycle.IsFrozen(ctx, cwd) {
			continue // paused/archived projects stop feeding the graph
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
	// Collapse cross-project duplicate tech/tool entities into global
	// singletons (idempotent — a clean graph is a no-op).
	if n, err := a.store.MergeDuplicateGlobalEntities(ctx); err != nil {
		a.log.Warn("merge duplicate entities failed", "err", err)
	} else if n > 0 {
		a.log.Info("merged cross-project duplicate entities", "merged", n)
	}
	return nil
}
