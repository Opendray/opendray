package cortex

import (
	"strings"
	"testing"
)

func TestParseBlueprintProposal(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		wantN   int
	}{
		{
			name: "clean json",
			raw: `{"project_type":"mobile app","reason":"Flutter app detected","sections":[
				{"slug":"overview","title":"Overview","description":"d","position":0,"maintainer_mode":"ai","prompt_hint":"","pinned":true,"inject":false}]}`,
			wantN: 1,
		},
		{
			name:  "fenced with preamble",
			raw:   "Here is the blueprint:\n```json\n{\"project_type\":\"cli\",\"reason\":\"r\",\"sections\":[{\"slug\":\"overview\",\"title\":\"O\",\"description\":\"\",\"position\":0,\"maintainer_mode\":\"ai\",\"prompt_hint\":\"\",\"pinned\":true,\"inject\":false}]}\n```",
			wantN: 1,
		},
		{name: "empty sections", raw: `{"project_type":"x","reason":"r","sections":[]}`, wantErr: true},
		{name: "garbage", raw: "not json at all", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBlueprintProposal(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got.Sections) != tt.wantN {
				t.Errorf("sections = %d, want %d", len(got.Sections), tt.wantN)
			}
		})
	}
}

func TestBlueprintSystemPromptMentionsReservedSlugs(t *testing.T) {
	for _, want := range []string{`"overview"`, `"goal"`, `"plan"`, "kb_"} {
		if !strings.Contains(blueprintSystemPrompt, want) {
			t.Errorf("blueprint system prompt missing %q", want)
		}
	}
}
