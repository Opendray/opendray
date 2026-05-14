package memquery

import (
	"math"
	"testing"
	"time"
)

func TestDecayScore(t *testing.T) {
	tests := []struct {
		name string
		sim  float32
		age  time.Duration
		want float32
	}{
		{"brand new keeps full score", 0.9, 0, 0.9},
		{"30d old: ~0.83 of original", 1.0, 30 * 24 * time.Hour, 1 - 30.0/180},
		{"180d hits floor", 0.8, 180 * 24 * time.Hour, 0.5 * 0.8},
		{"way past floor", 1.0, 365 * 24 * time.Hour, 0.5},
		{"negative age clamps to now", 1.0, -time.Hour, 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decayScore(tc.sim, tc.age)
			if math.Abs(float64(got-tc.want)) > 0.001 {
				t.Errorf("got %f, want %f", got, tc.want)
			}
		})
	}
}

func TestPgvecLiteral(t *testing.T) {
	tests := []struct {
		name string
		in   []float32
		want string
	}{
		{"empty", nil, "[]"},
		{"single", []float32{0.5}, "[0.5]"},
		{"multiple", []float32{0.1, 0.2, 0.3}, "[0.1,0.2,0.3]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pgvecLiteral(tc.in)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNew_ValidationErrors(t *testing.T) {
	if _, err := New(nil, nil, nil); err == nil {
		t.Error("expected error for all-nil deps")
	}
}
