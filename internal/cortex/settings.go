package cortex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// ─── Cortex runtime settings (Phase: AI-drive optimization) ─────
//
// Operator-tunable knobs editable from the Cortex settings page.
// Stored as KV rows (0052) so new knobs don't need migrations.

// Spawn modes.
const (
	SpawnModeFull = "full" // inject everything inject-flagged, in full
	SpawnModeLean = "lean" // inject guardrails + a compact index; fetch on demand
)

const settingSpawnMode = "spawn_mode"

// SettingsStore reads/writes cortex_settings.
type SettingsStore struct {
	pool *pgxpool.Pool
}

// NewSettingsStore wires the store.
func NewSettingsStore(pool *pgxpool.Pool) *SettingsStore {
	return &SettingsStore{pool: pool}
}

func (s *SettingsStore) get(ctx context.Context, key, fallback string) string {
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT value FROM cortex_settings WHERE key = $1`, key).Scan(&v)
	if err != nil || strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (s *SettingsStore) set(ctx context.Context, key, value string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cortex_settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		key, value)
	if err != nil {
		return fmt.Errorf("cortex settings: set %s: %w", key, err)
	}
	return nil
}

// SpawnMode returns the configured spawn injection mode (full|lean).
func (s *SettingsStore) SpawnMode(ctx context.Context) string {
	mode := s.get(ctx, settingSpawnMode, SpawnModeFull)
	if mode != SpawnModeLean {
		return SpawnModeFull
	}
	return SpawnModeLean
}

// SetSpawnMode persists the spawn injection mode.
func (s *SettingsStore) SetSpawnMode(ctx context.Context, mode string) error {
	if mode != SpawnModeFull && mode != SpawnModeLean {
		return fmt.Errorf("cortex settings: spawn_mode must be full|lean, got %q", mode)
	}
	return s.set(ctx, settingSpawnMode, mode)
}

// SpawnModeSource adapts the store to projectdoc's spawn-config hook.
func (s *SettingsStore) SpawnModeSource() projectdoc.SpawnModeSource {
	return func(ctx context.Context) string { return s.SpawnMode(ctx) }
}

// mountSettings registers the settings routes under the
// already-entered /cortex group:
//
//	GET /settings  → {spawn_mode}
//	PUT /settings  body: {spawn_mode} → {spawn_mode}
func (h *Handlers) mountSettings(r interface {
	Get(pattern string, handler http.HandlerFunc)
	Put(pattern string, handler http.HandlerFunc)
}) {
	r.Get("/settings", h.getSettings)
	r.Put("/settings", h.putSettings)
}

func (h *Handlers) settingsReady(w http.ResponseWriter) bool {
	if h.settings == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "settings not configured"})
		return false
	}
	return true
}

func (h *Handlers) getSettings(w http.ResponseWriter, r *http.Request) {
	if !h.settingsReady(w) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"spawn_mode": h.settings.SpawnMode(r.Context()),
	})
}

func (h *Handlers) putSettings(w http.ResponseWriter, r *http.Request) {
	if !h.settingsReady(w) {
		return
	}
	var body struct {
		SpawnMode string `json:"spawn_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if body.SpawnMode != "" {
		if err := h.settings.SetSpawnMode(r.Context(), body.SpawnMode); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"spawn_mode": h.settings.SpawnMode(r.Context()),
	})
}
