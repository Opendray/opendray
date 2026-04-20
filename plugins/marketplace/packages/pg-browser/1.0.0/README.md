# pg-browser — PostgreSQL Browser (v1 sidecar)

A v1 rewrite of OpenDray's legacy in-process database panel.
Everything that used to live inside the gateway binary — the `pg`
connection, the SQL, the custom HTTP routes — is now a Node sidecar
that the platform installs, permission-gates, and sandboxes like any
third-party plugin.

## Commands

All commands are zero-argument and return JSON:

| Command | Method | Returns |
|---|---|---|
| Postgres: Connection info | `info` | `{status, config, serverVersion}` |
| Postgres: List databases | `listDatabases` | `{count, databases: [{name, size}]}` |
| Postgres: List schemas | `listSchemas` | `{count, schemas: [{name, owner}]}` |
| Postgres: List tables (public schema) | `listTables` | `{count, tables: [{schema, name, type}]}` |
| Postgres: Sample SELECT | `sampleQuery` | `{sql, rowCount, rows}` |

All queries are read-only. `listDatabases` / `listSchemas` /
`listTables` hit pg_catalog; `sampleQuery` runs one `SELECT COUNT(*)
GROUP BY schemaname` against `pg_catalog.pg_tables`.

## Configuration

Connection parameters are set through `manifest.host.env`:

```json
{
  "host": {
    "env": {
      "PG_HOST": "db.example.com",
      "PG_PORT": "5432",
      "PG_USER": "readonly_user",
      "PG_PASSWORD": "...",
      "PG_DATABASE": "postgres",
      "PG_SSLMODE": "disable"
    }
  }
}
```

The supervisor filters host env aggressively (only `PATH`, `HOME`,
`USER`, `LANG`, `TMPDIR`, plus whatever `host.env` declares) so
arbitrary host secrets can't leak into the sidecar.

The committed manifest in this repo uses `__REPLACE_ME__` placeholders
— the marketplace operator rewrites these with real values before
publishing. The plugin refuses to connect while any `__REPLACE_ME__`
is still present, so a forgotten edit surfaces immediately in the
`info` command.

## Security model

- `secret: true` permission is declared in the manifest to signal
  intent (storing connection-related secrets via the secret cap in a
  future version). No other capabilities are requested.
- All queries are read-only by convention. A dedicated read-only
  role on the target Postgres is still strongly recommended — the
  plugin trusts its `host.env` creds.
- The sidecar never opens a listening socket. Communication is
  JSON-RPC 2.0 over stdio with LSP Content-Length framing (same as
  `fs-readme`).

## Why this replaces the legacy panel

The old `plugins/panels/database/` plugin was a manifest-only shell
— all logic lived in-process at `gateway/database/postgres.go` with
custom `/api/database/{plugin}/*` HTTP routes and a bespoke
`features/database/database_page.dart` UI. It was not installable,
not sandboxed, and not extensible.

This version is a self-contained bundle installed through the
standard marketplace pipeline. Delete it via uninstall → the sidecar
stops, the DB connection closes, and the on-disk bundle is removed.
No gateway-side cleanup, no schema migration.

## Roadmap

- **Configuration UI**: a `configure` command that takes args and
  writes to the `storage` cap, so the user doesn't need to edit
  `host.env` manually. Depends on a Flutter-side arg-taking command
  runner.
- **Query UI**: `form: webview` rendering a SQL editor + result
  table. The current command-palette-only flow is sufficient for
  demoing the v1 architecture but isn't a replacement for the legacy
  panel's interactive browsing experience.
