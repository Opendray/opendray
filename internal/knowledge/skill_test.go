package knowledge

import (
	"strings"
	"testing"
)

func TestSkillSlug(t *testing.T) {
	cases := map[string]string{
		"Deploy a Go service to Proxmox LXC": "deploy-a-go-service-to-proxmox-lxc",
		"  Trim & punctuate!!  ":             "trim-punctuate",
		"":                                   "skill",
		"###":                                "skill",
	}
	for in, want := range cases {
		if got := skillSlug(in); got != want {
			t.Errorf("skillSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderSkillMarkdown(t *testing.T) {
	md := renderSkillMarkdown("deploy-go", `He said "hi"`+"\nsecond line", "## Steps\n1. do it")
	for _, want := range []string{"---", "name: deploy-go", `description: "He said 'hi' second line"`, "# deploy-go", "## Steps", "1. do it"} {
		if !strings.Contains(md, want) {
			t.Fatalf("SKILL.md missing %q:\n%s", want, md)
		}
	}
}

func TestSkillDescription(t *testing.T) {
	if got := skillDescription("T", "**Applies when:** shipping a Go API\n## Steps"); got != "shipping a Go API" {
		t.Errorf("got %q", got)
	}
	if got := skillDescription("Fallback", ""); got != "Fallback" {
		t.Errorf("got %q", got)
	}
}
