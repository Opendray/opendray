package main

import "testing"

func TestMemoryResultText(t *testing.T) {
	tests := []struct {
		name     string
		result   any
		wantText string
		wantErr  bool
	}{
		{
			name: "single text content block ([]map form, as callX returns)",
			result: map[string]any{
				"content": []map[string]any{{"type": "text", "text": "hello"}},
			},
			wantText: "hello",
		},
		{
			name: "multiple blocks joined by newline",
			result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": "line1"},
					{"type": "text", "text": "line2"},
				},
			},
			wantText: "line1\nline2",
		},
		{
			name: "isError flagged ([]any form, as JSON round-trip yields)",
			result: map[string]any{
				"content": []any{map[string]any{"type": "text", "text": "tool error: boom"}},
				"isError": true,
			},
			wantText: "tool error: boom",
			wantErr:  true,
		},
		{
			name:     "non-map result falls back to JSON",
			result:   []string{"a", "b"},
			wantText: `["a","b"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, isErr := memoryResultText(tt.result)
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if isErr != tt.wantErr {
				t.Errorf("isErr = %v, want %v", isErr, tt.wantErr)
			}
		})
	}
}
