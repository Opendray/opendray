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
	// Format: <short>|<full>|<upstream>|<head_marker>.
	// Simulated for-each-ref output: local main (current) +
	// feat/x with upstream + remote refs (origin/main,
	// origin/feat/x) + an origin/HEAD symref that should be
	// filtered + a bare "origin" remote symref that some setups
	// produce (also filtered).
	raw := `main|refs/heads/main|origin/main|*
feat/x|refs/heads/feat/x|origin/feat/x|
origin/main|refs/remotes/origin/main||
origin/feat/x|refs/remotes/origin/feat/x||
origin/HEAD|refs/remotes/origin/HEAD||
origin|refs/remotes/origin||
`
	got := parseBranchRefs(raw, "main")
	if len(got) != 4 {
		t.Fatalf("expected 4 refs (HEAD + bare origin filtered), got %d: %+v", len(got), got)
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

func TestParseBranchRefs_FiltersBareRemoteSymref(t *testing.T) {
	// A bare "refs/remotes/origin" with no branch suffix should
	// be dropped — it's a remote symref, not a switchable branch.
	// This was the bug: it was rendering as a local branch
	// called "origin" because the short form has no slash.
	raw := `origin|refs/remotes/origin||
`
	got := parseBranchRefs(raw, "")
	if len(got) != 0 {
		t.Errorf("expected empty (bare origin symref filtered), got %+v", got)
	}
}

func TestParseBranchRefs_HeadSymrefFiltered(t *testing.T) {
	raw := `origin/HEAD|refs/remotes/origin/HEAD||
`
	got := parseBranchRefs(raw, "")
	if len(got) != 0 {
		t.Errorf("expected empty when only HEAD symref present, got %+v", got)
	}
}

func TestParseBranchRefs_LocalNamedOrigin(t *testing.T) {
	// An operator can (regrettably) name a local branch "origin".
	// `refname:full` disambiguates: refs/heads/origin is local.
	raw := `origin|refs/heads/origin||
`
	got := parseBranchRefs(raw, "")
	if len(got) != 1 || got[0].IsRemote {
		t.Errorf("local branch named 'origin' misclassified: %+v", got)
	}
	if got[0].Name != "origin" {
		t.Errorf("name wrong: %q", got[0].Name)
	}
}
