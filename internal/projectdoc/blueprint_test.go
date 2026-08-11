package projectdoc

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestValidSectionSlug(t *testing.T) {
	tests := []struct {
		slug string
		want bool
	}{
		{"overview", true},
		{"goal", true},
		{"tech_stack", true},
		{"api_surface", true},
		{"release_notes2", true},
		{"kb_lessons", false},            // reserved global prefix
		{"kb_anything", false},           // reserved global prefix
		{"Goal", false},                  // uppercase
		{"a", false},                     // too short
		{"1abc", false},                  // must start with a letter
		{"has-dash", false},              // dashes not allowed
		{"", false},                      //
		{strings.Repeat("a", 49), false}, // too long
		{strings.Repeat("a", 48), true},
	}
	for _, tt := range tests {
		if got := ValidSectionSlug(tt.slug); got != tt.want {
			t.Errorf("ValidSectionSlug(%q) = %v, want %v", tt.slug, got, tt.want)
		}
	}
}

func TestValidMaintainerMode(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"ai", true},
		{"human", true},
		{"scanner", true},
		{"session", true}, // in-session agent maintains the page
		{"", false},
		{"librarian", false},
		{"AI", false},
	}
	for _, tt := range tests {
		if got := ValidMaintainerMode(tt.mode); got != tt.want {
			t.Errorf("ValidMaintainerMode(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

// The Go-side set and the DB CHECK constraint must agree. They are two
// declarations of one rule, and when they drift the failure surfaces only
// at write time as a raw 23514 in front of the operator — which is
// exactly what happened when "session" was added here without widening
// the constraint.
func TestMaintainerModesMatchMigration(t *testing.T) {
	dbModes := modesFromMigrations(t)
	if len(dbModes) == 0 {
		t.Fatal("no maintainer_mode CHECK found in the migrations — did the file move?")
	}
	for _, m := range MaintainerModes {
		if !dbModes[m] {
			t.Errorf("maintainer_mode %q is valid in Go but rejected by the DB constraint — add a migration widening doc_blueprint_sections_maintainer_mode_check", m)
		}
	}
	for m := range dbModes {
		if !ValidMaintainerMode(m) {
			t.Errorf("maintainer_mode %q is allowed by the DB but rejected in Go — the two declarations have drifted", m)
		}
	}
}

// modesFromMigrations returns the values allowed by the LAST migration
// that (re)declares the maintainer_mode CHECK, which is the one in force.
func modesFromMigrations(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "store", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files) // numeric prefixes sort chronologically
	re := regexp.MustCompile(`maintainer_mode\s+IN\s*\(([^)]*)\)`)

	out := map[string]bool{}
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		m := re.FindSubmatch(body)
		if m == nil {
			continue
		}
		out = map[string]bool{} // a later migration REPLACES the constraint
		for _, raw := range strings.Split(string(m[1]), ",") {
			if v := strings.Trim(strings.TrimSpace(raw), "'"); v != "" {
				out[v] = true
			}
		}
	}
	return out
}

func TestSessionWritable(t *testing.T) {
	// The whole point: writability is decided by the page's own
	// blueprint metadata, never by a hardcoded slug list. A user who
	// never creates a session-maintained page gets nothing writable.
	tests := []struct {
		name string
		sec  Section
		want bool
	}{
		{
			name: "session-maintained emergent page",
			sec:  Section{Cwd: GlobalCwd, Slug: "kb_projects", MaintainerMode: "session", Nature: "emergent"},
			want: true,
		},
		{
			name: "any custom slug qualifies — no hardcoded allowlist",
			sec:  Section{Cwd: GlobalCwd, Slug: "kb_a", MaintainerMode: "session", Nature: "emergent"},
			want: true,
		},
		{
			name: "ai-maintained page belongs to the Cortex sweep",
			sec:  Section{Cwd: GlobalCwd, Slug: "kb_lessons", MaintainerMode: "ai", Nature: "emergent"},
			want: false,
		},
		{
			name: "human-maintained page is the operator's alone",
			sec:  Section{Cwd: GlobalCwd, Slug: "kb_integrations", MaintainerMode: "human", Nature: "emergent"},
			want: false,
		},
		{
			name: "foundational pages are binding rules — never session-writable",
			sec:  Section{Cwd: GlobalCwd, Slug: "kb_conventions", MaintainerMode: "session", Nature: "foundational"},
			want: false,
		},
		{
			name: "pinned pages are reserved even if mislabelled session",
			sec:  Section{Cwd: GlobalCwd, Slug: "kb_lessons", MaintainerMode: "session", Nature: "emergent", Pinned: true},
			want: false,
		},
		{
			name: "per-project sections are out of scope (global KB only)",
			sec:  Section{Cwd: "/repo", Slug: "goal", MaintainerMode: "session"},
			want: false,
		},
		{
			name: "non-kb slug under global cwd",
			sec:  Section{Cwd: GlobalCwd, Slug: "overview", MaintainerMode: "session"},
			want: false,
		},
	}
	for _, tt := range tests {
		if got := SessionWritable(tt.sec); got != tt.want {
			t.Errorf("%s: SessionWritable() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestGlobalKBWriteGuard(t *testing.T) {
	sections := []Section{
		{Cwd: GlobalCwd, Slug: "kb_conventions", MaintainerMode: "ai", Nature: "foundational", Pinned: true},
		{Cwd: GlobalCwd, Slug: "kb_lessons", MaintainerMode: "ai", Nature: "emergent", Pinned: true},
		{Cwd: GlobalCwd, Slug: "kb_integrations", MaintainerMode: "human", Nature: "emergent", Pinned: true},
		{Cwd: GlobalCwd, Slug: "kb_projects", MaintainerMode: "session", Nature: "emergent"},
	}
	tests := []struct {
		name    string
		kind    Kind
		wantErr bool
	}{
		{"session-maintained page passes", "kb_projects", false},
		{"sweep-owned page refused", "kb_lessons", true},
		{"operator-owned page refused", "kb_integrations", true},
		{"foundational page refused", "kb_conventions", true},
		{"unknown page refused — fail closed", "kb_ghost", true},
	}
	for _, tt := range tests {
		err := globalKBWriteGuard(sections, tt.kind)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: globalKBWriteGuard(%q) err = %v, wantErr %v",
				tt.name, tt.kind, err, tt.wantErr)
		}
		if tt.wantErr && err != nil && !errors.Is(err, ErrNotSessionWritable) {
			t.Errorf("%s: got %v, want ErrNotSessionWritable", tt.name, err)
		}
	}
}

// A blueprint we failed to read must never fall through to a write:
// an unknown page is refused, so a lookup failure has to be too.
func TestGlobalKBWriteGuard_EmptyBlueprintFailsClosed(t *testing.T) {
	if err := globalKBWriteGuard(nil, "kb_projects"); err == nil {
		t.Error("globalKBWriteGuard(nil, …) = nil, want refusal")
	}
}

// The spawn index must mark session-maintained pages, so the injected
// instructions never point an agent at a page it has no tool for.
func TestRenderKnowledgeIndex_MarksSessionPages(t *testing.T) {
	out := renderKnowledgeIndex([]Section{
		{Cwd: GlobalCwd, Slug: "kb_conventions", Title: "Conventions", Nature: "foundational", Inject: true, Pinned: true},
		{Cwd: GlobalCwd, Slug: "kb_lessons", Title: "Lessons", Nature: "emergent", Pinned: true, MaintainerMode: "ai"},
		{Cwd: GlobalCwd, Slug: "kb_projects", Title: "Local Projects", Nature: "emergent",
			MaintainerMode: "session", Description: "Per-project records."},
	})

	if strings.Contains(out, "kb_conventions") {
		t.Error("a foundational injected page is already in the banner; it must not repeat in the index")
	}
	projects := indexLine(out, "kb_projects")
	if !strings.Contains(projects, "kb_page_set") {
		t.Errorf("session-maintained page is not marked writable:\n%s", projects)
	}
	if lessons := indexLine(out, "kb_lessons"); strings.Contains(lessons, "kb_page_set") {
		t.Errorf("sweep-owned page must not be marked writable:\n%s", lessons)
	}
	if !strings.Contains(out, "yours to keep current") {
		t.Error("index should explain the marked pages when at least one is writable")
	}
}

// With nothing session-maintained the index stays exactly as it was —
// no tool mentioned, no trailing instruction.
func TestRenderKnowledgeIndex_NoSessionPagesNoInstruction(t *testing.T) {
	out := renderKnowledgeIndex([]Section{
		{Cwd: GlobalCwd, Slug: "kb_lessons", Title: "Lessons", Nature: "emergent", Pinned: true, MaintainerMode: "ai"},
	})
	if strings.Contains(out, "kb_page_set") || strings.Contains(out, "yours to keep current") {
		t.Errorf("no page is session-maintained; index must not mention the tool:\n%s", out)
	}
}

// indexLine returns the index line mentioning slug (empty if absent).
func indexLine(out, slug string) string {
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, slug) {
			return ln
		}
	}
	return ""
}

