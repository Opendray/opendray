# Worktree Isolation for Concurrent Sessions

Status: **DRAFT — awaiting operator review**

## 1. Problem

Two sessions pointed at the same `cwd` share one working tree and one git
index. Both CLIs edit the same files, race the index, and overwrite each
other's half-finished work. Today nothing prevents this — it is merely rare
because a single operator seldom runs two agents on one project. As soon as
several people share the gateway (one account, many humans), same-project
concurrency becomes the normal case, not the accident.

This design gives each session an **opt-in isolated git worktree**: the CLI
runs in a private checkout on a private branch, and changes flow back through
the repo's normal branch → PR path. Multi-account/RBAC is explicitly out of
scope here.

## 2. Current state (code audit)

A session has exactly one path field, used for everything:

| Consumer | Site | Uses `cwd` as |
|---|---|---|
| Process spawn | `internal/session/manager.go:785` (`cmd.Dir = sess.Cwd`) | execution dir |
| Memory MCP scope | `internal/catalog/adapter.go:786` (`OPENDRAY_MEMORY_SCOPE_KEY = cwd`) | project identity |
| Headless memory attach | `internal/catalog/memory_mcp.go:34` (`AttachMemoryMCP(runCwd, scopeKey, …)`) | **already split** into execution dir + identity — every caller currently passes the same value |
| Ambient injector | `internal/memory/injector/inject.go:43` (`Render(sessionID, cwd)`) | project identity (scope key + search query) |
| Project docs / journal / Cortex | `project_docs`, `session_logs` tables keyed by `cwd` | project identity |
| antigravity scope derivation | `cmd/opendray/mcp_memory.go:147` (`SCOPE_FROM_CWD=1` → `Getwd()`) | **execution dir standing in for identity** — breaks under a worktree |
| dbtool MCP scope | `cmd/opendray/mcp_dbtool.go:104` (same `Getwd()` pattern) | same problem |
| Claude transcript / carryover / input history | `manager.go:1212`, `manager.go:1527` (Claude keys `~/.claude/projects/<encoded-cwd>`) | execution dir (the CLI records under its *actual* cwd) |
| agy conversation resume | `manager.go:733` (`ConversationIDForCwd(home, cwd)`) | execution dir |
| Claude local memory mirror | `internal/memory/mirror.go` (reads `<cwd>/.claude/**`, ingests with `ScopeKey: cwd`) | **both**: source = execution dir, scope = identity |

The core of this design is making that third column explicit: every consumer
is classified as needing the **physical execution path** or the **logical
project identity**, and the session model carries both.

## 3. Design

### 3.1 Two paths per session

```
Session.Cwd      — logical project path (unchanged; the main checkout).
                   Keys memory, project docs, Cortex, journal, custom
                   tasks, UI grouping. NEVER changes meaning.
Session.WorkDir  — physical execution path. Defaults to Cwd. When
                   isolation is on, points at the session's worktree.
```

`CreateRequest` gains `isolation: "" | "worktree"`. Empty (default) keeps
today's behaviour exactly: `WorkDir == Cwd`, zero new moving parts.

Consumers are rewired once, by classification:

- **WorkDir** (physical): `cmd.Dir`, Claude transcript/carryover/input
  history lookups, agy conversation-id lookup + copy, mirror *source*
  directory, dbtool `Getwd` result (then canonicalized, §3.4).
- **Cwd** (logical): memory scope key, ambient injector, project docs,
  session logs, Cortex capture, custom-task grouping, default session name,
  `AttachMemoryMCP`'s `scopeKey` argument (its `runCwd` gets WorkDir).

### 3.2 Worktree lifecycle

New package `internal/workspace` owning creation, tracking, and reclamation.

**Create (at spawn, before the PTY starts):**

```
repoRoot = git -C <cwd> rev-parse --show-toplevel      (reject non-git)
path     = ~/.opendray/worktrees/<repoBase>-<sessionID>
branch   = opendray/<sessionID>                        (e.g. opendray/ses_x1y2z3)
git -C <repoRoot> worktree add -b <branch> <path> HEAD
WorkDir  = path (+ the cwd-relative subdir, if Cwd was inside the repo
           rather than at its root)
```

