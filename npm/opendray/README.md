# opendray

Self-hosted gateway that runs Claude Code · Codex · Gemini · shell sessions
on your own infrastructure, with persistent sessions, local-first memory, and
six chat channels (Telegram, Slack, Discord, Feishu, DingTalk, WeCom).

This package is the npm distribution of the official Go release binary. The
project itself lives at [github.com/Opendray/opendray](https://github.com/Opendray/opendray).

## Install

```sh
npm install -g opendray
```

Or with `pnpm` / `yarn`:

```sh
pnpm add -g opendray
yarn global add opendray
```

The right platform binary (Linux x64/arm64, macOS x64/arm64) is selected
automatically via `optionalDependencies` — no post-install network call.

## Usage

`opendray` ships the whole gateway — the web admin is embedded, so there is no
Node runtime or separate web server at deploy time. You bring a PostgreSQL 15+
database (with the `pgvector` extension) and start it yourself:

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:pw@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"
opendray migrate        # apply schema (idempotent)
opendray serve          # run in foreground → http://127.0.0.1:8770/admin/
```

See the [binary install guide](https://github.com/Opendray/opendray/blob/main/docs/install-binary.md)
for the full first-run walkthrough — pgvector setup, `config.toml`, and running
as a systemd / launchd service.

## Supported platforms

| OS     | Architecture |
| ------ | ------------ |
| Linux  | x64, arm64   |
| macOS  | x64, arm64   |

Windows is not yet packaged — track [opendray#XXX](https://github.com/Opendray/opendray/issues).

## License

Apache-2.0 — see [LICENSE](./LICENSE).