func TestValidKind_BlueprintSemantics(t *testing.T) {
	// Global KB pages remain valid kinds.
	for _, k := range []Kind{KindInfrastructure, KindConventions, KindLessons, KindReusable} {
		if !ValidKind(k) {
			t.Errorf("ValidKind(%q) = false, want true", k)
		}
	}
	// The retired handbook is now merely a syntactically valid kb_*
	// slug — writes are gated by knowledge-blueprint membership, and
	// it is seeded nowhere.
	if !ValidKind(KindHandbook) {
		t.Errorf("ValidKind(kb_handbook) = false, want true (syntax-level only)")
	}
	// Custom knowledge pages are valid kinds (knowledge blueprint).
	if !ValidKind("kb_network_topology") {
		t.Errorf("ValidKind(kb_network_topology) = false, want true")
	}
	// Arbitrary well-formed section slugs are now valid kinds.
	if !ValidKind("api_surface") {
		t.Errorf("ValidKind(api_surface) = false, want true")
	}
	if ValidKind("Bad Slug") {
		t.Errorf("ValidKind('Bad Slug') = true, want false")
	}
}

func TestValidateKindForCwd(t *testing.T) {
	if err := validateKindForCwd(GlobalCwd, KindLessons); err != nil {
		t.Errorf("kb page under GlobalCwd should validate, got %v", err)
	}
	if err := validateKindForCwd("/proj", KindLessons); err == nil {
		t.Errorf("kb page under a project cwd must be rejected")
	}
	if err := validateKindForCwd(GlobalCwd, KindPlan); err == nil {
		t.Errorf("per-project slug under GlobalCwd must be rejected")
	}
	if err := validateKindForCwd("/proj", "custom_section"); err != nil {
		t.Errorf("custom slug under a project cwd should validate, got %v", err)
	}
}

