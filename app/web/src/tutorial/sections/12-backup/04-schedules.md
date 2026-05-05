# Backup — schedules

A *schedule* is "take a backup automatically every N seconds and
prune anything beyond N kept rows."

## Cadence

v1 uses simple **interval** scheduling, not cron expressions.
You pick "every 24 hours / 6 hours / 30 minutes" via the New
schedule dialog (Schedules tab on `/backups`). cron syntax is
deferred to v1.1.

A scheduler goroutine wakes every 30 seconds and:

1. Atomically claims one due schedule via
   `SELECT … FOR UPDATE SKIP LOCKED LIMIT 1`. Multiple opendray
   instances cooperate safely on the same table.
2. Runs the backup synchronously (sequential per opendray
   instance — no parallel runs of the same schedule).
3. After success applies retention: keep N most recent
   `succeeded` backups *for that target*, soft-delete the rest
   (their blob is removed from the target; row stays at
   `status='deleted'` for audit).

## Retention semantics

`retention = 7` means "always keep the 7 newest succeeded
backups per target." Failed and deleted rows don't count — only
`succeeded` ones. The default in the UI is 7; the floor is 0
(meaning "delete every successful backup immediately after
making it," which only makes sense as a smoke test).

## What happens on crash

- A backup row stuck at `status='running'` (because opendray
  crashed mid-pipeline) is reset to `failed` with a clear
  marker error after 1 hour. Reset runs automatically at next
  startup.
- A schedule that crashed mid-run already had its
  `next_run_at` bumped; the failed run is in `backups` so you
  can see why it failed without it blocking the cadence.

## Disabling vs deleting

A schedule has `enabled` toggle separate from delete. Toggle
to off when you want to pause without losing the configuration.
Delete removes the schedule but leaves prior backups (they're
linked via `schedule_id` which becomes NULL — schema uses
`ON DELETE SET NULL`).

A target referenced by an enabled schedule **cannot be deleted**
(`ON DELETE RESTRICT`). Disable / delete the schedule first.
