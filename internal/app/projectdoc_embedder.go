package app

import (
	"context"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// projectdocEmbedderAdapter bridges memory.Embedder to
// projectdoc.LogEmbedder. Both interfaces share the same shape but
// live in different packages on purpose — projectdoc must not
// import internal/memory directly (it'd create a dependency from
// the operator-owned doc layer to the agent-owned fact layer,
// which we keep one-way for the same reasons captured in the
// projectdoc package comment).
type projectdocEmbedderAdapter struct {
	emb memory.Embedder
}

var _ projectdoc.LogEmbedder = projectdocEmbedderAdapter{}

func (a projectdocEmbedderAdapter) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if a.emb == nil {
		return make([][]float32, len(texts)), nil
	}
	return a.emb.Embed(ctx, texts)
}

func (a projectdocEmbedderAdapter) Name() string {
	if a.emb == nil {
		return ""
	}
	return a.emb.Name()
}
