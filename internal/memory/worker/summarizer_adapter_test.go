package worker

import (
	"testing"

	"github.com/opendray/opendray-v2/internal/memory/summarizer"
)

func TestParseFactsJSON_Clean(t *testing.T) {
	raw := `{"facts":[{"text":"User prefers pnpm","category":"preference","confidence":0.9}]}`
	got, err := parseFactsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 || got[0].Text != "User prefers pnpm" ||
		got[0].Category != summarizer.CategoryPreference || got[0].Confidence != 0.9 {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseFactsJSON_Fenced(t *testing.T) {
	raw := "```json\n{\"facts\":[]}\n```"
	got, err := parseFactsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestParseFactsJSON_PreambleAndTrailingProse(t *testing.T) {
	raw := `Sure, here you go:
{"facts":[{"text":"DB host db.example.com","category":"identifier","confidence":0.95}]}
— end of output`
	got, err := parseFactsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 || got[0].Category != summarizer.CategoryIdentifier {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseFactsJSON_EmptyTextEntriesSkipped(t *testing.T) {
	raw := `{"facts":[
    {"text":"keep","category":"other","confidence":0.5},
    {"text":"   ","category":"other","confidence":0.4}
  ]}`
	got, err := parseFactsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 || got[0].Text != "keep" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseFactsJSON_Garbage(t *testing.T) {
	if _, err := parseFactsJSON("not json"); err == nil {
		t.Error("expected error for garbage input")
	}
	if _, err := parseFactsJSON(""); err == nil {
		t.Error("expected error for empty input")
	}
}

func TestStripJSONFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no fence", "{}", ""},
		{"json-tagged", "```json\n{\"x\":1}\n```", `{"x":1}`},
		{"untagged", "```\n{\"x\":1}\n```", `{"x":1}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripJSONFence(tc.in)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSummarizerAdapter_NameAndKind(t *testing.T) {
	a := NewSummarizerProvider(nil, TaskCapture)
	if name := a.Name(); name != "worker:capture" {
		t.Errorf("Name: got %q", name)
	}
	if kind := a.Kind(); kind != "worker:capture" {
		t.Errorf("Kind: got %q", kind)
	}
}
