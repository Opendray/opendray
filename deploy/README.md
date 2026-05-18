# Deploy artefacts

Reference configurations for running OpenDray v2 outside of `go run`. Pick
one path:

| Artefact | When to use |
|---|---|
| [`systemd/opendray.service`](systemd/opendray.service) | Bare-metal, VM, or LXC. Standard Linux deploy path — sandboxed unit with `migrate`-then-`serve` boot. |
| [`launchd/com.opendray.opendray.plist`](launchd/com.opendray.opendray.plist) | macOS LaunchDaemon — Mac mini / Mac Studio as 24/7 home server. Starts at boot before any user login, restarts on crash. |
| [`lxc/proxmox-pty-notes.md`](lxc/proxmox-pty-notes.md) | Proxmox LXC specifics — PTY allocation in unprivileged containers, networking, secrets. |
| [`.goreleaser.yml`](../.goreleaser.yml) (repo root) | Cut a tagged release of pre-built binaries + checksums. |

> **Docker is intentionally not listed.** opendray is a host-resident
> gateway — it spawns AI CLIs via PTYs and shares process state
> (`~/.claude`, ssh-agent, project files) with them, which is
> incompatible with the container isolation that a production Docker
> deploy would impose. Use the systemd / launchd paths above instead.

The systemd unit and the LXC notes are designed to compose: a Proxmox
LXC running OpenDray uses both — the unit for service management,
the LXC notes for the host-side container config that lets PTYs work.

## Out of scope

- Reverse-proxy configs (nginx / Caddy / Traefik / Cloudflare Tunnel) —
  see [`docs/operator-guide.md`](../docs/operator-guide.md) §Topology.
- Postgres bootstrapping. OpenDray expects an external Postgres 15+;
  use whatever flow you already have.
- Backups *of* the OpenDray host. opendray's encrypted-DB-dump
  backup subsystem is a different layer (see operator guide).

## Verifying the deploy

After install + start, three smoke checks confirm things are working:

```bash
# 1. The binary is on PATH and prints the build banner.
opendray version

# 2. The HTTP server is up and the SPA is reachable on loopback.
curl -fsS http://127.0.0.1:8770/admin/ | head -1

# 3. Login round-trips.
curl -fsS -X POST http://127.0.0.1:8770/api/v1/auth/login \
  -H 'content-type: application/json' \
  -d '{"user":"admin","password":"<your-admin-password>"}'
```

If any of those fails, `journalctl -u opendray -n 100 --no-pager` is
the next stop.
