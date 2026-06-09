package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
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
	KBKindReusable       = "kb_reusable"
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

// ProposalSink files an update PROPOSAL for a human-locked Knowledge page
// (B3 — Iterate). When new evidence diverges from a page the operator has
// locked, the drafter does not overwrite it; it proposes the refreshed draft
// for the operator to approve. The app adapts projectdoc's proposal flow.
type ProposalSink interface {
	HasPendingKBProposal(ctx context.Context, cwd, kind string) (bool, error)
	ProposeKBDoc(ctx context.Context, cwd, kind, content, reason string) error
}

// KBDrafter distils Memory + the graph into the global Knowledge pages.
type KBDrafter struct {
	store     *Store
	llm       LLM
	mem       MemorySource // P-G: declarative facts come straight from Memory
	docs      DocSink
	proposals ProposalSink // B3: propose updates to locked pages instead of skipping
	log       *slog.Logger
}

// NewKBDrafter builds a drafter. docs + llm are required to do anything.
func NewKBDrafter(store *Store, llm LLM, docs DocSink, log *slog.Logger) *KBDrafter {
	if log == nil {
		log = slog.Default()
	}
	return &KBDrafter{store: store, llm: llm, docs: docs, log: log.With("component", "knowledge.kb")}
}

// WithMemory wires episodic Memory as the declarative-fact feedstock for the
// infrastructure / conventions / reusable pages (P-G — fact nodes retired).
// Optional: without it those pages distil from entities/playbooks alone.
func (d *KBDrafter) WithMemory(src MemorySource) *KBDrafter {
	d.mem = src
	return d
}

// WithProposals enables B3 iteration: a locked page whose feedstock has
// diverged gets a refreshed draft filed as a proposal (not an overwrite).
func (d *KBDrafter) WithProposals(p ProposalSink) *KBDrafter {
	d.proposals = p
	return d
}

const kbSafety = `
Output human-readable GitHub-flavoured Markdown. Use "## " section headers and tight bullet lists.
Deduplicate aggressively — merge restatements of the same fact into one line.
NEVER include secrets: passwords, API keys, tokens, certificates' private material. If a value looks like a credential, omit it (you may name WHERE it is stored, never the value).
When information conflicts across time, describe only the CURRENT state — if something was renamed, deprecated, or replaced (e.g. an old tool/host/path superseded by a new one), present the current one and note the predecessor as deprecated; never present superseded state as current.
Be concise and factual. No preamble, no "here is", no markdown code fences around the whole document.`

// kbInfraSystem + kbConvSystem produce FOUNDATIONAL pages: standing ground
// truth + the rules for using it. They are injected into every project as
// BINDING constraints, so each page must end with an explicit, imperative
// "## Rules (MUST follow)" section separated from the descriptive facts.

const kbInfraSystem = `You curate the home-lab / ecosystem INFRASTRUCTURE reference from a developer's accumulated facts and entities.
Organize the FACTS into sections such as: Hosts & network, Databases, Gateways & services, Credential stores (names/locations only), Build & deploy targets, Domains.
Include concrete values that are NOT secrets (IPs, ports, hostnames, container names, paths, ID ranges).
Then end with a "## Rules (MUST follow)" section: the imperative rules for USING this infrastructure (e.g. which account to connect as, where credentials must be stored, ID ranges to allocate from). These are binding — phrase them as commands.` + kbSafety

const kbConvSystem = `You curate the DEVELOPMENT CONVENTIONS & policies the developer follows — the binding "how we work" rules.
Organize into sections such as: Package manager & stack, Source control (commits / PR / branching), Coding rules, Release & deploy process, Naming, Workflow, Language & model preferences.
Phrase every item as an imperative RULE the developer/agent must follow (not a description). End with a "## Rules (MUST follow)" section collecting the hardest must/never constraints.` + kbSafety

const kbLessonsSystem = `You curate a LESSONS / playbooks reference from already-distilled playbooks.
Group related playbooks under thematic "## " sections. For each, give a one-line how-to and the key pitfall. Keep it skimmable — this is the "what we learned the hard way" index.` + kbSafety

const kbReusableSystem = `You curate a REUSABLE FEATURES catalog from what has been built across the developer's projects.
List features / components / patterns / integrations that could be LIFTED into a NEW project, grouped under "## " themes. For each: what it is, which project it came from, and how to reuse it.
Only include things genuinely reusable across projects — skip one-off project specifics.` + kbSafety

// KBDraftResult reports the outcome of drafting one KB page (returned by the
// manual draft endpoint so failures are observable without log access).
type KBDraftResult struct {
	Kind   string `json:"kind"`
	Cwd    string `json:"cwd"`
	Status string `json:"status"` // written | skipped-empty | skipped-locked | skipped-unchanged | error
	Bytes  int    `json:"bytes,omitempty"`
	Err    string `json:"error,omitempty"`
}

