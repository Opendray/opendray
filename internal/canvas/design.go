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

// DesignSystem is a project's canvas styling contract.
type DesignSystem struct {
	Cwd    string            `json:"cwd"`
	Tokens map[string]string `json:"tokens"`
	Notes  string            `json:"notes"`
}

// IsEmpty reports whether there is nothing worth injecting.
func (d DesignSystem) IsEmpty() bool {
	return len(d.Tokens) == 0 && strings.TrimSpace(d.Notes) == ""
}

// designTokens are the tokens the Canvas understands. Anything else the
// operator adds is kept and passed through, but these are the ones with a
// documented CSS variable and a place in the prompt. Order matters: it is the
// order the operator and the agent read them in.
var designTokens = []struct{ Key, Label string }{
	{"primary", "primary / accent colour"},
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

// StyleBlock renders the tokens as a CSS custom-property block to inject into a
// canvas document. Empty when there are no tokens.
func (d DesignSystem) StyleBlock() string {
	toks := d.sortedTokens()
	if len(toks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<style id=\"od-design-system\">:root{")
	for _, kv := range toks {
		fmt.Fprintf(&b, "%s:%s;", cssVarName(kv[0]), kv[1])
	}
	b.WriteString("}</style>")
	return b.String()
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
	if n := strings.TrimSpace(d.Notes); n != "" {
		fmt.Fprintf(&b, "Style rules: %s\n", n)
	}
	b.WriteString("Anything the system doesn't cover is your call — but stay consistent with what it does cover.\n")
	return b.String()
}

// GetDesign returns the project's design system (zero value when unset).
func (s *Store) GetDesign(ctx context.Context, cwd string) (DesignSystem, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return DesignSystem{}, errors.New("canvas: cwd is required")
	}
	var raw []byte
	var notes string
	err := s.pool.QueryRow(ctx, `
		SELECT tokens, notes FROM canvas_design_systems WHERE cwd = $1`, cwd).
		Scan(&raw, &notes)
	if errors.Is(err, pgx.ErrNoRows) {
		return DesignSystem{Cwd: cwd, Tokens: map[string]string{}}, nil
	}
	if err != nil {
		return DesignSystem{}, fmt.Errorf("canvas: read design system: %w", err)
	}
	tokens := map[string]string{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &tokens); err != nil {
			// A hand-edited row shouldn't break rendering — fall back to notes.
			tokens = map[string]string{}
		}
	}
	return DesignSystem{Cwd: cwd, Tokens: tokens, Notes: notes}, nil
}

// SetDesign replaces the project's design system.
func (s *Store) SetDesign(ctx context.Context, d DesignSystem) (DesignSystem, error) {
	cwd := strings.TrimSpace(d.Cwd)
	if cwd == "" {
		return DesignSystem{}, errors.New("canvas: cwd is required")
	}
	tokens := map[string]string{}
	for k, v := range d.Tokens {
		if k = strings.TrimSpace(k); k != "" && strings.TrimSpace(v) != "" {
			tokens[k] = strings.TrimSpace(v)
		}
	}
	raw, err := json.Marshal(tokens)
	if err != nil {
		return DesignSystem{}, fmt.Errorf("canvas: encode tokens: %w", err)
	}
	notes := strings.TrimSpace(d.Notes)
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO canvas_design_systems (cwd, tokens, notes)
		VALUES ($1, $2, $3)
		ON CONFLICT (cwd) DO UPDATE SET
			tokens = EXCLUDED.tokens,
			notes = EXCLUDED.notes,
			updated_at = NOW()`, cwd, raw, notes); err != nil {
		return DesignSystem{}, fmt.Errorf("canvas: write design system: %w", err)
	}
	return DesignSystem{Cwd: cwd, Tokens: tokens, Notes: notes}, nil
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
