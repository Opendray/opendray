package catalog

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a provider id does not exist in the
// embedded manifest set.
var ErrNotFound = errors.New("provider not found")

// Catalog is the read-mostly facade over the embedded manifests and
// the per-id user state in the providers table.
type Catalog struct {
	log       *slog.Logger
	store     *catalogStore
	manifests map[string]Manifest
	hashes    map[string]string
	mu        sync.RWMutex
}

// New loads the embedded manifests but does not touch the DB. Call
// Sync once during startup to upsert the provider rows.
func New(pool *pgxpool.Pool, log *slog.Logger) (*Catalog, error) {
	if log == nil {
		log = slog.Default()
	}
	manifests, hashes, err := LoadBuiltin()
	if err != nil {
		return nil, err
	}
	return &Catalog{
		log:       log.With("component", "catalog"),
		store:     newStore(pool),
		manifests: manifests,
		hashes:    hashes,
	}, nil
}

// Sync upserts every embedded manifest into the providers table. Run
// once at startup so the FK from sessions.provider_id is satisfied
// and existing rows pick up manifest changes via the hash field.
func (c *Catalog) Sync(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for id, hash := range c.hashes {
		if err := c.store.Upsert(ctx, id, hash); err != nil {
			return err
		}
	}
	c.log.Info("catalog synced", "count", len(c.manifests))
	return nil
}

// Provider is the public view returned by REST.
type Provider struct {
	Manifest     Manifest       `json:"manifest"`
	ManifestHash string         `json:"manifest_hash"`
	Config       map[string]any `json:"config"`
	Enabled      bool           `json:"enabled"`
	// Runtime is the live probed CLI state (installed?, real version).
	// Set by the HTTP handler, not the store — nil when not probed.
	Runtime *RuntimeInfo `json:"runtime,omitempty"`
}

func (c *Catalog) Get(ctx context.Context, id string) (Provider, error) {
	c.mu.RLock()
	m, ok := c.manifests[id]
	hash := c.hashes[id]
	c.mu.RUnlock()
	if !ok {
		return Provider{}, ErrNotFound
	}
	row, err := c.store.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		// Manifest exists in embed but DB row missing (Sync not yet
		// run). Return defaults so callers can still render the UI.
		return Provider{Manifest: m, ManifestHash: hash, Config: map[string]any{}, Enabled: true}, nil
	}
	if err != nil {
		return Provider{}, err
	}
	return Provider{
		Manifest:     m,
		ManifestHash: row.ManifestHash,
		Config:       row.Config,
		Enabled:      row.Enabled,
	}, nil
}

func (c *Catalog) List(ctx context.Context) ([]Provider, error) {
	c.mu.RLock()
	ids := SortedIDs(c.manifests)
	c.mu.RUnlock()
	out := make([]Provider, 0, len(ids))
	for _, id := range ids {
		p, err := c.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (c *Catalog) UpdateConfig(ctx context.Context, id string, cfg map[string]any) error {
	c.mu.RLock()
	_, ok := c.manifests[id]
	c.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	return c.store.UpdateConfig(ctx, id, cfg)
}

func (c *Catalog) Toggle(ctx context.Context, id string, enabled bool) error {
	c.mu.RLock()
	_, ok := c.manifests[id]
	c.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	return c.store.UpdateEnabled(ctx, id, enabled)
}
