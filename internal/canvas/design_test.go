package canvas

import "testing"

// The swatch map is what both panels draw from, so what belongs in it — and
// what must stay out of it — is the contract.
func TestViewResolvesOnlyColourTokens(t *testing.T) {
	d := DesignSystem{Tokens: map[string]string{
		"primary":      "oklch(0.65 0.20 35)",
		"text":         "#14161c",
		"accent":       "hsl(210 90% 60%)", // an operator-added extra
		"radius":       "0.375rem",
		"baseSize":     "12px",
		"font":         `"Inter Variable", "Inter", sans-serif`,
		"shadow":       "0 1px 2px oklch(0 0 0 / 0.06)",
		"stateRunning": "oklch(0.62 0.16 145)",
	}}
	got := d.View().TokensResolved

	// Extras get a swatch on the same terms as documented tokens.
	for _, k := range []string{"primary", "text", "accent", "stateRunning"} {
		if got[k] == "" {
			t.Errorf("token %q should have resolved to a colour", k)
		}
	}
	// Anything that isn't a colour must be absent, or the panel draws a swatch
	// next to a radius and a shadow renders as black.
	for _, k := range []string{"radius", "baseSize", "font", "shadow"} {
		if v, ok := got[k]; ok {
			t.Errorf("token %q is not a colour but resolved to %q", k, v)
		}
	}
}

func TestAchromaticWarning(t *testing.T) {
	tests := []struct {
		name   string
		tokens map[string]string
		want   bool
	}{
		{
			// The exact mistake this exists to catch: shadcn's ink copied into
			// `primary`, so the whole palette is opendray's neutral ramp.
			name: "shadcn ink mapped by name",
			tokens: map[string]string{
				"primary":    "oklch(0.20 0.012 270)",
				"secondary":  "oklch(0.95 0.005 270)",
				"background": "oklch(0.99 0.004 270)",
				"surface":    "oklch(0.97 0.004 270)",
				"text":       "oklch(0.20 0.012 270)",
				"muted":      "oklch(0.46 0.012 270)",
				"border":     "oklch(0.88 0.008 270)",
			},
			want: true,
		},
		{
			// Same palette, mapped correctly — one brand colour is enough.
			name: "brand colour present",
			tokens: map[string]string{
				"primary":    "oklch(0.65 0.20 35)",
				"background": "oklch(0.99 0.004 270)",
				"surface":    "oklch(0.97 0.004 270)",
				"text":       "oklch(0.20 0.012 270)",
				"border":     "oklch(0.88 0.008 270)",
			},
			want: false,
		},
		{
			// A half-filled system is someone mid-edit, not a mistake.
			name:   "too few colours to judge",
			tokens: map[string]string{"text": "#000000", "background": "#ffffff"},
			want:   false,
		},
		{
			name:   "no colours at all",
			tokens: map[string]string{"radius": "8px", "baseSize": "14px", "font": "Inter"},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := makeAchromaticWarning(tt.tokens)
			if got != tt.want {
				t.Errorf("makeAchromaticWarning() fired = %v, want %v", got, tt.want)
			}
		})
	}
}

// A picker edit must change the colour without changing how the palette is
// written; everything it didn't touch must come through byte-identical.
func TestKeepNotation(t *testing.T) {
	prev := map[string]string{
		"primary":    "oklch(0.20 0.012 270)",
		"background": "oklch(0.99 0.004 270)",
		"surface":    "#f4f5f8",
		"radius":     "0.375rem",
	}
	in := map[string]string{
		"primary":    "#f0532b",             // picked in the OS colour chooser
		"background": "oklch(0.99 0.004 0)", // typed out by hand
		"surface":    "#ffffff",             // picked, but this token was hex already
		"radius":     "0.5rem",
		"accent":     "#00ff00", // brand new token, nothing to preserve
	}
	got := keepNotation(in, prev)

	if n := notationOf(got["primary"]); n != notationOKLCH {
		t.Errorf("picked colour should have been written back as oklch, got %q (%s)", got["primary"], n)
	}
	if ResolvedHex(got["primary"]) != "#f0532b" {
		t.Errorf("re-encoding changed the colour: %q resolves to %q", got["primary"], ResolvedHex(got["primary"]))
	}
	for _, k := range []string{"background", "surface", "radius", "accent"} {
		if got[k] != in[k] {
			t.Errorf("token %q should have passed through as %q, got %q", k, in[k], got[k])
		}
	}
}
