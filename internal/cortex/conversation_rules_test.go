package cortex

import (
	"reflect"
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

func TestRuleSuggestionFrom(t *testing.T) {
	mk := func(action, guidance string, add, remove []string) curationReply {
		var r curationReply
		r.Reply = "ok"
		r.Rules.Action = action
		r.Rules.Guidance = guidance
		r.Rules.ExclusionsAdd = add
		r.Rules.ExclusionsRemove = remove
		r.Rules.Reason = "because"
		return r
	}

	t.Run("action none yields nil", func(t *testing.T) {
		if got := ruleSuggestionFrom(mk("none", "g", nil, nil), TargetKBPage); got != nil {
			t.Fatalf("want nil, got %+v", got)
		}
	})
	t.Run("suggest with all-empty payload yields nil", func(t *testing.T) {
		if got := ruleSuggestionFrom(mk("suggest", "  ", []string{" ", ""}, nil), TargetKBPage); got != nil {
			t.Fatalf("want nil, got %+v", got)
		}
	})
	t.Run("suggest trims and keeps payload", func(t *testing.T) {
		got := ruleSuggestionFrom(mk("suggest", " keep it short ", []string{" rcc ", ""}, []string{"ntc"}), TargetKBPage)
		if got == nil {
			t.Fatal("want suggestion, got nil")
		}
		if got.Guidance != "keep it short" {
			t.Errorf("guidance = %q", got.Guidance)
		}
		if !reflect.DeepEqual(got.ExclusionsAdd, []string{"rcc"}) {
			t.Errorf("add = %v", got.ExclusionsAdd)
		}
		if !reflect.DeepEqual(got.ExclusionsRemove, []string{"ntc"}) {
			t.Errorf("remove = %v", got.ExclusionsRemove)
		}
	})
	t.Run("doc_section target drops exclusions", func(t *testing.T) {
		got := ruleSuggestionFrom(mk("suggest", "hint", []string{"rcc"}, []string{"ntc"}), TargetDocSection)
		if got == nil {
			t.Fatal("want suggestion, got nil")
		}
		if len(got.ExclusionsAdd) != 0 || len(got.ExclusionsRemove) != 0 {
			t.Errorf("exclusions should be dropped for doc sections, got +%v -%v", got.ExclusionsAdd, got.ExclusionsRemove)
		}
	})
	t.Run("doc_section exclusions-only suggestion yields nil", func(t *testing.T) {
		if got := ruleSuggestionFrom(mk("suggest", "", []string{"rcc"}, nil), TargetDocSection); got != nil {
			t.Fatalf("want nil, got %+v", got)
		}
	})
	t.Run("blueprint target yields nil", func(t *testing.T) {
		if got := ruleSuggestionFrom(mk("suggest", "hint", nil, nil), TargetBlueprint); got != nil {
			t.Fatalf("want nil, got %+v", got)
		}
	})
}

func TestMergeRuleSuggestion(t *testing.T) {
	base := projectdoc.Section{
		PromptHint: "old hint",
		Exclusions: []string{"RCC", "ntc", "keep"},
	}

	t.Run("guidance replaces, empty leaves unchanged", func(t *testing.T) {
		got := mergeRuleSuggestion(base, RuleSuggestion{Guidance: "new hint"})
		if got.PromptHint != "new hint" {
			t.Errorf("hint = %q", got.PromptHint)
		}
		got = mergeRuleSuggestion(base, RuleSuggestion{})
		if got.PromptHint != "old hint" {
			t.Errorf("hint = %q, want unchanged", got.PromptHint)
		}
	})
	t.Run("remove is case-insensitive, add dedupes", func(t *testing.T) {
		got := mergeRuleSuggestion(base, RuleSuggestion{
			ExclusionsAdd:    []string{"Keep", "fresh"},
			ExclusionsRemove: []string{"rcc"},
		})
		if !reflect.DeepEqual(got.Exclusions, []string{"ntc", "keep", "fresh"}) {
			t.Errorf("exclusions = %v", got.Exclusions)
		}
	})
	t.Run("does not mutate the input section", func(t *testing.T) {
		_ = mergeRuleSuggestion(base, RuleSuggestion{ExclusionsAdd: []string{"x"}, ExclusionsRemove: []string{"keep"}})
		if !reflect.DeepEqual(base.Exclusions, []string{"RCC", "ntc", "keep"}) {
			t.Errorf("input mutated: %v", base.Exclusions)
		}
	})
}

func TestParseCurationReplyRulesBlock(t *testing.T) {
	raw := `{"reply":"done","revision":{"action":"none","content":"","reason":""},` +
		`"rules":{"action":"suggest","guidance":"track hosts only","exclusions_add":["rcc"],"exclusions_remove":[],"reason":"operator deleted rcc twice"}}`
	got, err := parseCurationReply(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Rules.Action != "suggest" || got.Rules.Guidance != "track hosts only" {
		t.Errorf("rules = %+v", got.Rules)
	}
	// Replies without the block (older/looser models) still parse.
	if _, err := parseCurationReply(`{"reply":"ok","revision":{"action":"none","content":"","reason":""}}`); err != nil {
		t.Errorf("reply without rules block should parse: %v", err)
	}
}

func TestCurationSchemaHasRules(t *testing.T) {
	for _, want := range []string{`"rules"`, `"suggest"`, `"exclusions_add"`, `"exclusions_remove"`} {
		if !strings.Contains(curationJSONSchema, want) {
			t.Errorf("curation schema missing %s", want)
		}
	}
}

func TestDescribeRuleApply(t *testing.T) {
	got := describeRuleApply(RuleSuggestion{
		Guidance:         "new",
		ExclusionsAdd:    []string{"rcc", "ntc"},
		ExclusionsRemove: []string{"old"},
	})
	for _, want := range []string{"guidance", "rcc", "ntc", "old"} {
		if !strings.Contains(got, want) {
			t.Errorf("describeRuleApply() = %q, missing %q", got, want)
		}
	}
}
