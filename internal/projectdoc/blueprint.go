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
	// Pinned sections sort first and cannot be deleted (overview).
	Pinned bool `json:"pinned"`
	// Inject includes this section's doc in the spawn banner.
	Inject    bool      `json:"inject"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SlugOverview is the reserved front-page section: present in every
// blueprint, pinned, undeletable.
const SlugOverview = "overview"

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,47}$`)

// ValidSectionSlug reports whether s is a legal blueprint slug. kb_*
// is reserved for the global Knowledge pages and never valid inside a
// per-project blueprint.
func ValidSectionSlug(s string) bool {
	return slugRe.MatchString(s) && !strings.HasPrefix(s, "kb_")
}

// ValidMaintainerMode reports whether m is ai|human|scanner.
func ValidMaintainerMode(m string) bool {
	return m == "ai" || m == "human" || m == "scanner"
}

// ErrReservedSection is returned when a caller tries to delete or
// demote the reserved overview section.
var ErrReservedSection = errors.New("projectdoc: the overview section is reserved")

// defaultSections is the blueprint a never-configured project gets —
// a 1:1 map of the legacy fixed kinds, so pre-blueprint behaviour is
// preserved exactly until the operator (or the AI proposer) reshapes it.
func defaultSections(cwd string) []Section {
	return []Section{
		{Cwd: cwd, Slug: SlugOverview, Title: "Overview", Position: 0, MaintainerMode: "ai", Pinned: true, Inject: false,
			Description: "The project's official document — the comprehensive page a developer reads to understand the whole project."},
		{Cwd: cwd, Slug: "goal", Title: "Goal", Position: 1, MaintainerMode: "ai", Inject: true,
			Description: "Long-term intent: what this project is for and what done looks like."},
		{Cwd: cwd, Slug: "plan", Title: "Plan", Position: 2, MaintainerMode: "ai", Inject: true,
			Description: "The current roadmap / work-in-progress arc."},
		{Cwd: cwd, Slug: "tech_stack", Title: "Tech stack", Position: 3, MaintainerMode: "scanner", Inject: true,
			Description: "Architecture, stack and repo structure — rebuilt mechanically by the project scanner."},
		{Cwd: cwd, Slug: "recent_activity", Title: "Recent activity", Position: 4, MaintainerMode: "scanner", Inject: true,
			Description: "Narrative summary of recent git history — rebuilt mechanically by the activity scanner."},
	}
}

// ListSections returns the blueprint for cwd ordered pinned-first then
// by position. A never-seen cwd is lazily seeded with the default
// blueprint (idempotent — ON CONFLICT DO NOTHING), so every project
// always has a usable section set without an explicit setup step.
func (s *Service) ListSections(ctx context.Context, cwd string) ([]Section, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, ErrEmptyCwd
	}
	if cwd == GlobalCwd {
		return nil, fmt.Errorf("projectdoc: %q has no blueprint (global KB pages are fixed)", GlobalCwd)
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
	if IsEphemeralCwd(cwd) {
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
		       prompt_hint, pinned, inject, created_at, updated_at
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
			&sec.Position, &sec.MaintainerMode, &sec.PromptHint,
			&sec.Pinned, &sec.Inject, &sec.CreatedAt, &sec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("projectdoc: scan section: %w", err)
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

func (s *Service) seedSections(ctx context.Context, cwd string) error {
	for _, sec := range defaultSections(cwd) {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO doc_blueprint_sections
				(cwd, slug, title, description, position, maintainer_mode, prompt_hint, pinned, inject)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (cwd, slug) DO NOTHING`,
			sec.Cwd, sec.Slug, sec.Title, sec.Description, sec.Position,
			sec.MaintainerMode, sec.PromptHint, sec.Pinned, sec.Inject); err != nil {
			return fmt.Errorf("projectdoc: seed section %s: %w", sec.Slug, err)
		}
	}
	return nil
}

// PutSection upserts one blueprint section (create a new section or
// retitle / reorder / re-mode an existing one). The reserved overview
// keeps its pinned flag no matter what the caller sends.
func (s *Service) PutSection(ctx context.Context, sec Section) (Section, error) {
	if strings.TrimSpace(sec.Cwd) == "" {
		return Section{}, ErrEmptyCwd
	}
	if sec.Cwd == GlobalCwd {
		return Section{}, fmt.Errorf("projectdoc: %q has no blueprint", GlobalCwd)
	}
	if IsEphemeralCwd(sec.Cwd) {
		return Section{}, ErrEphemeralCwd
	}
	if !ValidSectionSlug(sec.Slug) {
		return Section{}, fmt.Errorf("%w: bad slug %q", ErrInvalidKind, sec.Slug)
	}
	if !ValidMaintainerMode(sec.MaintainerMode) {
		return Section{}, fmt.Errorf("projectdoc: maintainer_mode must be ai|human|scanner, got %q", sec.MaintainerMode)
	}
	if strings.TrimSpace(sec.Title) == "" {
		return Section{}, errors.New("projectdoc: section title is required")
	}
	if sec.Slug == SlugOverview {
		sec.Pinned = true // the front page stays pinned
	}
	// Lazy-seed first so a brand-new project's custom section lands in
	// a complete blueprint rather than an otherwise-empty one.
	if _, err := s.ListSections(ctx, sec.Cwd); err != nil {
		return Section{}, err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO doc_blueprint_sections
			(cwd, slug, title, description, position, maintainer_mode, prompt_hint, pinned, inject)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (cwd, slug) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			position = EXCLUDED.position,
			maintainer_mode = EXCLUDED.maintainer_mode,
			prompt_hint = EXCLUDED.prompt_hint,
			pinned = EXCLUDED.pinned,
			inject = EXCLUDED.inject,
			updated_at = NOW()
		RETURNING cwd, slug, title, description, position, maintainer_mode,
		          prompt_hint, pinned, inject, created_at, updated_at`,
		sec.Cwd, sec.Slug, sec.Title, sec.Description, sec.Position,
		sec.MaintainerMode, sec.PromptHint, sec.Pinned, sec.Inject)
	var out Section
	if err := row.Scan(&out.Cwd, &out.Slug, &out.Title, &out.Description,
		&out.Position, &out.MaintainerMode, &out.PromptHint,
		&out.Pinned, &out.Inject, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return Section{}, fmt.Errorf("projectdoc: put section: %w", err)
	}
	return out, nil
}

// DeleteSection removes a section from the blueprint. The overview is
// reserved and cannot be deleted. The section's project_docs row is
// deliberately KEPT — deleting a section hides it; re-adding the same
// slug resurrects the content. Accidental deletes lose nothing.
func (s *Service) DeleteSection(ctx context.Context, cwd, slug string) error {
	if strings.TrimSpace(cwd) == "" {
		return ErrEmptyCwd
	}
	if slug == SlugOverview {
		return ErrReservedSection
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM doc_blueprint_sections WHERE cwd = $1 AND slug = $2`, cwd, slug)
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
	if cwd == GlobalCwd {
		return nil
	}
	// Temp dirs are not projects — refuse to create doc footprint for
	// them (third-party consumers + tests spawn there constantly).
	if IsEphemeralCwd(cwd) {
		return ErrEphemeralCwd
	}
	ok, err := s.HasSection(ctx, cwd, string(kind), "")
	if err != nil {
		s.log.Warn("projectdoc: blueprint check failed — allowing write",
			"cwd", cwd, "kind", kind, "err", err)
		return nil
	}
	if !ok {
		return fmt.Errorf("%w: section %q is not in this project's blueprint", ErrInvalidKind, kind)
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
