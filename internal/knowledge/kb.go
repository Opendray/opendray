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
	// MaintainerMode + PromptHint come from the page's blueprint section and
	// let the OPERATOR own the page's form: "human" hands the page entirely to
	// the operator (the drafter never touches it); PromptHint steers the AI
	// maintainer's shape/scope when the mode is "ai". Empty MaintainerMode is
	// treated as "ai" (the historical default).
	MaintainerMode string
	PromptHint     string
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
	// RejectedSigs returns the feedstock signatures of proposals the
	// operator has already REJECTED for this page. The drafter uses them to
	// avoid re-proposing an identical refresh every cycle: a rejection with
	// no memory would otherwise resurface the same draft on the next sweep
	// (the ~15m consolidation interval), nagging the operator forever.
	RejectedSigs(ctx context.Context, cwd, kind string) ([]string, error)
}

// KBDrafter distils Memory + the graph into the global Knowledge pages.
type KBDrafter struct {
	store     *Store
	llm       LLM
	mem       MemorySource // P-G: declarative facts come straight from Memory
	docs      DocSink
	proposals ProposalSink    // B3: propose updates to locked pages instead of skipping
	lifecycle LifecycleFilter // Cortex Phase 2: frozen projects' facts leave the feedstock
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

// WithLifecycle installs the lifecycle filter so paused/archived projects'
// facts stop feeding the cross-project Knowledge pages (Cortex Phase 2 —
// previously DraftAll distilled from ALL memories regardless of status).
func (d *KBDrafter) WithLifecycle(f LifecycleFilter) *KBDrafter {
	d.lifecycle = f
	return d
}

// filterFrozenFacts drops facts whose project (scope key) is frozen.
// IsFrozen results are memoised per cwd — feedstock batches repeat the
// same handful of projects hundreds of times.
func filterFrozenFacts(ctx context.Context, lc LifecycleFilter, facts []MemoryRow) []MemoryRow {
	if lc == nil || len(facts) == 0 {
		return facts
	}
	frozen := make(map[string]bool)
	out := make([]MemoryRow, 0, len(facts))
	for _, f := range facts {
		st, ok := frozen[f.ScopeKey]
		if !ok {
			st = lc.IsFrozen(ctx, f.ScopeKey)
			frozen[f.ScopeKey] = st
		}
		if st {
			continue
		}
		out = append(out, f)
	}
	return out
}

const kbSafety = `
Output human-readable GitHub-flavoured Markdown. Use "## " section headers and tight bullet lists.
Deduplicate aggressively — merge restatements of the same fact into one line.
NEVER include secrets: passwords, API keys, tokens, certificates' private material. If a value looks like a credential, omit it (you may name WHERE it is stored, never the value).
When information conflicts across time, describe only the CURRENT state — if something was renamed, deprecated, or replaced (e.g. an old tool/host/path superseded by a new one), present the current one and note the predecessor as deprecated; never present superseded state as current.
Be concise and factual. No preamble, no "here is", no markdown code fences around the whole document.`

// kbBaseSystem is the shared, FORM-NEUTRAL maintainer prompt for the four
// global KB pages. It deliberately mandates NO section skeleton — the page's
// shape belongs to the operator, expressed by editing the page itself and/or
// its blueprint prompt_hint. The drafter only folds genuinely new, on-topic
// evidence into whatever structure the operator has established. (Historically
// each page had a hardcoded skeleton here — Databases/Domains/Rules/… — which
// re-manufactured the same shape on every sweep and made operator curation
// impossible to keep. That is intentionally gone.)
const kbBaseSystem = `You maintain a long-lived, operator-owned knowledge page.
You receive the page's CURRENT content and fresh EVIDENCE (facts / entities / playbooks) from recent work.
Keep the page accurate by folding in genuinely new, on-topic evidence — nothing more.
RULES:
- PRESERVE the operator's existing structure, section headings, ordering, and voice EXACTLY. Never re-title, reorder, split, or merge sections the operator established, and never impose a template of your own.
- Only ADD or CORRECT individual items the evidence supports; leave everything else as-is.
- If (and only if) the current page is EMPTY, create a minimal, clean starting point — a short intro and a few obvious bullets. Do not pad it into an elaborate template.
- The operator's maintainer hint (if present below) is AUTHORITATIVE on this page's form and scope.` + kbSafety

// kbTopics is the one-line SCOPE descriptor per page — what the page is about,
// NOT how it must be structured. Used only to tell the maintainer which
// evidence is on-topic; the operator owns the actual form.
var kbTopics = map[string]string{
	KBKindInfrastructure: "PAGE TOPIC: infrastructure ground truth — hosts, networks, databases, gateways/services, and the binding rules for using them.",
	KBKindConventions:    "PAGE TOPIC: the developer's binding development conventions & policies (how we work).",
	KBKindLessons:        "PAGE TOPIC: distilled lessons and playbooks from past work — reference guidance.",
	KBKindReusable:       "PAGE TOPIC: features / components / patterns / integrations reusable across projects.",
}

// KBDraftResult reports the outcome of drafting one KB page (returned by the
// manual draft endpoint so failures are observable without log access).
type KBDraftResult struct {
	Kind   string `json:"kind"`
	Cwd    string `json:"cwd"`
	Status string `json:"status"` // written | proposed | skipped-empty | skipped-human | skipped-locked | skipped-pending | skipped-rejected | skipped-unchanged | error
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
		facts = filterFrozenFacts(ctx, d.lifecycle, facts)
	}
	entities, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindEntity, Limit: 400})
	playbooks, _ := d.store.ListNodes(ctx, NodeFilter{Kind: KindPlaybook, Limit: 200})

	var out []KBDraftResult
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindInfrastructure, buildInfraFeedstock(facts, entities)))
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindConventions, buildConvFeedstock(facts)))
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindLessons, buildLessonsFeedstock(playbooks)))
	out = append(out, d.draftOne(ctx, GlobalKBCwd, KBKindReusable, buildReusableFeedstock(playbooks, facts)))
	return out, nil
}

