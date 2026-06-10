package projectdoc

import (
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

func TestValidKind_BlueprintSemantics(t *testing.T) {
	// Global KB pages remain valid kinds.
	for _, k := range []Kind{KindInfrastructure, KindConventions, KindLessons, KindReusable} {
		if !ValidKind(k) {
			t.Errorf("ValidKind(%q) = false, want true", k)
		}
	}
	// Retired handbook stays invalid.
	if ValidKind(KindHandbook) {
		t.Errorf("ValidKind(kb_handbook) = true, want false (retired)")
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
	if len(secs) != 5 {
		t.Fatalf("default blueprint has %d sections, want 5", len(secs))
	}
	if secs[0].Slug != SlugOverview || !secs[0].Pinned || secs[0].Inject {
		t.Errorf("overview must be first, pinned, and not injected: %+v", secs[0])
	}
	for _, sec := range secs {
		if !ValidSectionSlug(sec.Slug) {
			t.Errorf("default slug %q fails its own validation", sec.Slug)
		}
		if !ValidMaintainerMode(sec.MaintainerMode) {
			t.Errorf("default section %q has bad mode %q", sec.Slug, sec.MaintainerMode)
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
