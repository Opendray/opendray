package summarizer

import (
	"math"
	"testing"
)

func TestEstimateUSD_KnownModels(t *testing.T) {
	cases := []struct {
		model         string
		input, output int
		want          float64
		tolerancePerM float64 // micro-USD tolerance
	}{
		// Haiku: $1/MTok input, $5/MTok output
		{"claude-haiku-4-5", 1_000_000, 0, 1.00, 1e-6},
		{"claude-haiku-4-5", 0, 1_000_000, 5.00, 1e-6},
		// 500 input * $1/M = 0.0005 ; 100 output * $5/M = 0.0005 → total 0.001
		{"claude-haiku-4-5", 500, 100, 0.001, 1e-6},
		// Sonnet: $3 input, $15 output
		{"claude-sonnet-4-6", 100_000, 50_000, 0.300 + 0.750, 1e-6},
		// Opus
		{"claude-opus-4-7", 1_000_000, 0, 15.00, 1e-6},
		// Local (zero)
		{"ollama:*", 1_000_000, 1_000_000, 0, 1e-9},
	}
	for _, c := range cases {
		got := EstimateUSD(c.model, c.input, c.output)
		if math.Abs(got-c.want) > c.tolerancePerM {
			t.Errorf("EstimateUSD(%q, %d, %d) = %g, want %g", c.model, c.input, c.output, got, c.want)
		}
	}
}

func TestEstimateUSD_UnknownModel(t *testing.T) {
	got := EstimateUSD("nonexistent-model-xyz", 999_999, 999_999)
	if got != 0 {
		t.Errorf("unknown model should cost 0, got %g", got)
	}
}

func TestIsLocalModel(t *testing.T) {
	if !IsLocalModel("ollama") {
		t.Error("ollama should be local")
	}
	if IsLocalModel("anthropic") {
		t.Error("anthropic is not local")
	}
}
