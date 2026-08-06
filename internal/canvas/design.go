package canvas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// designBlockRE matches a previously injected token block so a re-render
// replaces it instead of stacking another one.
var designBlockRE = regexp.MustCompile(`(?is)<style id="od-design-system">.*?</style>`)

// The design system is the Canvas's answer to "why does every render look
// different?". A project gets one: a small set of design TOKENS plus free-text
// NOTES. The gateway then makes it binding in two ways at once —
//
//	1. every canvas request/feedback prompt carries it, so the agent never has
//	   to remember to look it up;
//	2. the tokens are injected into the rendered document as CSS variables, and
//	   the agent is told to use those variables instead of inventing values.
//
// Prompt alone would leave a model free to drift; variables alone would not
// reach an agent that writes its own colours. Together they pin the look down.

// DesignSystem is a project's canvas styling contract. Colour tokens come in
// two sets because the canvas preview follows the operator's theme — a single
// fixed palette looks wrong the moment they switch. Keys absent from TokensDark
// fall back to their light value, so a one-theme project sets nothing there.
type DesignSystem struct {
	Cwd        string            `json:"cwd"`
	Tokens     map[string]string `json:"tokens"`
	TokensDark map[string]string `json:"tokens_dark"`
	Notes      string            `json:"notes"`
}

// IsEmpty reports whether there is nothing worth injecting.
func (d DesignSystem) IsEmpty() bool {
	return len(d.Tokens) == 0 && len(d.TokensDark) == 0 && strings.TrimSpace(d.Notes) == ""
}

// DesignWarning is advice about a stored system, never a reason to reject one.
// Code is what a panel localises on; Message is the English fallback and is
// what an agent reads back through the canvas_design tool.
type DesignWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WarningAchromaticPalette fires when a system has colour tokens but none of
// them is actually a colour.
const WarningAchromaticPalette = "achromatic_palette"

// DesignSystemView is a design system plus everything computed from it. The
// stored shape stays exactly what the operator or the agent wrote; the derived
// parts ride alongside so neither panel has to understand CSS colours.
type DesignSystemView struct {
	DesignSystem
	// TokensResolved maps each token that IS a colour to its #rrggbb, for the
	// swatches. Tokens that aren't colours (radius, font, the shadow triple)
	// are simply absent, which is how a panel knows not to draw one.
	TokensResolved     map[string]string `json:"tokens_resolved,omitempty"`
	TokensDarkResolved map[string]string `json:"tokens_dark_resolved,omitempty"`
	Warnings           []DesignWarning   `json:"warnings,omitempty"`
}

// View decorates a stored system with resolved swatch colours and warnings.
func (d DesignSystem) View() DesignSystemView {
	return DesignSystemView{
		DesignSystem:       d,
		TokensResolved:     resolveSwatches(d.Tokens),
		TokensDarkResolved: resolveSwatches(d.TokensDark),
		Warnings:           d.warnings(),
	}
}

