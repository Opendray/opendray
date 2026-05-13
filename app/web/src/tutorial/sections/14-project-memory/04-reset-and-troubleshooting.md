# Reset & troubleshooting

## Reset project memory

A first-class action that wipes all per-cwd state in one
transaction. Useful when:

- You're done with a project and want to clear its memory.
- A project's goal / plan got fundamentally wrong and the
  fastest fix is "start over".
- You want a clean slate for a benchmark / demo.

**Where**:

- Web: Project screen header → outlined **Reset** button (red text)
- Mobile: Project screen AppBar → 🔄 IconButton

**Dialog**:

The reset confirm dialog shows the cwd in a red banner with the
warning **"Always deleted: goal, plan, proposals, journal,
cleanup decisions"**. Two opt-in checkboxes:

| Checkbox | What it does | Default |
|---|---|---|
| Also delete scanner docs | Wipes `tech_stack` + `recent_activity` too | **off** — they auto-rebuild on next spawn anyway |
| Also delete pgvector memories | Wipes the section 11 facts for this scope_key | **off** — these are long-term agent-stored facts that may be valuable; explicit opt-in |

The button is labeled **"Delete forever"** and styled
destructive. Confirmation is a single click — there's no
"type the cwd to confirm" because the explicit checkbox flow
plus the cwd-in-banner is enough for this risk level. The toast
on success shows concrete deletion counts per table.

Under the hood: `POST /api/v1/project-docs/reset` runs the doc /
proposal / journal / cleanup-decision delete in a single
transaction; the optional memory wipe is a separate
`POST /api/v1/memory/delete-by-scope` call from the UI when the
checkbox is on.

## Orphan scope_keys

When inspecting the **Cleanup inbox** or the Project picker you
may see entries marked `orphan`:

```
PROJECT  /Users/    [orphan]    15 pending
```

These are bug data from an old mirror import that truncated
source paths down to fragments like `/Users/`. They're not real
project cwds — opening them as a project lands on an empty
ProjectScreen, which is pointless.

The orphan heuristic: a scope_key with fewer than 2 non-empty
path segments. `/Users/` has 1 (just "Users"). Real projects
like `/tmp/foo` or `/Users/me/work/repo` have ≥ 2.

**Handling**:

- **Cleanup inbox**: orphan groups show the `orphan` badge and
  drop the "Open project" deep-link. You can still approve /
  reject the underlying decisions row-by-row — `stale` verdicts
  on orphan entries are the quickest way to clear the rot.
- **Project picker**: orphans are sorted to the bottom of the
  list, visually muted (opacity-60), and marked with a warning
  icon. You can still navigate into them if you really need to.

To bulk-delete: nothing built-in yet. SQL works:

```sql
DELETE FROM memories
  WHERE scope='project'
    AND array_length(string_to_array(trim(both '/' from scope_key), '/'), 1) < 2;
DELETE FROM memory_cleanup_decisions
  WHERE memory_scope='project'
    AND array_length(string_to_array(trim(both '/' from memory_scope_key), '/'), 1) < 2;
```

## Project isolation guarantees (M22)

The journaler that auto-writes session-end summaries (L4 of L5)
runs an LLM over the transcript. A misconfigured reader could
pick up an unrelated session's jsonl and the LLM would
confidently summarise the wrong work — silent
misinformation worse than no summary.

Three defenses in `internal/session/transcript.go`:

1. **Fail-closed on missing UUID file**: if the session row's
   `claude_session_id` is set but the named `*.jsonl` doesn't
   exist, the reader returns nil — never substitutes "latest
   mtime in dir".
2. **Time-window filter**: even when the right file is opened,
   each parsed turn must fall within `[startedAt - 30s,
   endedAt + 30s]`. Accumulated content from earlier spawns is
   filtered out.
3. **Cwd canary**: the first jsonl entry with a `cwd` field must
   exactly match the calling session's cwd. One mismatch and
   the whole file is rejected.

When any defense trips, the journaler degrades to metadata-only
(no `Agent activity summary` section appended). "We don't know
what happened" is the correct failure mode — never a
confidently-wrong summary.

If you suspect a contaminated journal entry, the SQL check:

```sql
SELECT id, cwd, content FROM session_logs
 WHERE content LIKE '%Agent activity summary%'
   AND content LIKE '%<some file path you didn''t expect to see>%';
```

…and cross-reference the mentioned file paths against the actual
session's jsonl tool_use blocks. New entries on M22 build should
be clean; old data from before M22 may need manual deletion.

## SQL recipes

### See everything for one cwd

```sql
SELECT 'project_docs' AS source, COUNT(*) AS n FROM project_docs
 WHERE cwd = '/your/path'
UNION ALL
SELECT 'session_logs', COUNT(*) FROM session_logs
 WHERE cwd = '/your/path'
UNION ALL
SELECT 'cleanup_decisions', COUNT(*) FROM memory_cleanup_decisions
 WHERE memory_scope = 'project' AND memory_scope_key = '/your/path'
UNION ALL
SELECT 'memories', COUNT(*) FROM memories
 WHERE scope = 'project' AND scope_key = '/your/path';
```

### Check tech_stack staleness

```sql
SELECT cwd,
       NOW() - updated_at AS age,
       length(content) AS bytes
  FROM project_docs
 WHERE kind = 'tech_stack'
 ORDER BY updated_at;
```

Anything `age > 6 hours` will trigger a sync re-scan on the next
spawn into that cwd.

### Cleaner decision quality skim

```sql
SELECT verdict, status,
       substring(memory_text_snapshot, 1, 50) AS preview,
       substring(reason, 1, 80) AS llm_reason
  FROM memory_cleanup_decisions
 ORDER BY created_at DESC LIMIT 20;
```

Skim the `llm_reason` column. If ≥ 80% read as sensible to you,
the librarian is calibrated. If not, the cleaner prompt or model
needs adjustment.

## When to ask for help

- Spawn injection text looks corrupt or wrong project → check
  the cwd matches the spawn's cwd (case-sensitive!) and the
  scanner has refreshed.
- Journal entries missing the `Agent activity summary` section
  → could be M22 isolation kicked in (good!), or summarizer
  provider not configured, or session was too short for the
  LLM "too sparse" guard. Check the server log for
  `journaler: transcript fetch failed` or `journaler: llm
  summarise failed`.
- Cleanup decisions pile up → either you're behind on triage or
  the librarian is misclassifying. Skim 20 and decide.
