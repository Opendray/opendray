package projectdoc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ─── doc blueprints (Cortex Phase 3) ───────────────────────────
//
// A blueprint is the per-project declaration of which doc sections
// exist, their order, and who maintains each one. project_docs.kind
// is the section slug; the blueprint is the schema the UI renders,
// the drift detector iterates, and the spawn banner follows. Each
// project type (mobile app / service / CLI / …) can carry a different
// section set instead of being trapped in one fixed tab layout.

// Section is one row of doc_blueprint_sections.
type Section struct {
	Cwd         string `json:"cwd"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Position    int    `json:"position"`
	// MaintainerMode: "ai" (drift auto-drafts; proposals when locked),
	// "human" (operator-authored; AI only proposes), "scanner"
	// (mechanically rebuilt).
	MaintainerMode string `json:"maintainer_mode"`
	// PromptHint steers the AI maintainer ("track the public API
	// surface", "keep this a one-page pitch").
	PromptHint string `json:"prompt_hint,omitempty"`
	// WritePolicy governs the AGENT-SIDE explicit MCP write
	// (project_*_set / current_objective_set):
	//   "proposal" — the write lands as a proposal the operator approves
	//                (goal/plan — long-term, deliberate changes).
	//   "direct"   — the in-session agent writes the live doc directly
	//                when it is unlocked (current_objective — short-term,
	//                high-context, churns every session). A human edit
	//                still re-gates the next write back to a proposal.
	// Empty defaults to "proposal". Does NOT affect the post-session
	// drift path, which keeps its own ai+unlocked→apply rule.
	WritePolicy string `json:"write_policy,omitempty"`
	// Pinned sections sort first and cannot be deleted (overview; the
	// four classic knowledge pages).
	Pinned bool `json:"pinned"`
	// Inject includes this section's doc in the spawn banner. Pages
	// with inject=false are reached on demand via cross-layer search
	// instead — agents index only what a task needs.
	Inject bool `json:"inject"`
	// Nature classifies GLOBAL knowledge pages: "foundational"
	// (binding ground truth + rules, injected as guardrails) or
	// "emergent" (distilled guidance, injected as reference). Always
	// empty for per-project sections.
	Nature    string    `json:"nature,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SlugOverview is the reserved front-page section: present in every
// blueprint, pinned, undeletable.
const SlugOverview = "overview"

var (
	slugRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,47}$`)
	// kbSlugRe requires a real name after the kb_ prefix.
	kbSlugRe = regexp.MustCompile(`^kb_[a-z0-9][a-z0-9_]{0,44}$`)
)

// ValidSectionSlug reports whether s is a legal PER-PROJECT blueprint
// slug. kb_* is reserved for the global Knowledge pages and never
// valid inside a per-project blueprint.
func ValidSectionSlug(s string) bool {
	return slugRe.MatchString(s) && !strings.HasPrefix(s, "kb_")
}

// ValidGlobalKBSlug reports whether s is a legal GLOBAL knowledge page
// slug: kb_-prefixed, same syntax. The knowledge base is extensible —
// any new knowledge domain gets its own kb_* page instead of being
// crammed into the four classics.
func ValidGlobalKBSlug(s string) bool {
	return kbSlugRe.MatchString(s)
}

// ValidMaintainerMode reports whether m is ai|human|scanner.
func ValidMaintainerMode(m string) bool {
	return m == "ai" || m == "human" || m == "scanner"
}

// ValidWritePolicy reports whether p is direct|proposal.
func ValidWritePolicy(p string) bool {
	return p == "direct" || p == "proposal"
}

// normalizeWritePolicy defaults an empty/unknown policy to the safe
// "proposal" gate so a section never silently grants direct write.
func normalizeWritePolicy(p string) string {
	if p == "direct" {
		return "direct"
	}
	return "proposal"
}

// ValidNature reports whether n is a legal knowledge nature.
func ValidNature(n string) bool {
	return n == "foundational" || n == "emergent"
}

// ErrReservedSection is returned when a caller tries to delete or
// demote the reserved overview section.
var ErrReservedSection = errors.New("projectdoc: the overview section is reserved")

// defaultSections is the blueprint a never-configured project gets —
// a 1:1 map of the legacy fixed kinds, so pre-blueprint behaviour is
// preserved exactly until the operator (or the AI proposer) reshapes it.
func defaultSections(cwd string) []Section {
	return []Section{
		{Cwd: cwd, Slug: SlugOverview, Title: "Overview", Position: 0, MaintainerMode: "ai", WritePolicy: "proposal", Pinned: true, Inject: false,
			Description: "The project's official document — the comprehensive page a developer reads to understand the whole project."},
		{Cwd: cwd, Slug: "goal", Title: "Goal", Position: 1, MaintainerMode: "ai", WritePolicy: "proposal", Inject: true,
			Description: "Long-term intent: what this project is for and what done looks like."},
		{Cwd: cwd, Slug: "plan", Title: "Plan", Position: 2, MaintainerMode: "ai", WritePolicy: "proposal", Inject: true,
			Description: "The current roadmap / work-in-progress arc."},
		{Cwd: cwd, Slug: "current_objective", Title: "Current Objective", Position: 3, MaintainerMode: "ai", WritePolicy: "direct", Inject: true,
			Description: "The short-term objective we are working on right now and its immediate steps — set in-session, completed, then rolled to the next. Not permanent.",
			PromptHint:  "This is the CURRENT short-term objective plus its immediate steps. It is expected to change FREQUENTLY: when a session establishes a new immediate objective, replace this with it; when a session completes it, roll it forward to the next objective and note what was just finished. Do NOT treat it as long-term intent — that is the Goal."},
		{Cwd: cwd, Slug: "tech_stack", Title: "Tech stack", Position: 4, MaintainerMode: "scanner", WritePolicy: "proposal", Inject: true,
			Description: "Architecture, stack and repo structure — rebuilt mechanically by the project scanner."},
		{Cwd: cwd, Slug: "recent_activity", Title: "Recent activity", Position: 5, MaintainerMode: "scanner", WritePolicy: "proposal", Inject: true,
			Description: "Narrative summary of recent git history — rebuilt mechanically by the activity scanner."},
	}
}

// kbDefaultSections is the global knowledge blueprint a fresh install
// gets — the four classic pages, pinned (the KB drafter re-drafts them
// and the spawn guardrails build on the foundational pair). Mirrors
// the 0050 seed.
func kbDefaultSections() []Section {
	return []Section{
		{Cwd: GlobalCwd, Slug: string(KindInfrastructure), Title: "Infrastructure", Position: 0, MaintainerMode: "ai", WritePolicy: "proposal", Pinned: true, Inject: true, Nature: "foundational",
			Description: "Standing ground truth about the home-lab/ecosystem: hosts, networks, databases, gateways — plus the binding rules for using them."},
		{Cwd: GlobalCwd, Slug: string(KindConventions), Title: "Conventions", Position: 1, MaintainerMode: "ai", WritePolicy: "proposal", Pinned: true, Inject: true, Nature: "foundational",
			Description: "The binding development conventions & policies: stack, source control, coding rules, release process."},
		{Cwd: GlobalCwd, Slug: string(KindLessons), Title: "Lessons", Position: 2, MaintainerMode: "ai", WritePolicy: "proposal", Pinned: true, Inject: true, Nature: "emergent",
			Description: "Distilled playbooks and hard-won lessons from past work — reference guidance, not law."},
		{Cwd: GlobalCwd, Slug: string(KindReusable), Title: "Reusable features", Position: 3, MaintainerMode: "ai", WritePolicy: "proposal", Pinned: true, Inject: true, Nature: "emergent",
			Description: "Catalog of features/components/patterns liftable into new projects."},
	}
}

// ListSections returns the blueprint for cwd ordered pinned-first then
// by position. A never-seen cwd is lazily seeded with the default
// blueprint (idempotent — ON CONFLICT DO NOTHING), so every project
// always has a usable section set without an explicit setup step.
// GlobalCwd returns the KNOWLEDGE blueprint — the knowledge base's
// page set, extensible exactly like a project's sections.
func (s *Service) ListSections(ctx context.Context, cwd string) ([]Section, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, ErrEmptyCwd
	}
	sections, err := s.querySections(ctx, cwd)
	if err != nil {
		return nil, err
	}
	if len(sections) > 0 {
		return sections, nil
	}
	// Ephemeral cwds (tmp dirs from third-party consumers / tests) are
	// not projects: serve the in-memory defaults so a UI that opens one
	// still renders, but never persist a blueprint for them.
	if cwd != GlobalCwd && IsEphemeralCwd(cwd) {
		return defaultSections(cwd), nil
	}
	if err := s.seedSections(ctx, cwd); err != nil {
		return nil, err
	}
	return s.querySections(ctx, cwd)
}

func (s *Service) querySections(ctx context.Context, cwd string) ([]Section, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cwd, slug, title, description, position, maintainer_mode,
		       COALESCE(write_policy, 'proposal'), prompt_hint, pinned, inject,
		       COALESCE(nature, ''), created_at, updated_at
		  FROM doc_blueprint_sections
		 WHERE cwd = $1
		 ORDER BY pinned DESC, position ASC, slug ASC`, cwd)
	if err != nil {
		return nil, fmt.Errorf("projectdoc: list sections: %w", err)
	}
	defer rows.Close()
	var out []Section
	for rows.Next() {
		var sec Section
		if err := rows.Scan(&sec.Cwd, &sec.Slug, &sec.Title, &sec.Description,
			&sec.Position, &sec.MaintainerMode, &sec.WritePolicy, &sec.PromptHint,
			&sec.Pinned, &sec.Inject, &sec.Nature, &sec.CreatedAt, &sec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("projectdoc: scan section: %w", err)
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

func (s *Service) seedSections(ctx context.Context, cwd string) error {
	defaults := defaultSections(cwd)
	if cwd == GlobalCwd {
		defaults = kbDefaultSections()
	}
	for _, sec := range defaults {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO doc_blueprint_sections
				(cwd, slug, title, description, position, maintainer_mode, write_policy, prompt_hint, pinned, inject, nature)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (cwd, slug) DO NOTHING`,
			sec.Cwd, sec.Slug, sec.Title, sec.Description, sec.Position,
			sec.MaintainerMode, normalizeWritePolicy(sec.WritePolicy), sec.PromptHint, sec.Pinned, sec.Inject, sec.Nature); err != nil {
			return fmt.Errorf("projectdoc: seed section %s: %w", sec.Slug, err)
		}
	}
	return nil
}

// PutSection upserts one blueprint section (create a new section or
// retitle / reorder / re-mode an existing one). Per-project sections
// must not use the kb_ prefix; GLOBAL knowledge pages must — and carry
// a nature (foundational|emergent). Pinned reserved sections keep
// their flag no matter what the caller sends.
func (s *Service) PutSection(ctx context.Context, sec Section) (Section, error) {
	if strings.TrimSpace(sec.Cwd) == "" {
		return Section{}, ErrEmptyCwd
	}
	if sec.Cwd == GlobalCwd {
		if !ValidGlobalKBSlug(sec.Slug) {
			return Section{}, fmt.Errorf("%w: knowledge pages need a kb_* slug, got %q", ErrInvalidKind, sec.Slug)
		}
		if !ValidNature(sec.Nature) {
			return Section{}, fmt.Errorf("projectdoc: knowledge nature must be foundational|emergent, got %q", sec.Nature)
		}
		if IsGlobalKBKind(Kind(sec.Slug)) {
			sec.Pinned = true // the classic four stay pinned
		}
	} else {
		if IsEphemeralCwd(sec.Cwd) {
			return Section{}, ErrEphemeralCwd
		}
		if !ValidSectionSlug(sec.Slug) {
			return Section{}, fmt.Errorf("%w: bad slug %q", ErrInvalidKind, sec.Slug)
		}
		sec.Nature = "" // nature is a knowledge-layer concept
		if sec.Slug == SlugOverview {
			sec.Pinned = true // the front page stays pinned
		}
	}
	if !ValidMaintainerMode(sec.MaintainerMode) {
		return Section{}, fmt.Errorf("projectdoc: maintainer_mode must be ai|human|scanner, got %q", sec.MaintainerMode)
	}
	// Empty defaults to the safe proposal gate; a bad explicit value is
	// rejected so a caller can't silently mistype direct-write away.
	if sec.WritePolicy != "" && !ValidWritePolicy(sec.WritePolicy) {
		return Section{}, fmt.Errorf("projectdoc: write_policy must be direct|proposal, got %q", sec.WritePolicy)
	}
	sec.WritePolicy = normalizeWritePolicy(sec.WritePolicy)
	if strings.TrimSpace(sec.Title) == "" {
		return Section{}, errors.New("projectdoc: section title is required")
	}
	// Lazy-seed first so a brand-new blueprint's custom section lands
	// in a complete set rather than an otherwise-empty one.
	if _, err := s.ListSections(ctx, sec.Cwd); err != nil {
		return Section{}, err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO doc_blueprint_sections
			(cwd, slug, title, description, position, maintainer_mode, write_policy, prompt_hint, pinned, inject, nature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (cwd, slug) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			position = EXCLUDED.position,
			maintainer_mode = EXCLUDED.maintainer_mode,
			write_policy = EXCLUDED.write_policy,
			prompt_hint = EXCLUDED.prompt_hint,
			pinned = EXCLUDED.pinned,
			inject = EXCLUDED.inject,
			nature = EXCLUDED.nature,
			updated_at = NOW()
		RETURNING cwd, slug, title, description, position, maintainer_mode,
		          COALESCE(write_policy, 'proposal'), prompt_hint, pinned, inject,
		          COALESCE(nature, ''), created_at, updated_at`,
		sec.Cwd, sec.Slug, sec.Title, sec.Description, sec.Position,
		sec.MaintainerMode, sec.WritePolicy, sec.PromptHint, sec.Pinned, sec.Inject, sec.Nature)
	var out Section
	if err := row.Scan(&out.Cwd, &out.Slug, &out.Title, &out.Description,
		&out.Position, &out.MaintainerMode, &out.WritePolicy, &out.PromptHint,
		&out.Pinned, &out.Inject, &out.Nature, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return Section{}, fmt.Errorf("projectdoc: put section: %w", err)
	}
	return out, nil
}

// DeleteSection removes a section from the blueprint. Pinned reserved
// sections (overview; the classic knowledge four) cannot be deleted.
// The section's project_docs row is deliberately KEPT — deleting a
// section hides it; re-adding the same slug resurrects the content.
func (s *Service) DeleteSection(ctx context.Context, cwd, slug string) error {
	if strings.TrimSpace(cwd) == "" {
		return ErrEmptyCwd
	}
	if slug == SlugOverview || (cwd == GlobalCwd && IsGlobalKBKind(Kind(slug))) {
		return ErrReservedSection
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM doc_blueprint_sections WHERE cwd = $1 AND slug = $2 AND NOT pinned`, cwd, slug)
	if err != nil {
		return fmt.Errorf("projectdoc: delete section: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// validateWriteTarget is the write-path validation for PutDoc /
// ProposeDoc: syntax + kind↔cwd pairing, plus blueprint membership
// for per-project sections. A blueprint lookup failure (e.g. the 0046
// table missing in an un-migrated test DB) degrades to syntax-only —
// a doc write must never be blocked by blueprint plumbing.
func (s *Service) validateWriteTarget(ctx context.Context, cwd string, kind Kind) error {
	if err := validateKindForCwd(cwd, kind); err != nil {
		return err
	}
	// Temp dirs are not projects — refuse to create doc footprint for
	// them (third-party consumers + tests spawn there constantly).
	if cwd != GlobalCwd && IsEphemeralCwd(cwd) {
		return ErrEphemeralCwd
	}
	// Both project docs AND global knowledge pages must exist in their
	// blueprint (lazily seeded with the defaults).
	ok, err := s.HasSection(ctx, cwd, string(kind), "")
	if err != nil {
		s.log.Warn("projectdoc: blueprint check failed — allowing write",
			"cwd", cwd, "kind", kind, "err", err)
		return nil
	}
	if !ok {
		return fmt.Errorf("%w: %q is not in this blueprint", ErrInvalidKind, kind)
	}
	return nil
}

// HasSection reports whether the blueprint for cwd currently contains
// slug with the given maintainer mode (mode "" = any). Used by the
// scanners so a removed scanner section silently stops being written.
func (s *Service) HasSection(ctx context.Context, cwd, slug, mode string) (bool, error) {
	sections, err := s.ListSections(ctx, cwd)
	if err != nil {
		return false, err
	}
	for _, sec := range sections {
		if sec.Slug == slug && (mode == "" || sec.MaintainerMode == mode) {
			return true, nil
		}
	}
	return false, nil
}