func TestDefaultSectionsShape(t *testing.T) {
	secs := defaultSections("/p")
	if len(secs) != 6 {
		t.Fatalf("default blueprint has %d sections, want 6", len(secs))
	}
	if secs[0].Slug != SlugOverview || !secs[0].Pinned || secs[0].Inject {
		t.Errorf("overview must be first, pinned, and not injected: %+v", secs[0])
	}
	var sawDirect bool
	for _, sec := range secs {
		if !ValidSectionSlug(sec.Slug) {
			t.Errorf("default slug %q fails its own validation", sec.Slug)
		}
		if !ValidMaintainerMode(sec.MaintainerMode) {
			t.Errorf("default section %q has bad mode %q", sec.Slug, sec.MaintainerMode)
		}
		if !ValidWritePolicy(sec.WritePolicy) {
			t.Errorf("default section %q has bad write_policy %q", sec.Slug, sec.WritePolicy)
		}
		if sec.Slug == "current_objective" {
			if sec.WritePolicy != "direct" {
				t.Errorf("current_objective must be direct-write, got %q", sec.WritePolicy)
			}
			sawDirect = true
		}
	}
	if !sawDirect {
		t.Error("default blueprint is missing the direct-write current_objective section")
	}
}

func TestKBDefaultSectionsShape(t *testing.T) {
	secs := kbDefaultSections()
	if len(secs) != 4 {
		t.Fatalf("knowledge blueprint defaults = %d sections, want 4", len(secs))
	}
	natures := map[string]int{}
	for _, sec := range secs {
		if sec.Cwd != GlobalCwd {
			t.Errorf("section %q cwd = %q, want %q", sec.Slug, sec.Cwd, GlobalCwd)
		}
		if !ValidGlobalKBSlug(sec.Slug) {
			t.Errorf("slug %q fails ValidGlobalKBSlug", sec.Slug)
		}
		if !ValidNature(sec.Nature) {
			t.Errorf("section %q nature %q invalid", sec.Slug, sec.Nature)
		}
		if !sec.Pinned {
			t.Errorf("classic page %q must be pinned (drafter + guardrails depend on it)", sec.Slug)
		}
		if !sec.Inject {
			t.Errorf("classic page %q should inject by default", sec.Slug)
		}
		natures[sec.Nature]++
	}
	if natures["foundational"] != 2 || natures["emergent"] != 2 {
		t.Errorf("natures = %v, want 2 foundational + 2 emergent", natures)
	}
}

func TestValidGlobalKBSlug(t *testing.T) {
	for slug, want := range map[string]bool{
		"kb_infrastructure":   true,
		"kb_network_topology": true,
		"kb_x":                true,
		"kb_":                 false, // nothing after the prefix
		"goal":                false, // no prefix
		"kb_Bad":              false, // uppercase
		"kb_has-dash":         false,
		"":                    false,
	} {
		if got := ValidGlobalKBSlug(slug); got != want {
			t.Errorf("ValidGlobalKBSlug(%q) = %v, want %v", slug, got, want)
		}
	}
}

func TestSectionDriftSystemPrompt(t *testing.T) {
	// Built-ins keep their tuned prompts.
	if got := SectionDriftSystemPrompt(DriftInput{Kind: KindGoal}); got != GoalDriftSystemPrompt {
		t.Errorf("goal drift prompt not the tuned one")
	}
	if got := SectionDriftSystemPrompt(DriftInput{Kind: KindPlan}); got != PlanDriftSystemPrompt {
		t.Errorf("plan drift prompt not the tuned one")
	}
	if got := SectionDriftSystemPrompt(DriftInput{}); got != PlanDriftSystemPrompt {
		t.Errorf("empty kind should default to the plan prompt")
	}
	// Custom sections get a parameterized prompt carrying their metadata.
	got := SectionDriftSystemPrompt(DriftInput{
		Kind:               "api_surface",
		SectionTitle:       "Public API",
		SectionDescription: "The HTTP surface third parties consume.",
		SectionPromptHint:  "List every route with auth requirements.",
	})
	for _, want := range []string{"Public API", "The HTTP surface", "List every route", "should_propose"} {
		if !strings.Contains(got, want) {
			t.Errorf("custom section prompt missing %q", want)
		}
	}
}
