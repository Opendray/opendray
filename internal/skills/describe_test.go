package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDescribeBuiltinIsBodyOnly(t *testing.T) {
	s := Skill{ID: "x", Body: "---\nname: x\n---\nbody\n"}
	if got := Describe(s); got != s.Body {
		t.Fatalf("built-in describe must be the body verbatim, got %q", got)
	}
}

func TestDescribeVaultAppendsDirectory(t *testing.T) {
	s := Skill{ID: "x", Body: "body without newline", SourcePath: "/vault/skills/x"}
	got := Describe(s)
	if !strings.HasPrefix(got, "body without newline\n") {
		t.Fatalf("body must come first, unchanged: %q", got)
	}
	if !strings.Contains(got, "Skill directory: /vault/skills/x\n") {
		t.Fatalf("missing skill directory line: %q", got)
	}
}

func TestDescribeVaultSkillFromLoader(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "demo", "---\nname: demo\ndescription: d\n---\nsee references/a.md\n")
	s, err := NewLoader(root).Get("demo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(Describe(s), "Skill directory: "+root+"/demo\n") {
		t.Fatalf("loader skill must report its vault directory: %q", Describe(s))
	}
}

func writeSkill(t *testing.T, root, id, body string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
