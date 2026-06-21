package memquery

import (
	"strings"
	"testing"
)

func TestKBSection(t *testing.T) {
	// H1 container wrapping focused H2 sections — mirrors the real
	// kb_integrations shape where the H1 slice is the whole page.
	doc := "# Integration Guide\n" +
		"intro mentioning auth and scopes and sessions briefly\n" +
		"## Authentication\n" +
		"send the bearer token on every request. auth auth auth.\n" +
		"## Scopes & authorization\n" +
		"the canonical scope list and enforcement. scope scope.\n" +
		"## Driving a session\n" +
		"create a session and poll for the reply.\n"

	tests := []struct {
		name        string
		query       string
		wantSection string
		notWhole    bool
	}{
		{"heading-term match wins", "how do I authenticate", "Authentication", true},
		{"body-density picks the right focused section", "canonical scope enforcement", "Scopes & authorization", true},
		{"never returns the whole-page H1 container", "auth scopes sessions", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, section := kbSection(doc, tt.query)
			if tt.notWhole && len(text) == len(doc) {
				t.Fatalf("returned the whole page (section=%q) — container leaked", section)
			}
			if tt.wantSection != "" && section != tt.wantSection {
				t.Errorf("section = %q, want %q", section, tt.wantSection)
			}
			if section == "Integration Guide" {
				t.Errorf("picked the H1 container heading")
			}
		})
	}
}

func TestKBSection_NoHeadingsReturnsWhole(t *testing.T) {
	doc := "just a flat paragraph with no headings at all.\n"
	text, section := kbSection(doc, "anything")
	if text != doc || section != "" {
		t.Errorf("flat doc: got (%q, %q), want whole content + empty section", text, section)
	}
}

func TestKBSection_AllShortTokensReturnsTeaser(t *testing.T) {
	doc := "# Title\n## First\nalpha body\n## Second\nbeta body\n"
	text, section := kbSection(doc, "is a to the") // all tokens < 4 chars
	if section != "First" {
		t.Errorf("section = %q, want First (first non-container teaser)", section)
	}
	if len(text) == len(doc) {
		t.Error("returned the whole page for a no-term query")
	}
}

func TestKBSection_SingleH1LargePageIsCapped(t *testing.T) {
	body := strings.Repeat("a line about widgets and gadgets\n", 200) // ~6.6K
	doc := "# Only Title\n" + body
	text, section := kbSection(doc, "widgets")
	if section != "" {
		t.Errorf("section = %q, want empty (page has no subsections)", section)
	}
	if len(text) >= len(doc) {
		t.Errorf("whole page not capped: len(text)=%d len(doc)=%d", len(text), len(doc))
	}
	if !strings.Contains(text, "truncated") {
		t.Error("capped fallback missing truncation marker")
	}
}

func TestKBSection_NoSignalReturnsFirstNonContainer(t *testing.T) {
	doc := "# Title\n## First\nalpha\n## Second\nbeta\n"
	text, section := kbSection(doc, "zzz-no-match")
	if section != "First" {
		t.Errorf("section = %q, want First", section)
	}
	if strings.Contains(text, "beta") {
		t.Errorf("teaser leaked a later section")
	}
}
