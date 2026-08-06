package canvas

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Colour handling for design-system tokens.
//
// A project writes its palette in whatever notation its theme uses — a Tailwind
// v4 codebase is usually all oklch(), an older one hex or hsl(). The gateway
// stores that string verbatim and injects it into canvases as a CSS variable,
// where the browser does the work and the notation never matters.
//
// It matters everywhere ELSE, and that is why this file exists:
//
//   - the panels draw a swatch per token, and neither surface can read a
//     colour on its own. The web panel used to scrape getComputedStyle().color
//     for "rgb(…)", which only legacy sRGB notations serialise to — an oklch
//     value round-trips unchanged, so an all-oklch project got a grid of
//     identical grey fallbacks. Mobile was worse: it parsed nothing but
//     #rrggbb, so those projects got no swatches at all. Flutter has no CSS
//     parser to borrow and React and Flutter share no runtime, so resolving in
//     both would mean writing this twice and letting the two drift.
//   - the operator's colour picker only speaks #rrggbb, so writing its result
//     back would convert a project's theme to hex one field at a time.
//   - a palette with no chromatic colour in it is the signature of a mapping
//     mistake (see designTokens), and detecting that needs real chroma.
//
// Resolving once, here, serves all three. Named CSS colours are deliberately
// NOT supported: the list is 148 entries and nobody names a brand token
// `rebeccapurple`. Everything degrades gracefully instead — an unresolvable
// value simply has no resolved form, the web panel falls back to asking the
// browser, and the achromatic check skips it rather than guessing.

// srgb is a colour resolved to sRGB. Components are 0..1 and may fall outside
// that range: oklch describes colours sRGB cannot, and clipping is deferred to
// the point where an 8-bit value is actually needed.
type srgb struct {
	R, G, B float64
}

// Colour notations we can both read and write. The name doubles as the value
// stored alongside a token so a picker result can be written back in kind.
const (
	notationHex   = "hex"
	notationRGB   = "rgb"
	notationHSL   = "hsl"
	notationOKLCH = "oklch"
	notationOKLAB = "oklab"
)

// parseColor resolves a CSS colour string to sRGB and reports which notation
// it was written in. Alpha is parsed to keep the syntax accepted but is then
// dropped: every consumer here wants an opaque colour (a swatch, a chroma
// reading), and `<input type="color">` has no alpha channel anyway.
func parseColor(value string) (srgb, string, bool) {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return srgb{}, "", false
	}
	if strings.HasPrefix(v, "#") {
		c, ok := parseHex(v)
		return c, notationHex, ok
	}
	fn, args, ok := splitFunc(v)
	if !ok {
		return srgb{}, "", false
	}
	switch fn {
	case "rgb", "rgba":
		c, ok := parseRGBFunc(args)
		return c, notationRGB, ok
	case "hsl", "hsla":
		c, ok := parseHSLFunc(args)
		return c, notationHSL, ok
	case "oklch":
		c, ok := parseOKLCHFunc(args)
		return c, notationOKLCH, ok
	case "oklab":
		c, ok := parseOKLabFunc(args)
		return c, notationOKLAB, ok
	}
	return srgb{}, "", false
}

// ResolvedHex renders a CSS colour as the #rrggbb a swatch or a native colour
// picker needs, or "" when the value isn't a colour we can read. Components
// outside sRGB are clipped, which is what any 8-bit rendering of a wide-gamut
// colour has to do.
func ResolvedHex(value string) string {
	c, _, ok := parseColor(value)
	if !ok {
		return ""
	}
	return hexOf(c)
}

// notationOf reports the notation a stored token value is written in, or "".
func notationOf(value string) string {
	_, n, ok := parseColor(value)
	if !ok {
		return ""
	}
	return n
}

