//go:build !local_onnx

// Default-build stub for the LocalONNX embedder. Kept under a
// negated build tag so callers can reference the type + builder
// without picking up the cgo dependency. Operators who set
// backend=local in [memory] but didn't compile with -tags
// local_onnx get a clear error pointing at the docs.

package memory

import (
	"context"
	"errors"
)

// LocalONNXConfig keeps the same shape as the cgo build so
// config-marshalling code doesn't need to branch.
type LocalONNXConfig struct {
	LibraryPath   string
	ModelPath     string
	TokenizerPath string
	MaxSeqLen     int
	Dimensions    int
}

// LocalONNXEmbedder in the stub build is a placeholder that always
// errors. Real implementation lives in embedder_onnx.go behind the
// `local_onnx` build tag.
type LocalONNXEmbedder struct{}

func NewLocalONNXEmbedder(_ LocalONNXConfig) (*LocalONNXEmbedder, error) {
	return nil, errors.New(
		"memory: LocalONNX embedder not compiled into this binary. " +
			"Rebuild with `-tags local_onnx` and provide CGO_LDFLAGS for onnxruntime + libtokenizers. " +
			"See docs/adr/0014-memory-subsystem.md and the Memory tutorial for setup.",
	)
}

func (e *LocalONNXEmbedder) Close() error                                   { return nil }
func (e *LocalONNXEmbedder) Name() string                                   { return "local-onnx-stub" }
func (e *LocalONNXEmbedder) Dimensions() int                                { return 0 }
func (e *LocalONNXEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, errors.New("memory: LocalONNX stub — see NewLocalONNXEmbedder")
}
