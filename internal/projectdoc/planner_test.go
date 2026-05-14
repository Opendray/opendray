package projectdoc

import (
	"errors"
	"strings"
	"testing"
)

func TestParseDriftResponse(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    DriftOutput
		wantErr error
	}{
		{
			name: "clean json schema response — propose",
			raw:  `{"should_propose": true, "new_plan": "## Phase 1\n- done", "reason": "M5 landed"}`,
			want: DriftOutput{ShouldPropose: true, NewPlan: "## Phase 1\n- done", Reason: "M5 landed"},
		},
		{
			name: "clean json schema response — no change",
			raw:  `{"should_propose": false, "new_plan": "", "reason": ""}`,
			want: DriftOutput{ShouldPropose: false},
		},
		{
			name: "fenced code block ```json ... ```",
			raw:  "```json\n{\"should_propose\": true, \"new_plan\": \"x\", \"reason\": \"y\"}\n```",
			want: DriftOutput{ShouldPropose: true, NewPlan: "x", Reason: "y"},
		},
		{
			name: "fenced code block without language hint",
			raw:  "```\n{\"should_propose\": false, \"new_plan\": \"\", \"reason\": \"nothing changed\"}\n```",
			want: DriftOutput{ShouldPropose: false, Reason: "nothing changed"},
		},
		{
			name: "leading prose then json",
			raw:  "Here is my answer:\n{\"should_propose\": true, \"new_plan\": \"plan body\", \"reason\": \"ok\"}",
			want: DriftOutput{ShouldPropose: true, NewPlan: "plan body", Reason: "ok"},
		},
		{
			name: "trailing prose after json",
			raw:  `{"should_propose": true, "new_plan": "a", "reason": "b"} -- end`,
			want: DriftOutput{ShouldPropose: true, NewPlan: "a", Reason: "b"},
		},
		{
			name: "whitespace trimmed from fields",
			raw:  `{"should_propose": true, "new_plan": "  body  ", "reason": "  why  "}`,
			want: DriftOutput{ShouldPropose: true, NewPlan: "body", Reason: "why"},
		},
		{
			name:    "empty response is unparseable",
			raw:     "",
			wantErr: ErrDetectorParse,
		},
		{
			name:    "whitespace only is unparseable",
			raw:     "   \n  ",
			wantErr: ErrDetectorParse,
		},
		{
			name:    "non-json garbage",
			raw:     "I don't know what to do here",
			wantErr: ErrDetectorParse,
		},
		{
			name:    "should_propose=true but empty new_plan is rejected",
			raw:     `{"should_propose": true, "new_plan": "", "reason": "vague"}`,
			wantErr: ErrDetectorParse,
		},
		{
			name:    "should_propose=true with whitespace new_plan is rejected",
			raw:     `{"should_propose": true, "new_plan": "   \n  ", "reason": "vague"}`,
			wantErr: ErrDetectorParse,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDriftResponse(tc.raw)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got err=%v out=%+v", tc.wantErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ShouldPropose != tc.want.ShouldPropose {
				t.Errorf("ShouldPropose: got %v, want %v", got.ShouldPropose, tc.want.ShouldPropose)
			}
			if got.NewPlan != tc.want.NewPlan {
				t.Errorf("NewPlan: got %q, want %q", got.NewPlan, tc.want.NewPlan)
			}
			if got.Reason != tc.want.Reason {
				t.Errorf("Reason: got %q, want %q", got.Reason, tc.want.Reason)
			}
		})
	}
}

func TestStripJSONFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no fence returns empty", "just text", ""},
		{"json language hint", "```json\n{\"x\":1}\n```", `{"x":1}`},
		{"no language hint", "```\n{\"x\":1}\n```", `{"x":1}`},
		{"unterminated fence returns tail", "```json\n{\"x\":1}", `{"x":1}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripJSONFence(tc.in)
			if strings.TrimSpace(got) != strings.TrimSpace(tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
