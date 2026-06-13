package catalog

import "testing"

// TestLoadBuiltin_AllPresent guards the catalog's builtin set: the M2
// scope (claude/codex/gemini/shell, ADR 0004) plus antigravity (agy),
// added as a gemini-lineage cloud CLI. If you intentionally add another
// manifest, update ADR 0004 and this test together.
func TestLoadBuiltin_AllPresent(t *testing.T) {
	manifests, hashes, err := LoadBuiltin()
	if err != nil {
		t.Fatalf("LoadBuiltin: %v", err)
	}
	want := map[string]bool{
		"claude": true, "codex": true, "gemini": true,
		"shell": true, "antigravity": true,
	}
	if len(manifests) != len(want) {
		t.Errorf("got %d manifests, want %d", len(manifests), len(want))
	}
	for id := range want {
		if _, ok := manifests[id]; !ok {
			t.Errorf("missing manifest: %s", id)
		}
		if hashes[id] == "" {
			t.Errorf("missing hash for %s", id)
		}
	}
	for id := range manifests {
		if !want[id] {
			t.Errorf("unexpected manifest: %s (update ADR 0004)", id)
		}
	}
}

func TestLoadBuiltin_ExecutablesNonEmpty(t *testing.T) {
	manifests, _, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	for id, m := range manifests {
		if m.Executable == "" {
			t.Errorf("%s: executable is empty", id)
		}
		if m.DisplayName == "" {
			t.Errorf("%s: displayName is empty", id)
		}
		if m.Kind != "cli" && m.Kind != "shell" {
			t.Errorf("%s: kind=%q (want cli|shell)", id, m.Kind)
		}
	}
}

func TestLoadBuiltin_HashStable(t *testing.T) {
	_, h1, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	_, h2, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	for id := range h1 {
		if h1[id] != h2[id] {
			t.Errorf("hash for %s changed across loads: %s vs %s", id, h1[id], h2[id])
		}
	}
}

func TestSortedIDs(t *testing.T) {
	manifests, _, _ := LoadBuiltin()
	ids := SortedIDs(manifests)
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			t.Errorf("not sorted at %d: %s >= %s", i, ids[i-1], ids[i])
		}
	}
}
