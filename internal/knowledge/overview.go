package knowledge

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// --- Overview: the project's rich, AI-maintained official document -----------
//
// Notes' structured fields (goal/plan/tech/journal) are too thin to be "the
// project's official documentation" — the thing a developer reads to understand
// the whole project's features and foundations. The Overview is that document:
// one comprehensive, human-readable page per project, AI-drafted from the
// project's own signals (tech-stack scan + goal/plan + journal + memory) and
// kept current. It is a Notes doc (projectdoc kind "overview"), per-project —
// NOT cross-project Knowledge.
//
// Same discipline as the KB pages: dirty-checked (skip when feedstock is
// unchanged), lock-aware (a human edit freezes it; further AI updates arrive as
// proposals via B3), and frozen-project-aware.

// OverviewKind is the projectdoc kind the Overview is stored under.
const OverviewKind = "overview"

const overviewSystem = `You write a software project's OFFICIAL OVERVIEW — the single document a new developer reads to understand the whole project.
You are given the project's path, its goal + current plan, a scan of its tech stack / structure, its recent work log, and accumulated facts.
Produce comprehensive, accurate GitHub-flavoured Markdown with these sections (omit a section only if there is genuinely nothing to say):

## What it is
One short paragraph: what the project does and who/what it's for.

## Features & capabilities
The concrete things it does, as a bulleted list grouped where helpful.

## Architecture & key components
How it's put together — the main parts/modules/services and how they relate. Name real components/paths from the scan + log.

## Tech stack
Languages, frameworks, datastores, key libraries.

## How to build / run / deploy
Concrete commands / entry points from the log + scan.

## Foundations & infrastructure it relies on
External services, databases, hosts, conventions it must follow.

## Status & current focus
Where the project is now (from the plan + recent log).

Rules:
- Ground everything in the supplied material; do NOT invent features or commands. If something is unknown, omit it rather than guess.
- NEVER include secrets (passwords, keys, tokens). Name where a credential lives, never its value.
- Describe only the CURRENT state; if something was replaced/deprecated, present the current one.
- No preamble, no "here is", no outer code fence.`

// OverviewDrafter writes the per-project Overview doc. It reuses the KB
// drafter's dependencies (LLM, DocSink, ProposalSink) plus the project's own
// Notes/journal/memory as feedstock.
type OverviewDrafter struct {
	store     *Store
	llm       LLM
	mem       MemorySource
	journal   JournalSource
	docs      DocSink
	proposals ProposalSink
	lifecycle LifecycleFilter
	log       *slog.Logger
}

// NewOverviewDrafter builds the drafter. docs + llm are required; the rest are
// optional and degrade the feedstock gracefully when absent.
func NewOverviewDrafter(store *Store, llm LLM, docs DocSink, log *slog.Logger) *OverviewDrafter {
	if log == nil {
		log = slog.Default()
	}
	return &OverviewDrafter{store: store, llm: llm, docs: docs, log: log.With("component", "knowledge.overview")}
}

func (o *OverviewDrafter) WithMemory(m MemorySource) *OverviewDrafter { o.mem = m; return o }
func (o *OverviewDrafter) WithJournal(j JournalSource) *OverviewDrafter {
	o.journal = j
	return o
}
func (o *OverviewDrafter) WithProposals(p ProposalSink) *OverviewDrafter {
	o.proposals = p
	return o
}
func (o *OverviewDrafter) WithLifecycle(f LifecycleFilter) *OverviewDrafter {
	o.lifecycle = f
	return o
}

// DraftAll refreshes the Overview for every known, active, non-ephemeral
// project. Returns per-project results (same shape as the KB drafter).
func (o *OverviewDrafter) DraftAll(ctx context.Context) ([]KBDraftResult, error) {
	if o.llm == nil || o.docs == nil {
		return nil, nil
	}
	keys, err := o.store.ListProjectScopeKeys(ctx)
	if err != nil {
		return nil, err
	}
	var out []KBDraftResult
	for _, k := range keys {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		if isEphemeralCwd(k) || k == GlobalKBCwd {
			continue
		}
		if o.lifecycle != nil && o.lifecycle.IsFrozen(ctx, k) {
			continue
		}
		feedstock := o.buildFeedstock(ctx, k)
		// Overview keeps its historical behaviour: regenerate from the project
		// feedstock (no operator-owned-form opts — those are KB-only).
		out = append(out, draftOrPropose(ctx, o.llm, o.docs, o.proposals, o.log, k, OverviewKind, overviewSystem, feedstock, draftOpts{}))
	}
	return out, nil
}

// buildFeedstock assembles the project's own signals for the overview: its
// goal + plan + tech-stack scan (read back through the DocSink), a slice of
// recent journal, and accumulated project memory facts.
func (o *OverviewDrafter) buildFeedstock(ctx context.Context, cwd string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PROJECT PATH: %s\n", cwd)

	for _, kind := range []string{"goal", "plan", "tech_stack"} {
		if d, err := o.docs.GetKBDoc(ctx, cwd, kind); err == nil {
			if body := strings.TrimSpace(stripKBSigText(d.Content)); body != "" {
				fmt.Fprintf(&b, "\n%s:\n%s\n", strings.ToUpper(kind), body)
			}
		}
	}

	if o.journal != nil {
		if js, err := o.journal.ListJournal(ctx, cwd, 40); err == nil && len(js) > 0 {
			b.WriteString("\nWORK LOG (recent session traces):\n")
			for _, j := range js {
				b.WriteString("- ")
				if t := strings.TrimSpace(j.Title); t != "" {
					b.WriteString(t)
					b.WriteString(": ")
				}
				c := strings.TrimSpace(j.Content)
				if len(c) > 400 {
					c = c[:400] + "…"
				}
				b.WriteString(strings.ReplaceAll(c, "\n", " "))
				b.WriteByte('\n')
			}
		}
	}

	if o.mem != nil {
		if ms, err := o.mem.ListProjectMemories(ctx, cwd, 200); err == nil && len(ms) > 0 {
			b.WriteString("\nFACTS:\n")
			writeFactTitles(&b, ms)
		}
	}
	return b.String()
}

// stripKBSigText removes the hidden kb-sig marker line so the overview feedstock
// reflects only real content (the goal/plan docs may carry one from drafting).
func stripKBSigText(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "kb-sig:") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}
