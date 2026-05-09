# Running OpenDray in a Proxmox LXC

OpenDray spawns child PTYs for every session
([`internal/session/manager.go:362`](../../internal/session/manager.go)).
Proxmox LXC containers can run those PTYs, but the default unprivileged
profile blocks the syscalls OpenDray needs. This page captures the
host-side container config that makes it work, plus a few smaller
LXC-specific gotchas.

For everything that isn't LXC-specific (binary install, systemd unit,
config) see [`../README.md`](../README.md) and
[`../systemd/opendray.service`](../systemd/opendray.service).

## TL;DR

| LXC mode | What works out of the box | Action required |
|---|---|---|
| **Privileged** (`unprivileged: 0`) | Everything, including PTY. | None. Recommended for personal homelab use. |
| **Unprivileged** (default) | HTTP, WS, DB, channels, integrations. **Sessions / PTY fail with `EPERM`.** | Apply the cgroup + mount entries below. |

If you'd rather not edit the host's LXC config, just run a privileged
container. The trade-off is the standard Proxmox guidance: a process
escape inside the container has more authority on the host. For a
single-admin AI cockpit, privileged is fine.

## Privileged container (recommended)

In the Proxmox web UI, when creating the container, untick **"Unprivileged
container"**. Or in `/etc/pve/lxc/<vmid>.conf`:

```
unprivileged: 0
```

Recreate the container after toggling — Proxmox doesn't migrate the
existing rootfs across the privilege boundary. There's nothing else to
configure for OpenDray.

## Unprivileged container — making PTY work

If you must stay unprivileged (shared-tenancy hosts, compliance, etc.),
add the following to `/etc/pve/lxc/<vmid>.conf`:

```
features: nesting=1
lxc.cgroup2.devices.allow: c 5:2 rwm
lxc.cgroup2.devices.allow: c 136:* rwm
lxc.mount.entry: /dev/ptmx dev/ptmx none bind,create=file 0 0
```

What these do:

| Line | Effect |
|---|---|
| `features: nesting=1` | Lets unprivileged containers create their own user namespaces. Required for some PTY allocation paths. |
| `c 5:2 rwm` | Grants `/dev/ptmx` read/write/mknod (major 5, minor 2). |
| `c 136:* rwm` | Grants `/dev/pts/*` (major 136, any minor) — the PTY slaves. |
| `lxc.mount.entry … /dev/ptmx` | Bind-mounts the host's `/dev/ptmx` into the container so `pty.Open` finds it. |

Restart the container after editing:

```bash
pct stop <vmid>
pct start <vmid>
```

## Verifying PTY before installing OpenDray

Run this **inside the container** before you do anything else. If it
errors with permission, fix the LXC config; don't touch OpenDray yet.

```bash
# Spawn a child PTY running /bin/bash; type `exit` to quit.
python3 -c "import pty; pty.spawn(['/bin/bash'])"
```

Successful output looks like a normal interactive shell. Failure looks
like `OSError: [Errno 1] Operation not permitted` or
`OSError: [Errno 25] Inappropriate ioctl for device`. Either of those
means the cgroup/mount config above didn't take.

If `python3` isn't available:

```bash
apt install -y python3-minimal
# or, if you have script(1):
script -qc 'echo hi' /dev/null
```

## Networking

Standard LXC networking covers OpenDray. The defaults from the Proxmox
template work. Two things to confirm:

1. **The Postgres host is reachable from the container.** OpenDray has
   no bundled Postgres; you point `[database].url` at an external one.
   Test with `psql -h <host> -U <user> -d <db>` or
   `pg_isready -h <host> -p 5432` from inside the container.
2. **Loopback (`127.0.0.1:8770`) is fine for the OpenDray bind.**
   Don't bind to `0.0.0.0` unless you have a TLS-terminating reverse
   proxy *outside* this container. Cloudflare Tunnel running in the
   same container is a common pattern.

## Memory subsystem (pgvector)

