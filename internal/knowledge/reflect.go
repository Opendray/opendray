package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Reflector is the Phase 3 graduation engine: it reads a project's facts and,
// when a repeatable how-to emerges, drafts a playbook node — the fact ->
// playbook step on the maturity axis (the literal "self-evolution"). LLM-
// driven and best-effort; a no-op without an LLM or below the fact threshold.
// JournalEntry is one project work-trace (a session journal entry) — the real
// record of what was done / fixed / learned. It is the PRIMARY feedstock for
// distilling procedural playbooks (declarative facts alone are too thin).
type JournalEntry struct {
	Title     string
	Content   string
	Kind      string
	CreatedAt time.Time
}

// JournalSource is the read-only view of the project journal the reflector
// needs. The app wires a projectdoc-backed adapter (one-way dependency rule).
type JournalSource interface {
	ListJournal(ctx context.Context, scopeKey string, limit int) ([]JournalEntry, error)
}

// LifecycleFilter reports whether a project (cwd) is frozen (paused/archived).
// P-D: a frozen project is excluded from per-project distillation. Optional —
// a nil filter treats every project as active. The app adapts projectdoc's
// GetStatus to this; knowledge keeps no projectdoc import.
type LifecycleFilter interface {
	IsFrozen(ctx context.Context, cwd string) bool
}

type Reflector struct {
	store     *Store
	llm       LLM
	journal   JournalSource   // optional; work-trace feedstock for playbooks
	mem       MemorySource    // P-G: declarative facts come straight from Memory now
	lifecycle LifecycleFilter // P-D: skip frozen projects
	log       *slog.Logger
}

// NewReflector builds a Reflector over the shared pool and an LLM.
func NewReflector(pool *pgxpool.Pool, llm LLM, log *slog.Logger) *Reflector {
	if log == nil {
		log = slog.Default()
	}
	return &Reflector{store: NewStore(pool), llm: llm, log: log.With("component", "knowledge.reflect")}
}

// WithJournal wires the project journal as the primary reflection feedstock so
// playbooks are distilled from real work-traces, not just declarative facts.
func (r *Reflector) WithJournal(src JournalSource) *Reflector {
	r.journal = src
	return r
}

// WithMemory wires episodic Memory as the declarative-fact feedstock (P-G —
// fact nodes are retired; Memory is the fact store). Optional: without it the
// reflector distils from the journal alone.
func (r *Reflector) WithMemory(src MemorySource) *Reflector {
	r.mem = src
	return r
}

// WithLifecycle installs the optional P-D lifecycle filter so frozen
// (paused/archived) projects are skipped during distillation.
func (r *Reflector) WithLifecycle(f LifecycleFilter) *Reflector {
	r.lifecycle = f
	return r
}

type draftPlaybook struct {
	Title       string   `json:"title"`
	AppliesWhen string   `json:"applies_when"`
	Steps       []string `json:"steps"`
	Pitfalls    []string `json:"pitfalls"`
}

const reflectSystem = `You distill durable, reusable PROCEDURAL knowledge (playbooks) from a project's work log.
The WORK LOG holds real session traces — what was built, how a bug was root-caused and fixed, what was decided. Mine it for repeatable how-tos.
Output ONLY new, genuinely reusable playbooks as JSON:
{"playbooks":[{"title":"...","applies_when":"...","steps":["..."],"pitfalls":["..."]}]}
Rules:
- Emit a playbook ONLY when the log supports a CONCRETE, repeatable procedure someone could follow next time. If nothing rises to that bar, return {"playbooks":[]}.
- Prefer fix-with-root-cause and multi-step how-tos. NEVER turn a single declarative fact (a preference, a value, a config) into a playbook.
- title = short imperative, e.g. "Ship an unreleased build to the local Mac".
- applies_when = the trigger/situation. steps = concrete + ordered, reusing the real commands / paths / file names from the log. pitfalls = the actual failure modes that were hit.
- Never duplicate an existing playbook title.
- Reflect the CURRENT way of doing things — if the log shows a tool / path / process was replaced or deprecated, base the playbook on the current one, not the superseded predecessor.
- JSON only: no prose, no markdown fences.`