Branching from the main checkout's current `HEAD` — not `origin/main` — so
the session starts from exactly what the operator sees. Worktrees share the
object store, so creation is fast and cheap; only untracked build artifacts
(node_modules, .env, caches) cost anything, and those are per-worktree by
nature (§6).

**Guards at create:**

- `cwd` not inside a git repo → `400`, UI disables the toggle (see §3.6).
- `cwd` is itself a *linked* worktree (opendray-managed or hand-made) →
  reject; isolation must anchor on the main checkout.
- Branch/path collision (stale leftovers) → `git worktree prune`, then
  regenerate; session IDs are random so real collisions don't occur.

**Restart / resume:** `WorkDir` and `Branch` are persisted; a restarted or
gateway-crash-resumed session re-enters the same worktree. If the directory
vanished from disk (manual deletion), Start fails with an explicit error and
the UI offers "recreate worktree from current HEAD" (fresh tree, same
branch if it still exists).

**Reclaim (on session Remove, not on mere Stop/End):**

- tree clean AND branch has no commits beyond its base, or branch fully
  merged into the base branch → `git worktree remove` + `git branch -d`.
- dirty tree or unmerged commits → keep everything, flag the session row
  `worktree: needs attention` in the UI. Never auto-delete work.
- A settings-page maintenance view lists orphaned worktrees under
  `~/.opendray/worktrees/` with per-item "open shell / remove" actions.

### 3.3 Merge-back

Nothing new is invented: the worktree's branch is a normal local branch of
the shared repo. The agent (or operator) commits on it and lands it through
the existing convention — push, open a PR, merge. Concurrent sessions'
conflicts surface at PR merge time, reviewed by a human, instead of as
silent file clobbering.

v1 ships no auto-PR machinery. A later iteration can add a one-click
"create PR from this session" and merged-branch detection to drive
auto-reclaim, but the manual path must work first.

### 3.4 Scope-key canonicalization at the gateway boundary

Per-session env (`OPENDRAY_MEMORY_SCOPE_KEY`) covers claude/codex/opencode:
the adapter simply keeps passing the *logical* Cwd there while `runCwd`
becomes WorkDir. Two providers can't be handled that way:

- **antigravity** — HOME-global `mcp_config.json` shared across sessions;
  scope key is derived from the MCP subprocess's `Getwd()`, which inside a
  worktree is the worktree path.
- **dbtool MCP** — same `Getwd()` fallback.

Fix at the gateway, generically: the gateway *created* every managed
worktree and persists `(work_dir → cwd)` on the sessions table, so the
memory/dbtool HTTP handlers canonicalize any incoming scope key with one
map lookup — if it matches a known managed `work_dir` (or a subdirectory of
one), rewrite it to the owning session's `Cwd`. Backed by an in-memory map
maintained by `internal/workspace`, rebuilt from the DB at boot.

Hand-made worktrees the gateway didn't create are *not* canonicalized —
out of scope, same behaviour as today.

The mirror keeps its shape: scan `<WorkDir>/.claude/**` as the source,
ingest with `ScopeKey: <Cwd>`.

### 3.5 Persistence

Migration `009x_session_worktrees.sql`:

```sql
ALTER TABLE sessions ADD COLUMN work_dir        TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN worktree_branch TEXT NOT NULL DEFAULT '';
-- work_dir = '' means "not isolated; execution dir is cwd" (all existing rows).
```

No new table: the sessions row *is* the worktree registry. The
canonicalization map is `SELECT work_dir, cwd FROM sessions WHERE work_dir <> ''`.

### 3.6 API + UI

- `POST /api/v1/sessions` body: `"isolation": "worktree"` (optional).
- Session JSON exposes `work_dir`, `worktree_branch` when set.
- A lightweight `GET /api/v1/fs/git-info?path=` (or extension of the
  existing fs endpoints) reports `{is_repo, repo_root, current_branch}` so
  the create dialog can enable/disable the toggle live.