// reencode rewrites a colour into the notation `like` is written in, so the
// panel's hex-only picker can edit an oklch theme without converting it. A
// value already in the target notation, an unreadable input or an unreadable
// reference all leave the input untouched.
func reencode(value, like string) string {
	c, from, ok := parseColor(value)
	if !ok {
		return value
	}
	to := notationOf(like)
	if to == "" || to == from {
		return value
	}
	switch to {
	case notationHex:
		return hexOf(c)
	case notationRGB:
		r, g, b := clip8(c)
		return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	case notationHSL:
		h, s, l := hslOf(c)
		return fmt.Sprintf("hsl(%s %s%% %s%%)", trimFloat(h, 1), trimFloat(s*100, 1), trimFloat(l*100, 1))
	case notationOKLCH:
		l, ch, h := oklchOf(c)
		return fmt.Sprintf("oklch(%s %s %s)", trimFloat(l, 4), trimFloat(ch, 4), trimFloat(h, 2))
	case notationOKLAB:
		l, a, b := oklabOf(c)
		return fmt.Sprintf("oklab(%s %s %s)", trimFloat(l, 4), trimFloat(a, 4), trimFloat(b, 4))
	}
	return value
}

// chromaticThreshold is the OKLCh chroma at which a colour stops being a
// neutral and starts being a colour someone chose. Calibrated against real
// palettes rather than picked: the most tinted neutral in common use is
// Tailwind's slate-500 and the least saturated brand colour in the panel's own
// starting palettes is amber-600, and this sits between them. See
// TestChromaSeparatesNeutralsFromBrandColours.
const chromaticThreshold = 0.05

// chromaOf returns the OKLCh chroma of a colour — how far it is from grey,
// independent of how light it is. Used to tell a palette that merely looks
// restrained from one that has no colour in it at all.
func chromaOf(value string) (float64, bool) {
	c, _, ok := parseColor(value)
	if !ok {
		return 0, false
	}
	_, ch, _ := oklchOf(c)
	return ch, true
}

// --- parsing -------------------------------------------------------------

func parseHex(v string) (srgb, bool) {
	h := v[1:]
	// 4 and 8 digit forms carry alpha, which we accept and drop.
	switch len(h) {
	case 3, 4:
		var sb strings.Builder
		for _, r := range h[:3] {
			sb.WriteRune(r)
			sb.WriteRune(r)
		}
		h = sb.String()
	case 6:
	case 8:
		h = h[:6]
	default:
		return srgb{}, false
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return srgb{}, false
	}
	return srgb{
		R: float64((n>>16)&0xff) / 255,
		G: float64((n>>8)&0xff) / 255,
		B: float64(n&0xff) / 255,
	}, true
}

// splitFunc pulls "oklch" and "0.65 0.2 35" out of "oklch(0.65 0.2 35)".
func splitFunc(v string) (string, string, bool) {
	i := strings.IndexByte(v, '(')
	if i <= 0 || !strings.HasSuffix(v, ")") {
		return "", "", false
	}
	return strings.TrimSpace(v[:i]), strings.TrimSpace(v[i+1 : len(v)-1]), true
}

// splitArgs handles both CSS argument syntaxes at once — the legacy comma form
// and the modern space form with an optional `/ alpha` — returning the
// components with alpha removed.
func splitArgs(args string) []string {
	if i := strings.IndexByte(args, '/'); i >= 0 {
		args = args[:i]
	}
	args = strings.ReplaceAll(args, ",", " ")
	return strings.Fields(args)
}

