# Install & run from a prebuilt binary

For when you already have — or want — just the `opendray` binary, with no
installer wizard touching your machine. This is the path for:

- **`npm install -g opendray` / `npx opendray`** — the npm package ships the
  official Go release binary (see [README → npm / npx](../README.md#install)).
- **Release downloads** — grab `opendray_*_<os>_<arch>.tar.gz` from the
  [Releases page](https://github.com/Opendray/opendray/releases).
- **Scripted / ephemeral environments** — CI runners, golden images, config
  management (Ansible, Nix, Docker), or any host where you already run your
  own Postgres and process supervisor.

The binary is the *whole* gateway — the web admin SPA is embedded, so there
is no Node runtime, no separate static server, and nothing to build. What it
does **not** do is set anything up for you. That is the trade: you bring a
PostgreSQL database and a way to keep the process running, and in exchange
nothing is installed, configured, or registered behind your back.

> **Want it all done for you instead?** On a fresh Linux / macOS box, the
> one-line installer provisions Postgres, installs the AI CLIs, writes the
> config, and registers a service in ~5–10 minutes. See
> [README → One-line installer](../README.md#install) or the manual
> [getting-started.md](getting-started.md) walkthrough.

This guide takes you from "binary on `PATH`" to "running gateway" in five
steps, then shows how to keep it running as a service.

---

## Step 1 — Get the binary

### Via npm (any OS with Node ≥ 18)

```sh
npm install -g opendray        # global install, puts `opendray` on PATH
# or, without installing:
npx opendray --help
# or with another package manager:
pnpm add -g opendray
yarn global add opendray
```

The right platform binary (`opendray-{linux,darwin}-{x64,arm64}`) is selected
automatically via `optionalDependencies` — there is no `postinstall` hook and
no network call at install time. Do **not** pass `--no-optional`: it skips the
platform package and leaves the launcher with no binary to exec.

### Via release archive

```sh
# Pick the archive matching your OS/arch from the Releases page, then:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### Verify

```sh
opendray version          # prints version, commit, build date
opendray --help           # lists every subcommand
```

Supported platforms: **Linux** (x64, arm64) and **macOS** (x64, arm64).
Native Windows is not packaged — use WSL2 and follow the Linux path.

---

## Step 2 — Provide PostgreSQL 15+ with pgvector

opendray stores everything (sessions, memory, audit log) in PostgreSQL, and
its memory subsystem needs the [`pgvector`](https://github.com/pgvector/pgvector)
extension. Supported server versions: **15, 16, 17**.

If you already run Postgres, create a database and a CRUD-only role, then
enable the extension once with a superuser:

```sh
# 1. Install pgvector (once per host).
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    Other / source: https://github.com/pgvector/pgvector#installation

# 2. Create the database + a project-scoped role.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. Enable pgvector inside that database (one-off, needs superuser).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

Once the extension exists, opendray's CRUD-only role runs migrations without
any further superuser access. **Never point opendray at a superuser role for
runtime** — give it a project-scoped account and rotate its password out of
band.

---

## Step 3 — Configure

opendray reads its config from a TOML file **or** purely from environment
variables (12-factor) — env always wins over the file. The only hard
requirement is the database URL; everything else has a default.

### Option A — environment variables (good for containers / ephemeral hosts)

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # admin login
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # optional; this is the default
```

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **yes** | — | Postgres DSN |
| `OPENDRAY_ADMIN_PASSWORD` | recommended | — | Web/mobile admin password |
| `OPENDRAY_ADMIN_USER` | no | `admin` | Admin username |
| `OPENDRAY_LISTEN` | no | `127.0.0.1:8770` | Bind address |
| `OPENDRAY_LOG_LEVEL` | no | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | no | `text` | `text`/`json` |

Run `opendray serve` with no `-config` flag and it loads entirely from the
environment.

### Option B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # set [database].url and [admin].password
```

The minimum to edit:

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

See [`config.example.toml`](../config.example.toml) for the fully annotated
file (logging, session idle detection, backups, vault, MCP). Pass it with
`-config config.toml` to the commands below. Keep secrets out of the TOML on
shared hosts — set `OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD` via env
and leave the file non-secret.

---

## Step 4 — Apply the schema

```sh
opendray migrate                          # env-only config
# or
opendray migrate -config config.toml
```

Idempotent — re-running is a no-op once the schema is current. This must
succeed before the first `serve`.

---

## Step 5 — Run it

```sh
opendray serve                            # env-only config
# or
opendray serve -config config.toml
```

This runs in the **foreground** (Ctrl-C stops it). You should now have:

| URL | What |
|---|---|
| `http://127.0.0.1:8770/admin/` | Web admin — log in with `admin` + your password |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket API |

That is a complete, running gateway. For anything beyond a quick test, run it
under a supervisor so it survives reboots and restarts on crash — next.

---

## Run it as a service

`opendray serve` is exactly what a service unit's start command should call.
opendray ships hardened, ready-to-use units; the steps below are the same as
[README → Production deploy](../README.md#production-deploy), which is the
authoritative reference (full bootstrap, sandboxing notes, reverse-proxy/TLS).

### Linux — systemd

The repo ships a hardened unit at
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
(runs `migrate` as `ExecStartPre`, secrets via an `EnvironmentFile`,
`on-failure` restart, syscall/filesystem sandboxing).

```sh
# Binary at /usr/local/bin/opendray, service user, state dir:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# Config (non-secret) + secrets file (env, mode 0640):
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# Install + enable the unit:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

No systemd? (LXC without it, OpenRC, runit, s6, supervisord…) Point your
supervisor at `opendray serve -config /etc/opendray/config.toml` and run
`opendray migrate` once as a pre-start step. See
[README → Production deploy §B](../README.md#option-b--direct-binary--your-own-process-supervisor).

### macOS — launchd

The repo ships a LaunchDaemon at
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
(starts at boot, restarts on crash, logs to `/usr/local/var/log/opendray/`).

```sh
sudo install -d -m 0755 /usr/local/etc/opendray /usr/local/var/lib/opendray /usr/local/var/log/opendray
sudo install -m 0640 config.toml /usr/local/etc/opendray/config.toml
sudo /usr/local/bin/opendray migrate -config /usr/local/etc/opendray/config.toml

sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl print system/com.opendray.opendray
```

Restart: `sudo launchctl kickstart -k system/com.opendray.opendray`.
Unload: `sudo launchctl bootout system/com.opendray.opendray`.

> Both units are documented in full — including the secrets layout and why
> `MemoryDenyWriteExecute` is left off — in
> [`deploy/README.md`](../deploy/README.md).

---

## Keeping it updated

How you update depends on how you installed:

- **Installed via npm** — update with your package manager. `opendray update`
  would replace the binary *inside* `node_modules` behind npm's back and get
  clobbered on the next install, so don't use it here.

  ```sh
  npm install -g opendray@latest
  ```

- **Release download / wizard install** — the binary self-updates in place
  (downloads the latest release, verifies its SHA-256, atomically swaps
  itself):

  ```sh
  opendray update --check          # report-only version probe
  sudo opendray update --restart   # apply, then restart the service
  ```

---

## Troubleshooting

**`the matching platform package "opendray-…" was not installed`**
npm was run with `--no-optional`, or the install was interrupted. Re-run
`npm install -g opendray` (without `--no-optional`).

**`unsupported platform`**
The npm package covers Linux/macOS on x64/arm64 only. On other targets, build
from source — see [quickstart.md](quickstart.md).

**`config: database.url is empty`**
Neither `OPENDRAY_DATABASE_URL` nor `[database].url` is set. Set one (Step 3).

**`connection refused` on migrate/serve**
Postgres isn't running or the DSN is wrong. Confirm the server is up and the
host/port/credentials in your DSN are correct.

**pgvector / `extension "vector" is not available`**
The extension isn't installed on the server, or wasn't enabled in the
opendray database. Re-do Step 2 (install the OS package, then
`CREATE EXTENSION vector` as a superuser).

**Port already in use**
Change `OPENDRAY_LISTEN` (or `listen` in config.toml) to a free port.

---

## Next steps

- [README → Production deploy](../README.md#production-deploy) — full deploy
  reference (systemd / launchd / own-supervisor, hardening, reverse proxy)
- [`docs/operator-guide.md`](operator-guide.md) — ops: reverse-proxy/TLS
  topology, encrypted DB backups, data export/import
- [`docs/integration-guide.md`](integration-guide.md) — build an external
  integration against the REST + WebSocket API
- [`docs/getting-started.md`](getting-started.md) — the guided, all-in-one
  setup if you'd rather not assemble the pieces yourself
