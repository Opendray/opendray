package knowledge

import (
	"context"
	"strconv"
	"strings"
)

// Embedder is the vector surface the knowledge index needs. memory.Embedder
// satisfies it structurally; the app injects memorySvc.Embedder() so knowledge
// reuses the exact embedder memory already runs (same vector space).
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
	Name() string
}

// ParseVector parses pgvector's text output format ("[a,b,c]") back into a
// []float32. Inverse of vectorLiteral; nil on empty / malformed input. Used
// by the app's episode adapter to hand stored journal vectors to the
// experience compiler without a pgvector client dependency.
func ParseVector(s string) []float32 {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil
	}
	parts := strings.Split(s[1:len(s)-1], ",")
	out := make([]float32, 0, len(parts))
	for _, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil
		}
		out = append(out, float32(f))
	}
	return out
}

// vectorLiteral renders a []float32 in pgvector's text input format: [a,b,c].
func vectorLiteral(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
