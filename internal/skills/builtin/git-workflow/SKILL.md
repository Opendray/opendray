---
name: git-workflow
description: Run git status / log / diff and list pull requests for the session's repo via the opendray gateway. Use whenever the user asks about "what's changed", "status", "diff", "open PRs", or wants to commit / push.
---

# git-workflow

The opendray gateway exposes read-only git inspection endpoints
backed by the `git` binary in the session's cwd, plus PR listing
across configured hosts (GitHub, Gitea, GitLab). Use them instead of
shelling out to `git` directly when you want structured output that's
easy to summarise back to the user — falls back to plain `git` for
anything not covered.

## When to use

- User asks "what's changed", "status", "what files did I touch"
- User wants to see a recent commit, a file's diff, or a PR list
- Before committing, to verify staged vs unstaged
- Before opening a PR, to check the diff against `HEAD` or origin

For destructive ops (commit, push, branch, rebase, …) keep using the
plain `git` CLI — those are not exposed via the gateway and are safer
to run with the user's explicit confirmation.

## Endpoints (gateway runs at http://127.0.0.1:8770 by default)

All endpoints accept `path=<absolute cwd>` and require the admin
session token. Inside an opendray-spawned session, you can rely on the
local gateway being reachable; `curl -s` works.

```
GET /api/v1/git/status?path=<dir>           → branch, ahead/behind, files
GET /api/v1/git/log?path=<dir>&n=20         → recent commits
GET /api/v1/git/diff?path=<dir>&scope=all   → unified diff (HEAD)
GET /api/v1/git/diff?path=<dir>&scope=staged&file=<rel>
GET /api/v1/git/show?path=<dir>&hash=<sha>  → commit + patch
GET /api/v1/git/remote?path=<dir>           → owner/repo + token state
GET /api/v1/git/prs?path=<dir>&state=open   → list pull requests
```

Most of the time the plain CLI is faster and the user expects
familiar git output. Reach for the gateway endpoints when:

- you want to surface a structured PR list (no `gh`/`tea` setup needed)
- the user wants you to *summarise* a long diff without dumping all
  of it into the chat — fetch the JSON, then paraphrase

## Patterns

### Quick status summary

```bash
git -C "$PWD" status --short
git -C "$PWD" log --oneline -5
```

### Review the working tree before committing

```bash
git diff --stat HEAD
git diff --cached  # staged
git diff           # unstaged
```

### Show recent open PRs without per-CLI auth

```bash
curl -s "http://127.0.0.1:8770/api/v1/git/prs?path=$PWD&state=open" \
  | jq '.prs[] | {number, title, author, head, base}'
```

If `need_token: true` comes back, tell the user to add a token in the
opendray Plugins page (Git hosts section) for the matching host.

## Conventions

This repo uses **conventional commits**:
`<type>: <description>` where type ∈ {feat, fix, refactor, docs, test, chore, perf, ci}.

Branch naming: `<type>/<short-slug>` (e.g. `feat/notes-vault`).

PRs should:
- have a 1-3 sentence summary at the top
- include a Test Plan checklist
- reference any related issue / decision note in `projects/<basename>/decisions/`

## Safety

- Never run destructive git ops without confirming the working tree
  state and asking the user
- For force-push, rebase main, branch -D — always pause and confirm
- Read-only endpoints are safe to call freely; PR listing hits remote
  APIs (GitHub/Gitea/GitLab), which counts against rate limits
