# Troubleshooting

When memory misbehaves, this is where to look.

## Symptom: agent never calls memory tools

**Likely cause**: the system-prompt guidance wasn't injected, or
the MCP server didn't get auto-attached.

**Check**:
```bash
ls /var/folders/.../opendray-sess-<your-session-id>/
cat /var/folders/.../opendray-sess-<your-session-id>/claude-mcp.json
```

You should see an `opendray-memory` entry. If not:

1. Was the session spawned BEFORE memory was enabled? Restart the
   session.
2. Does the provider support MCP? Check
   `internal/catalog/builtin/<provider>.json: capabilities.supportsMcp`.
3. Did `app.New` log `memory MCP auto-attach enabled` at startup?
   If the line is missing, the integration key wasn't minted —
   check for `init memory:` errors in `/tmp/opendray.log`.

## Symptom: `tool error: 401 unauthorized`

**Cause**: the API key in the session's `mcp.json` is stale (e.g.
opendray was bounced and the cache file was wiped). The agent's
mcp-memory subprocess can't authenticate.

**Recovery**: end + restart the session. opendray re-renders
`mcp.json` with the live cached key.

If 401 persists across a restart:
```bash
rm ~/.opendray/memory.key
pkill -f "opendray serve"
go run ./cmd/opendray serve -config config.toml
```
This forces a fresh rotate-and-cache cycle.

## Symptom: search returns no hits even though I just stored

**Likely causes**:

1. **BM25 token mismatch**. Sparse hash vectors only match exact
   tokens. `query="package manager"` against
   `text="opendray prefers pnpm"` returns 0 because no shared
   words. Try a query that contains a literal word from the stored
   text. Solution: switch to `backend = "http"` with a real
   embedder.
2. **Wrong scope_key**. Inspector → set `scope_key` to the cwd
   you stored under. Different cwd = different scope = no hits.
3. **Threshold too high**. Settings → Memory → Similarity
   threshold. Default is 0.1. Set to 0 (or `min_similarity=-1` in
   the API call) to see every hit ranked.

## Symptom: memories I deleted in Inspector keep coming back

**Cause**: the mirror is re-ingesting them from Claude's local
`.md` files on the next session spawn.

**Recovery**:
```bash
rm ~/.claude/projects/-encoded-cwd/memory/<topic>.md
rm ~/.claude-accounts/*/projects/-encoded-cwd/memory/<topic>.md
```

Then re-delete the row in Inspector. Phase 2 may add a "blocklist"
that tells the mirror to skip specific source paths.

## Symptom: opendray won't start, log says `pgvector` related error

**Cause**: pgvector extension not installed in the `opendray_v2`
database, or installed but binary missing on the PG container.

**Recovery**:
```bash
ssh -i ~/.ssh/home_lab_key root@<pg-host> \
  "docker exec mypostgresql_container psql -U <superuser> -d opendray_v2 \
    -c 'CREATE EXTENSION IF NOT EXISTS vector;'"
```

If `CREATE EXTENSION` fails with "could not open extension control file",
the pgvector binary needs to be installed inside the container — see
`docs/setup/pgvector.md` for the rebuild path.

## Symptom: `memory ready` log line missing entirely

**Cause**: `app.New` returned an error from `init memory: …`.

**Check**: scroll up in the log. Common reasons:

- `database url` empty → `[database.url]` not configured
- migration 0011 didn't run yet → `go run ./cmd/opendray migrate -config config.toml`
- DB unreachable → check Postgres is up at the configured URL

## Symptom: `gateway returned 502/503 from mcp-memory`

**Cause**: the `mcp-memory` subprocess can't reach opendray's
gateway. Either the gateway crashed, or the BASE_URL it's calling
is wrong.

**Check** the rendered `mcp.json`:

```bash
cat /var/folders/.../opendray-sess-<id>/claude-mcp.json | jq '.mcpServers["opendray-memory"].env'
```

`OPENDRAY_BASE_URL` should be `http://127.0.0.1:<your-port>`. If
opendray is bound to `0.0.0.0`, opendray writes `127.0.0.1`
internally so subprocesses can reach it. If the URL is wrong,
restart opendray (it re-derives from `[listen]` on startup).

## Symptom: 401 on /memory/* even with admin token

**Cause**: routes moved between admin-only and dual-auth groups
during phase 2. Old admin clients hitting `/admin/memory/*` get
404; new path is `/memory/*`.

**Update**: any custom scripts using the old path should switch to
`/api/v1/memory/*`. The `mcp-memory` subprocess already uses the
new path.

## Recovery escape hatch: nuke everything

If memory is in a really weird state and you just want a clean
slate:

```sql
DELETE FROM memories;
DELETE FROM memory_index_state;
```

Then `rm ~/.opendray/memory.key` and restart opendray. You'll
lose all stored memories but keep the integration row + UI
configuration.

To also remove the `opendray-memory` integration row (forces a
brand-new register on next start):

```bash
# Web UI: Integrations → opendray-memory → Delete
# Or:
TOKEN=$(curl -s -X POST .../auth/login ... | jq -r .token)
ID=$(curl -s ... /integrations | jq -r '.integrations[]|select(.name=="opendray-memory").id')
curl -X DELETE -H "Authorization: Bearer $TOKEN" .../integrations/$ID
```

## When to file a bug

If the symptom isn't here, file at
<https://github.com/Opendray/opendray_v2/issues> with:

1. Output of `tail -100 /tmp/opendray.log`
2. The session's `mcp.json` (redact api_key first)
3. The agent's tool-call output if it surfaced an error
4. SQL: `SELECT count(*), embedder FROM memories GROUP BY embedder;`
