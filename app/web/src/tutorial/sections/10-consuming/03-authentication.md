# Authentication

opendray accepts **Bearer tokens** of two kinds on every protected
endpoint. The middleware checks them in this order:

1. **Admin token** — issued by `POST /api/v1/auth/login`. Full
   access to every endpoint. Lifetime configured by
   `[admin].token_ttl` in `config.toml` (default 24h, empty = no
   expiry). Used by the web UI and by your one-off setup scripts.
2. **Integration API key** — minted by `POST /api/v1/integrations`,
   plaintext shown **once**. Scoped to the integration's allowed
   API surface. No lifetime — they live until rotated or deleted.

Both go in the `Authorization` header:

```
Authorization: Bearer <token>
```

For browser WebSockets that can't add headers, opendray also
accepts `?token=<token>` query parameter on the WS upgrade request.

## How identity is resolved

```
HTTP request lands  →  CombinedMiddleware (admin first, integration
                       fallback) → resolves Principal:
                       { kind: admin | integration, id, scopes }
                                  ↓
                       per-endpoint scope check (if integration)
                                  ↓
                       handler runs; integration_id flows into the
                       call log (see Activity → Call log)
```

For integrations specifically: the gateway iterates every enabled
integration's `api_key_hash`, bcrypt-comparing against the bearer.
First match wins. After the first match a tiny in-memory cache
maps plaintext → integration_id so subsequent requests skip
bcrypt; this cache is cleared on rotate.

## Where to store your API key

| Environment | Recommended store |
|---|---|
| Local CLI tool | `~/.config/<your-app>/credentials` (mode 0600) |
| macOS desktop app | Keychain (`security` CLI, `node-keytar`) |
| Linux desktop app | libsecret / GNOME Keyring |
| Linux daemon | systemd `LoadCredential=` or file mode 0600 |
| Server / container | AWS Secrets Manager, GCP Secret Manager, Vault |
| CI / CD | encrypted env var (GitHub Actions secrets etc.) |

The reference [demo-client](#consuming-typescript-sdk) uses a flat
JSON file (`.demo-state.json`, mode 0600, gitignored) because it's
a single-machine learning example. Don't ship that into production.

## Authorization failures

| Status | Meaning |
|---|---|
| `401 unauthorized` | Bearer missing, malformed, or no integration's hash matches. |
| `401` (after a working call) | Key was rotated server-side. Recover by getting a fresh key (see [Key rotation](#consuming-key-rotation)). |
| `403 forbidden` | Authenticated but the integration's scopes don't include this endpoint. |
| `503 service unavailable` | The integration row is disabled or `health_status=unhealthy` (proxy mode only). |

The failing JSON body always has `{"error": "<reason>"}` so you can
log the underlying cause without parsing prose.

## CORS + browser callers

opendray's `CheckOrigin` is permissive — every origin is allowed
on the WebSocket upgrade. The HTTP server doesn't add CORS headers
either (no `Access-Control-Allow-Origin`), so calling directly
from a browser at a different origin will fail preflight unless
you put a reverse proxy in front. For now: only call opendray
from the **same origin** the admin UI is served from, or from a
server-side runtime.
