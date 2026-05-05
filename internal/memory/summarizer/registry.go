package summarizer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Registry resolves a ProviderRow into a runnable Provider. Phase A
// builds providers on demand (no caching) — construction is cheap
// (just an http.Client) and a future Phase C optimisation can
// memoize once the read patterns stabilise.
//
// IntegrationLookup (optional, set via WithIntegrationLookup) is
// used to resolve integration-kind rows by looking up the
// integration's base_url. nil leaves integration-kind rows
// unbuildable.
type Registry struct {
	store        *Store
	integrations IntegrationLookup
	log          *slog.Logger
}

func NewRegistry(store *Store, log *slog.Logger) *Registry {
	if log == nil {
		log = slog.Default()
	}
	return &Registry{store: store, log: log}
}

// WithIntegrationLookup lets the app wire integration resolution
// after construction. Returns the registry for chaining.
func (r *Registry) WithIntegrationLookup(lookup IntegrationLookup) *Registry {
	r.integrations = lookup
	return r
}

// Build constructs a runnable Provider from the row at id. Returns
// ErrProviderNotFound when missing, or ErrProviderDisabled when
// the row exists but enabled=false (so callers can distinguish
// "operator paused this" from "this never existed").
func (r *Registry) Build(ctx context.Context, id string) (Provider, error) {
	row, err := r.store.GetProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	if !row.Enabled {
		return nil, ErrProviderDisabled
	}
	return r.buildFromRow(row)
}

// Default returns the row with is_default=TRUE. When none is set
// but at least one enabled row exists, returns the oldest enabled
// row as a fallback. Returns ErrNoProviderConfigured when nothing
// exists.
func (r *Registry) Default(ctx context.Context) (Provider, error) {
	row, err := r.store.GetDefaultProvider(ctx)
	if err == nil && row.Enabled {
		return r.buildFromRow(row)
	}
	if err != nil && !errors.Is(err, ErrProviderNotFound) {
		return nil, err
	}

	// No default flag set or default is disabled — fall back to
	// the oldest enabled row.
	all, err := r.store.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range all {
		if p.Enabled {
			return r.Build(ctx, p.ID)
		}
	}
	return nil, ErrNoProviderConfigured
}

// ListEnabledRows is a thin convenience for handlers that just
// want to render a select dropdown — never decrypts plaintext.
func (r *Registry) ListEnabledRows(ctx context.Context) ([]ProviderRow, error) {
	rows, err := r.store.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, p := range rows {
		if p.Enabled {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *Registry) buildFromRow(row ProviderRow) (Provider, error) {
	switch row.Kind {
	case "anthropic":
		// row.APIKeyPlaintext was decrypted during scanProvider for
		// single-row fetches.
		if row.APIKeyPlaintext == "" {
			return nil, fmt.Errorf("registry: anthropic provider %q has no decrypted api_key (cipher missing?)", row.ID)
		}
		cfg := AnthropicConfig{
			APIKey: row.APIKeyPlaintext,
			Model:  row.Model,
			Name:   row.Name,
		}
		if v, ok := row.ExtraConfig["base_url"].(string); ok && v != "" {
			cfg.BaseURL = v
		}
		if v, ok := row.ExtraConfig["max_tokens"].(float64); ok && v > 0 {
			cfg.MaxTokens = int(v)
		}
		return NewAnthropicProvider(cfg)
	case "ollama":
		cfg := OllamaConfig{
			Model:   row.Model,
			BaseURL: row.BaseURL,
			Name:    row.Name,
		}
		if v, ok := row.ExtraConfig["max_tokens"].(float64); ok && v > 0 {
			cfg.MaxTokens = int(v)
		}
		return NewOllamaProvider(cfg)
	case "openai", "lmstudio":
		// LM Studio + OpenAI share wire format; only auth presence
		// + base_url defaults differ. The OpenAICompatProvider
		// handles both based on Kind.
		cfg := OpenAICompatConfig{
			Kind:    row.Kind,
			APIKey:  row.APIKeyPlaintext, // empty OK for lmstudio
			Model:   row.Model,
			BaseURL: row.BaseURL,
			Name:    row.Name,
		}
		if v, ok := row.ExtraConfig["max_tokens"].(float64); ok && v > 0 {
			cfg.MaxTokens = int(v)
		}
		return NewOpenAICompatProvider(cfg)
	case "integration":
		if r.integrations == nil {
			return nil, errors.New("registry: integration kind requires IntegrationLookup wired into the registry (app.go)")
		}
		integID, _ := row.ExtraConfig["integration_id"].(string)
		if integID == "" {
			return nil, errors.New("registry: integration provider extra_config missing integration_id")
		}
		baseURL, enabled, err := r.integrations.LookupBaseURL(context.Background(), integID)
		if err != nil {
			return nil, fmt.Errorf("registry: lookup integration %q: %w", integID, err)
		}
		if !enabled {
			return nil, fmt.Errorf("registry: integration %q is disabled", integID)
		}
		// row.APIKeyPlaintext (when present) is the outbound bearer
		// token opendray sends to the integration; left empty means
		// no auth (relies on network trust).
		return NewIntegrationProvider(IntegrationConfig{
			IntegrationID: integID,
			BaseURL:       baseURL,
			OutboundToken: row.APIKeyPlaintext,
			Name:          row.Name,
		})
	default:
		return nil, fmt.Errorf("registry: unknown kind %q", row.Kind)
	}
}

// Sentinel errors specific to the registry (alongside the row-level
// ones in store.go).
var (
	ErrProviderDisabled    = errors.New("summarizer registry: provider exists but is disabled")
	ErrNoProviderConfigured = errors.New("summarizer registry: no enabled provider configured")
)
