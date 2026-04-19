package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Per-key and per-plugin storage quotas. Callers above this layer
// should rely on KVSet's ErrQuota* returns rather than re-checking.
const (
	MaxValueBytes     = 1 << 20    // 1 MiB per key
	MaxPerPluginBytes = 100 << 20  // 100 MiB total per plugin
)

// Sentinels — use errors.Is.
var (
	ErrValueTooLarge       = errors.New("store: plugin_kv value exceeds 1 MiB per key")
	ErrPluginQuotaExceeded = errors.New("store: plugin_kv quota exceeded (100 MiB per plugin)")
)

// PluginKV is one row in plugin_kv (migration 011).
type PluginKV struct {
	PluginName string
	Key        string
	Value      json.RawMessage
	SizeBytes  int
	UpdatedAt  time.Time
}

// KVGet fetches a key. Returns (nil, false, nil) when the key is
// absent so callers can distinguish "not found" from errors.
func (d *DB) KVGet(ctx context.Context, pluginName, key string) (json.RawMessage, bool, error) {
	var raw []byte
	err := d.Pool.QueryRow(ctx,
		`SELECT value FROM plugin_kv WHERE plugin_name = $1 AND key = $2`,
		pluginName, key,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("store: kv get %q/%q: %w", pluginName, key, err)
	}
	return json.RawMessage(raw), true, nil
}

// KVSet upserts. Per-key quota (1 MiB) is checked against len(value)
// BEFORE any DB call. Per-plugin quota (100 MiB) is enforced by a
// pre-UPSERT SELECT for the plugin's current total — the subsequent
// UPSERT holds no lock that would block concurrent sets, so this is
// best-effort: race detector + t.Parallel-safe, NOT absolute
// (atomic check-and-set is M6).
//
// Errors: ErrValueTooLarge, ErrPluginQuotaExceeded, plus wrapped DB errors.
func (d *DB) KVSet(ctx context.Context, pluginName, key string, value json.RawMessage) error {
	// Per-key quota: checked before any DB work.
	if len(value) > MaxValueBytes {
		return ErrValueTooLarge
	}

	// Per-plugin quota: best-effort SELECT of current total.
	// We subtract any existing size for this key (the upsert will replace it)
	// so we don't double-count the old value.
	var currentTotal int
	err := d.Pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(size_bytes), 0) FROM plugin_kv
		 WHERE plugin_name = $1 AND key != $2`,
		pluginName, key,
	).Scan(&currentTotal)
	if err != nil {
		return fmt.Errorf("store: kv quota check %q: %w", pluginName, err)
	}

	if currentTotal+len(value) >= MaxPerPluginBytes {
		return ErrPluginQuotaExceeded
	}

	_, err = d.Pool.Exec(ctx,
		`INSERT INTO plugin_kv (plugin_name, key, value, size_bytes, updated_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (plugin_name, key) DO UPDATE
		     SET value      = EXCLUDED.value,
		         size_bytes = EXCLUDED.size_bytes,
		         updated_at = now()`,
		pluginName, key, []byte(value), len(value),
	)
	if err != nil {
		return fmt.Errorf("store: kv set %q/%q: %w", pluginName, key, err)
	}
	return nil
}

// KVDelete removes one key. Idempotent — missing key returns nil.
func (d *DB) KVDelete(ctx context.Context, pluginName, key string) error {
	_, err := d.Pool.Exec(ctx,
		`DELETE FROM plugin_kv WHERE plugin_name = $1 AND key = $2`,
		pluginName, key,
	)
	if err != nil {
		return fmt.Errorf("store: kv delete %q/%q: %w", pluginName, key, err)
	}
	return nil
}

// KVList returns keys under the given prefix (prefix="" lists everything).
// Sorted ascending. Caller should paginate above this layer; KVList
// caps hard at 1000 keys per call for safety.
func (d *DB) KVList(ctx context.Context, pluginName, prefix string) ([]string, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT key FROM plugin_kv
		 WHERE plugin_name = $1 AND key LIKE $2 || '%'
		 ORDER BY key ASC
		 LIMIT 1000`,
		pluginName, prefix,
	)
	if err != nil {
		return nil, fmt.Errorf("store: kv list %q prefix=%q: %w", pluginName, prefix, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("store: kv list scan: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: kv list rows: %w", err)
	}
	return out, nil
}
