package session

import (
	"strings"
	"testing"
)

func TestFilterClaudeChrome_DropsBoilerplate(t *testing.T) {
	in := strings.Join([]string{
		"之: 一个用宪法级规范严格约束的、纯 Apple 技术栈的承包商业务管理 App。",
		"________________________",
		"",
		"* Brewed for 15s",
		"________________________________",
		"________________________________",
		"________________________________",
		"________________________",
		"›",
		"________________________________",
		"________________________________",
		"________________________________",
		"________________________________",
		"  ▶▶ bypass permissions on (shift+tab to cycle)",
	}, "\n")
	out := FilterClaudeChrome(in)

	if !strings.Contains(out, "Apple") || !strings.Contains(out, "App") {
		t.Errorf("user content lost:\n%s", out)
	}
	for _, gone := range []string{
		"Brewed for 15s",
		"bypass permissions",
		"shift+tab",
		"›",
		"_______",
	} {
		if strings.Contains(out, gone) {
			t.Errorf("filter should have dropped %q in:\n%s", gone, out)
		}
	}
}

func TestFilterClaudeChrome_KeepsAssistantBody(t *testing.T) {
	in := strings.Join([]string{
		"⏺ Got it — let's design the API.",
		"",
		"1. POST /api/v1/sessions",
		"2. GET  /api/v1/sessions/{id}",
		"",
		"❯ Continue?",
		"________________________",
		"  ✢ Cultivating · ↓ 1.2k tokens · esc to interrupt",
	}, "\n")
	out := FilterClaudeChrome(in)
	for _, want := range []string{
		"Got it",
		"POST /api/v1/sessions",
		"GET",
		"Continue?",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("kept content missing %q in:\n%s", want, out)
		}
	}
	for _, gone := range []string{
		"Cultivating",
		"tokens",
		"esc to interrupt",
		"⏺",
		"❯",
		"_______",
	} {
		if strings.Contains(out, gone) {
			t.Errorf("filter should have dropped %q in:\n%s", gone, out)
		}
	}
}

func TestIsSeparatorLine(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"________________", true},
		{"---- ---- ----", true},
		{"= = = = = =", true},
		{"x x x", false}, // contains letters
		{"--", false},    // too short
		{"hello", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isSeparatorLine(c.in); got != c.want {
			t.Errorf("isSeparatorLine(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestHasReadableContent(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"hello", true},
		{"测试", true},
		{"日本語", true},
		{"한글", true},
		{"123", true},
		{"·····", false},
		{"▶▶▶", false},
		{"   ", false},
	}
	for _, c := range cases {
		if got := hasReadableContent(c.in); got != c.want {
			t.Errorf("hasReadableContent(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
