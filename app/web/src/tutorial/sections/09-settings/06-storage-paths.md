# Storage paths

Three Server sub-sections in the sidebar configure where opendray
looks for each upstream CLI's on-disk data:

- **Storage · Claude** — `~/.claude/projects` + multi-account roots
- **Storage · Codex** — `~/.codex/sessions`
- **Storage · Gemini** — `~/.gemini/tmp` + `projects.json`

These paths feed two features:

1. **History panel** in the Sessions Inspector (shows every
   prompt the operator has sent in the current project, pooled
   across all sessions ever spawned).
2. **New-account defaults** (Claude only) — when you create a
   Claude account via the API and don't specify `ConfigDir`, it's
   derived from `accounts_dir`.

Every field is optional — empty values fall back to the upstream
CLI's standard layout under `$HOME`. Override only when:

- The CLI is installed in a non-default location.
- You set `CLAUDE_CONFIG_DIR` / `CODEX_HOME` / `GEMINI_HOME` on
  the shell that launched opendray.
- You vendored a CLI under `/opt/<tool>` etc.

## Claude

| Field | toml key | Default behaviour |
|---|---|---|
| History roots | `providers.claude.history_roots` | Empty list = scan `~/.claude/projects` **plus** every `~/.claude-accounts/*/projects` (auto-globbed, dedup'd via EvalSymlinks). |
| Accounts directory | `providers.claude.accounts_dir` | `~/.claude-accounts` |

The History roots field is a list — add one row per directory.
Order doesn't matter (results are merged and sorted by
timestamp, dedup'd by file inode).

The default behaviour was chosen because opendray's
multi-account support uses `~/.claude-accounts/<name>/projects`
which is typically a symlink to `~/.claude-accounts/shared/projects`
— so the ring of dirs all point at the same actual files. The
EvalSymlinks dedup ensures each transcript shows up once.

## Codex

| Field | toml key | Default |
|---|---|---|
| Sessions root | `providers.codex.sessions_root` | `~/.codex/sessions` |

opendray walks the entire tree under this root, reads the first
line of each `*.jsonl` (the `session_meta` envelope), and keeps
the file only if its `payload.cwd` matches the current session's
working directory.

The synthetic AGENTS.md / `<environment_context>` bootstrap
that Codex injects at the start of every session is filtered
out — you only see prompts the operator actually typed.

## Gemini

| Field | toml key | Default |
|---|---|---|
| Tmp directory | `providers.gemini.tmp_root` | `~/.gemini/tmp` |
| projects.json | `providers.gemini.projects_file` | `~/.gemini/projects.json` |

Gemini's per-project storage lives in `<tmp_root>/<dir>/logs.json`
where `<dir>` can be:

1. The "short name" mapped from `cwd` in `projects.json`
   (preferred; written by the Gemini CLI on first run).
2. The lowercase hex SHA-256 of the cwd (older Gemini versions).
3. Anything — opendray scans `<dir>/.project_root` files as a
   last-resort fallback.

If your Gemini layout is unusual (custom `GEMINI_HOME`, mounted
volume, etc.) the scan-fallback handles it as long as Gemini
itself wrote the `.project_root` marker.

## Test paths

Every path field has a **Test** button. Click it to:

- Resolve `~/` against `$HOME` for the running gateway
  process.
- Stat the path — show "not found" or "✓ N children" inline.
- Verify it's a directory (when expected).

Paths are **not** restart-required — they're read on every
History API call. Save the change and the next History tab
refresh picks up the new value immediately.

## Cross-platform note

The current implementation assumes opendray and the CLI tools
run **on the same machine, as the same OS user**. Cross-machine
or cross-user setups (opendray in a container, CLI on the host)
need volume-mounting + the configured paths must be valid from
inside the gateway process.

Windows isn't supported (`os.Getenv("HOME")` returns empty;
fix would need to fall back to `USERPROFILE`).