If you plan to use OpenDray's [Memory subsystem](../../docs/adr/0014-memory-subsystem.md),
the Postgres your `[database].url` points at must have the `pgvector`
extension installed. Check:

```sql
SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';
```

If empty:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

The OpenDray migration runner does NOT create extensions — that
typically requires `superuser`, which the project-scoped role
shouldn't have. Install the extension with a DBA / superuser one-shot
and OpenDray's project role will use it.

## Backup subsystem

If you set `[backup].enabled = true`, **`pg_dump` and `pg_restore`
must match the major version of the Postgres they target.** A common
mistake is installing `pg_dump` 15 in the LXC while the database is
PG 17 — `pg_dump` aborts with `server version mismatch`.

Install the matching client tools and point at them:

```bash
# Debian/Ubuntu LXC, targeting an external PG 17:
apt install -y postgresql-client-17

# Then in /etc/opendray/env.d/secrets:
OPENDRAY_BACKUP_PG_DUMP_PATH=/usr/bin/pg_dump
OPENDRAY_BACKUP_PG_RESTORE_PATH=/usr/bin/pg_restore
OPENDRAY_BACKUP_KEY=$(openssl rand -base64 32)
```

`OPENDRAY_BACKUP_KEY` **must** live in env, never in the config TOML.
Lose the key → encrypted backups become unrecoverable.

## Secrets in systemd EnvironmentFile

The systemd unit ships an `EnvironmentFile=/etc/opendray/env.d/secrets`
directive. In an LXC the file lives inside the container's filesystem;
that's fine for self-host. For tighter isolation, two upgrades:

1. **systemd-creds** — encrypted-at-rest credentials decrypted only at
   service start. Requires systemd 250+ (Debian 12 ships 252, OK).
2. **Vault Agent / OS keychain** — render the EnvironmentFile from a
   secret manager at boot. More moving parts; needed for multi-host
   fleets.

For a single homelab LXC, mode-0640 root-owned `secrets` file is
proportionate.

## Reverse proxy

OpenDray binds to `127.0.0.1:8770` by default. You need TLS terminated
*somewhere*. Three common patterns on Proxmox:

| Pattern | Where TLS lives | Notes |
|---|---|---|
| **Cloudflare Tunnel inside the LXC** | Cloudflare edge | `cloudflared service install` in the same LXC. Zero firewall config; the tunnel registers a public hostname. |
| **Caddy/nginx in a separate LXC or VM** | Reverse-proxy LXC | Forwards to the OpenDray LXC over the cluster's private network. Standard practice if you already run a reverse-proxy box. |
| **Proxmox's built-in `pveproxy`** | Not really | `pveproxy` is for Proxmox itself; don't try to multiplex OpenDray onto it. |

The reverse proxy must forward the full URL path including
`/api/v1/sessions/{id}/ws` (the WebSocket terminal stream) **without
buffering** and with a long idle timeout. nginx config sketch:

```nginx
location / {
    proxy_pass http://127.0.0.1:8770;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_buffering off;
    proxy_read_timeout 86400s;
}
```

## Sanity checklist before going live

- [ ] PTY smoke (`python3 -c "import pty; pty.spawn(['/bin/bash'])"`) works inside the container
- [ ] `pg_isready -h <db-host>` from inside the container exits 0
- [ ] `pgvector` extension created (if using Memory)
- [ ] `pg_dump --version` matches the PG server major (if using Backup)
- [ ] `OPENDRAY_ADMIN_PASSWORD` set in `/etc/opendray/env.d/secrets`
- [ ] `OPENDRAY_BACKUP_KEY` set if `[backup].enabled = true`
- [ ] systemd unit shows `active (running)` and `journalctl -u opendray` is clean
- [ ] `curl -fsS http://127.0.0.1:8770/admin/` returns the SPA HTML
- [ ] Reverse proxy fronts the LXC and the SPA loads over TLS in a browser
- [ ] Login + create one test session — confirm the terminal renders

If all eight are green, the deploy is complete.
