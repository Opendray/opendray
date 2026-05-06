# Security Policy

## Threat Model

OpenDray spawns pseudo-terminal (PTY) sessions that run AI coding CLIs (Claude
Code, Codex, Gemini, OpenCode, shell) **with the same privileges as the
OpenDray process**. A PTY is root-equivalent for the user running the server.

Concretely:

- Any authenticated admin can execute arbitrary commands on the host.
- The WebSocket terminal stream carries raw PTY I/O; intercepting it is
  equivalent to a shell session hijack.
- Integration API keys can proxy arbitrary HTTP traffic through OpenDray to
  any registered upstream — operators choose what gets registered.

**Always run OpenDray behind authentication.** It is not designed to be
exposed to the public internet without a reverse proxy and TLS.

## Default Security Posture

| Setting | Default | Notes |
|---|---|---|
| `listen` | `127.0.0.1:8770` | Loopback only; safe for local dev. |
| Admin auth | Bearer token, 24h TTL | Issued by `/api/v1/auth/login` against `[admin]` config. |
| Admin password | Constant-time compared to plaintext value in config | Single-admin self-host model. Prefer `OPENDRAY_ADMIN_PASSWORD` env over committing to `config.toml`. |
| Integration API keys | bcrypt-hashed in DB | One key per integration; revocable. |
| Backup encryption | AES-256-GCM, PBKDF2-HMAC-SHA256, 200,000 iterations | Key derived from `OPENDRAY_BACKUP_KEY`; **must** live in env, never in config. |
| WebSocket auth | Bearer token in query parameter | All WS endpoints require auth before any data flows. |
| Database | External Postgres (no bundled mode) | Operator chooses TLS posture via `?sslmode=` in DSN. |

When `[admin].password` is unset (and no env override), the server **rejects
every login attempt** rather than booting with no auth.

## Deployment Checklist

1. **Reverse proxy.** Place OpenDray behind Cloudflare Tunnel, nginx, Caddy,
   or Traefik. Terminate TLS at the proxy. Keep `listen = "127.0.0.1:8770"`.
2. **Strong admin password.** ≥ 16 random characters. Prefer
   `OPENDRAY_ADMIN_PASSWORD` env var over `config.toml`.
3. **Backup key in env only.** Set `OPENDRAY_BACKUP_KEY` via the OS secret
   manager, systemd `LoadCredential`, or Vault. Never commit it. Losing the
   key makes encrypted backups unrecoverable.
4. **Postgres SSL.** Use `sslmode=verify-full` with a non-localhost database.
5. **Least privilege.** Run OpenDray as a dedicated unprivileged user.
6. **Token TTL.** Keep `[admin].token_ttl ≤ 24h`. Rotate on compromise via
   server restart (active tokens are invalidated).
7. **Cloudflare Tunnel binding.** Bind to `127.0.0.1`, not `0.0.0.0` —
   `middleware.RealIP` trusts `X-Forwarded-For`, which is spoofable on a
   public bind.

## Reporting a Vulnerability

If you discover a security vulnerability in OpenDray, please report it
privately. Do **not** open a public GitHub issue.

- **GitHub Security Advisories:** [Open a private advisory](https://github.com/Opendray/opendray_v2/security/advisories)
- **Email:** security@opendray.dev

We aim to acknowledge reports within 48 hours and ship fixes for critical
issues within 7 days.

## Supported Versions

| Version | Supported |
|---|---|
| v1.0-rc onward | ✅ |
| v1 (`Opendray/opendray`) | Maintenance only — security fixes for the cutover quarter, then archived per ADR 0001. |

## CVE History

None as of v1.0-rc.
