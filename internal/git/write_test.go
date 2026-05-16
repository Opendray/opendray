package git

import "testing"

func TestValidBranchName(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"main", true},
		{"feat/foo-bar", true},
		{"release-1.2.3", true},
		{"", false},
		{"   ", false},
		{"-rm", false},     // leading dash → would parse as flag
		{"foo bar", false}, // whitespace
		{"foo\tbar", false},
		{"foo\nbar", false},
	}
	for _, c := range cases {
		if got := validBranchName(c.in); got != c.want {
			t.Errorf("validBranchName(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValidRelativePath(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"file.go", true},
		{"sub/dir/file.go", true},
		{"", false},
		{"/abs/path", false},
		{"../escape", false},
		{"sub/../etc/passwd", false},
		{"sub/./file", true}, // "." segments are harmless
	}
	for _, c := range cases {
		if got := validRelativePath(c.in); got != c.want {
			t.Errorf("validRelativePath(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseBranchRefs(t *testing.T) {
	// Simulated for-each-ref output: local main (current) +
	// feat/x with upstream + a remote ref (origin/main) + an
	// origin/HEAD symref that should be filtered.
	raw := `main|origin/main|*
feat/x|origin/feat/x|
origin/main||
origin/feat/x||
origin/HEAD||
`
	got := parseBranchRefs(raw, "main")
	if len(got) != 4 {
		t.Fatalf("expected 4 refs (HEAD symref filtered), got %d: %+v", len(got), got)
	}
	// Find each by traits.
	var mainRef, featRef, originMain, originFeat *BranchRef
	for i := range got {
		switch {
		case got[i].Name == "main" && !got[i].IsRemote:
			mainRef = &got[i]
		case got[i].Name == "feat/x" && !got[i].IsRemote:
			featRef = &got[i]
		case got[i].Name == "main" && got[i].IsRemote && got[i].Remote == "origin":
			originMain = &got[i]
		case got[i].Name == "feat/x" && got[i].IsRemote && got[i].Remote == "origin":
			originFeat = &got[i]
		}
	}
	if mainRef == nil || !mainRef.IsCurrent {
		t.Errorf("main should be present + current, got %+v", mainRef)
	}
	if featRef == nil || featRef.IsCurrent {
		t.Errorf("feat/x should be present + NOT current, got %+v", featRef)
	}
	if featRef != nil && featRef.Upstream != "origin/feat/x" {
		t.Errorf("feat/x upstream wrong: %q", featRef.Upstream)
	}
	if originMain == nil {
		t.Error("origin/main remote ref missing")
	}
	if originFeat == nil {
		t.Error("origin/feat/x remote ref missing")
	}
}

func TestParseBranchRefs_HeadSymrefFiltered(t *testing.T) {
	// HEAD symref alone — only entry; output must be empty.
	got := parseBranchRefs("origin/HEAD||\n", "")
	if len(got) != 0 {
		t.Errorf("expected empty when only HEAD symref present, got %+v", got)
	}
}

func TestIsLikelyRemote(t *testing.T) {
	cases := map[string]bool{
		"origin":   true,
		"upstream": true,
		"fork":     true,
		"feat":     false, // local branch named "feat/..."
		"":         false,
	}
	for in, want := range cases {
		if got := isLikelyRemote(in); got != want {
			t.Errorf("isLikelyRemote(%q) = %v, want %v", in, got, want)
		}
	}
}
