# Git workflow

The **Git** tab on the Inspector turns the session's working
directory into a self-contained PR command center: switch
branches, stage files, commit, push, open a pull request, watch
its CI checks, and merge — without leaving opendray.

The tab only renders when the cwd is inside a git repo. If `git
rev-parse` fails on the directory, the panel collapses to a
short "not a git repo" hint.

## Status header

The top of the tab summarises where you are:

- **Current branch** and its upstream (e.g. `feat/foo →
  origin/feat/foo`). When there's no upstream tracked the row
  reads "no upstream" — the **Push** button below will use
  `--set-upstream` on first push.
- **Ahead / behind counts** vs the upstream. `+3 / -1` means
  three local commits not on the remote and one remote commit
  not on you.
- A working-tree summary — counts of staged, unstaged, and
  untracked files. The numbers are clickable; they jump to the
  staging area below.

## Branch controls

The strip under the status header has every branch operation:

| Element | Purpose |
|---|---|
| Branch dropdown | Switch to any other local branch. The list is fetched once on tab open and refreshes when any other branch op succeeds. |
| **+ New** | Create a branch from current HEAD and switch to it. One-input modal. |
| **Push** | Push to upstream. First push (no upstream) uses `--set-upstream`. Disabled when ahead == 0 and an upstream is already set. |

### Stash & switch

Switching with a dirty tree is not blocked. If the server
detects uncommitted changes during a `checkout`, the response is
a **409 Conflict** whose body carries a `dirty_files: [...]`
array. The web UI catches that and surfaces a Sonner toast like:

> Uncommitted changes block switch to `main`
> &nbsp;&nbsp;`app/foo.ts, internal/bar.go, app/baz.tsx`
> &nbsp;&nbsp;[ Stash & switch ]

Clicking **Stash & switch** retries the checkout with
`stash: true`. The server runs `git stash push --include-untracked
-m "opendray-auto: switch to <branch>"` first, then performs the
checkout, and returns the stash short ref. The success toast
shows it explicitly so recovery is one terminal command away:

> Switched to `main` (stashed as `abc1234`)

Recover the stashed work later with `git stash pop` (or
`git stash apply` if you want to keep the stash around).

## Staging and commit

The middle pane lists every working-tree change with porcelain
status codes (`M`, `A`, `??`, etc.). Per-row actions:

- **Stage** moves an unstaged change to the index.
- **Unstage** is the reverse for staged rows.
- **Diff** opens a unified diff in a side drawer (read-only).

Bulk actions:

- **Stage all** = `git add .`
- **Unstage all** = `git reset`

The commit form sits below the file list:

- Multi-line **message** field. **Cmd / Ctrl + Enter** submits
  without needing to mouse over to the button.
- Live staged-count badge — the **Commit** button is disabled
  until at least one file is staged.
- On success the form clears, the working-tree status refreshes,
  and the new HEAD shows in the log strip on the right.

## Pull requests

Below the commit form is the **PR command center**. It's the
same surface for GitHub, Gitea, and GitLab — the gateway
normalises the API differences into one shape.

### Listing

The default view lists **open** PRs for the current remote.
Switch to **Closed** or **All** via the pill toggle above the
list. Each row shows:

| Column | Notes |
|---|---|
| `#NN` | Click → opens the PR in a new tab on the host. |
| Title | Truncated; full title on hover. |
| Author | Avatar + login. |
| Branches | `head → base` (e.g. `feat/foo → main`). |
| Aggregate checks | Coloured pill — `success`, `failure`, `pending`, `mixed`. See the Checks subsection. |

Clicking a row **expands it in place** to reveal the Checks list
and the Merge form.

### Creating

The **+ Create PR** button opens an inline form:

- **Title** — defaults to the last commit subject.
- **Body** — multi-line.
- **Head** — the current branch (read-only).
- **Base** — defaults to the host's default branch (`main` /
  `master` / whatever the repo is configured with). Resolved via
  the host API, not a hard-coded string. Override freely.

Submit → opens the PR, the new row appears at the top of the
list, expanded.

### Merging

Each expanded PR row carries a Merge form with the standard
GitHub merge methods:

- **Merge commit** — default `--merge`.
- **Squash and merge** — `--squash`. Recommended for the
  one-commit-per-PR workflow this project uses.
- **Rebase and merge** — `--rebase`. Use sparingly; rebasing
  loses the merge anchor.

**Delete branch on merge** is a checkbox below the method
toggle. Defaults to **on** — branches accumulate fast otherwise.

### Checks

Once a row is expanded, opendray polls the PR's checks every
30 s and shows them as a list. Vocabulary is normalised to
GitHub's:

| Status | Meaning |
|---|---|
| `success` | Job finished without errors. |
| `failure` | Job ran and failed. |
| `pending` | Queued or in progress. |
| `neutral` | Job finished without a pass/fail verdict. |
| `cancelled` | Killed before finishing. |
| `skipped` | Conditionally skipped by the workflow. |

Gitea and GitLab's commit-status endpoints map onto this
vocabulary so you can use the same UI without worrying about
which host emitted what.

The aggregate verdict on the list row is the worst-case
rollup: any `failure` → `failure`; otherwise any `pending` →
`pending`; otherwise `success`. Hover the badge for a tooltip
with the breakdown.

## When something goes wrong

The Git tab surfaces server errors as Sonner toasts in
red. Common ones:

- **`fatal: not a git repository`** — the cwd isn't tracked;
  the panel will go back to "not a git repo" empty state.
- **`Permission denied (publickey)`** on push — the gateway's
  Git credentials don't have write access to that remote.
  Check the PAT in **Settings → Git hosts**.
- **`branch '<name>' not fully merged`** on delete — git's
  safe-delete refusal. The toast offers a force-delete
  confirmation that upgrades to `git branch -D`.

## Backend routes

For wire-level debugging, the endpoints powering this tab live
under `/api/v1/git/...`:

| Method | Path | Purpose |
|---|---|---|
| GET | `/git/status` | Working-tree + branch summary. |
| GET | `/git/log` | Recent commits on HEAD. |
| GET | `/git/write/branches` | All branches (local + remote refs). |
| POST | `/git/write/branches` | Create + optional checkout. |
| DELETE | `/git/write/branches` | Delete (query: `name`, `force`). |
| POST | `/git/write/checkout` | Switch branch. `stash: true` body field auto-stashes a dirty tree. |
| POST | `/git/write/stage` `/unstage` | Index ops. |
| POST | `/git/write/commit` | Create a commit. |
| POST | `/git/write/push` | `--set-upstream` on first push. |
| GET | `/git/prs` | List PRs for the cwd's remote. |
| POST | `/git/prs` | Create. |
| POST | `/git/prs/{n}/merge` | Merge with method options. |
| GET | `/git/prs/{n}/checks` | Normalised check runs. |

Each route is auth-gated; the embedded web client wires the
session token through automatically.
