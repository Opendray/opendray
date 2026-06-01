# Install + uninstall wizards

Interactive, guided opendray lifecycle scripts for Linux, macOS, and
Windows (WSL2 setup helper).

| Script | What it does | One-liner |
|---|---|---|
| `install.sh` | Install everything (Postgres, AI CLIs, admin creds, service). | `curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh \| bash` |
| `uninstall.sh` | Remove the gateway. Keeps DB + data by default. | `curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh \| bash` |
| `uninstall.sh --purge` | Remove everything: DB, role, config, data, logs. | `curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh \| bash -s -- --purge` |

> Want the manual deploy paths instead?
> See [`docs/getting-started.md`](../docs/getting-started.md) and
> [`README.md` § Install](../README.md#install).

---

## What the installer covers

The wizard is **end-to-end**: by the time it returns, opendray is
running under `systemd` / `launchd` and the admin UI is reachable.

| Phase | What you decide | What gets installed |
|---|---|---|
| 1 | – | `curl`, `tar`, `build-essential`, `postgresql-client` (Linux) / verifies macOS tools |
| 2 | – | Node.js 22 LTS + `pnpm` (needed for AI CLIs) |
| 3 | which AI CLIs (Claude / Codex / Gemini) | `npm install -g` for each |
| 4 | existing Postgres host **or** install local one | `postgresql-16` + `pgvector` if local |
| 5 | DB name, app user, app password | `CREATE DATABASE`, `CREATE USER`, `CREATE EXTENSION vector`, grants |
| 6 | admin password, listen address | `config.toml`, schema migrations, systemd unit / launchd plist |

What it **doesn't** do (intentionally — these are interactive):

- Log into the AI CLIs (`claude login`, `gemini auth login`, …)
- Configure providers, channels, or backup destinations
- TLS / reverse proxy setup

Those happen after the wizard, in the admin UI.

---

## Quick start

### Linux / macOS / WSL2 — one-liner

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

`install.sh` is dual-mode:

- **Piped from curl** (no local checkout): it shallow-clones
  `Opendray/opendray` to `${TMPDIR:-/tmp}/opendray-install-$$`,
  installs `git` first via `apt` / `brew` if it's missing, then
  re-execs itself from the clone so the rest of the wizard sees a
  real working directory.
- **Run from a clone** (already inside the repo): it just `exec`s
  the matching OS installer (`install-linux.sh` or
  `install-macos.sh`).

Either way you end up with `install-linux.sh` (or `install-macos.sh`)
walking through the same wizard. The one-liner just removes the
`git clone` + `cd` step.

### Linux / macOS — from a checkout

If you'd rather inspect the code before running anything:

```sh
git clone https://github.com/Opendray/opendray.git
cd opendray
bash scripts/install-linux.sh          # Linux (Ubuntu/Debian)
# or:
bash scripts/install-macos.sh          # macOS (Intel + Apple Silicon)
```

The wizard prompts before each privileged step. macOS default scope
is a **user LaunchAgent** (runs at login). Pass `--launchd-daemon` for
`/Library/LaunchDaemons` (boot-time, all users — needs admin).
Homebrew is required for the macOS path; if it's missing the wizard
prints the install command and exits.

### Windows

opendray does **not** support native Windows. The session subsystem
spawns AI CLIs via Unix PTYs, which Windows does not expose to
opendray. The PowerShell helper sets up WSL2 + Ubuntu for you:

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

Or, from a local checkout:

```powershell
pwsh -ExecutionPolicy Bypass -File scripts/install-windows.ps1
```

It either:

1. Detects existing WSL and prints the inside-WSL command to run, or
2. Installs WSL2 + Ubuntu via `wsl --install -d Ubuntu` (needs admin
   + reboot) and tells you to rerun afterwards.

Inside the Ubuntu shell, run the Linux one-liner above — WSL2's
loopback forwarding makes the admin UI reachable at
`http://localhost:8770/admin/` from the Windows host.

---

## Options

All three Unix wizards take the same flags:

| Flag | What it does |
|---|---|
| `--from-source` | Build opendray from this checkout (`go build`) instead of downloading the GitHub release tarball. Needs `go` 1.25+ on PATH. |
| `--skip-service` | Install the binary + config but don't register systemd / launchd. Useful when you want to wire your own supervisor. |
| `-h`, `--help` | Print built-in help and exit. |

macOS only:

| Flag | What it does |
|---|---|
| `--launchd-daemon` | Install a `/Library/LaunchDaemons` plist instead of a user LaunchAgent. |

---

## Defaults the wizard picks

If you just press Enter at every prompt:

- **Database name**: `opendray`
- **App DB user**: `opendray_user`
- **App DB password**: random 24-char alphanumeric (printed at the end)
- **Listen address**: `127.0.0.1:8770` (loopback only — safest)
- **Admin password**: random 20-char alphanumeric (printed at the end)

The two random passwords are printed exactly once, at the end of the
wizard. **Save them somewhere safe** — opendray hashes the admin one
into a bcrypt keyfile after your first UI password change, so you
can rotate the admin password later, but the DB password is what
`config.toml` keeps.

---

## Re-running

The wizard is idempotent on the parts that matter:

- `apt install` / `brew install` skip already-installed packages
- `CREATE DATABASE … IF NOT EXISTS` / `CREATE EXTENSION IF NOT EXISTS`
- `migrate` is a no-op if you're already at the latest schema
- The systemd unit / launchd plist gets overwritten

But re-running with **different** prompt answers will:

- Overwrite `config.toml`
- Reset the app DB user's password (if you change it)
- Restart the running service

So treat re-running as "I want to change something." If you only need
to flip a config value, edit `config.toml` directly and `systemctl
restart opendray` / `launchctl kickstart -k …`.

---

## File layout the wizard creates

### Linux

```
/usr/local/bin/opendray              # binary
/etc/opendray/config.toml            # 0644 — non-secret (listen, log, admin.user)
/etc/opendray/opendray.env           # 0640 root:opendray — SECRETS (db URL, admin pw)
/var/lib/opendray/                   # runtime data (bcrypt keyfile, sessions, …)
/var/log/opendray/opendray.{log,err} # logs
/etc/systemd/system/opendray.service # systemd unit (loads opendray.env)
```

A system user `opendray` owns runtime data and logs. The systemd
unit's `EnvironmentFile=/etc/opendray/opendray.env` injects DB URL and
admin password into the service environment at start, so `config.toml`
stays safe to read.

### macOS

```
~/.opendray/bin/opendray
~/.opendray/bin/opendray-launcher.sh   # 0700 — sources env, exec opendray
~/.opendray/config.toml                # 0644 — non-secret
~/.opendray/opendray.env               # 0600 — SECRETS
~/.opendray/data/                      # runtime data
~/.opendray/logs/opendray.{log,err}
~/Library/LaunchAgents/com.opendray.opendray.plist
```

(`/Library/LaunchDaemons/...` instead, if you passed `--launchd-daemon`.)
launchd doesn't support `EnvironmentFile`, so the plist runs
`opendray-launcher.sh`, a tiny wrapper that sources `opendray.env`
and execs the binary. Same security model as Linux: secrets in one
strictly-permissioned file, never in the unit definition or
`config.toml`.

### Secrets model (both OSes)

opendray's config loader treats `OPENDRAY_…` environment variables as
overrides — see `internal/config/config.go`. The wizard takes
advantage of this: `OPENDRAY_DATABASE_URL` and
`OPENDRAY_ADMIN_PASSWORD` live in `opendray.env`, everything else
lives in `config.toml`. If you ever need to rotate the DB password
or the bootstrap admin password, edit `opendray.env` and restart the
service — no `config.toml` touch required.

---

## Uninstall

```sh
# Linux / macOS — stops + removes the gateway, KEEPS DB + data
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash

# Or from a checkout:
bash scripts/uninstall.sh
```

Default mode (no flags) only removes the gateway runtime: stops the
service, deletes the systemd unit / launchd plist, removes the binary.
Your `config.toml`, `data/` directory (bcrypt keyfile, sessions, notes,
vault), logs, and the PostgreSQL database all stay put. Re-running the
installer later picks up where you left off.

### Purge (delete everything)

```sh
# Method 1: env var (safer for copy-paste — survives accidental newlines)
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash

# Method 2: flag via bash -s (must be one line, no manual line break!)
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash -s -- --purge

# From a checkout:
bash scripts/uninstall.sh --purge
```

> **Heads up on `bash -s -- --purge`:** the `--` and `--purge` MUST
> be on the same line as `bash -s`. If your terminal soft-wraps the
> command across two lines and you actually paste a newline, the
> shell parses it as two commands — the first one is `bash -s` with
> no args (default-mode uninstall, keeps your data), the second is
> `-- --purge` (which fails with `--: command not found`). The
> env-var form (`OPENDRAY_PURGE=1`) doesn't have this footgun.

Adds these destructive steps to the default flow:

- `DROP DATABASE` for the opendray database
- `DROP ROLE` for the app DB user
- delete `/etc/opendray/` (Linux) or `~/.opendray` (macOS)
- delete `/var/lib/opendray/` and `/var/log/opendray/` (Linux)
- remove the `opendray` service account (Linux)

The PG drop step prompts for a superuser (local peer auth or
host/user/password). The uninstaller does **not** touch:

- Homebrew, Node.js, pnpm
- The Claude / Codex / Gemini CLIs or their credentials
- PostgreSQL itself or any other databases on it
- apt / brew packages

### Other options

| Flag | Env var | Effect |
|---|---|---|
| `--purge` | `OPENDRAY_PURGE=1` | Also drop the DB, role, config, data, logs, and the service user. |
| `--yes`, `-y` | `OPENDRAY_YES=1` | Skip all confirmations. Combine with `--purge` for unattended cleanup. |
| `-h`, `--help` | — | Built-in help. |

Env-var forms are equivalent to the flag forms; pick whichever
survives your copy-paste flow.

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `Cannot connect as <superuser>@<host>` during PG step | wrong host / port / password, or PG not listening | Verify `psql -h <host> -p <port> -U <super> -d postgres` works manually |
| `pgvector extension is not available on this PG server` | missing OS package | `apt install postgresql-<ver>-pgvector` / `brew install pgvector`, then rerun |
| `cannot connect to <host>:<port>` after bootstrap | `pg_hba.conf` doesn't allow the app user from this host | Add a `host opendray opendray_user <client-ip>/32 md5` line; reload PG |
| `Health check did not succeed within 20s` | gateway crashed during startup | `journalctl -u opendray -n 50` (Linux) or `tail ~/.opendray/logs/opendray.err` (macOS) |
| AI CLI says `command not found` after wizard | `npm bin -g` is not on your shell PATH | `echo $PATH` — make sure it contains the output of `npm bin -g` |

---

## Verifying the install yourself

After the wizard claims success:

```sh
# 1) Health endpoint
curl -fsSL http://127.0.0.1:8770/api/v1/health
# {"status":"ok",…}

# 2) Service status
systemctl status opendray              # Linux
launchctl list | grep opendray         # macOS

# 3) Database schema present
psql "<your DSN>" -tAc \
  "SELECT count(*) FROM pg_tables WHERE schemaname='public'"
# Should print 30+ (current schema has ~31 tables on v2.0.0).
```

If all three pass, the wizard did its job. Open the admin UI and
proceed with [docs/getting-started.md § Step 4 — first login](
../docs/getting-started.md#step-4--first-login--change-admin-password).


## `enable-cli-updates.sh` — one-shot helper to enable in-app CLI updates

On a non-root install the opendray service user can't write to the global
npm prefix (`/usr/lib/node_modules` or similar), so the per-provider
**Update** button on the Providers page returns *"In-app updates aren't
available here — npm global prefix isn't writable."*

`scripts/enable-cli-updates.sh` flips that by creating an opendray-owned
npm prefix at `$OPENDRAY_DATA_DIR/.npm-global`, dropping a systemd unit
fragment that pins `NPM_CONFIG_PREFIX` + prepends the prefix's `bin` onto
`PATH`, reinstalling `@anthropic-ai/claude-code` / `@google/gemini-cli` /
`@openai/codex` into it as the service user, and restarting the daemon.

```sh
sudo bash scripts/enable-cli-updates.sh
```

Idempotent — re-running just refreshes the CLIs in the prefix.

**Skip individual providers** with env vars:

```sh
sudo SKIP_GEMINI=1 SKIP_CODEX=1 bash scripts/enable-cli-updates.sh
```

After it finishes, **Providers → Update** on each CLI writes into the
new prefix and one-tap upgrades work as designed.
