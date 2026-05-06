package memory

import (
	"context"
	"strings"
	"unicode"
)

// BM25Embedder is the pure-Go fallback embedder. It produces fixed-
// dimension term-frequency vectors using a hash-projected bag-of-
// tokens scheme — basically a stripped-down BM25 ranker reframed
// as an embedder.
//
// Why this works: a hash-bucket sparse vector has the same
// cosine-similarity semantics as a "real" embedding for the narrow
// case of "did these two pieces of text mention overlapping
// terms?" That's most of what we want from the v1 memory subsystem
// anyway — code paths, file names, library names, decision
// keywords. It loses to a transformer on paraphrase / semantic
// similarity, but it's deterministic, allocation-light, and ships
// with zero dependencies. LocalONNX (bge-m3) picks up where this
// gives out for operators who care about paraphrase recall.
//
// The hash-projection (FNV-1a → modulo `dim`) is a well-known
// trick from large-scale ML: it's how scikit-learn's HashingVectorizer
// works. Collisions are noise; the cosine still ranks honestly.
//
// Tokenization is deliberately naive: lowercase + Unicode-letter
// runs. Code identifiers like `parseCwd` get emitted as a single
// token (no camelCase splitting) which matches how a developer
// types them when searching.
type BM25Embedder struct {
	dim int
}

// NewBM25Embedder returns a BM25 embedder of the given dimension.
// 384 is the sweet spot — small enough to stay cheap, large enough
// to keep collisions tolerable on a per-operator memory store of a
// few thousand entries.
func NewBM25Embedder(dim int) *BM25Embedder {
	if dim <= 0 {
		dim = 384
	}
	return &BM25Embedder{dim: dim}
}

func (e *BM25Embedder) Name() string    { return "bm25" }
func (e *BM25Embedder) Dimensions() int { return e.dim }

// Embed computes a hash-bucketed term-frequency vector for each
// input. Vectors are L2-normalised so cosine similarity equals
// the dot product, which is what every Store implementation
// expects.
func (e *BM25Embedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = e.embedOne(t)
	}
	return out, nil
}

func (e *BM25Embedder) embedOne(text string) []float32 {
	v := make([]float32, e.dim)
	for _, tok := range tokenize(text) {
		bucket := fnv1aMod(tok, uint32(e.dim))
		v[bucket]++
	}
	return l2Normalise(v)
}

// tokenize lowercases + splits on non-letter runs. Unicode-aware so
// CJK / accented text contributes too.
func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	out := make([]string, 0, 16)
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			out = append(out, strings.ToLower(b.String()))
			b.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

// fnv1aMod is FNV-1a 32-bit, then modulo. Adequate hash for a
// hashing-trick projection; not adequate for security.
func fnv1aMod(s string, mod uint32) uint32 {
	const offset uint32 = 2166136261
	const prime uint32 = 16777619
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	return h % mod
}

// l2Normalise scales the vector to unit length. Operates in-place
// on a fresh slice; safe to mutate the input.
func l2Normalise(v []float32) []float32 {
	var sumSq float32
	for _, x := range v {
		sumSq += x * x
	}
	if sumSq == 0 {
		return v
	}
	inv := 1 / sqrt32(sumSq)
	for i := range v {
		v[i] *= inv
	}
	return v
}