// draftOne maintains one global KB page. The operator owns its form, so the
// drafter honours the page's blueprint maintainer_mode (human → hands off),
// steers on its prompt_hint, and PRESERVES the current content's structure —
// it edits the page rather than regenerating it from a fixed skeleton.
func (d *KBDrafter) draftOne(ctx context.Context, cwd, kind, feedstock string) KBDraftResult {
	system := kbBaseSystem + "\n\n" + kbTopics[kind]
	return draftOrPropose(ctx, d.llm, d.docs, d.proposals, d.log, cwd, kind, system, feedstock,
		draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true})
}

// draftOpts turns on the operator-owned-form behaviours. They default OFF so
// the Overview drafter (which shares this path) is unaffected; only the global
// KB drafter opts in.
type draftOpts struct {
	// honorMaintainerMode: a page whose blueprint maintainer_mode is "human"
	// is the operator's to write by hand — the drafter never touches it.
	honorMaintainerMode bool
	// preserveCurrent: feed the current page content to the LLM so it EDITS
	// the operator's structure in place instead of regenerating from scratch.
	preserveCurrent bool
	// applyPromptHint: append the page's blueprint prompt_hint to the system
	// prompt so the operator can steer form/scope without a code change.
	applyPromptHint bool
}

// draftOrPropose is the shared lock-aware, dirty-checked draft path used by the
// KB drafter and the Overview drafter. An unlocked page is rewritten in place;
// a human-locked page whose feedstock diverged is filed as an update proposal
// (B3 — Iterate) instead of overwritten. opts enables the KB-only
// operator-owned-form behaviours (Overview passes the zero value).
func draftOrPropose(ctx context.Context, llm LLM, docs DocSink, proposals ProposalSink, log *slog.Logger, cwd, kind, system, feedstock string, opts draftOpts) KBDraftResult {
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
	// The operator fully owns a human-maintained page — never draft or propose.
	if opts.honorMaintainerMode && cur.MaintainerMode == "human" {
		res.Status = "skipped-human"
		return res
	}
	// Fold the operator's prompt_hint into the dirty-check signature. The hint
	// steers the page's form, so changing it must invalidate the cached draft —
	// otherwise (sig keys on feedstock alone) a new hint would never take effect
	// until the feedstock happened to diverge. Overview passes applyPromptHint
	// off, so its signature stays feedstock-only (byte-identical behaviour).
	hint := ""
	if opts.applyPromptHint {
		hint = strings.TrimSpace(cur.PromptHint)
	}
	sig := kbSig(feedstock)
	if hint != "" {
		sig = kbSig(feedstock + "\x00prompt_hint:" + hint)
	}
	if cur.Exists && extractKBSig(cur.Content) == sig {
		res.Status = "skipped-unchanged"
		return res // feedstock + hint unchanged since last draft — nothing to do
	}
	if hint != "" {
		system += "\n\nOPERATOR MAINTAINER HINT (authoritative on this page's form and scope):\n" + hint
	}
	current := ""
	if opts.preserveCurrent {
		current = cur.Content
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
		// A rejected refresh must stay rejected: if the operator already said
		// no to the draft for THIS feedstock, don't re-file it. Otherwise the
		// same proposal resurfaces every consolidation cycle. On lookup error
		// we fail open (proceed to propose) rather than silently drop a real
		// divergence.
		if rejected, err := proposals.RejectedSigs(ctx, cwd, kind); err != nil {
			log.Warn("draft: rejected-sig lookup failed", "kind", kind, "cwd", cwd, "err", err)
		} else {
			for _, rs := range rejected {
				if rs == sig {
					res.Status = "skipped-rejected"
					return res
				}
			}
		}
		body, err := draftPageBody(ctx, llm, log, system, current, feedstock, sig)
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
	body, err := draftPageBody(ctx, llm, log, system, current, feedstock, sig)
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

// draftUserMessage builds the LLM user turn. With no current content (Overview,
// or a first-time KB page) it is just the feedstock — byte-identical to the
// historical behaviour. When current content is supplied (KB edit-in-place) it
// frames the current page as the structure to preserve and the feedstock as
// new evidence to fold in.
func draftUserMessage(current, feedstock string) string {
	cur := stripKBSigText(strings.TrimSpace(current))
	if cur == "" {
		return feedstock
	}
	return "CURRENT PAGE (preserve this structure exactly; change only what the evidence updates):\n" +
		cur + "\n\nNEW EVIDENCE (fold in only what is genuinely new and on-topic):\n" + feedstock
}

// draftPageBody runs the LLM and returns the cleaned page body with the
// feedstock signature appended (so the next sweep's dirty-check can skip it).
func draftPageBody(ctx context.Context, llm LLM, log *slog.Logger, system, current, feedstock, sig string) (string, error) {
	body, err := llm.Complete(ctx, system, draftUserMessage(current, feedstock))
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

// ExtractKBSig reads the feedstock signature a drafted page carries in its
// trailing `<!-- kb-sig:… -->` marker, or "" if absent. Exported so the app's
// proposal sink can recover the signature of a rejected proposal's content
// (which was produced by draftPageBody and therefore carries the marker).
func ExtractKBSig(content string) string { return extractKBSig(content) }

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
