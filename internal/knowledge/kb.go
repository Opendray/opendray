package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// --- M-KB: curated, human-readable Knowledge Base pages -----------------------
//
// The KB fuses INTO the note system (projectdoc): each page is a projectdoc Doc
// (cwd, kind, content). The drafter distils the graph's facts/entities/playbooks
// + the project journal into clean Markdown pages — the human-readable side of
// the knowledge brain — while the same content is what the AI reads on new work.
//
// Three GLOBAL pages (under GlobalKBCwd): Infrastructure, Conventions, Lessons.
// One HANDBOOK per real project cwd. AI drafts (author=agent); a human edit
// (author=operator) locks a page from further AI overwrite.

// GlobalKBCwd mirrors projectdoc.GlobalCwd. knowledge owns no projectdoc import
// (one-way rule); the app guarantees the two constants match.
const GlobalKBCwd = "__global__"

const (
	KBKindInfrastructure = "kb_infrastructure"
	KBKindConventions    = "kb_conventions"
	KBKindLessons        = "kb_lessons"
	KBKindHandbook       = "kb_handbook"
)

// KBDoc is the current state of a KB page as the drafter sees it.
type KBDoc struct {
	Content     string
	HumanLocked bool // a human edit (operator-authored) locks it from AI overwrite
	Exists      bool
}

// DocSink persists curated KB pages into the note system (projectdoc-backed in
// the app). Kinds are the KBKind* constants; cwd is GlobalKBCwd for globals.
type DocSink interface {
	GetKBDoc(ctx context.Context, cwd, kind string) (KBDoc, error)
	PutKBDoc(ctx context.Context, cwd, kind, content string) error
}

// KBDrafter distils graph + journal into curated KB pages via the LLM.
type KBDrafter struct {
	store   *Store
	llm     LLM
	journal JournalSource
	docs    DocSink
	log     *slog.Logger
}

// NewKBDrafter builds a drafter. journal is optional (handbooks degrade to
// facts-only without it); docs + llm are required to do anything.
func NewKBDrafter(store *Store, llm LLM, journal JournalSource, docs DocSink, log *slog.Logger) *KBDrafter {
	if log == nil {
		log = slog.Default()
	}
	return &KBDrafter{store: store, llm: llm, journal: journal, docs: docs, log: log.With("component", "knowledge.kb")}
}

const kbSafety = `
Output human-readable GitHub-flavoured Markdown. Use "## " section headers and tight bullet lists.
Deduplicate aggressively — merge restatements of the same fact into one line.
NEVER include secrets: passwords, API keys, tokens, certificates' private material. If a value looks like a credential, omit it (you may name WHERE it is stored, never the value).
Be concise and factual. No preamble, no "here is", no markdown code fences around the whole document.`

const kbInfraSystem = `You curate the home-lab / ecosystem INFRASTRUCTURE reference from a developer's accumulated facts and entities.
Organize into sections such as: Hosts & network, Databases, Gateways & services, Credential stores (names/locations only), Build & deploy targets, Domains.
Include concrete values that are NOT secrets (IPs, ports, hostnames, container names, paths, ID ranges).` + kbSafety

const kbConvSystem = `You curate the DEVELOPMENT CONVENTIONS & habits reference from a developer's accumulated facts.
Organize into sections such as: Package manager & stack, Source control (commits / PR / branching), Coding rules, Release & deploy process, Naming, Workflow, Language & model preferences.
Capture the RULES the developer follows, as imperative bullets.` + kbSafety

const kbLessonsSystem = `You curate a LESSONS / playbooks reference from already-distilled playbooks.
Group related playbooks under thematic "## " sections. For each, give a one-line how-to and the key pitfall. Keep it skimmable — this is the "what we learned the hard way" index.` + kbSafety

const kbHandbookSystem = `You curate a PROJECT HANDBOOK from one project's work log (journal), facts, and playbooks.
Organize into: What it is, Tech stack & layout, How to build / run / deploy, Infrastructure it uses, Collaboration boundaries, Key lessons & pitfalls.
Prefer concrete commands / paths / hosts from the log.` + kbSafety

// DraftAll refreshes every KB page once: the three global pages + one handbook
// per non-ephemeral project. Each page is lock-aware and dirty-checked, so an
// unchanged or human-edited page costs no LLM call.
func (d *KBDrafter) DraftAll(ctx context.Context) error {
	if d.llm == nil || d.docs == nil {
		return nil
	}
	facts, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindFact, Limit: 400})
	entities, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindEntity, Limit: 400})
	playbooks, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindPlaybook, Limit: 200})

	d.draftOne(ctx, GlobalKBCwd, KBKindInfrastructure, kbInfraSystem, buildInfraFeedstock(facts, entities))
	d.draftOne(ctx, GlobalKBCwd, KBKindConventions, kbConvSystem, buildConvFeedstock(facts))
	d.draftOne(ctx, GlobalKBCwd, KBKindLessons, kbLessonsSystem, buildLessonsFeedstock(playbooks))

	keys, err := d.store.ListProjectScopeKeys(ctx)
	if err != nil {
		return err
	}
	for _, k := range keys {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if isEphemeralCwd(k) || k == GlobalKBCwd {
			continue
		}
		pf, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindFact, Scope: ScopeProject, ScopeKey: k, Limit: 200})
		pp, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindPlaybook, Scope: ScopeProject, ScopeKey: k, Limit: 100})
		var jr []JournalEntry
		if d.journal != nil {
			jr, _ = d.journal.ListJournal(ctx, k, 50)
		}
		if len(pf) == 0 && len(jr) == 0 {
			continue
		}
		d.draftOne(ctx, k, KBKindHandbook, kbHandbookSystem, buildHandbookFeedstock(k, pf, jr, pp))
	}
	return nil
}

