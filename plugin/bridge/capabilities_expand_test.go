package bridge

import "testing"

func TestExpandPathVars_Table(t *testing.T) {
	t.Parallel()
	ctx := PathVarCtx{
		Workspace: "/home/kev/proj",
		Home:      "/home/kev",
		DataDir:   "/var/lib/opendray/plugins/fs-readme/1.0.0/data",
		Tmp:       "/tmp",
	}
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"no vars", "/etc/passwd", "/etc/passwd"},
		{"workspace glob", "${workspace}/**", "/home/kev/proj/**"},
		{"home sshdir", "${home}/.ssh/**", "/home/kev/.ssh/**"},
		{"data files", "${dataDir}/cache/*.json", "/var/lib/opendray/plugins/fs-readme/1.0.0/data/cache/*.json"},
		{"tmp sub", "${tmp}/opendray-*.log", "/tmp/opendray-*.log"},
		{"unknown var stays literal", "${unknown}/foo", "${unknown}/foo"},
		{"multiple vars", "${home}/x/${workspace}/y", "/home/kev/x//home/kev/proj/y"},
		{"no-op short-circuit", "plain/path", "plain/path"},
		{"traversal text in workspace stays after match", "${workspace}/../../etc/passwd",
			"/home/kev/proj/../../etc/passwd"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ExpandPathVars(tc.pattern, ctx)
			if got != tc.want {
				t.Errorf("ExpandPathVars(%q)\n  got  %q\n  want %q", tc.pattern, got, tc.want)
			}
		})
	}
}

func TestExpandPathVars_EmptyCtxKeepsVarsLiteral(t *testing.T) {
	t.Parallel()
	got := ExpandPathVars("${workspace}/README.md", PathVarCtx{})
	want := "${workspace}/README.md"
	if got != want {
		t.Errorf("empty ctx should leave vars literal: got %q, want %q", got, want)
	}
}

func TestExpandPathVars_OnlyWorkspaceSet(t *testing.T) {
	t.Parallel()
	ctx := PathVarCtx{Workspace: "/w"}
	got := ExpandPathVars("${workspace}/a/${home}/b", ctx)
	want := "/w/a/${home}/b"
	if got != want {
		t.Errorf("partial ctx: got %q, want %q", got, want)
	}
}