- Create dialog (web **and** mobile, i18n both): a "Run in isolated
  worktree" toggle with one explanatory line; disabled with a reason for
  non-git paths.
- Session list/card: branch badge (`⎇ opendray/ses_x1y2z3`) and a
  "needs attention" flag for dirty reclaims.
- Sessions grouped under their *logical* project regardless of isolation.

### 3.7 Child sessions and inspector surfaces

The Inspector operates on "the session's files and repo state" — under
isolation those live in the worktree, so every *physical* surface follows
WorkDir:

- **Files / preview / download** (`internal/fs`, paths anchored on the
  session's dir) → WorkDir.
- **Git tab** (`internal/git` read + write: status, log, branches, stage,
  commit, push) → WorkDir. This is how the operator commits/pushes a
  session's branch from the UI.
- **Task runner** (`TaskRunnerPanel` lists manifests via fs and spawns
  shell children) → WorkDir; a `make build` must build the worktree.
- **Shell children**: a child session whose parent is isolated inherits the
  parent's WorkDir server-side (manager checks `ParentSessionID` at
  create) — the client never has to know. Children keep the parent's
  logical Cwd; no second worktree is created. A worktree is not
  reclaimable while any session (parent or child) still references it.

Session JSON exposes `work_dir`; web/mobile use `work_dir || cwd` for the
physical surfaces above and `cwd` for naming/grouping.

### 3.8 Companion guard (independent, cheap)

Even with isolation available, two *non-isolated* sessions on one cwd stay
dangerous. At create time, if an active session already holds the same
logical Cwd without isolation, return the conflict in the create response
and let the UI show "this project already has an active session — consider
worktree isolation". Advisory, never blocking: single-operator workflows
that intentionally share a tree keep working.

## 4. Compatibility matrix — Cortex and every other cwd consumer

Full sweep of the codebase for features keyed on or operating in `cwd`.
Design invariant #1: **to Cortex and the whole memory system, an isolated
session is indistinguishable from a normal one** — same scope key, same
injections, same capture. Invariant #2: **no worktree path ever enters the
scope-key namespace**; the memory/dbtool HTTP boundary canonicalization
(§3.4) is the single enforcement point, so every enumerator downstream
stays clean for free.

