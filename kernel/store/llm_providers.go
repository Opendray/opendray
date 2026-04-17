package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// LLMProvider is one OpenAI-compatible endpoint the NTC agent plugins
// can route to (Mac Ollama, LM Studio, Groq, Gemini, custom). The
// actual model weights live behind BaseURL — this row is pure address
// book. APIKeyEnv (if set) names an env var on the NTC host that the
// gateway reads at spawn time and forwards as Bearer token; the key
// value itself never enters the DB.
type LLMProvider struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"displayName"`
	ProviderType string    `json:"providerType"` // ollama | lmstudio | openai-compat | groq | gemini | custom
	BaseURL      string    `json:"baseUrl"`
	APIKeyEnv    string    `json:"apiKeyEnv"`
	Description  string    `json:"description"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

const llmProviderCols = `id, name, display_name, provider_type, base_url, api_key_env, description, enabled, created_at, updated_at`

func (d *DB) CreateLLMProvider(ctx context.Context, p LLMProvider) (LLMProvider, error) {
	if p.ProviderType == "" {
		p.ProviderType = "openai-compat"
	}
	row := d.Pool.QueryRow(ctx,
		`INSERT INTO llm_providers
		    (name, display_name, provider_type, base_url, api_key_env, description, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+llmProviderCols,
		p.Name, p.DisplayName, p.ProviderType, p.BaseURL, p.APIKeyEnv, p.Description, p.Enabled,
	)
	return scanLLMProvider(row)
}

func (d *DB) GetLLMProvider(ctx context.Context, id string) (LLMProvider, error) {
	row := d.Pool.QueryRow(ctx,
		`SELECT `+llmProviderCols+` FROM llm_providers WHERE id = $1`, id)
	return scanLLMProvider(row)
}

func (d *DB) ListLLMProviders(ctx context.Context) ([]LLMProvider, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT `+llmProviderCols+` FROM llm_providers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("store: list llm providers: %w", err)
	}
	defer rows.Close()

	out := []LLMProvider{}
	for rows.Next() {
		p, err := scanLLMProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (d *DB) UpdateLLMProvider(ctx context.Context, id string, p LLMProvider) (LLMProvider, error) {
	if p.ProviderType == "" {
		p.ProviderType = "openai-compat"
	}
	row := d.Pool.QueryRow(ctx,
		`UPDATE llm_providers SET
		    name = $2, display_name = $3, provider_type = $4, base_url = $5,
		    api_key_env = $6, description = $7, enabled = $8, updated_at = now()
		 WHERE id = $1
		 RETURNING `+llmProviderCols,
		id, p.Name, p.DisplayName, p.ProviderType, p.BaseURL,
		p.APIKeyEnv, p.Description, p.Enabled,
	)
	return scanLLMProvider(row)
}

func (d *DB) SetLLMProviderEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE llm_providers SET enabled = $2, updated_at = now() WHERE id = $1`,
		id, enabled)
	if err != nil {
		return fmt.Errorf("store: set llm provider enabled: %w", err)
	}
	return nil
}

func (d *DB) DeleteLLMProvider(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM llm_providers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete llm provider: %w", err)
	}
	return nil
}

func scanLLMProvider(s scannable) (LLMProvider, error) {
	var p LLMProvider
	err := s.Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.ProviderType, &p.BaseURL,
		&p.APIKeyEnv, &p.Description, &p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return LLMProvider{}, fmt.Errorf("store: llm provider not found")
		}
		return LLMProvider{}, fmt.Errorf("store: scan llm provider: %w", err)
	}
	return p, nil
}
