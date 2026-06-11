# The Experience Compiler — ground-up rework of distillation + graph

> Status: DESIGN APPROVED FOR IMPLEMENTATION (operator demanded a
> qualitative leap, not patches). Replaces the reflect→playbook→
> skillify pipeline and the Impact/graph surface. Implementation runs
> from this document.

## 0. The paradigm error being corrected

The current pipeline distills "whatever a single project's recent log
supports". That is **distillation for distillation's sake**: even with
quality gates, it produces *descriptions of things that happened once*.
The operator's definition of value is different:

> A skill is something we REPEAT. Its value = how much manual time it
> removes from future work.

So the unit of distillation is not "a session log" — it is a
**recurring procedure across the whole history of work**, and the
output is not prose — it is a **compiled, executable, feedback-tracked
artifact**.

## 1. The pipeline (replaces Reflector)

```
        all sessions, all projects (journal + transcripts)
                          │
            ① TRACE EXTRACTION (cheap LLM, per session-end)
                "procedure trace": {intent, steps[], outcome ok/fail, duration}
                stored in procedure_traces table
                          │
            ② RECURRENCE CLUSTERING (no LLM — embeddings)
                cluster traces by embedding similarity (pgvector,
                threshold ~0.83) ACROSS projects
                          │
            ③ CANDIDATE SYNTHESIS (strong LLM, only for clusters
                with ≥2 SUCCESSFUL traces)
                merge the cluster's traces into ONE canonical procedure:
                trigger, parameterised steps (what varies between runs
                becomes a parameter), pitfalls (from the failed traces!),
                verification, evidence = trace excerpts + session ids
                          │
            ④ COMPILATION (strong LLM + operator review)
                pick target form, best first:
                  a. script (.sh/.mjs) with --dry-run     → vault/skills/<slug>/run.sh
                  b. opendray custom task                  → one-click from UI
                  c. SKILL.md instructions (fallback)      → as today
                          │
            ⑤ FEEDBACK LOOP (no LLM)
                skill loaded → session outcome recorded → effectiveness
                score; loaded-but-abandoned twice ⇒ auto-propose retire;
                failed twice after env change ⇒ auto-propose re-compile
```

Key mechanical guarantees (not prompt hopes):
- **nothing with <2 successful occurrences is ever synthesised** —
  recurrence is computed by clustering, the LLM never decides it;
- pitfall candidates keep today's gate but require either recurrence
  OR a failed trace with a root cause + working alternative;
- every candidate carries `saves_minutes_estimate = avg(trace
  duration) × recurrence` — the ranking key.

## 2. Data model

```sql
-- ① raw traces (new)
procedure_traces (
  id, session_id, cwd, intent TEXT, steps JSONB, outcome TEXT
    CHECK (outcome IN ('success','failure','unclear')),
  duration_seconds INT, embedding vector, created_at
)
-- ② clusters are computed, not stored; cluster id = hash of member ids
-- ③ candidates: knowledge_nodes kind='playbook' gains
--    provenance: {cluster_traces:[ids], recurrence:N, failures:N,
--                 saves_minutes:M, parameters:[...]}
-- ④ compiled skills: kind='skill' gains provenance.compiled_form
--    ('script'|'task'|'instructions') + artifact path
-- ⑤ outcomes: skill_outcomes (skill_id, session_id, loaded BOOL,
--    outcome TEXT, created_at)
```

## 3. UI — the workbench becomes a decision queue

Distillation tab (full replacement of the two-column view):

```
┌─ 候选（按省时价值排序）────────────────────────────────┐
│ ▸ Deploy a Nuxt app update to PDA-web LXC               │
│   出现 4 次 · 3 成功 1 失败 · 均耗时 ~12min · 估省 48min │
│   [证据 ▾]  [参数: app_dir, lxc_ip]                      │
│   ┌──────────────────────────────────────────┐          │
│   │ 编译为: ●脚本 ○自定义任务 ○指令文档        │          │
│   └──────────────────────────────────────────┘          │
│   [编译并启用]   [仅存为文档]   [丢弃]                    │
└─────────────────────────────────────────────────────────┘
┌─ 已编译技能 ────────────────────────────────────────────┐
│ ssh-lxc-provision   脚本 · 用过 7 次 · 成功率 100% · [开关]│
│ release-to-testflight 任务 · 用过 3 次 · 2 周未用 [退役?] │
└─────────────────────────────────────────────────────────┘
```

- Every candidate answers "为什么值得": recurrence, success ratio,
  time saved, evidence quotes inline-expandable.
- Compile choice is the operator's; scripts always generated with
  `--dry-run` and shown for review before "编译并启用".
- Skill rows show effectiveness (loaded→succeeded ratio), not just
  use count; retire proposals appear here.

## 4. Graph → the compiler's x-ray (replaces Impact-as-list)

The graph earns its place by serving the pipeline, not by existing:

1. **Recurrence map** (primary view): clusters as rows — which
   procedures recur, across which projects, trend over time. This IS
   the graph (trace→cluster→project edges) rendered as a worklist.
2. **Blast radius** (kept from Impact view): entity → dependents.
   Wired into compilation: a skill that touches entity X lists X's
   dependents as pre-flight warnings in its compiled script.
3. The raw node browser stays deleted.

## 5. What gets deleted

- Reflector's per-project playbook drafting (reflect.go synthesis
  path) — trace extraction replaces it. The structural gate
  (validateDraftPlaybook) survives as the floor for ③'s output.
- 'Automate:' title-prefix heuristic — recurrence clustering IS the
  automation detector now.
- The skills/playbooks two-column workbench.

## 6. Implementation order (each step green + committed)

1. `procedure_traces` table + trace extractor (worker task `trace`,
   prompt + strict schema + outcome heuristics) hooked into
   session-end (journaler), embedding via memory embedder.
2. Clustering job in the consolidation engine (pgvector neighbour
   query, union-find; no LLM).
3. Candidate synthesis (worker task `synthesize`) for qualifying
   clusters; structural gate floor; saves_minutes computation.
4. Compilation: script generation with dry-run + review; custom-task
   emission; SKILL.md fallback. skill_outcomes + effectiveness.
5. Workbench UI (decision queue) + recurrence map + blast-radius
   pre-flight; retire/recompile proposals.
6. Backfill: run trace extraction over existing session_logs history
   so the queue is warm on day one.

## 7. Cost control

Stage ① uses the capture-grade cheap model per session end (one call);
②⑤ are SQL/embedding-only; ③④ use strong models but fire only for
clusters with proven recurrence — by construction, the expensive calls
happen exactly where value is already demonstrated.
