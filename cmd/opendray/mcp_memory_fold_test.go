package main

import (
	"strings"
	"testing"
)

// mem builds a search-hit memory map shaped like /api/v1/memory/search returns
// (memory object with an optional metadata.merged_from audit list).
func mem(text string, mergedFrom ...string) map[string]any {
	m := map[string]any{"id": "mem_x", "text": text}
	if len(mergedFrom) > 0 {
		var arr []any
		for _, t := range mergedFrom {
			arr = append(arr, map[string]any{"text": t, "similarity": 0.9})
		}
		m["metadata"] = map[string]any{"merged_from": arr}
	}
	return m
}

func TestMergedFromTexts(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		want []string
	}{
		{"no metadata", map[string]any{"text": "a"}, nil},
		{"metadata but no merged_from", map[string]any{"text": "a", "metadata": map[string]any{"type": "fact"}}, nil},
		{"two variants", mem("canonical", "from=antigravity", "from=codex"), []string{"from=antigravity", "from=codex"}},
		{
			"skips malformed entries",
			map[string]any{"metadata": map[string]any{"merged_from": []any{
				map[string]any{"text": "keep"},
				map[string]any{"nope": 1},    // no text
				"raw-string",                 // not a map
				map[string]any{"text": "  "}, // blank
			}}},
			[]string{"keep"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mergedFromTexts(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("[%d] got %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestFoldedVariantsBlock(t *testing.T) {
	// No fold → empty suffix (unchanged output for the common case).
	if got := foldedVariantsBlock(mem("solo"), "  "); got != "" {
		t.Errorf("expected empty block for un-folded memory, got %q", got)
	}

	// Folded → surfaces count + every absorbed variant, indented, newlines
	// collapsed so each variant stays on one line.
	block := foldedVariantsBlock(mem("canonical", "port 8770", "port 9090\nextra line"), "  ")
	for _, want := range []string{"2 earlier", "port 8770", "port 9090 extra line"} {
		if !strings.Contains(block, want) {
			t.Errorf("block missing %q\n--- block ---\n%s", want, block)
		}
	}
	if strings.Contains(block, "port 9090\nextra line") {
		t.Errorf("variant newline not collapsed:\n%s", block)
	}
	if !strings.HasPrefix(block, "  ") {
		t.Errorf("block not indented with the given prefix:\n%s", block)
	}
}