func (d *KBDrafter) draftOne(ctx context.Context, cwd, kind, system, feedstock string) {
	if strings.TrimSpace(feedstock) == "" {
		return
	}
	cur, err := d.docs.GetKBDoc(ctx, cwd, kind)
	if err != nil {
		d.log.Warn("kb: get doc failed", "kind", kind, "cwd", cwd, "err", err)
		return
	}
	if cur.HumanLocked {
		return // operator owns this page — never overwrite
	}
	sig := kbSig(feedstock)
	if cur.Exists && extractKBSig(cur.Content) == sig {
		return // feedstock unchanged since last draft — skip the LLM call
	}
	body, err := d.llm.Complete(ctx, system, feedstock)
	if err != nil {
		d.log.Warn("kb: draft failed", "kind", kind, "cwd", cwd, "err", err)
		return
	}
	body = stripFences(strings.TrimSpace(body))
	if body == "" {
		return
	}
	body += fmt.Sprintf("\n\n<!-- kb-sig:%s -->\n", sig)
	if err := d.docs.PutKBDoc(ctx, cwd, kind, body); err != nil {
		d.log.Warn("kb: put doc failed", "kind", kind, "cwd", cwd, "err", err)
		return
	}
	d.log.Info("kb page drafted", "kind", kind, "cwd", cwd)
}

// --- feedstock builders ---

func buildInfraFeedstock(facts, entities []Node) string {
	var b strings.Builder
	b.WriteString("ENTITIES (grouped by type):\n")
	byType := map[EntityType][]string{}
	for _, e := range entities {
		byType[e.EntityType] = append(byType[e.EntityType], e.Title)
	}
	for _, t := range []EntityType{"host", "service", "tool", "tech"} {
		if v := byType[t]; len(v) > 0 {
			fmt.Fprintf(&b, "%s: %s\n", t, strings.Join(dedupSorted(v), ", "))
		}
	}
	b.WriteString("\nFACTS (mine the infrastructure-relevant ones):\n")
	writeFactTitles(&b, facts)
	return b.String()
}

func buildConvFeedstock(facts []Node) string {
	var b strings.Builder
	b.WriteString("FACTS (mine the conventions / habits / rules):\n")
	writeFactTitles(&b, facts)
	return b.String()
}

func buildLessonsFeedstock(playbooks []Node) string {
	if len(playbooks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("PLAYBOOKS:\n")
	for _, p := range playbooks {
		b.WriteString("\n## ")
		b.WriteString(p.Title)
		b.WriteByte('\n')
		if body := strings.TrimSpace(p.Body); body != "" {
			if len(body) > 700 {
				body = body[:700] + "…"
			}
			b.WriteString(body)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func buildHandbookFeedstock(cwd string, facts []Node, journal []JournalEntry, playbooks []Node) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PROJECT: %s\n\n", cwd)
	if len(journal) > 0 {
		b.WriteString("WORK LOG (session traces):\n")
		for _, j := range journal {
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
	if len(facts) > 0 {
		b.WriteString("\nFACTS:\n")
		writeFactTitles(&b, facts)
	}
	if len(playbooks) > 0 {
		b.WriteString("\nPLAYBOOKS (lessons):\n")
		for _, p := range playbooks {
			b.WriteString("- ")
			b.WriteString(p.Title)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func writeFactTitles(b *strings.Builder, facts []Node) {
	for _, f := range facts {
		t := strings.TrimSpace(f.Title)
		if t == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(t)
		b.WriteByte('\n')
	}
}

func dedupSorted(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		k := strings.ToLower(strings.TrimSpace(s))
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}
	return out
}

// --- signature + fence helpers ---

func kbSig(feedstock string) string {
	sum := sha256.Sum256([]byte(feedstock))
	return hex.EncodeToString(sum[:])[:16]
}

func extractKBSig(content string) string {
	const marker = "<!-- kb-sig:"
	i := strings.LastIndex(content, marker)
	if i < 0 {
		return ""
	}
	rest := content[i+len(marker):]
	if j := strings.IndexByte(rest, ' '); j >= 0 {
		return strings.TrimSpace(rest[:j])
	}
	return ""
}

// stripFences removes a leading ```lang / trailing ``` wrapper if the model
// wrapped the whole document in a code fence.
func stripFences(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimSuffix(strings.TrimRight(s, "\n"), "```")
	return strings.TrimSpace(s)
}

// KBSweepConfig tunes the background KB-drafting loop.
type KBSweepConfig struct {
	Interval     time.Duration // between sweeps (default 1h)
	InitialDelay time.Duration // before the first sweep (default 7m)
}

func (c KBSweepConfig) withDefaults() KBSweepConfig {
	if c.Interval <= 0 {
		c.Interval = time.Hour
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = 7 * time.Minute
	}
	return c
}

// RunKBSweep blocks until ctx is cancelled, periodically refreshing all KB
// pages. Dirty-checked + lock-aware, so steady-state cost is ~0 LLM calls.
func (d *KBDrafter) RunKBSweep(ctx context.Context, cfg KBSweepConfig) {
	if d.llm == nil || d.docs == nil {
		return
	}
	cfg = cfg.withDefaults()
	d.log.Info("knowledge KB sweep running", "interval", cfg.Interval)
	timer := time.NewTimer(cfg.InitialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if err := d.DraftAll(ctx); err != nil && ctx.Err() == nil {
			d.log.Warn("kb sweep cycle failed", "err", err)
		}
		timer.Reset(cfg.Interval)
	}
}
