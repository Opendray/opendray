# General

The General section combines the gateway's network binding with
the operator account. Two sub-groups live here:

- **Network** — `listen` address (host:port the HTTP server
  binds to)
- **Operator account** — login user + password + bearer token
  TTL

All four fields require a **Restart** for the new value to take
effect. The page surfaces a yellow "restart required" pill the
moment any of them goes dirty.

## Network

| Field | toml key | Notes |
|---|---|---|
| Listen address | `listen` | `0.0.0.0:8770` exposes on every interface; `127.0.0.1:8770` is loopback-only. |

If you change the port and your reverse proxy / browser is
still pointed at the old one, the page reload after restart
will fail. Either fix the proxy first or note the new URL
before saving.

## Operator account

| Field | toml key | Notes |
|---|---|---|
| Username | `admin.user` | Login name accepted by `/login`. Changing it forces re-login. |
| Password | `admin.password` | Masked. Leave blank to keep current; type a value to overwrite. |
| Token TTL | `admin.token_ttl` | Bearer-token lifetime as a Go duration (`24h`, `30m`). Empty = never expire. |

### Password storage

The password is stored as plain text in `config.toml` so the
gateway can validate `/login` requests without a separate hash
DB. Two consequences:

1. **`config.toml` permissions matter.** Make sure it's mode
   `600` (the gateway writes it back as `0o600` on every Save,
   but if you copied it from elsewhere check the perms).
2. **The `Reveal` toggle** lets you see what you typed before
   saving. The browser never receives the existing password
   from the server — only what you type into the field.

### Forgot the password?

Edit `config.toml` directly:

```toml
[admin]
user     = "admin"
password = "new-temporary-password"
```

Save the file, restart opendray (`pkill -f "opendray serve"` +
the same launch command), log in, then rotate via the UI.

### Confirm dialog on dangerous changes

Saving the General section after touching `listen`, `admin.user`,
or `admin.password` triggers a confirm dialog warning that
re-authentication or a different URL may be required. This is
specifically because the post-restart browser may not be able
to reach the running gateway with the credentials it has cached.

## Bearer tokens vs integration API keys

The Token TTL applies to the **operator's** browser session
bearer token (issued by `/login`). It does **not** affect:

- **Integration API keys** — managed under Integrations, never
  expire by default
- **Channel webhooks** — verified per-platform (Telegram bot
  token, Slack signing secret, etc.)
- **Claude account OAuth tokens** — managed under Providers →
  Claude accounts

So setting `token_ttl = "1h"` only logs YOU out every hour; it
doesn't affect bots or webhooks.
