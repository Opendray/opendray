package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrConsentNotFound is returned by GetConsent when no consent row exists
// for the requested plugin name. Use errors.Is to check.
var ErrConsentNotFound = errors.New("plugin consent not found")

// PluginConsent is the granted-permissions row for an installed plugin,
// pinned to the manifest hash the user actually consented to.
type PluginConsent struct {
	PluginName   string
	ManifestHash string
	PermsJSON    json.RawMessage // opaque — full PermissionsV1 as granted
	GrantedAt    time.Time
	UpdatedAt    time.Time
}

// AuditEntry is one row in plugin_audit. Append-only.
//
// Caps is the capability set asserted by the call (e.g. ["exec"]).
// Result is "ok" | "denied" | "error".
// Ns is the bridge namespace ("exec" | "fs" | "http" | "install" | "command").
// ArgsHash is a sha256-prefix of the call args so we can correlate without
// logging secrets. Message is optional free-form text (errors, deny reason).
type AuditEntry struct {
	Ts         time.Time
	PluginName string
	Ns         string
	Method     string
	Caps       []string
	Result     string
	DurationMs int
	ArgsHash   string
	Message    string
}

// UpsertConsent inserts or updates the consent row. updated_at is bumped on
// every upsert; granted_at stays at the first grant timestamp.
func (d *DB) UpsertConsent(ctx context.Context, c PluginConsent) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO plugin_consents (plugin_name, manifest_hash, perms_json)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (plugin_name) DO UPDATE
		     SET manifest_hash = EXCLUDED.manifest_hash,
		         perms_json    = EXCLUDED.perms_json,
		         updated_at    = now()`,
		c.PluginName, c.ManifestHash, []byte(c.PermsJSON),
	)
	if err != nil {
		return fmt.Errorf("store: upsert consent: %w", err)
	}
	return nil
}

// GetConsent returns the stored consent for a plugin. Returns ErrConsentNotFound
// (wrapped) when no row exists; all other DB errors are wrapped separately so
// callers can distinguish them with errors.Is(err, ErrConsentNotFound).
func (d *DB) GetConsent(ctx context.Context, name string) (PluginConsent, error) {
	row := d.Pool.QueryRow(ctx,
		`SELECT plugin_name, manifest_hash, perms_json, granted_at, updated_at
		 FROM plugin_consents
		 WHERE plugin_name = $1`,
		name,
	)

	var c PluginConsent
	var permsBytes []byte
	err := row.Scan(&c.PluginName, &c.ManifestHash, &permsBytes, &c.GrantedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PluginConsent{}, fmt.Errorf("store: get consent %q: %w", name, ErrConsentNotFound)
		}
		return PluginConsent{}, fmt.Errorf("store: get consent: %w", err)
	}
	c.PermsJSON = json.RawMessage(permsBytes)
	return c, nil
}

// DeleteConsent removes a consent row. Idempotent — deleting a non-existent
// row is not an error.
func (d *DB) DeleteConsent(ctx context.Context, name string) error {
	_, err := d.Pool.Exec(ctx,
		`DELETE FROM plugin_consents WHERE plugin_name = $1`, name)
	if err != nil {
		return fmt.Errorf("store: delete consent: %w", err)
	}
	return nil
}

// AppendAudit writes one audit row. Must not fail silently — audit is
// load-bearing for security review and post-incident forensics.
// All fields are passed as parameterised values; no string interpolation.
func (d *DB) AppendAudit(ctx context.Context, e AuditEntry) error {
	caps := e.Caps
	if caps == nil {
		caps = []string{}
	}
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO plugin_audit
		     (plugin_name, ns, method, caps, result, duration_ms, args_hash, message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.PluginName, e.Ns, e.Method, caps, e.Result, e.DurationMs, e.ArgsHash, e.Message,
	)
	if err != nil {
		return fmt.Errorf("store: append audit: %w", err)
	}
	return nil
}

// TailAudit returns the most-recent limit audit rows for a plugin, newest-first.
//
// The limit is clamped server-side to [1, 1000] to prevent pagination abuse:
//   - limit <= 0 is clamped to 1
//   - limit > 1000 is clamped to 1000
func (d *DB) TailAudit(ctx context.Context, name string, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 1
	}
	if limit > 1000 {
		limit = 1000
	}

	rows, err := d.Pool.Query(ctx,
		`SELECT ts, plugin_name, ns, method, caps, result, duration_ms, args_hash,
		        COALESCE(message, '')
		 FROM plugin_audit
		 WHERE plugin_name = $1
		 ORDER BY ts DESC, id DESC
		 LIMIT $2`,
		name, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: tail audit: %w", err)
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var caps []string
		if err := rows.Scan(
			&e.Ts, &e.PluginName, &e.Ns, &e.Method, &caps,
			&e.Result, &e.DurationMs, &e.ArgsHash, &e.Message,
		); err != nil {
			return nil, fmt.Errorf("store: tail audit scan: %w", err)
		}
		if caps == nil {
			caps = []string{}
		}
		e.Caps = caps
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: tail audit rows: %w", err)
	}
	return out, nil
}
