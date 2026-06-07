package knowledge

import (
	"strings"
	"testing"
)

func TestParsePlaybooks(t *testing.T) {
	raw := "```json\n{\"playbooks\":[{\"title\":\"Deploy Go to Proxmox\",\"applies_when\":\"shipping a Go API\",\"steps\":[\"read infra registry\",\"create LXC\"],\"pitfalls\":[\"TCC denial\"]}]}\n```"
	got := parsePlaybooks(raw)
	if len(got) != 1 {
		t.Fatalf("got %d playbooks, want 1", len(got))
	}
	if got[0].Title != "Deploy Go to Proxmox" || len(got[0].Steps) != 2 || len(got[0].Pitfalls) != 1 {
		t.Fatalf("unexpected playbook: %+v", got[0])
	}
	if parsePlaybooks("not json") != nil {
		t.Fatal("garbage should parse to nil")
	}
	if len(parsePlaybooks(`{"playbooks":[]}`)) != 0 {
		t.Fatal("empty list should be empty")
	}
}

func TestRenderPlaybookBody(t *testing.T) {
	body := renderPlaybookBody(draftPlaybook{
		AppliesWhen: "shipping a Go API",
		Steps:       []string{"read infra registry", "create LXC"},
		Pitfalls:    []string{"TCC denial"},
	})
	for _, want := range []string{"Applies when:", "## Steps", "1. read infra registry", "2. create LXC", "## Pitfalls", "- TCC denial"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}
