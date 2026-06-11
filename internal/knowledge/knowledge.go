// Package knowledge implements opendray's structured, self-evolving
// knowledge graph (arc M-KG; see docs/knowledge-graph-redesign.md).
//
// It is a DB-native typed graph that grows ON TOP of the M-U episodic
// memory. Dependency direction is strictly one-way: this package may read
// internal/memory, but internal/memory never imports this package — so the
// just-shipped memory system stays a stable island and disabling the
// [knowledge] feature flag returns exact memory-only behaviour.
//
// Phase 0 (this file set) is the substrate only: typed nodes + edges over a
// closed ontology, with CRUD. No reflection / consolidation / graduation
// yet — those land in later phases.
package knowledge

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NodeKind is the closed set of structured-knowledge node types. Together
// they cover declarative (entity + fact), procedural (playbook) and skill
// knowledge.
type NodeKind string

const (
	KindEntity NodeKind = "entity" // declarative backbone — a thing
	// KindFact is RETIRED (P-G): fact nodes were a 1:1 mirror of episodic
	// Memory and are no longer produced (migration 0041 deleted existing ones).
	// The constant is kept so Valid() still accepts legacy rows mid-migration
	// and the node-validation surface stays stable. Memory is the fact store.
	KindFact     NodeKind = "fact"
	KindPlaybook NodeKind = "playbook" // procedural know-how
	KindSkill    NodeKind = "skill"    // an invocable capability
)

// Valid reports whether k is a known node kind.
func (k NodeKind) Valid() bool {
	switch k {
	case KindEntity, KindFact, KindPlaybook, KindSkill:
		return true
	}
	return false
}

// EntityType sub-classifies KindEntity nodes. Closed vocabulary.
type EntityType string

const (
	EntityService  EntityType = "service"
	EntityHost     EntityType = "host"
	EntityProject  EntityType = "project"
	EntityTool     EntityType = "tool"
	EntityDecision EntityType = "decision"
	EntityTech     EntityType = "tech"
	EntityPerson   EntityType = "person"
)

// Valid reports whether e is a known entity type.
func (e EntityType) Valid() bool {
	switch e {
	case EntityService, EntityHost, EntityProject, EntityTool,
		EntityDecision, EntityTech, EntityPerson:
		return true
	}
	return false
}

// Scope is a node's transfer radius. It subsumes M-U's all-or-nothing
// project isolation: project-scoped nodes stay isolated to one cwd, while
// domain/global nodes transfer across projects.
type Scope string

const (
	ScopeProject Scope = "project" // cwd-local (isolated, like today's memory)
	ScopeDomain  Scope = "domain"  // shared across a class of projects
	ScopeGlobal  Scope = "global"  // operator-wide
)

// Valid reports whether s is a known scope.
func (s Scope) Valid() bool {
	switch s {
	case ScopeProject, ScopeDomain, ScopeGlobal:
		return true
	}
	return false
}

// Maturity is the self-evolution axis: knowledge graduates upward as
// evidence accumulates (candidate -> fact -> playbook -> skill).
type Maturity string

const (
	MaturityCandidate Maturity = "candidate"
	MaturityFact      Maturity = "fact"
	MaturityPlaybook  Maturity = "playbook"
	MaturitySkill     Maturity = "skill"
)

// Valid reports whether m is a known maturity level.
func (m Maturity) Valid() bool {
	switch m {
	case MaturityCandidate, MaturityFact, MaturityPlaybook, MaturitySkill:
		return true
	}
	return false
}

// EdgeType is the closed relation vocabulary. Free-form relation types are
// forbidden by design — an untyped free-association graph is exactly what
// rots at scale.
type EdgeType string

const (
	EdgeRunsOn      EdgeType = "runs_on"
	EdgeUses        EdgeType = "uses"
	EdgeAbout       EdgeType = "about"
	EdgePartOf      EdgeType = "part_of"
	EdgeDependsOn   EdgeType = "depends_on"
	EdgeSupersedes  EdgeType = "supersedes"
	EdgeDerivedFrom EdgeType = "derived_from"
	EdgeUsedBy      EdgeType = "used_by"
)

// Valid reports whether e is a known edge type.
func (e EdgeType) Valid() bool {
	switch e {
	case EdgeRunsOn, EdgeUses, EdgeAbout, EdgePartOf, EdgeDependsOn,
		EdgeSupersedes, EdgeDerivedFrom, EdgeUsedBy:
		return true
	}
	return false
}

// Node is one vertex in the knowledge graph.
type Node struct {
	ID         string         `json:"id"`
	Kind       NodeKind       `json:"kind"`
	EntityType EntityType     `json:"entity_type,omitempty"`
	Title      string         `json:"title"`
	Body       string         `json:"body"`
	Scope      Scope          `json:"scope"`
	ScopeKey   string         `json:"scope_key,omitempty"`
	Maturity   Maturity       `json:"maturity"`
	Confidence *float64       `json:"confidence,omitempty"`
	Provenance map[string]any `json:"provenance,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	ArchivedAt *time.Time     `json:"archived_at,omitempty"`

	// Usage tracking (skills): how often a session transcript actually
	// referenced this skill, and when last. Surfaces never-used skills
	// as retirement candidates so the injected set stays lean.
	UseCount   int        `json:"use_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Outcome tracking (skills): of the sessions that referenced this
	// skill, how many ended in success (exit 0 / clean stop) vs failure.
	// Skills that get loaded but keep failing are retirement candidates.
	SuccessCount int `json:"success_count"`
	FailureCount int `json:"failure_count"`

	// Enabled (skills): a disabled skill keeps its node but its
	// SKILL.md is removed from the vault so no session loads it.
	Enabled bool `json:"enabled"`
}

// Validate enforces the closed ontology before a write. It mirrors the DB
// CHECK constraints so callers get a clear error before hitting Postgres.
func (n Node) Validate() error {
	if !n.Kind.Valid() {
		return fmt.Errorf("knowledge: invalid node kind %q", n.Kind)
	}
	if n.Kind == KindEntity {
		if !n.EntityType.Valid() {
			return fmt.Errorf("knowledge: entity node needs a valid entity_type, got %q", n.EntityType)
		}
	} else if n.EntityType != "" {
		return fmt.Errorf("knowledge: entity_type is only valid on entity nodes")
	}
	if n.Title == "" {
		return fmt.Errorf("knowledge: node title is required")
	}
	if !n.Scope.Valid() {
		return fmt.Errorf("knowledge: invalid scope %q", n.Scope)
	}
	if n.Maturity != "" && !n.Maturity.Valid() {
		return fmt.Errorf("knowledge: invalid maturity %q", n.Maturity)
	}
	return nil
}

// Edge is one typed, directed relation between two nodes.
type Edge struct {
	SrcID     string    `json:"src_id"`
	EdgeType  EdgeType  `json:"edge_type"`
	DstID     string    `json:"dst_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Validate enforces the closed edge vocabulary and rejects self-edges.
func (e Edge) Validate() error {
	if e.SrcID == "" || e.DstID == "" {
		return fmt.Errorf("knowledge: edge needs both src_id and dst_id")
	}
	if e.SrcID == e.DstID {
		return fmt.Errorf("knowledge: self-edge is not allowed")
	}
	if !e.EdgeType.Valid() {
		return fmt.Errorf("knowledge: invalid edge_type %q", e.EdgeType)
	}
	return nil
}

// NewID mints a node id when the caller does not supply a semantic one
// (e.g. "svc-postgres-dev"). prefix is usually the node kind.
func NewID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	if prefix == "" {
		prefix = "kn"
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
