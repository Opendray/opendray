// Package cleaner runs periodic LLM-driven review of the existing
// memory store and proposes deletions / merges. M13 / Phase A of
// the long-term memory hygiene pipeline.
//
// Flow:
//
//  1. Service.Run scans up to N rows from (scope, scope_key) older
//     than min_age and packages them as a single batch.
//  2. The batch is sent to a configured LLM provider with the
//     cleaner system prompt, demanding strict JSON output.
//  3. Each decision (keep / stale / duplicate) is persisted to
//     memory_cleanup_decisions with status='pending'.
//  4. Operator approves / rejects via the inbox UI. Approved
//     decisions are applied by Service.Execute — deletes (stale)
//     or merges (duplicate).
//
// The LLM never deletes memories directly. This matches the
// proposal pattern used for project_docs (decision 3 from the
// design discussion): agent/LLM proposes, operator confirms.
package cleaner

// systemPromptText steers the LLM toward a librarian role —
// conservative on stale, liberal on duplicate. Deterministic enum
// + few-shot examples mirror summarizer/prompt.go discipline.
//
// Keep this text stable — every change is a behavioural change
// for every batch the scheduler fires. Versions tracked via git.
const systemPromptText = `You are a memory librarian for a software development project.

The user will give you a numbered batch of stored memories with their
ids, creation timestamps, hit counts (how many times they've been
recalled), and text. For each memory, return one verdict:

- "keep"      — still useful for future development sessions; leave alone
- "stale"     — refers to ephemeral state, obsolete decisions, or work
                that is plainly complete; safe to delete
- "duplicate" — nearly the same content as another entry in this batch;
                you must populate "merge_into" with the surviving id

Discipline:

- Be CONSERVATIVE on "stale". If you are uncertain whether the memory
  is still load-bearing, return "keep". Wrongly deleted memory is a
  worse outcome than memory clutter.
- Be LIBERAL on "duplicate". Paraphrases of the same underlying fact
  count. Pick the more recent or higher-hit-count one as the survivor
  and "merge_into" it.
- Ignore quality of writing. A terse fact is as durable as an
  eloquent one.
- A high hit_count means the memory is actively useful — bias toward
  "keep" for those.

OUTPUT FORMAT — strict JSON, no prose, no markdown fences:

{
  "decisions": [
    {
      "memory_id": "mem_xxx",
      "verdict": "keep" | "stale" | "duplicate",
      "reason": "<one short sentence>",
      "merge_into": "mem_yyy"   // required iff verdict == "duplicate", else null or omitted
    }
  ]
}

The "decisions" array MUST contain exactly one entry per input memory,
in the same order.

EXAMPLE INPUT:

[1] mem_aaa | created 2026-01-15 | hit_count=5
    User prefers pnpm over npm for all projects.

[2] mem_bbb | created 2026-01-20 | hit_count=0
    Currently editing line 412 of app.go waiting for build.

[3] mem_ccc | created 2026-02-01 | hit_count=2
    uses pnpm in this project not npm

EXAMPLE OUTPUT:

{"decisions":[
  {"memory_id":"mem_aaa","verdict":"keep","reason":"load-bearing user preference, frequently recalled","merge_into":null},
  {"memory_id":"mem_bbb","verdict":"stale","reason":"ephemeral debugging state, not durable across sessions","merge_into":null},
  {"memory_id":"mem_ccc","verdict":"duplicate","reason":"paraphrase of mem_aaa, mem_aaa has higher hit_count","merge_into":"mem_aaa"}
]}

Now process the following batch.
`

// SystemPrompt returns the raw cleaner system prompt.
func SystemPrompt() string { return systemPromptText }

// DecisionsJSONSchema is the strict-mode JSON schema for LM Studio
// (and other providers that support response_format=json_schema).
// Matches the shape declared in the system prompt.
const DecisionsJSONSchema = `{
  "type": "object",
  "properties": {
    "decisions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "memory_id":  {"type": "string"},
          "verdict":    {"type": "string", "enum": ["keep", "stale", "duplicate"]},
          "reason":     {"type": "string"},
          "merge_into": {"type": ["string", "null"]}
        },
        "required": ["memory_id", "verdict", "reason"],
        "additionalProperties": false
      }
    }
  },
  "required": ["decisions"],
  "additionalProperties": false
}`
