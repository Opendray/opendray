// Package memory implements opendray's optional cross-session
// memory layer — the "remember things across runs" RAG subsystem
// exposed to agents through an in-process MCP server.
//
// The package is split into two replaceable abstractions so the
// runtime can mix and match without touching call sites:
//
//   - Embedder turns natural-language text into vectors.
//     Implementations: BM25 (pure Go, default), OpenAICompatibleHTTP
//     (covers ollama / LM Studio / OpenAI / LocalAI / vLLM), and
//     LocalONNX (bge-m3 via cgo, behind the `local_onnx` build tag).
//
//   - Store persists vectors and answers similarity queries.
//     Implementations: pgvector (default, reuses opendray's
//     existing PG).
//
// Higher layers (the MCP server, the admin debug API) talk to a
// memory.Service which wires one Embedder + one Store together.
package memory

import (
	"context"
	"math"
)

// stdlibSqrt is named separately so sqrt32 can stay a small inline
// helper without importing math at the call site.
func stdlibSqrt(x float64) float64 { return math.Sqrt(x) }

// Embedder turns text into a dense vector. All implementations
// must produce vectors of the same Dimensions() across calls so
// stored vectors stay comparable through the lifetime of the
// process.
//
// Embed accepts a slice for batch efficiency: HTTP backends save
// per-call latency by sending many texts at once, and BM25 can
// share its document-frequency index across the batch. The
// returned slice has one vector per input in the same order.
type Embedder interface {
	// Embed computes vectors for each input string. len(out) ==
	// len(in) on success; partial results are not returned —
	// callers retry the whole batch on error.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Dimensions reports the fixed vector dimensionality this
	// embedder produces. Stable across calls.
	Dimensions() int
	// Name is a short identifier shown in logs / settings UI.
	Name() string
}

// CosineSimilarity computes the cosine similarity between two
// equally-sized vectors. Returns 0 when sizes mismatch or either
// vector is the zero vector — the caller treats that as "no match"
// without having to special-case the inputs.
//
// Used by every Store implementation that doesn't have a native
// HNSW or IVF index (i.e. chromem-go and brute-force fallback).
// pgvector pushes this down to the DB via the `<=>` operator.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float32
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	// Use float64 sqrt for numerical stability; cast back.
	return dot / (sqrt32(na) * sqrt32(nb))
}

// sqrt32 wraps math.Sqrt to keep the float32 surface clean.
func sqrt32(x float32) float32 {
	if x <= 0 {
		return 0
	}
	// Newton's method one or two iterations is enough for our use
	// case, but stdlib math.Sqrt is fine — we go through float64
	// for portability.
	return float32(stdlibSqrt(float64(x)))
}