// resolveSwatches renders every token that is a colour as #rrggbb. It runs over
// ALL tokens, not just the documented ones, so an operator-added `accent` gets
// a swatch on the same terms as `primary`.
func resolveSwatches(tokens map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range tokens {
		if hex := ResolvedHex(v); hex != "" {
			out[k] = hex
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (d DesignSystem) warnings() []DesignWarning {
	var out []DesignWarning
	if w, ok := makeAchromaticWarning(d.Tokens); ok {
		out = append(out, w)
	}
	return out
}

// makeAchromaticWarning catches the mapping mistake described on designTokens:
// a palette where every colour resolves to a grey. A restrained product still
// has one colour it paints its main action with, so a whole system without one
// is far more likely to be `--primary` copied off a shadcn theme than a
// deliberate choice. It is advisory — a genuinely monochrome design is legal,
// and the operator just ignores it.
func makeAchromaticWarning(tokens map[string]string) (DesignWarning, bool) {
	// Below a few colours this is a half-filled system, not a mapping mistake.
	const minColours = 3
	seen := 0
	for _, v := range tokens {
		c, ok := chromaOf(v)
		if !ok {
			continue
		}
		seen++
		if c >= chromaticThreshold {
			return DesignWarning{}, false
		}
	}
	if seen < minColours {
		return DesignWarning{}, false
	}
	return DesignWarning{
		Code: WarningAchromaticPalette,
		Message: "Every colour in this system resolves to a grey, so canvases built from it will have no brand colour. " +
			"If this project uses shadcn/ui or a Tailwind template, `--primary` there is an ink (near-black or near-white), not a brand colour — " +
			"take the brand hue from `--accent` (or whatever the project paints its main action with) and put the ink in `text`. " +
			"Ignore this if the design really is monochrome.",
	}, true
}

// themedTokens are the tokens that differ between light and dark. The rest
// (type, radius, spacing) are theme-independent, so a dark override for them
// is ignored rather than silently doubling the vocabulary.
var themedTokens = map[string]bool{
	"primary": true, "secondary": true, "background": true, "surface": true,
	"text": true, "muted": true, "border": true, "shadow": true,
}

// designTokens are the tokens the Canvas understands. Anything else the
// operator adds is kept and passed through, but these are the ones with a
// documented CSS variable and a place in the prompt. Order matters: it is the
// order the operator and the agent read them in.
var designTokens = []struct{ Key, Label string }{
	// `primary` is the project's BRAND colour — the one a main action is
	// painted with. Saying so is load-bearing, because the obvious way to fill
	// this in is to copy a CSS variable of the same name, and in shadcn/ui and
	// the Tailwind templates built on it `--primary` is a near-black or
	// near-white INK, not a brand colour at all: the brand hue lives in
	// `--accent`. Name-matching a shadcn project therefore produces a palette
	// with no colour in it, which is what makeAchromaticWarning catches.
	{"primary", "brand colour — what a main action is painted with (NOT shadcn's ink `--primary`)"},
	{"secondary", "secondary colour"},
	{"background", "page background"},
	{"surface", "card / surface background"},
	{"text", "body text colour"},
	{"muted", "muted / secondary text colour"},
	{"border", "border colour"},
	{"font", "body font family"},
	{"headingFont", "heading font family"},
	{"baseSize", "base font size"},
	{"radius", "corner radius"},
	{"spacing", "spacing base unit"},
	{"shadow", "elevation / shadow"},
}

func knownToken(key string) bool {
	for _, t := range designTokens {
		if t.Key == key {
			return true
		}
	}
	return false
}

// cssVarName maps a token key to its CSS custom property, e.g.
// "headingFont" -> "--od-heading-font". Agents are told to use these.
func cssVarName(key string) string {
	var b strings.Builder
	b.WriteString("--od-")
	for i, r := range key {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r - 'A' + 'a')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// sortedTokens returns the tokens in documented order, with any operator-added
// extras after them, so prompts and CSS are stable between renders.
func (d DesignSystem) sortedTokens() [][2]string {
	out := make([][2]string, 0, len(d.Tokens))
	for _, t := range designTokens {
		if v := strings.TrimSpace(d.Tokens[t.Key]); v != "" {
			out = append(out, [2]string{t.Key, v})
		}
	}
	extra := make([]string, 0)
	for k, v := range d.Tokens {
		if !knownToken(k) && strings.TrimSpace(v) != "" {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		out = append(out, [2]string{k, strings.TrimSpace(d.Tokens[k])})
	}
	return out
}

// StyleBlock renders the tokens as CSS custom properties to inject into a
// canvas document: the light set on :root, the dark overrides behind
// prefers-color-scheme, so one canvas serves both themes. Empty when there are
// no tokens.
func (d DesignSystem) StyleBlock() string {
	toks := d.sortedTokens()
	if len(toks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<style id=\"od-design-system\">:root{color-scheme:light dark;")
	for _, kv := range toks {
		fmt.Fprintf(&b, "%s:%s;", cssVarName(kv[0]), kv[1])
	}
	b.WriteString("}")
	if dark := d.darkOverrides(); len(dark) > 0 {
		b.WriteString("@media (prefers-color-scheme:dark){:root{")
		for _, kv := range dark {
			fmt.Fprintf(&b, "%s:%s;", cssVarName(kv[0]), kv[1])
		}
		b.WriteString("}}")
	}
	b.WriteString("</style>")
	return b.String()
}

// darkOverrides returns the dark values that actually differ from the light
// ones, in documented order — so the media query stays as small as it can be.
func (d DesignSystem) darkOverrides() [][2]string {
	out := make([][2]string, 0, len(d.TokensDark))
	for _, t := range designTokens {
		if !themedTokens[t.Key] {
			continue
		}
		v := strings.TrimSpace(d.TokensDark[t.Key])
		if v == "" || v == strings.TrimSpace(d.Tokens[t.Key]) {
			continue
		}
		out = append(out, [2]string{t.Key, v})
	}
	return out
}

// PromptBlock renders the design system for a seeded prompt: the tokens with
// the variable the agent must use, then the free-text rules.
func (d DesignSystem) PromptBlock() string {
	if d.IsEmpty() {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n[Design system for this project — follow it; it is why our canvases look like one product]\n")
	if toks := d.sortedTokens(); len(toks) > 0 {
		b.WriteString("Tokens (already injected into the canvas document as CSS variables — USE THE VARIABLE, do not hard-code the value, so a later token change re-styles every canvas):\n")
		for _, kv := range toks {
			fmt.Fprintf(&b, "  %s = %s   → var(%s)\n", kv[0], kv[1], cssVarName(kv[0]))
		}
	}
	if dark := d.darkOverrides(); len(dark) > 0 {
		b.WriteString("Dark mode is already handled: the same variables are re-declared under @media (prefers-color-scheme: dark), so a canvas built from them works in BOTH themes — never hard-code a light background or write your own dark-mode rules for these.\n")
		for _, kv := range dark {
			fmt.Fprintf(&b, "  %s (dark) = %s\n", kv[0], kv[1])
		}
	}
	if n := strings.TrimSpace(d.Notes); n != "" {
		fmt.Fprintf(&b, "Style rules: %s\n", n)
	}
	b.WriteString("Anything the system doesn't cover is your call — but stay consistent with what it does cover.\n")
	return b.String()
}

// Design tasks the operator can hand to the agent from the panel. Both are
// things people ask for in words and get inconsistently; making them buttons
// with a gateway-owned prompt means the wording stays in step with the token
// vocabulary instead of drifting per surface.
const (
	// TaskExtract — read the project's real theme and record it as the system.
	TaskExtract = "extract"
	// TaskShowcase — render the system as a canvas the operator can look at.
	TaskShowcase = "showcase"
)

// DesignTaskPrompt returns the seeded prompt for a design task, or "" if the
// task is unknown.
func DesignTaskPrompt(task string) string {
	switch task {
	case TaskExtract:
		return "[Canvas design system] The operator asked you to set up this project's canvas design system FROM THE REAL CODE — do not invent values. " +
			"Read what the project actually uses: the Tailwind config, CSS custom properties (`:root` / `.dark` / `@theme` blocks), the global stylesheet, existing components, and CLAUDE.md if it records conventions. " +
			"Then call the `canvas_design` MCP tool once with BOTH `tokens` (light) and `darkTokens` (the dark counterparts of the colour tokens), plus `notes` describing the style rules tokens can't express — density, what to avoid, how buttons and accents are used — inferred from the same code. " +
			"Map by MEANING, not by variable name — `primary` here is the project's BRAND colour, the one a main action is painted with. " +
			"Watch out for shadcn/ui and the Tailwind templates built on it: their `--primary` is a near-black or near-white INK, not a brand colour, and copying it here yields a palette of greys with no brand colour at all. " +
			"In that case put the brand hue (usually `--accent`) in `primary` and the ink in `text`. If the palette you are about to write has no chromatic colour in it, you have hit exactly this and should re-read the theme. " +
			"Prefer the project's own colour notation (keep oklch if that's what it uses). If the project has no theme yet, say so plainly and ask the operator to pick a palette in the panel instead of inventing one."
	case TaskShowcase:
		return "[Canvas design system] The operator wants to SEE this project's design system. Render it to the Canvas with `canvas_render` using kind=\"doc\" and slug=\"design-system\" (re-render that slug if it already exists). " +
			"Show, in this order: a swatch per colour token with its name and CSS variable; a table of the type / radius / spacing / shadow tokens with a live sample of each; a small set of components (buttons in both forms, a chip, an input, a card) built ONLY from the tokens; and the style rules written out. " +
			"Critical: use `var(--od-…)` for every single value — do not hard-code one colour, size or radius. That way the page doubles as proof the tokens are live, and it restyles itself when a token changes. It must read correctly in BOTH light and dark, which it will if you only use the variables."
	}
	return ""
}

// GetDesign returns the project's design system (zero value when unset).
func (s *Store) GetDesign(ctx context.Context, cwd string) (DesignSystem, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return DesignSystem{}, errors.New("canvas: cwd is required")
	}
	var raw, rawDark []byte
	var notes string
	err := s.pool.QueryRow(ctx, `
		SELECT tokens, tokens_dark, notes FROM canvas_design_systems WHERE cwd = $1`, cwd).
		Scan(&raw, &rawDark, &notes)
	if errors.Is(err, pgx.ErrNoRows) {
		return DesignSystem{Cwd: cwd, Tokens: map[string]string{}, TokensDark: map[string]string{}}, nil
	}
	if err != nil {
		return DesignSystem{}, fmt.Errorf("canvas: read design system: %w", err)
	}
	return DesignSystem{
		Cwd:        cwd,
		Tokens:     decodeTokens(raw),
		TokensDark: decodeTokens(rawDark),
		Notes:      notes,
	}, nil
}

// SetDesign replaces the project's design system.
//
// preserveNotation is for writes that came from a colour PICKER rather than
// from someone choosing a notation. The native picker on both surfaces can only
// emit #rrggbb, so without this, editing one swatch of an oklch theme converts
// that token to hex and the stored palette ends up in two notations. Agents
// writing through canvas_design pass false: they picked their notation on
// purpose (usually the project's own) and it is not ours to rewrite.
func (s *Store) SetDesign(ctx context.Context, d DesignSystem, preserveNotation bool) (DesignSystem, error) {
	cwd := strings.TrimSpace(d.Cwd)
	if cwd == "" {
		return DesignSystem{}, errors.New("canvas: cwd is required")
	}
	tokens := cleanTokens(d.Tokens, false)
	dark := cleanTokens(d.TokensDark, true)
	if preserveNotation {
		prev, err := s.GetDesign(ctx, cwd)
		if err != nil {
			return DesignSystem{}, err
		}
		tokens = keepNotation(tokens, prev.Tokens)
		dark = keepNotation(dark, prev.TokensDark)
	}
	raw, err := json.Marshal(tokens)
	if err != nil {
		return DesignSystem{}, fmt.Errorf("canvas: encode tokens: %w", err)
	}
	rawDark, err := json.Marshal(dark)
	if err != nil {
		return DesignSystem{}, fmt.Errorf("canvas: encode dark tokens: %w", err)
	}
	notes := strings.TrimSpace(d.Notes)
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO canvas_design_systems (cwd, tokens, tokens_dark, notes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (cwd) DO UPDATE SET
			tokens = EXCLUDED.tokens,
			tokens_dark = EXCLUDED.tokens_dark,
			notes = EXCLUDED.notes,
			updated_at = NOW()`, cwd, raw, rawDark, notes); err != nil {
		return DesignSystem{}, fmt.Errorf("canvas: write design system: %w", err)
	}
	return DesignSystem{Cwd: cwd, Tokens: tokens, TokensDark: dark, Notes: notes}, nil
}

// keepNotation rewrites incoming hex values back into the notation the same
// token already used. Only a bare hex is touched, and only when the previous
// value was written some other way — so typing a colour out in full still
// changes the notation, and a project that was already hex stays hex.
func keepNotation(in, prev map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
		old := prev[k]
		if old == "" || notationOf(v) != notationHex {
			continue
		}
		out[k] = reencode(v, old)
	}
	return out
}

// decodeTokens reads a tokens column; a hand-edited row shouldn't break
// rendering, so a bad value just yields an empty set.
func decodeTokens(raw []byte) map[string]string {
	out := map[string]string{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return map[string]string{}
		}
	}
	return out
}

// cleanTokens trims and drops empties. themedOnly keeps the dark set to the
// tokens that actually vary with the theme.
func cleanTokens(in map[string]string, themedOnly bool) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		if themedOnly && !themedTokens[k] {
			continue
		}
		out[k] = v
	}
	return out
}

// injectDesign puts the design system's CSS variables into a canvas document,
// replacing any block from a previous render so re-rendering can't stack them.
func injectDesign(html, style string) string {
	html = designBlockRE.ReplaceAllString(html, "")
	if style == "" {
		return html
	}
	if i := strings.Index(strings.ToLower(html), "<head>"); i >= 0 {
		return html[:i+len("<head>")] + style + html[i+len("<head>"):]
	}
	return style + html
}
