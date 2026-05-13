# Picking a worker per task

This is opinionated — the right answer depends on your hardware,
your Claude budget, and how often you spawn sessions. The
following recommendations come from running opendray on a Mac
Studio with LM Studio (qwen-3.5-9b) + a Claude Pro subscription.

## Gatekeeper — keep on Summarizer

Forced by design (see overview). Tweak: pin a small fast model
specifically for the gatekeeper if your default summarizer is a
larger one.

Config in **Memory → Workers → Gatekeeper**:

- Worker: `Summarizer` (locked)
- Provider: pick a small model row if you have one; otherwise
  leave as "Registry default"

## Cleaner — Summarizer (default), Agent only if local is too dumb

The cleaner judges memory entries as keep / stale / duplicate.
This is a structured-output task with a clear JSON schema and
clear criteria. A 7-13B local model handles it well in our
testing — judgement quality on aged ephemeral entries
("currently debugging X") is solid.

Switch to Agent (Claude) only when:

- Your local model returns inconsistent verdicts (mix of `keep`
  for clearly stale entries).
- Your memory store has lots of subtle near-duplicates that
  semantic similarity didn't merge but a smart reviewer would.

24h frequency means the cost per run is bounded. Each run
processes up to `batch_size` (default 20) memories in one prompt
— that's one Claude API call. With agent worker, ~10s + a few
cents per run.

## Gitactivity — Agent (Claude) recommended

This is the strongest case for agent worker. The summariser
reads 7 days of `git log` and produces a 2-3 paragraph narrative
that every subsequent agent reads on spawn. Quality compounds.

Local 9-13B models tend to produce generic summaries ("the
project worked on memory and mobile changes"). Claude Opus
produces specific, file-level insights ("the M16-M17 work
introduced auto-capture tech stack scanning across
`internal/projectscan/`, then M22 added three layers of
transcript isolation in `internal/session/transcript.go` after a
production bug surfaced via cross-session jsonl confusion").

Config in **Memory → Workers → Git activity summariser**:

- Worker: `Agent`
- CLI: `Claude`
- Claude account: whichever account you've authed for this
  workspace

Fires once per 24h or on stale-spawn (>12h since last refresh).
Cost: ~1 Claude API call per day per active project — bounded
and predictable.

## Transcript — Agent (Claude) recommended for active projects

Per session-end summarisation. The frequency is higher than
gitactivity (every session vs. once a day), but each call is
short (transcript is capped at 16 KB, so ~4k input tokens).

Config: same as gitactivity. The "too sparse" guard in the
system prompt means trivial sessions (a single "hi") cost
nothing — Claude returns empty `<summary></summary>` and the
journal stays metadata-only.

Switch back to summarizer if:

- You spawn dozens of micro-sessions per day and the cost adds
  up.
- You don't care about narrative-quality session summaries (the
  metadata-only fall-back is still useful).

## A reasonable starting config

| Task | Worker | Why |
|---|---|---|
| Gatekeeper | Summarizer (small model) | Hot path, locked anyway |
| Cleaner | Summarizer (default) | Local handles structured judgement fine |
| Gitactivity | **Agent — Claude** | Best quality / cost ratio; runs ≤ 1x/day |
| Transcript | **Agent — Claude** | Session journals you'll actually read |

Total Claude usage: ~5-15 API calls/day on an actively-used
project. Cost: well under $1/day at Pro tier.

## How to test before saving

For every row, the **Test** button runs a one-line synthetic
prompt and reports the round-trip latency. Use it to:

- Sanity-check that the picked Claude account is actually authed
  on this host before the next 24h tick.
- Compare summarizer vs. agent latency for a given task before
  flipping the row — gives you a real number, not a guess.
- Validate that a `gemini` agent worker even works on your box
  (gemini CLI presence is not pre-checked).
