package summarizer

// Pricing table — USD per 1M tokens, separated input vs output.
//
// We hard-code current public pricing rather than fetching from
// the provider's billing API because (a) most providers don't
// expose a price feed, (b) operators care more about a rough
// "what did this cost me yesterday" than penny-perfect accounting,
// and (c) writing the snapshot price into memory_summarizer_calls
// at call time gives us audit-stable history even when the
// upstream price changes.
//
// Sources (re-check periodically):
//   Anthropic: https://www.anthropic.com/pricing#anthropic-api
//   OpenAI:    https://openai.com/api/pricing/  (Phase B)
//
// Unknown model → 0 cost. ollama / local models → always 0
// (operator pays for hardware, not per-token).

type modelPrice struct {
	InputPerM  float64 // USD per 1M input tokens
	OutputPerM float64 // USD per 1M output tokens
}

// modelPrices: keyed by exact model name as the provider returns it.
// Add new rows when adding new model SKUs; lookups missing here
// silently return zero cost (no UI penalty for being slightly off,
// but obvious "$0 for this expensive model" signal in the panel).
var modelPrices = map[string]modelPrice{
	// ── Anthropic Haiku 4.5 (default summarizer model) ─────────
	"claude-haiku-4-5":          {InputPerM: 1.00, OutputPerM: 5.00},
	"claude-haiku-4-5-20251001": {InputPerM: 1.00, OutputPerM: 5.00},

	// ── Anthropic Sonnet 4.6 (operator opt-in for higher quality) ─
	"claude-sonnet-4-6": {InputPerM: 3.00, OutputPerM: 15.00},

	// ── Anthropic Opus 4.7 (rarely worth it for this task) ─────
	"claude-opus-4-7": {InputPerM: 15.00, OutputPerM: 75.00},

	// ── ollama / any local — operator pays for hardware ───────
	"ollama:*": {InputPerM: 0, OutputPerM: 0},
}

// EstimateUSD returns the predicted USD cost of a single call
// based on token counts + the configured model's price.
//
// Returns 0 for unknown models (graceful fallback) — the caller
// sees the price snapshot in memory_summarizer_calls.estimated_usd
// and can post-correct via a manual UPDATE if needed.
func EstimateUSD(model string, inputTokens, outputTokens int) float64 {
	p, ok := modelPrices[model]
	if !ok {
		return 0
	}
	return (float64(inputTokens)*p.InputPerM + float64(outputTokens)*p.OutputPerM) / 1_000_000
}

// IsLocalModel returns true when the model name belongs to a
// runs-on-operator-hardware backend (ollama). Used by the registry
// to skip cipher requirements (no API key to encrypt).
func IsLocalModel(kind string) bool {
	return kind == "ollama"
}
