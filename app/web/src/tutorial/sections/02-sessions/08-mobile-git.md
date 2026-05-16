# Mobile Git workflow

The mobile app mirrors the web's [Git workflow](#02-sessions-07-git-workflow)
with a touch-first layout. Everything in this section is the
mobile UI's equivalent of the web Git tab — the gateway routes
are identical, so the two clients can operate on the same repo
without stepping on each other.

Open it under **Session detail → Git tab** (the second pill at
the top of the inspector pages).

## Status pane

The header echoes the web tab: current branch + upstream,
ahead/behind counts, working-tree file count. Tap the working
tree row to expand it into the file list.

## Branch picker (bottom sheet)

Tapping the current-branch chip opens a **bottom sheet** that
fills 75% of the viewport — there's no room for the web's
dropdown on a phone screen, and the sheet gives each branch
space for its upstream subtitle and a delete affordance.

| Section | What's listed |
|---|---|
| **Local** | Every `refs/heads/*` ref. Current branch is pinned to the top with a checkmark icon; the rest are alphabetical. |
| **Remote** | Every `refs/remotes/<remote>/*` ref except `HEAD` symrefs. Tapping a remote ref checks out the local branch of the same name (same as `git checkout <name>`'s remote-tracking shortcut). |

Each row:

- **Tap** the row → checkout that branch.
- **Tap the trash icon** on the right → delete the branch
  (disabled on the current branch).

The "+ New" and "Push" buttons sit on the action strip
alongside the chip, so the common ops are one tap from any
screen.

### Stash & switch

If the working tree is dirty when you tap a branch, the server
returns 409 + a `dirty_files` array. The mobile UI opens an
AlertDialog showing the file list (scrollable, monospace) with
two buttons:

- **Cancel** — leave the tree alone.
- **Stash & switch** — auto-stashes the tree
  (`git stash push --include-untracked`) and switches.

On success a Snackbar toast confirms `Switched to <branch>
(stashed as abc1234)`. Recover the stash later from a terminal
with `git stash pop`.

### Delete with force fallback

`-d` is the default. If git refuses with "not fully merged" the
server returns 409, and the UI surfaces a **second** dialog —
"Force delete? Branch is not fully merged. Forcing deletion
will lose any commits unique to this branch." The confirm
upgrades the call to `-D`. The two-step gating means a slip
can't blow away work.

## Commit form

A simplified form lives below the file list:

- **Stage all** button — there's no per-file stage UI on the
  mobile (no hover targets). Manage individual files from the
  web inspector or a terminal if you need finer control.
- **Multi-line message** field — the soft keyboard reveals an
  Enter button (added separately) so newlines work without
  hunting the toolbar.
- **Commit** button — disabled until something is staged.

After commit, the file list and status header refresh from the
server.

## Pull requests

Below the commit form is the **PR section**:

- A scrollable list of open PRs, polled every 60 seconds.
- **+ Create** opens a bottom-sheet form (title / body / base
  with default-branch resolution).
- Tapping a PR row expands an inline panel with the check
  runs (polled every 30 seconds while expanded) and the merge
  form.

The merge form's options match the web's: method (merge /
squash / rebase) and a "delete branch on merge" checkbox
defaulting to **on**.

## Polling cadence summary

The mobile app polls more aggressively than the web because
there's no event-bus subscription on iOS yet:

| Source | Cadence | Stops when |
|---|---|---|
| Branch list | On open + after any branch op | tab unmounts |
| Status / file list | 8 s | tab unmounts |
| PR list | 60 s | tab unmounts |
| PR checks (expanded row) | 30 s | row collapses |

These intervals are tuned to feel "live" without burning radio
on a phone. If you're plugged in and want faster updates,
trigger a manual refresh via the swipe-down on the list.

## Differences vs the web

The web Git tab has a few capabilities the mobile doesn't yet
mirror:

- **Per-file staging** — web has stage/unstage buttons next to
  every changed path. Mobile is stage-all only for now.
- **Diff drawer** — web opens a unified diff on tap. Mobile
  shows just the porcelain status code; viewing diffs needs the
  web or a terminal.
- **Force-with-lease push** — web has an opt-in checkbox.
  Mobile push always uses the safe path.

All other features (branches, PRs, stash & switch, etc.) are
on parity.