| Feature | Keying / operation | Under isolation | Change needed |
|---|---|---|---|
| Cortex capture pipeline | routes on `Origin` + session cwd | logical Cwd → same partition | none |
| Cortex spawn injection (KB, project docs, rules args) | rendered from `sess.Cwd` | logical Cwd → identical injections | none |
| Cortex blueprint proposer / project docs / current objective | `project_docs(cwd, kind)` | logical Cwd | none |
| Cortex conversation worker (AI discussions) | `TargetCwd`, headless run independent of sessions | keeps targeting main checkout | none |
| ExperienceCompiler / session_logs | `session_logs.cwd` | logical Cwd | none |
| Knowledge graph project entities | `ProjectEntityID = sha256(cwd)` (`knowledge/anchor.go:89`) | logical Cwd; canonicalization prevents phantom entities from worktree paths | none (protected by §3.4) |
| Memory conflict / mirror / KB enumerators | enumerate project scope keys | never see worktree paths (invariant #2) | none (protected by §3.4) |
| Ambient injector | `Render(sessionID, cwd)` | logical Cwd | call-site uses Cwd (already planned) |
| Claude local memory mirror | scans `<dir>/.claude/**`, ingests by scope | **source = WorkDir**, scope = Cwd | mirror call-site passes both |
| memory worker headless agents | `runCwd` = scratch (agy: project dir) | background worker, main checkout is correct | none |
| gitactivity (recent_activity doc) | `git log` with `cmd.Dir = cwd`, per-cwd scheduler | runs on main checkout; **session-branch commits appear only after merge** — accepted | none (documented) |
| projectscan (tech_stack doc) | scans cwd | main checkout | none |
| prwatcher | `uniqueLiveCwds(sessions)` → repo PR polling | logical Cwd; PRs opened from session branches belong to the same repo, so they're picked up automatically | none (synergy) |
| Custom tasks | list filtered by `cwd ==` session cwd; grouped by project | filter on logical Cwd; **execution in WorkDir** (§3.7) | run-side only |
| Canvas focus + artifacts | per-cwd focus, `canvas_artifacts(cwd, slug)` | logical Cwd | none |
| Round Table | `rt.Cwd` = memory scope for seats | logical Cwd; execute/handoff spawns may opt into isolation like any create | none |
| Notes vault project mapping | `basename(cwd)` → vault project dir | logical Cwd | none |
| dbtool connections | per-project registry keyed by cwd | logical Cwd; the MCP subprocess `Getwd()` is canonicalized (§3.4) | §3.4 covers it |
| Inspector fs / git tab / task runner / shell children | operate on session's directory | WorkDir (§3.7) | §3.7 |
| Claude transcripts, carryover, input history; codex history; agy conversation resume | keyed by the CLI's actual path | WorkDir | manager call-sites (§3.1) |
| Provider project config (`.claude/CLAUDE.md`, committed rc files) | read by the CLI from its actual dir | tracked files are present in the worktree checkout | none |
| Session default name / UI grouping | derived from cwd | logical Cwd | none |

The pattern the table confirms: **background/project-level features key on
logical Cwd and need zero changes; only surfaces that touch the session's
literal files switch to WorkDir.** The risky consumers are exactly the two
`Getwd()`-derived scope keys (antigravity, dbtool), both closed by one
gateway-side canonicalization.

## 5. Explicitly out of scope

- **Multi-account / RBAC / attribution** — separate track.
- **Runtime resource isolation** — shared dev DB, ports, dev servers, and
  the Mac's CPU stay shared; a worktree isolates *files*, nothing else.
- **Auto-PR / auto-merge** — v1 ends at "branch exists, normal git flow".
- **Non-git projects** — no isolation offered; advisory warning only.

## 6. Known limitations (accepted for v1, documented in operator guide)

- **Untracked state does not follow**: `.env`, `node_modules`, build caches
  are absent in a fresh worktree; sessions must reinstall deps. A future
  per-project "worktree init" custom task can automate this.
- **Submodules** are not auto-initialized by `git worktree add`.
- **skip-worktree flags don't carry over** (per-index): e.g. a locally
  pinned `pubspec.yaml` build counter appears at its *committed* value
  inside a worktree. Fine for code work; do release packaging from the main
  checkout.
- **Claude/codex project history starts empty** in a new worktree (history
  is keyed by the CLI's actual path). Correct for isolation semantics, but
  worth a line in the UI copy.

## 7. Test plan

- `internal/workspace` unit tests against throwaway git repos: create,
  subdir-cwd mapping, non-git rejection, linked-worktree rejection, clean
  vs dirty reclaim, prune of stale paths, recreate-after-manual-delete.
- Manager tests: `cmd.Dir` = WorkDir; scope-key env = Cwd; transcript/
  history/agy lookups use WorkDir (extend existing table-driven tests).
- Canonicalization tests: memory handler rewrites managed work_dir (and
  subdirs) → cwd; unmanaged paths pass through untouched.
- Migration on ephemeral pg17; existing rows behave identically
  (`work_dir=''`).
- Manual: two concurrent claude sessions on this repo, one isolated —
  verify no file interference, memory/journal land in one partition,
  branch merges back cleanly via PR. Repeat isolation-on with antigravity
  to verify canonicalization.

## 8. Phasing

1. **PR A — core**: migration, `internal/workspace`, session model +
   manager rewiring, scope-key canonicalization, create-API flag, reclaim
   logic, web UI toggle + badges, i18n. Includes the §3.8 advisory.
2. **PR B — mobile parity + maintenance**: mobile toggle/badges + i18n,
   orphaned-worktree maintenance view, merged-branch detection for
   auto-reclaim hints.
