package canvas

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// The oklch expectations below are not hand-computed: they are what WebKit and
// Chromium actually paint for these exact strings, read back off a 1x1 canvas.
// The gateway now draws the panels' swatches, so "matches the browser" is the
// property that matters — if this drifts, a swatch stops looking like the
// colour the canvas renders.
func TestResolvedHexMatchesBrowser(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		// opendray's own theme, light then dark.
		{"ink light", "oklch(0.20 0.012 270)", "#14161c"},
		{"ink dark", "oklch(0.96 0.005 270)", "#f0f2f5"},
		{"accent light", "oklch(0.65 0.20 35)", "#f0532b"},
		{"accent dark", "oklch(0.72 0.18 35)", "#ff7350"},
		{"background light", "oklch(0.99 0.004 270)", "#fbfcff"},
		{"background dark", "oklch(0.13 0.012 270)", "#06070c"},
		{"surface light", "oklch(0.97 0.004 270)", "#f4f5f8"},
		{"muted light", "oklch(0.46 0.012 270)", "#55585f"},
		{"border light", "oklch(0.88 0.008 270)", "#d5d7dd"},
		{"destructive", "oklch(0.55 0.22 25)", "#d40924"},
		{"state running", "oklch(0.62 0.16 145)", "#399e43"},
		{"state idle", "oklch(0.70 0.18 90)", "#ca9600"},
		// Alpha is accepted and dropped — the swatch shows the colour itself.
		{"alpha dropped", "oklch(0.65 0.20 35 / 0.4)", "#f0532b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvedHex(tt.value)
			if !hexNear(got, tt.want, 1) {
				t.Errorf("ResolvedHex(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolvedHexNotations(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"hex 6", "#5b9eff", "#5b9eff"},
		{"hex 3", "#abc", "#aabbcc"},
		{"hex 8 drops alpha", "#5b9eff80", "#5b9eff"},
		{"hex uppercase", "#5B9EFF", "#5b9eff"},
		{"rgb legacy", "rgb(12, 34, 56)", "#0c2238"},
		{"rgb modern", "rgb(12 34 56)", "#0c2238"},
		{"rgba with alpha", "rgba(12, 34, 56, 0.5)", "#0c2238"},
		{"rgb slash alpha", "rgb(12 34 56 / 50%)", "#0c2238"},
		{"hsl", "hsl(210 90% 60%)", "#3d99f5"},
		{"hsl legacy commas", "hsl(210, 90%, 60%)", "#3d99f5"},
		{"hsl grey", "hsl(0 0% 50%)", "#808080"},
		{"oklab", "oklab(0.65 0.15 0.09)", ""}, // just needs to resolve
		{"whitespace tolerated", "  oklch(0.65 0.20 35)  ", "#f0532b"},
		{"uppercase function", "OKLCH(0.65 0.20 35)", "#f0532b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvedHex(tt.value)
			if got == "" {
				t.Fatalf("ResolvedHex(%q) = \"\", want a colour", tt.value)
			}
			if tt.want != "" && !hexNear(got, tt.want, 1) {
				t.Errorf("ResolvedHex(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

// Non-colours must resolve to "" rather than to some default, or the radius and
// spacing rows would sprout swatches and an unreadable value would silently
// render as black.
func TestResolvedHexRejectsNonColours(t *testing.T) {
	for _, v := range []string{
		"", "   ", "0.375rem", "12px", "garbage",
		"rebeccapurple", // named colours are deliberately unsupported
		"#12", "#1234567", "#zzzzzz",
		"oklch(0.65 0.20)", "rgb(1 2)", "hsl()",
		"oklch(a b c)", "var(--primary)",
		`0 1px 2px oklch(0 0 0 / 0.06)`, // a shadow, not a colour
	} {
		if got := ResolvedHex(v); got != "" {
			t.Errorf("ResolvedHex(%q) = %q, want \"\"", v, got)
		}
	}
}

// The picker only speaks hex; re-encoding is what keeps it from converting a
// project's theme one field at a time.
func TestReencodeKeepsNotation(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		like       string
		wantPrefix string
	}{
		{"hex into oklch", "#f0532b", "oklch(0.20 0.012 270)", "oklch("},
		{"hex into hsl", "#3d99f5", "hsl(210 90% 60%)", "hsl("},
		{"hex into rgb", "#0c2238", "rgb(12, 34, 56)", "rgb("},
		{"hex into oklab", "#f0532b", "oklab(0.65 0.15 0.09)", "oklab("},
		{"hex stays hex", "#f0532b", "#5b9eff", "#f0532b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reencode(tt.value, tt.like)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Fatalf("reencode(%q, %q) = %q, want prefix %q", tt.value, tt.like, got, tt.wantPrefix)
			}
			// Whatever notation comes out must still be the same colour.
			if !hexNear(ResolvedHex(got), tt.value, 1) {
				t.Errorf("reencode(%q, %q) = %q, which resolves to %q", tt.value, tt.like, got, ResolvedHex(got))
			}
		})
	}
}

// A value we can't read, or a reference we can't read, must pass through
// untouched rather than being replaced by a guess.
func TestReencodePassesThroughUnreadable(t *testing.T) {
	for _, tt := range []struct{ value, like string }{
		{"garbage", "oklch(0.2 0.012 270)"},
		{"#f0532b", "0.375rem"},
		{"#f0532b", ""},
	} {
		if got := reencode(tt.value, tt.like); got != tt.value {
			t.Errorf("reencode(%q, %q) = %q, want it unchanged", tt.value, tt.like, got)
		}
	}
}

// The achromatic guard exists to catch a token-mapping mistake, so the numbers
// it keys off have to separate a genuinely restrained neutral scale from a
// brand colour. These are the real values on both sides of that line.
func TestChromaSeparatesNeutralsFromBrandColours(t *testing.T) {
	neutrals := []string{
		"oklch(0.20 0.012 270)", // opendray ink
		"oklch(0.99 0.004 270)", // background
		"oklch(0.46 0.012 270)", // muted text
		"oklch(0.88 0.008 270)", // border
		"#ffffff", "#000000", "#71717a", // zinc-500
		"#64748b", // slate-500, the most tinted neutral in common use
	}
	brands := []string{
		"oklch(0.65 0.20 35)", // opendray accent
		"#4f46e5",             // indigo-600
		"#0284c7",             // sky-600
		"#059669",             // emerald-600
		"#d97706",             // amber-600
		"#e11d48",             // rose-600
		"#7c3aed",             // violet-600
		"#f97316",             // orange-500
	}
	for _, v := range neutrals {
		c, ok := chromaOf(v)
		if !ok {
			t.Fatalf("chromaOf(%q) failed", v)
		}
		if c >= chromaticThreshold {
			t.Errorf("chromaOf(%q) = %.4f, expected below the %.2f threshold", v, c, chromaticThreshold)
		}
	}
	for _, v := range brands {
		c, ok := chromaOf(v)
		if !ok {
			t.Fatalf("chromaOf(%q) failed", v)
		}
		if c < chromaticThreshold {
			t.Errorf("chromaOf(%q) = %.4f, expected at or above the %.2f threshold", v, c, chromaticThreshold)
		}
	}
}

// hexNear compares two #rrggbb strings allowing a per-channel tolerance, so a
// last-bit rounding difference against the browser isn't a failure.
func hexNear(got, want string, tol int) bool {
	if len(got) != 7 || len(want) != 7 {
		return false
	}
	for i := 1; i < 7; i += 2 {
		a, err1 := strconv.ParseInt(got[i:i+2], 16, 32)
		b, err2 := strconv.ParseInt(want[i:i+2], 16, 32)
		if err1 != nil || err2 != nil {
			return false
		}
		if math.Abs(float64(a-b)) > float64(tol) {
			return false
		}
	}
	return true
}
