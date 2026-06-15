package cortex

import (
	"strings"
	"testing"
)

func TestParseCurationReply(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantErr    bool
		wantAction string
	}{
		{
			name:       "reply with update revision",
			raw:        `{"reply":"Updated the tech stack section.","revision":{"action":"update","content":"# Tech\n- Go","reason":"operator asked"}}`,
			wantAction: "update",
		},
		{
			name:       "reply without revision",
			raw:        `{"reply":"The current doc already covers that.","revision":{"action":"none","content":"","reason":""}}`,
			wantAction: "none",
		},
		{
			name:       "fenced with preamble",
			raw:        "Sure!\n```json\n{\"reply\":\"done\",\"revision\":{\"action\":\"none\",\"content\":\"\",\"reason\":\"\"}}\n```",
			wantAction: "none",
		},
		{name: "empty reply text", raw: `{"reply":"","revision":{"action":"none","content":"","reason":""}}`, wantErr: true},
		{name: "garbage", raw: "no json here", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCurationReply(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Revision.Action != tt.wantAction {
				t.Errorf("action = %q, want %q", got.Revision.Action, tt.wantAction)
			}
		})
	}
}

func TestValidTargetKind(t *testing.T) {
	for _, k := range []string{TargetDocSection, TargetKBPage, TargetBlueprint} {
		if !validTargetKind(k) {
			t.Errorf("validTargetKind(%q) = false, want true", k)
		}
	}
	for _, k := range []string{"", "doc", "session", "kb"} {
		if validTargetKind(k) {
			t.Errorf("validTargetKind(%q) = true, want false", k)
		}
	}
}

func TestCurationSchemaIsStrict(t *testing.T) {
	// The schema must constrain action to none|update so workers with
	// structured output can never return a third action the apply
	// path would silently drop.
	for _, want := range []string{`"none"`, `"update"`, `"strict": true`} {
		if !strings.Contains(curationJSONSchema, want) {
			t.Errorf("curation schema missing %s", want)
		}
	}
}
