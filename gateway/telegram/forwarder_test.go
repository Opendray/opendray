package telegram

import (
	"strings"
	"testing"
)

func TestForwarder_ResetSnapshot_ClearsLast(t *testing.T) {
	f := &Forwarder{last: map[string]string{"sid": "previous turn content"}}
	f.ResetSnapshot("sid")
	if _, ok := f.last["sid"]; ok {
		t.Fatalf("ResetSnapshot did not clear last[sid]")
	}
	// Idempotent + safe on unknown ids.
	f.ResetSnapshot("does-not-exist")
}

func TestRenderTable_VerticalKeyValue(t *testing.T) {
	rows := []string{
		"| Node | IP |",
		"|------|-----|",
		"| node-a | 192.0.2.21 |",
		"| node-b | 192.0.2.22 |",
	}
	got := renderTable(rows)

	wants := []string{
		"<b>Node:</b> node-a",
		"<b>IP:</b> 192.0.2.21",
		"<b>Node:</b> node-b",
		"<b>IP:</b> 192.0.2.22",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in:\n%s", w, got)
		}
	}
	if strings.Contains(got, "---") {
		t.Errorf("separator row leaked into output:\n%s", got)
	}
}

func TestRenderTable_InlineMarkdownInCells(t *testing.T) {
	rows := []string{
		"| Key | Value |",
		"|-----|-------|",
		"| `DB_HOST` | **required** |",
	}
	got := renderTable(rows)
	if !strings.Contains(got, "<code>DB_HOST</code>") {
		t.Errorf("code span not rendered: %s", got)
	}
	if !strings.Contains(got, "<b>required</b>") {
		t.Errorf("bold not rendered: %s", got)
	}
}

func TestRenderTable_AlignmentSeparator(t *testing.T) {
	rows := []string{
		"| A | B | C |",
		"|:--|:-:|--:|",
		"| 1 | 2 | 3 |",
	}
	got := renderTable(rows)
	if !strings.Contains(got, "<b>A:</b> 1") ||
		!strings.Contains(got, "<b>B:</b> 2") ||
		!strings.Contains(got, "<b>C:</b> 3") {
		t.Errorf("alignment separator not recognized:\n%s", got)
	}
	if strings.Contains(got, ":--") || strings.Contains(got, "--:") {
		t.Errorf("alignment separator leaked:\n%s", got)
	}
}

func TestRenderTable_Malformed_FallbackToBullets(t *testing.T) {
	// No separator row → can't infer headers; fall back to bullets.
	rows := []string{
		"| a | b |",
		"| c | d |",
	}
	got := renderTable(rows)
	if !strings.Contains(got, "•") {
		t.Errorf("expected bullet fallback, got:\n%s", got)
	}
}

func TestFormatForTelegram_TableConversion(t *testing.T) {
	md := "Before table.\n\n| Node | IP |\n|------|-----|\n| node-a | 192.0.2.21 |\n\nAfter table."
	got := formatForTelegram(md)
	if !strings.Contains(got, "<b>Node:</b> node-a") {
		t.Errorf("table not converted in formatForTelegram output:\n%s", got)
	}
	if strings.Contains(got, "|------|") {
		t.Errorf("raw separator leaked into output:\n%s", got)
	}
	if !strings.Contains(got, "Before table.") || !strings.Contains(got, "After table.") {
		t.Errorf("surrounding content lost:\n%s", got)
	}
}