// number reads one component. `pct` is the value 100% stands for, letting the
// same parser take "50%" and "0.5". CSS Color 4's `none` keyword means "no
// value", which resolves to zero.
func number(tok string, pct float64) (float64, bool) {
	tok = strings.TrimSpace(tok)
	if tok == "none" {
		return 0, true
	}
	tok = strings.TrimSuffix(tok, "deg")
	if s, ok := strings.CutSuffix(tok, "%"); ok {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return f / 100 * pct, true
	}
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func parseRGBFunc(args string) (srgb, bool) {
	f := splitArgs(args)
	if len(f) < 3 {
		return srgb{}, false
	}
	out := make([]float64, 3)
	for i := range out {
		// Percentages are of 255 here, so both forms land on the same scale.
		n, ok := number(f[i], 255)
		if !ok {
			return srgb{}, false
		}
		out[i] = n / 255
	}
	return srgb{R: out[0], G: out[1], B: out[2]}, true
}

func parseHSLFunc(args string) (srgb, bool) {
	f := splitArgs(args)
	if len(f) < 3 {
		return srgb{}, false
	}
	h, ok1 := number(f[0], 360)
	s, ok2 := number(f[1], 1)
	l, ok3 := number(f[2], 1)
	if !ok1 || !ok2 || !ok3 {
		return srgb{}, false
	}
	// Bare numbers for saturation/lightness are already fractions in the
	// modern syntax; the percent form was scaled by number() above.
	return hslToRGB(h, clamp01(s), clamp01(l)), true
}

func parseOKLCHFunc(args string) (srgb, bool) {
	f := splitArgs(args)
	if len(f) < 3 {
		return srgb{}, false
	}
	l, ok1 := number(f[0], 1)
	c, ok2 := number(f[1], 0.4) // 100% chroma is defined as 0.4
	h, ok3 := number(f[2], 360)
	if !ok1 || !ok2 || !ok3 {
		return srgb{}, false
	}
	rad := h * math.Pi / 180
	return oklabToRGB(l, c*math.Cos(rad), c*math.Sin(rad)), true
}

func parseOKLabFunc(args string) (srgb, bool) {
	f := splitArgs(args)
	if len(f) < 3 {
		return srgb{}, false
	}
	l, ok1 := number(f[0], 1)
	a, ok2 := number(f[1], 0.4)
	b, ok3 := number(f[2], 0.4)
	if !ok1 || !ok2 || !ok3 {
		return srgb{}, false
	}
	return oklabToRGB(l, a, b), true
}

// --- conversions ---------------------------------------------------------

// OKLab ↔ sRGB use Björn Ottosson's published matrices, the same ones the CSS
// Color 4 specification defines the notation against.

func oklabToRGB(l, a, b float64) srgb {
	l_ := l + 0.3963377774*a + 0.2158037573*b
	m_ := l - 0.1055613458*a - 0.0638541728*b
	s_ := l - 0.0894841775*a - 1.2914855480*b
	lc, mc, sc := l_*l_*l_, m_*m_*m_, s_*s_*s_
	return srgb{
		R: gammaEncode(+4.0767416621*lc - 3.3077115913*mc + 0.2309699292*sc),
		G: gammaEncode(-1.2684380046*lc + 2.6097574011*mc - 0.3413193965*sc),
		B: gammaEncode(-0.0041960863*lc - 0.7034186147*mc + 1.7076147010*sc),
	}
}

func oklabOf(c srgb) (float64, float64, float64) {
	r, g, b := gammaDecode(c.R), gammaDecode(c.G), gammaDecode(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s
}

func oklchOf(c srgb) (float64, float64, float64) {
	l, a, b := oklabOf(c)
	h := math.Atan2(b, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return l, math.Hypot(a, b), h
}

func hslToRGB(h, s, l float64) srgb {
	h = math.Mod(math.Mod(h, 360)+360, 360)
	f := func(n float64) float64 {
		k := math.Mod(n+h/30, 12)
		a := s * math.Min(l, 1-l)
		return l - a*math.Max(-1, math.Min(math.Min(k-3, 9-k), 1))
	}
	return srgb{R: f(0), G: f(8), B: f(4)}
}

func hslOf(c srgb) (float64, float64, float64) {
	r, g, b := clamp01(c.R), clamp01(c.G), clamp01(c.B)
	maxc := math.Max(r, math.Max(g, b))
	minc := math.Min(r, math.Min(g, b))
	l := (maxc + minc) / 2
	d := maxc - minc
	if d == 0 {
		return 0, 0, l
	}
	s := d / (1 - math.Abs(2*l-1))
	var h float64
	switch maxc {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, clamp01(s), l
}

// sRGB transfer function, both directions.

func gammaEncode(x float64) float64 {
	if x <= 0.0031308 {
		return 12.92 * x
	}
	return 1.055*math.Pow(x, 1/2.4) - 0.055
}

func gammaDecode(x float64) float64 {
	x = clamp01(x)
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

// --- formatting ----------------------------------------------------------

func hexOf(c srgb) string {
	r, g, b := clip8(c)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func clip8(c srgb) (int, int, int) {
	to8 := func(x float64) int { return int(math.Round(clamp01(x) * 255)) }
	return to8(c.R), to8(c.G), to8(c.B)
}

func clamp01(x float64) float64 {
	return math.Max(0, math.Min(1, x))
}

// trimFloat formats without trailing zeros, so a converted value reads like
// something a person would have typed rather than "0.6500000".
func trimFloat(f float64, digits int) string {
	s := strconv.FormatFloat(f, 'f', digits, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "-0" {
		return "0"
	}
	return s
}
