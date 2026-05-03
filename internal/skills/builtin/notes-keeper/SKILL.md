---
name: notes-keeper
description: Read, search, and write notes in the opendray file-system vault. Use this whenever the user mentions "notes", "daily log", "project doc", "TODO list", or wants to persist context that should survive across sessions.
---

# notes-keeper

You have access to a markdown notes vault rooted on the gateway host. The
vault holds three conventional sections:

- `daily/YYYY-MM-DD.md` — daily logs / standup-style notes
- `projects/<basename>/<file>.md` — per-project documentation (one
  folder per repo). You write here: README.md, spec.md, architecture.md,
  decisions/0001-xxx.md, retros/2026-q2.md, etc. The user reads these
  in their inspector but rarely edits them by hand.
- `personal/<basename>.md` — the user's PERSONAL scratchpad for this
  project. **Do not write here** — this lane is owned by the user. If
  the user explicitly tells you "add this to my notes", write to
  projects/<basename>/<something>.md instead and tell them where it
  landed; let them copy across if they want.
- anywhere else under the vault — free-form library content

All operations go through the `opendray notes` CLI subcommand. The gateway
process does not need to be running; the CLI talks to the vault filesystem
directly.

## When to use this skill

- The user asks to "remember", "save", "note", "log", "track" something
- They reference a project's doc / journal / decisions
- They ask "what did I work on yesterday / last week"
- They want a persistent TODO list scoped to a project
- They mention `[[wiki-links]]`, tags (`#area-X`), or daily notes

Do **not** use this skill for ephemeral within-session scratch state — that
belongs in your normal context. Save things that are valuable across sessions.

## Commands

```
opendray notes path                       # vault root
opendray notes list [--prefix=projects/]  # list notes (newest first)
opendray notes read <path>                # body to stdout
opendray notes write <path>               # body from stdin (replace)
opendray notes append <path>              # body from stdin (append + newline)
opendray notes delete <path>
opendray notes daily                      # create-or-print today's daily note path
opendray notes project <basename>         # create-or-print a project note path
```

`<path>` is vault-relative (e.g. `projects/opendray.md`). Writes must end in
`.md`. Use `--json` on `list` / `read` for structured output.

## Patterns

### Append a TODO to a project

```bash
echo "- [ ] migrate auth middleware" | opendray notes append projects/opendray/README.md
```

### Add a spec / decision file alongside the project README

```bash
echo "# Auth migration plan\n\n..." | opendray notes write projects/opendray/auth-migration.md
```

### Open or initialize today's daily note

```bash
DAILY=$(opendray notes daily)            # creates if missing, returns path
opendray notes read "$DAILY"
```

### Search notes (use ripgrep against the vault root)

```bash
VAULT=$(opendray notes path)
rg --type md "auth middleware" "$VAULT"
```

### Cross-link with wiki-link syntax

Use `[[Project Name]]` or `[[projects/opendray]]` when referencing other
notes. Backlinks resolution is exact-match on basename (case-insensitive)
or full vault-relative path.

### Frontmatter

When creating notes from scratch, include YAML frontmatter so future tooling
(graph view, tag indexing) can pick up structured metadata:

```markdown
---
project: opendray
type: project
tags: [infra, multiplexer]
created: 2026-05-02
---

# Opendray
```

## Conventions

- **One note per concern** — split unrelated topics into separate files
- **Short titles, dated when temporal** — `2026-q2-plan.md`, not `Q2 2026 Strategic Planning Document.md`
- **Append rather than rewrite** for daily/log content — preserves history
- **Replace** when the file is structured (frontmatter / project doc / spec)

## Failure modes

- Path containing `..` → rejected (vault jail)
- Non-`.md` write → rejected
- Vault not configured → CLI exits with a clear error pointing to
  `OPENDRAY_VAULT_ROOT` or `[vault].root` in `config.toml`
