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
type Reflector struct {
	store *Store
	llm   LLM
	log   *slog.Logger
}

// NewReflector builds a Reflector over the shared pool and an LLM.
func NewReflector(pool *pgxpool.Pool, llm LLM, log *slog.Logger) *Reflector {
	if log == nil {
		log = slog.Default()
	}
	return &Reflector{store: NewStore(pool), llm: llm, log: log.With("component", "knowledge.reflect")}
}

type draftPlaybook struct {
	Title       string   `json:"title"`
	AppliesWhen string   `json:"applies_when"`
	Steps       []string `json:"steps"`
	Pitfalls    []string `json:"pitfalls"`
}

const reflectSystem = `You distill durable PROCEDURAL knowledge (playbooks) from a project's facts.
Given the facts and the titles of playbooks that already exist, output ONLY new, genuinely reusable playbooks as JSON:
{"playbooks":[{"title":"...","applies_when":"...","steps":["..."],"pitfalls":["..."]}]}
Rules:
- Emit a playbook ONLY when the facts support a repeatable how-to. If none, return {"playbooks":[]}.
- Never duplicate an existing playbook title.
- title = short imperative, e.g. "Deploy a Go service to Proxmox LXC".
- steps = concrete + ordered; pitfalls = known failure modes.
- JSON only: no prose, no markdown fences.`

// ReflectProject drafts new playbooks for one project from its facts. Returns
// the number of playbooks created.
func (r *Reflector) ReflectProject(ctx context.Context, scopeKey string, minFacts int) (int, error) {
	facts, err := r.store.ListNodes(ctx, NodeFilter{Kind: KindFact, Scope: ScopeProject, ScopeKey: scopeKey, Limit: 200})
	if err != nil {
		return 0, err
	}
	if len(facts) < minFacts {
		return 0, nil
	}
	existing, err := r.store.ListNodes(ctx, NodeFilter{Kind: KindPlaybook, Scope: ScopeProject, ScopeKey: scopeKey, Limit: 200})
	if err != nil {
		return 0, err
	}
	raw, err := r.llm.Complete(ctx, reflectSystem, buildReflectInput(facts, existing))
	if err != nil {
		return 0, err
	}
	existingTitles := map[string]struct{}{}
	for _, p := range existing {
		existingTitles[strings.ToLower(strings.TrimSpace(p.Title))] = struct{}{}
	}
	projID := ProjectEntityID(scopeKey)
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
	return n, nil
}

func buildReflectInput(facts, existing []Node) string {
	var b strings.Builder
	b.WriteString("FACTS:\n")
	for _, f := range facts {
		b.WriteString("- ")
		b.WriteString(f.Title)
		b.WriteByte('\n')
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

// ReflectSweepConfig tunes the background reflection loop.
type ReflectSweepConfig struct {
	Interval     time.Duration // between sweeps (default 30m)
	InitialDelay time.Duration // before the first sweep (default 5m)
	MinFacts     int           // skip projects with fewer facts (default 5)
}

func (c ReflectSweepConfig) withDefaults() ReflectSweepConfig {
	if c.Interval <= 0 {
		c.Interval = 30 * time.Minute
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = 5 * time.Minute
	}
	if c.MinFacts <= 0 {
		c.MinFacts = 5
	}
	return c
}

// RunReflectSweep blocks until ctx is cancelled, periodically drafting
// playbooks across all projects. Soft-fails every step.
func (r *Reflector) RunReflectSweep(ctx context.Context, cfg ReflectSweepConfig) {
	cfg = cfg.withDefaults()
	r.log.Info("knowledge reflect sweep running", "interval", cfg.Interval, "min_facts", cfg.MinFacts)
	timer := time.NewTimer(cfg.InitialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		keys, err := r.store.ListProjectScopeKeys(ctx)
		if err != nil {
			r.log.Warn("reflect: list projects failed", "err", err)
		} else {
			total := 0
			for _, k := range keys {
				select {
				case <-ctx.Done():
					return
				default:
				}
				n, err := r.ReflectProject(ctx, k, cfg.MinFacts)
				if err != nil {
					r.log.Warn("reflect project failed", "cwd", k, "err", err)
					continue
				}
				total += n
			}
			if total > 0 {
				r.log.Info("reflect sweep done", "playbooks", total)
			}
		}
		timer.Reset(cfg.Interval)
	}
}
