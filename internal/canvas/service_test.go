package canvas

import (
	"strings"
	"testing"
)

func TestFormatFeedback(t *testing.T) {
	art := Artifact{Slug: "default", Title: "Login page v2"}
	tests := []struct {
		name   string
		fb     Feedback
		wants  []string
		absent []string
	}{
		{
			name: "pin with selector and markup",
			fb: Feedback{
				Annotations: []Annotation{{
					N: 1, Kind: "pin", Note: "make it blue",
					Selector: ".hero > button.cta",
					HTML:     "<button class=\"cta\">Buy</button>",
				}},
			},
			wants: []string{
				"Login page v2",
				"1. (pin) element `.hero > button.cta`",
				"make it blue",
				"markup: <button class=\"cta\">Buy</button>",
			},
		},
		{
			name: "region uses percent coordinates",
			fb: Feedback{
				Annotations: []Annotation{{
					N: 2, Kind: "region", Note: "drop this block",
					X: 10, Y: 20, W: 30, H: 15,
				}},
			},
			wants: []string{
				"2. (region) region at ~x:10% y:20% w:30% h:15%",
				"drop this block",
			},
		},
		{
			name: "pin without selector falls back to point",
			fb: Feedback{
				Annotations: []Annotation{{N: 1, Kind: "pin", X: 42, Y: 7}},
			},
			wants:  []string{"point at ~x:42% y:7%"},
			absent: []string{"element `"},
		},
		{
			name:  "overall message only",
			fb:    Feedback{Message: "colors are too dark"},
			wants: []string{"Overall: colors are too dark"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatFeedback(art, tc.fb)
			for _, w := range tc.wants {
				if !strings.Contains(got, w) {
					t.Errorf("expected output to contain %q\n---\n%s", w, got)
				}
			}
			for _, a := range tc.absent {
				if strings.Contains(got, a) {
					t.Errorf("expected output to NOT contain %q\n---\n%s", a, got)
				}
			}
		})
	}
}

func TestFormatFeedbackFallsBackToSlug(t *testing.T) {
	art := Artifact{Slug: "checkout"}
	got := formatFeedback(art, Feedback{Message: "x"})
	if !strings.Contains(got, "\"checkout\"") {
		t.Errorf("expected slug in title position, got: %s", got)
	}
}

func TestOneLineCapsLength(t *testing.T) {
	in := strings.Repeat("a", 1000)
	got := oneLine(in)
	if len([]rune(got)) > 402 { // 400 + ellipsis rune + margin
		t.Errorf("oneLine did not cap length: got %d runes", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected truncation ellipsis, got tail: %q", got[len(got)-10:])
	}
}

func TestNormSlug(t *testing.T) {
	for in, want := range map[string]string{"": DefaultSlug, "  ": DefaultSlug, " a ": "a", "x": "x"} {
		if got := normSlug(in); got != want {
			t.Errorf("normSlug(%q) = %q, want %q", in, got, want)
		}
	}
}
