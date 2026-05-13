# Verification & metrics

After flipping a touchpoint to a new worker, you want to confirm
two things:

1. The new worker actually works (no auth / network issues).
2. The latency / quality tradeoff matches what the docs promised.

## Test connection (immediate)

Every worker card has a **Test** button. It runs:

```text
system: You are a connectivity test. Reply with the single word OK and nothing else.
user:   ping
```

through the configured worker with a 60s timeout and shows:

- **OK** + duration in ms + the first 200 chars of the response.
- **Error** + the exact error message (HTTP status / process
  exit / timeout / auth refusal).

Use cases:

- After picking an agent worker for the first time, before the
  next 24h tick fires. If `claude --print` isn't installed or
  the picked account isn't authed, the test surfaces that
  instantly.
- Before approving a config change for a high-frequency
  touchpoint (cleaner / transcript), do the test to see actual
  latency from your machine. Numbers > 10s suggest a misconfig
  (slow Claude account, network) — don't save.

## 24h rollup (mid-term)

Each worker card shows three stats for the last 24h:

```text
N calls · 24h
avg Xms
Y errors      (only when > 0)
```

These come from the `memory_worker_calls` audit table. Refresh
the page after a few hours of activity to see the rolling
window update.

What to watch for:

- **Avg ms** trending up over days = an upstream slowdown
  (Claude congested, LM Studio swapping). Compare with last
  week — if it's consistent, accept; if it's a regression, dig.
- **Error count > 0**: expand the "Recent calls" section to
  see which calls failed and what the error was. Common causes:
  - Agent CLI missing on host (`exec: "claude": executable file
    not found`)
  - Claude account quota exhausted (`HTTP 429`)
  - Network hiccups (`context deadline exceeded`)
- **Zero calls** for a touchpoint after enabling: maybe the
  scheduler hasn't fired yet (cleaner / gitactivity are 24h),
  or the trigger event hasn't happened (no session ended for
  transcript). Wait or force.

## Forcing a worker call

For tasks that have manual triggers:

- **Cleaner**: Project → Cleanup tab → "Run cleanup now"
- **Gitactivity**: `POST /api/v1/git-activity/run` against a
  cwd that has a stale row

Run one, then refresh the Workers page to see the new metric
row.

For tasks without manual triggers (gatekeeper, transcript) the
only validation path is the **Test** button + waiting for the
next natural event.

## Recent calls table

Each worker card has a collapsible "Recent calls (N)" section
showing the last 25 invocations with:

| Column | What |
|---|---|
| when | Local time of the call |
| worker | summarizer or agent · provider id |
| ms | Duration |
| ok | ✓ on success, ⚠ on error |

Drill into failed calls to read the full error_message — the
backend records it verbatim, including stderr from agent CLI
spawns.

## Rolling back

If a switch isn't working out:

1. **Memory → Workers** for the task.
2. Change worker back to the previous value.
3. **Save** — effective on the next call, no restart.

The metrics rollup recovers within minutes — old high-latency
calls drop out of the 24h window naturally.

## SQL access

For deeper analysis the audit table is plain:

```sql
SELECT task, worker_kind, provider_id,
       COUNT(*) AS n,
       ROUND(AVG(duration_ms)) AS avg_ms,
       COUNT(*) FILTER (WHERE NOT success) AS errors
  FROM memory_worker_calls
 WHERE started_at > NOW() - INTERVAL '7 days'
 GROUP BY task, worker_kind, provider_id
 ORDER BY task, avg_ms;
```

Use this when you want to compare past 7 days against the past
24h, or to compute cost / volume numbers the UI doesn't show.
