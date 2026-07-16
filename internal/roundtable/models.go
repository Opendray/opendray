package roundtable

import (
	"context"

	"github.com/opendray/opendray-v2/internal/providers"
)

// ModelOption re-exports the shared provider model option so existing callers
// (the handler, tests) keep working after the catalog moved to the neutral
// internal/providers package.
type ModelOption = providers.ModelOption

// ProviderModelOptions returns the selectable models per seat provider so the
// UI can offer a dropdown instead of a hand-typed model string (a one-char
// typo otherwise fails the whole seat). It delegates to the shared providers
// catalog — the single source of truth also used by the Cortex agent worker —
// so antigravity/opencode enumerate live from their CLIs and the curated
// providers stay in lockstep across features.
func ProviderModelOptions(ctx context.Context) map[string][]ModelOption {
	return providers.AgentModelOptions(ctx)
}