// ReflectProject drafts new playbooks for one project from its facts. Returns
// the number of playbooks created.
func (r *Reflector) ReflectProject(ctx context.Context, scopeKey string, minFacts int) (int, error) {
	var facts []MemoryRow
	if r.mem != nil {
		if fs, ferr := r.mem.ListProjectMemories(ctx, scopeKey, 200); ferr != nil {
			r.log.Warn("reflect: list memories failed", "cwd", scopeKey, "err", ferr)
		} else {
			facts = fs
		}
	}
	var journal []JournalEntry
	if r.journal != nil {
		if js, jerr := r.journal.ListJournal(ctx, scopeKey, 50); jerr != nil {
			r.log.Warn("reflect: list journal failed", "cwd", scopeKey, "err", jerr)
		} else {
			journal = js
		}
	}
	// Need real signal: a work-trace journal OR a minimum number of facts.
	if len(journal) == 0 && len(facts) < minFacts {
		return 0, nil
	}
	projID := ProjectEntityID(scopeKey)
	// Dirty-check — skip the (paid) LLM call when the feedstock is unchanged
	// since the last reflection. This is the main steady-state token saving.
	sig := reflectSignature(facts, journal)
	if prev, _ := r.store.ReflectSig(ctx, projID); prev != "" && prev == sig {
		return 0, nil
	}
	existing, err := r.store.ListNodes(ctx, NodeFilter{Kind: KindPlaybook, Scope: ScopeProject, ScopeKey: scopeKey, Limit: 200})
	if err != nil {
		return 0, err
	}
	raw, err := r.llm.Complete(ctx, reflectSystem, buildReflectInput(journal, facts, existing))
	if err != nil {
		return 0, err
	}
	existingTitles := map[string]struct{}{}
	for _, p := range existing {
		existingTitles[strings.ToLower(strings.TrimSpace(p.Title))] = struct{}{}
	}
	n := 0
	for _, d := range parsePlaybooks(raw) {
		title := strings.TrimSpace(d.Title)
		if title == "" {
			continue
		}
		if _, dup := existingTitles[strings.ToLower(title)]; dup {
			continue
		}
		node, err := r.store.CreateNode(ctx, Node{
			Kind:       KindPlaybook,
			Title:      title,
			Body:       renderPlaybookBody(d),
			Scope:      ScopeProject,
			ScopeKey:   scopeKey,
			Maturity:   MaturityPlaybook,
			Provenance: map[string]any{"source": "reflector", "from_facts": len(facts)},
		})
		if err != nil {
			r.log.Warn("playbook create failed", "title", title, "err", err)
			continue
		}
		_ = r.store.CreateEdge(ctx, Edge{SrcID: node.ID, EdgeType: EdgeAbout, DstID: projID})
		existingTitles[strings.ToLower(title)] = struct{}{}
		n++
	}
	// Record the feedstock signature so an unchanged project is skipped next sweep.
	if err := r.store.SetReflectSig(ctx, projID, sig); err != nil {
		r.log.Warn("reflect: set sig failed", "cwd", scopeKey, "err", err)
	}
	return n, nil
}

// reflectSignature fingerprints a project's feedstock so an unchanged project
// can be skipped (counts + newest journal time catch new facts / journal).
func reflectSignature(facts []MemoryRow, journal []JournalEntry) string {
	var newest int64
	for _, j := range journal {
		if u := j.CreatedAt.Unix(); u > newest {
			newest = u
		}
	}
	return fmt.Sprintf("f%d:j%d:%d", len(facts), len(journal), newest)
}

func buildReflectInput(journal []JournalEntry, facts []MemoryRow, existing []Node) string {
	var b strings.Builder
	if len(journal) > 0 {
		b.WriteString("WORK LOG (real session traces — what was built, root-caused, fixed, decided; the primary source for procedures):\n")
		for _, j := range journal {
			b.WriteString("- ")
			if t := strings.TrimSpace(j.Title); t != "" {
				b.WriteString(t)
				b.WriteString(": ")
			}
			c := strings.TrimSpace(j.Content)
			if len(c) > 600 {
				c = c[:600] + "…"
			}
			b.WriteString(strings.ReplaceAll(c, "\n", " "))
			b.WriteByte('\n')
		}
	}
	if len(facts) > 0 {
		b.WriteString("\nKNOWN FACTS (supporting context — do not turn a lone fact into a playbook):\n")
		for _, f := range facts {
			b.WriteString("- ")
			b.WriteString(factTitle(f.Text))
			b.WriteByte('\n')
		}
	}
	if len(existing) > 0 {
		b.WriteString("\nEXISTING PLAYBOOK TITLES (do not duplicate):\n")
		for _, p := range existing {
			b.WriteString("- ")
			b.WriteString(p.Title)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func renderPlaybookBody(d draftPlaybook) string {
	var b strings.Builder
	if strings.TrimSpace(d.AppliesWhen) != "" {
		b.WriteString("**Applies when:** ")
		b.WriteString(strings.TrimSpace(d.AppliesWhen))
		b.WriteString("\n\n")
	}
	if len(d.Steps) > 0 {
		b.WriteString("## Steps\n")
		for i, s := range d.Steps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, strings.TrimSpace(s))
		}
	}
	if len(d.Pitfalls) > 0 {
		b.WriteString("\n## Pitfalls\n")
		for _, p := range d.Pitfalls {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(p))
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}

func parsePlaybooks(raw string) []draftPlaybook {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '{'); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndexByte(raw, '}'); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	var parsed struct {
		Playbooks []draftPlaybook `json:"playbooks"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	return parsed.Playbooks
}

// ReflectAll runs one reflection pass across all projects, drafting playbooks
// for each. Exported so the consolidation engine (P-C) can sequence it after
// anchoring without owning the project-iteration logic. Returns the total
// playbook count. Per-project errors are logged and skipped.
func (r *Reflector) ReflectAll(ctx context.Context, minFacts int) (int, error) {
	keys, err := r.store.ListProjectScopeKeys(ctx)
	if err != nil {
		return 0, fmt.Errorf("reflect: list projects: %w", err)
	}
	total := 0
	for _, k := range keys {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		// P-D — a frozen (paused/archived) project is shelved: don't distil it.
		if r.lifecycle != nil && r.lifecycle.IsFrozen(ctx, k) {
			continue
		}
		n, err := r.ReflectProject(ctx, k, minFacts)
		if err != nil {
			r.log.Warn("reflect project failed", "cwd", k, "err", err)
			continue
		}
		total += n
	}
	if total > 0 {
		r.log.Info("reflect sweep done", "playbooks", total)
	}
	return total, nil
}
