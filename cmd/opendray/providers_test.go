package main

import (
	"reflect"
	"testing"
)

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"claude", []string{"claude"}},
		{"claude,gemini", []string{"claude", "gemini"}},
		{" claude , gemini , codex ", []string{"claude", "gemini", "codex"}},
		{",,claude,,", []string{"claude"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := splitCSV(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitCSV(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	cases := []struct {
		haystack []string
		needle   string
		want     bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "z", false},
		{nil, "anything", false},
		{[]string{}, "anything", false},
	}
	for _, tc := range cases {
		t.Run(tc.needle, func(t *testing.T) {
			if got := contains(tc.haystack, tc.needle); got != tc.want {
				t.Errorf("contains(%#v, %q) = %v, want %v", tc.haystack, tc.needle, got, tc.want)
			}
		})
	}
}

func TestProviderCatalogShape(t *testing.T) {
	// Defensive: rest of the wizard / CI assumes these three are
	// the catalog. If anyone adds a new provider, update this list.
	expected := map[string]string{
		"claude": "@anthropic-ai/claude-code",
		"gemini": "@google/gemini-cli",
		"codex":  "@openai/codex",
	}
	if len(providerCatalog) != len(expected) {
		t.Fatalf("providerCatalog has %d entries, expected %d", len(providerCatalog), len(expected))
	}
	for _, p := range providerCatalog {
		want, ok := expected[p.Bin]
		if !ok {
			t.Errorf("unexpected provider bin %q in catalog", p.Bin)
			continue
		}
		if p.NpmPkg != want {
			t.Errorf("provider %q npm pkg = %q, want %q", p.Bin, p.NpmPkg, want)
		}
		if p.Display == "" {
			t.Errorf("provider %q missing Display name", p.Bin)
		}
	}
}
