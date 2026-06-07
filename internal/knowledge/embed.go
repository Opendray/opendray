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
