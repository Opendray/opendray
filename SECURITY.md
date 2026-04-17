# Security Policy

## Threat Model

OpenDray spawns pseudo-terminal (PTY) sessions that run AI coding CLIs (Claude Code,
Codex, etc.) with the same privileges as the OpenDray process. **A PTY is
root-equivalent** for the user running the server. This means:

- Any authenticated user can execute arbitrary commands on the host.
- The WebSocket terminal stream carries raw PTY I/O; intercepting it is
  equivalent to a shell session hijack.

**Always run OpenDray behind authentication.** It is not designed to be
exposed to the public internet without a reverse proxy and TLS.

## Default Security Posture

| Setting | Default | Notes |
|---------|---------|-------|
| `LISTEN_ADDR` | `127.0.0.1:8640` | Loopback only; safe for local dev |
| `JWT_SECRET` | *(empty)* | Server refuses to start on non-loopback without it |
| Rate limiting | 10 req/min (session ops), 60 req/min (reads) | Per-IP token bucket |
| Body size cap | 1 MB | On POST/PUT/PATCH endpoints |

When `JWT_SECRET` is unset and the bind address is not loopback, OpenDray
exits with a fatal error. This prevents accidental exposure without auth.

## Deployment Checklist

1. **Reverse proxy** -- Place OpenDray behind nginx, Caddy, or Traefik.
   Terminate TLS at the proxy.
2. **TLS everywhere** -- Never expose the WebSocket endpoint over plain HTTP
   on a network you do not fully control.
3. **Firewall** -- Restrict access to the OpenDray port. Only the reverse proxy
   should reach it.
4. **`JWT_SECRET`** -- Set a strong, random secret (>= 32 bytes). Rotate
   periodically.
5. **Least privilege** -- Run OpenDray as a dedicated non-root user. The PTY
   sessions inherit this user's permissions.
6. **Database credentials** -- Store `DB_PASSWORD` and other secrets in a
   secrets manager (HashiCorp Vault, SOPS, etc.), not in `.env` files
   committed to version control.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |

## Reporting a Vulnerability

If you discover a security vulnerability in OpenDray, please report it
responsibly:

- **Email:** [security@opendray.dev](mailto:security@opendray.dev)
- **GitHub Security Advisories:**
  [github.com/opendray/opendray/security/advisories](https://github.com/opendray/opendray/security/advisories)

We will acknowledge your report within 48 hours and aim to provide a fix or
mitigation within 7 days for critical issues.

Please do **not** open a public GitHub issue for security vulnerabilities.

## CVE History

No CVEs have been assigned as of v0.1.0.
