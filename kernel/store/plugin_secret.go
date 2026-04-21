package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Package-level notes on plugin_secret + plugin_secret_kek:
//
// plugin_secret holds one row per (plugin, key) with ciphertext + nonce
// (AES-GCM). plugin_secret_kek holds one row per plugin with the wrapped
// DEK under the host KEK. This store layer is intentionally crypto-agnostic:
// the bridge.SecretAPI (T13) owns wrap/unwrap/encrypt/decrypt semantics and
// passes opaque bytes through these functions.
//
// Migration 012 created plugin_secret (ciphertext only). Migration 016 added
// the nonce column with DEFAULT ''::bytea so any pre-M3 rows (there are
// none — the namespace was scaffolded but never written to) remain
// identifiable. SecretSet always writes a non-empty nonce.
//
// Migration 014 created plugin_secret_kek.

// SecretGet fetches one encrypted value by (plugin, key). Returns
// (nil, nil, false, nil) when the key is absent.
func (d *DB) SecretGet(ctx context.Context, plugin, key string) (ciphertext, nonce []byte, found bool, err error) {
	err = d.Pool.QueryRow(ctx,
		`SELECT ciphertext, nonce FROM plugin_secret WHERE plugin_name = $1 AND key = $2`,
		plugin, key,
	).Scan(&ciphertext, &nonce)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("store: secret get %q/%q: %w", plugin, key, err)
	}
	return ciphertext, nonce, true, nil
}

// SecretSet upserts the encrypted value for (plugin, key). ciphertext and
// nonce are opaque — the bridge layer owns crypto. Caller MUST pass a
// non-empty nonce; we enforce that here as a defence against bridge-layer
// bugs that would otherwise produce unrecoverable rows.
func (d *DB) SecretSet(ctx context.Context, plugin, key string, ciphertext, nonce []byte) error {
	if len(nonce) == 0 {
		return fmt.Errorf("store: secret set %q/%q: nonce is required", plugin, key)
	}
	if len(ciphertext) == 0 {
		return fmt.Errorf("store: secret set %q/%q: ciphertext is required", plugin, key)
	}
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO plugin_secret (plugin_name, key, ciphertext, nonce, updated_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (plugin_name, key) DO UPDATE
		     SET ciphertext = EXCLUDED.ciphertext,
		         nonce      = EXCLUDED.nonce,
		         updated_at = now()`,
		plugin, key, ciphertext, nonce,
	)
	if err != nil {
		return fmt.Errorf("store: secret set %q/%q: %w", plugin, key, err)
	}
	return nil
}

// SecretDelete removes one secret row. Idempotent — missing key returns nil.
func (d *DB) SecretDelete(ctx context.Context, plugin, key string) error {
	_, err := d.Pool.Exec(ctx,
		`DELETE FROM plugin_secret WHERE plugin_name = $1 AND key = $2`,
		plugin, key,
	)
	if err != nil {
		return fmt.Errorf("store: secret delete %q/%q: %w", plugin, key, err)
	}
	return nil
}

// SecretList returns all keys for plugin, ascending. Never returns values.
func (d *DB) SecretList(ctx context.Context, plugin string) ([]string, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT key FROM plugin_secret WHERE plugin_name = $1 ORDER BY key ASC LIMIT 1000`,
		plugin,
	)
	if err != nil {
		return nil, fmt.Errorf("store: secret list %q: %w", plugin, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("store: secret list scan: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: secret list rows: %w", err)
	}
	return out, nil
}

// EnsureKEKRow upserts the wrapped DEK + kid for plugin. Called on first
// SecretSet for a plugin that has no KEK row yet, and again on KEK rotation.
func (d *DB) EnsureKEKRow(ctx context.Context, plugin string, wrappedDEK []byte, kid string) error {
	if len(wrappedDEK) == 0 {
		return fmt.Errorf("store: ensure kek row %q: wrappedDEK is required", plugin)
	}
	if kid == "" {
		return fmt.Errorf("store: ensure kek row %q: kid is required", plugin)
	}
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO plugin_secret_kek (plugin_name, wrapped_dek, kek_kid, created_at, updated_at)
		 VALUES ($1, $2, $3, now(), now())
		 ON CONFLICT (plugin_name) DO UPDATE
		     SET wrapped_dek = EXCLUDED.wrapped_dek,
		         kek_kid     = EXCLUDED.kek_kid,
		         updated_at  = now()`,
		plugin, wrappedDEK, kid,
	)
	if err != nil {
		return fmt.Errorf("store: ensure kek row %q: %w", plugin, err)
	}
	return nil
}

// GetWrappedDEK returns the plugin's wrapped DEK and kek_kid. Returns
// (nil, "", pgx.ErrNoRows-wrapped-error) when no row exists — callers use
// errors.Is(err, pgx.ErrNoRows) to branch into first-write flow.
func (d *DB) GetWrappedDEK(ctx context.Context, plugin string) (wrapped []byte, kid string, err error) {
	err = d.Pool.QueryRow(ctx,
		`SELECT wrapped_dek, kek_kid FROM plugin_secret_kek WHERE plugin_name = $1`,
		plugin,
	).Scan(&wrapped, &kid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", pgx.ErrNoRows
		}
		return nil, "", fmt.Errorf("store: get wrapped dek %q: %w", plugin, err)
	}
	return wrapped, kid, nil
}
