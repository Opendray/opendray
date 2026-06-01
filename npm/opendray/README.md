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

```sh
opendray --help
opendray serve --config /etc/opendray/config.toml
```

See the [quickstart](https://github.com/Opendray/opendray/blob/main/docs/quickstart.md)
for first-run setup (Postgres, admin credentials, channel configuration).

## Supported platforms

| OS     | Architecture |
| ------ | ------------ |
| Linux  | x64, arm64   |
| macOS  | x64, arm64   |

Windows is not yet packaged — track [opendray#XXX](https://github.com/Opendray/opendray/issues).

## License

Apache-2.0 — see [LICENSE](./LICENSE).