// DraftAll refreshes the global Knowledge pages once. Knowledge is
// cross-project only (Experience Flywheel): per-project documentation lives in
// Notes, not here — there is no per-project handbook. Each page is lock-aware
// and dirty-checked, so an unchanged or human-edited page costs no LLM call.
func (d *KBDrafter) DraftAll(ctx context.Context) ([]KBDraftResult, error) {
	if d.llm == nil || d.docs == nil {
		return nil, nil
	}
	var facts []MemoryRow
	if d.mem != nil {
		facts, _ = d.mem.ListAllMemories(ctx, 400)
	}
	entities, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindEntity, Limit: 400})
	playbooks, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindPlaybook, Limit: 200})

	var out []KBDraftResult
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindInfrastructure, kbInfraSystem, buildInfraFeedstock(facts, entities)))
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindConventions, kbConvSystem, buildConvFeedstock(facts)))
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindLessons, kbLessonsSystem, buildLessonsFeedstock(playbooks)))
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindReusable, kbReusableSystem, buildReusableFeedstock(playbooks, facts)))
	return out, nil
}

func (d *KBDrafter) draftOne(ctx context.Context, cwd, kind, system, feedstock string) KBDraftResult {
	return draftOrPropose(ctx, d.llm, d.docs, d.proposals, d.log, cwd, kind, system, feedstock)
}

// draftOrPropose is the shared lock-aware, dirty-checked draft path used by the
// KB drafter and the Overview drafter. An unlocked page is rewritten in place;
// a human-locked page whose feedstock diverged is filed as an update proposal
// (B3 — Iterate) instead of overwritten.
func draftOrPropose(ctx context.Context, llm LLM, docs DocSink, proposals ProposalSink, log *slog.Logger, cwd, kind, system, feedstock string) KBDraftResult {
	res := KBDraftResult{Kind: kind, Cwd: cwd}
	if strings.TrimSpace(feedstock) == "" {
		res.Status = "skipped-empty"
		return res
	}
	cur, err := docs.GetKBDoc(ctx, cwd, kind)
	if err != nil {
		log.Warn("draft: get doc failed", "kind", kind, "cwd", cwd, "err", err)
		res.Status, res.Err = "error", "get: "+err.Error()
		return res
	}
	sig := kbSig(feedstock)
	if cur.Exists && extractKBSig(cur.Content) == sig {
		res.Status = "skipped-unchanged"
		return res // feedstock unchanged since last draft — nothing to do
	}
	if cur.HumanLocked {
		if proposals == nil {
			res.Status = "skipped-locked"
			return res
		}
		if pending, _ := proposals.HasPendingKBProposal(ctx, cwd, kind); pending {
			res.Status = "skipped-pending"
			return res
		}
		body, err := draftPageBody(ctx, llm, log, system, feedstock, sig)
		if err != nil {
			res.Status, res.Err = "error", err.Error()
			return res
		}
		if err := proposals.ProposeKBDoc(ctx, cwd, kind, body,
			"New evidence has diverged from this locked page; review the refreshed draft."); err != nil {
			log.Warn("draft: propose failed", "kind", kind, "cwd", cwd, "err", err)
			res.Status, res.Err = "error", "propose: "+err.Error()
			return res
		}
		log.Info("update proposed for locked page", "kind", kind, "cwd", cwd)
		res.Status, res.Bytes = "proposed", len(body)
		return res
	}
	body, err := draftPageBody(ctx, llm, log, system, feedstock, sig)
	if err != nil {
		res.Status, res.Err = "error", err.Error()
		return res
	}
	if err := docs.PutKBDoc(ctx, cwd, kind, body); err != nil {
		log.Warn("draft: put doc failed", "kind", kind, "cwd", cwd, "err", err)
		res.Status, res.Err = "error", "put: "+err.Error()
		return res
	}
	log.Info("page drafted", "kind", kind, "cwd", cwd)
	res.Status, res.Bytes = "written", len(body)
	return res
}

// draftPageBody runs the LLM and returns the cleaned page body with the
// feedstock signature appended (so the next sweep's dirty-check can skip it).
func draftPageBody(ctx context.Context, llm LLM, log *slog.Logger, system, feedstock, sig string) (string, error) {
	body, err := llm.Complete(ctx, system, feedstock)
	if err != nil {
		log.Warn("draft: llm failed", "err", err)
		return "", fmt.Errorf("llm: %w", err)
	}
	body = stripFences(strings.TrimSpace(body))
	if body == "" {
		return "", fmt.Errorf("empty llm output")
	}
	return body + fmt.Sprintf("\n\n<!-- kb-sig:%s -->\n", sig), nil
}

// --- feedstock builders ---

func buildInfraFeedstock(facts []MemoryRow, entities []Node) string {
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

func buildConvFeedstock(facts []MemoryRow) string {
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

func buildReusableFeedstock(playbooks []Node, facts []MemoryRow) string {
	var b strings.Builder
	if len(playbooks) > 0 {
		b.WriteString("PLAYBOOKS (how things were built):\n")
		for _, p := range playbooks {
			b.WriteString("- ")
			b.WriteString(p.Title)
			b.WriteByte('\n')
		}
	}
	b.WriteString("\nFACTS (mine for built features / components / integrations worth reusing):\n")
	writeFactTitles(&b, facts)
	return b.String()
}

func writeFactTitles(b *strings.Builder, facts []MemoryRow) {
	for _, f := range facts {
		t := factTitle(f.Text)
		if t == "" || t == "(empty fact)" {
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
