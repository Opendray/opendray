package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// --- The experience compiler -------------------------------------------------
//
// Ground-up rework of skill distillation (replaces the per-project Reflector).
// The old pipeline asked an LLM "does this one project's log contain a
// procedure?" — which produced plausible prose from single occurrences. The
// compiler is outcome-driven instead:
//
//	mine    — pull work EPISODES (session journal entries + the session's
//	          real outcome and duration) across ALL projects
//	cluster — group episodes by embedding similarity: "the same kind of
//	          work happened more than once" is detected mechanically,
//	          not asserted by a model
//	qualify — only a cluster with ≥2 SUCCESSFUL sessions becomes a
//	          candidate; recurrence × the procedure's time cost ranks
//	          what is worth distilling first
//	compile — one LLM call per qualifying cluster drafts the procedure,
//	          quoting evidence per episode (quotes are verified against
//	          the feedstock — fabricated evidence is rejected), and, when
//	          the procedure is fully mechanical, an executable script
//	          with a mandatory validation step
//
// validateDraftPlaybook stays as the structural floor on top of all of this.

// Episode is one cross-project work trace: a session's journal summary plus
// the session's outcome. This is the compiler's only feedstock — skills come
// from what REALLY happened repeatedly, never from a single session.
type Episode struct {
	SessionID string
	Cwd       string
	Title     string
	Content   string
	CreatedAt time.Time
	Duration  time.Duration // how long the session ran — the manual time cost proxy
	Success   bool          // did the session end cleanly (exit 0 / operator stop)

	// Stored vector from the journal's embed backfill, when available.
	// Episodes whose Embedder doesn't match the compiler's are re-embedded.
	Embedding []float32
	Embedder  string
}

// EpisodeSource feeds the compiler. The app wires a SQL adapter joining
// session_logs with sessions; knowledge keeps zero table coupling.
type EpisodeSource interface {
	ListEpisodes(ctx context.Context, since time.Time, limit int) ([]Episode, error)
}

// ExperienceCompiler is the cross-project distillation engine.
type ExperienceCompiler struct {
	store     *Store
	llm       LLM
	emb       Embedder
	episodes  EpisodeSource
	lifecycle LifecycleFilter
	log       *slog.Logger

	window        time.Duration // episode look-back (default 90d)
	maxEpisodes   int           // feedstock cap per sweep (default 400)
	simThreshold  float64       // cosine floor for "same procedure" (default 0.80)
	minRecurrence int           // successful sessions required (default 2)
	maxPerSweep   int           // LLM-compiled clusters per sweep (default 3)

	lastSig string // in-memory dirty check — skips unchanged feedstock
}

// NewExperienceCompiler builds the compiler over the shared pool and an LLM.
func NewExperienceCompiler(pool *pgxpool.Pool, llm LLM, log *slog.Logger) *ExperienceCompiler {
	if log == nil {
		log = slog.Default()
	}
	return &ExperienceCompiler{
		store:         NewStore(pool),
		llm:           llm,
		log:           log.With("component", "knowledge.experience"),
		window:        90 * 24 * time.Hour,
		maxEpisodes:   400,
		simThreshold:  0.80,
		minRecurrence: 2,
		maxPerSweep:   3,
	}
}

// WithEpisodes wires the cross-project episode feedstock.
func (c *ExperienceCompiler) WithEpisodes(src EpisodeSource) *ExperienceCompiler {
	c.episodes = src
	return c
}

// WithEmbedder wires the embedder used to cluster episodes.
func (c *ExperienceCompiler) WithEmbedder(emb Embedder) *ExperienceCompiler {
	c.emb = emb
	return c
}

// WithLifecycle installs the optional P-D lifecycle filter so frozen
// (paused/archived) projects' episodes leave the feedstock.
func (c *ExperienceCompiler) WithLifecycle(f LifecycleFilter) *ExperienceCompiler {
	c.lifecycle = f
	return c
}

// cluster is a group of episodes doing "the same kind of work".
type cluster struct {
	episodes []Episode
	centroid []float64
}

// stats derived from a qualifying cluster — recurrence + time cost drive
// the workbench ranking (distill what saves the most operator time first).
type clusterStats struct {
	sessions   []string // distinct successful session ids (the recurrence evidence)
	projects   []string // distinct cwds among successful episodes
	recurrence int
	estMinutes int     // median successful-session duration
	score      float64 // recurrence × estMinutes
	sig        string  // fingerprint of the successful session set
}

// CompileAll runs one cross-project mining + compilation sweep. Returns the
// number of candidates created. A missing LLM / embedder / source is a no-op.
func (c *ExperienceCompiler) CompileAll(ctx context.Context) (int, error) {
	if c.llm == nil || c.emb == nil || c.episodes == nil {
		return 0, nil
	}
	eps, err := c.episodes.ListEpisodes(ctx, time.Now().Add(-c.window), c.maxEpisodes)
	if err != nil {
		return 0, fmt.Errorf("experience: list episodes: %w", err)
	}
	eps = c.filterFrozen(ctx, eps)
	if len(eps) < c.minRecurrence {
		return 0, nil
	}
	// Dirty check — the sweep runs every consolidation cycle; identical
	// feedstock means identical clusters, so skip the embed + LLM work.
	sig := episodesSignature(eps)
	if sig == c.lastSig {
		return 0, nil
	}
	if err := c.ensureVectors(ctx, eps); err != nil {
		return 0, err
	}
	clusters := clusterEpisodes(eps, c.simThreshold)

	existing, err := c.store.ListNodes(ctx, NodeFilter{Kind: KindPlaybook, Limit: 500})
	if err != nil {
		return 0, err
	}
	skills, err := c.store.ListNodes(ctx, NodeFilter{Kind: KindSkill, Limit: 500})
	if err != nil {
		return 0, err
	}
	titles := map[string]struct{}{}
	for _, n := range append(existing, skills...) {
		titles[strings.ToLower(strings.TrimSpace(n.Title))] = struct{}{}
	}

	// Qualify + rank, skipping clusters whose recurrence evidence was
	// already consumed by an existing candidate OR promoted skill (cluster
	// membership shifts as new episodes arrive — the session set is the
	// stable identity; skills inherit it at promotion).
	consumers := append(append([]Node{}, existing...), skills...)
	type ranked struct {
		cl cluster
		st clusterStats
	}
	var queue []ranked
	for _, cl := range clusters {
		st, ok := c.qualify(cl)
		if !ok {
			continue
		}
		if consumedBy(consumers, st.sessions, c.minRecurrence) {
			continue
		}
		queue = append(queue, ranked{cl: cl, st: st})
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i].st.score > queue[j].st.score })

	created := 0
	for i, q := range queue {
		if i >= c.maxPerSweep {
			c.log.Info("experience: per-sweep cap reached — remaining clusters wait",
				"compiled", c.maxPerSweep, "queued", len(queue)-c.maxPerSweep)
			break
		}
		if ctx.Err() != nil {
			return created, ctx.Err()
		}
		n, err := c.compileCluster(ctx, q.cl, q.st, titles)
		if err != nil {
			c.log.Warn("experience: cluster compile failed", "err", err)
			continue
		}
		created += n
	}
	c.lastSig = sig
	if created > 0 {
		c.log.Info("experience sweep done", "candidates", created, "clusters", len(clusters), "qualified", len(queue))
	}
	return created, nil
}

func (c *ExperienceCompiler) filterFrozen(ctx context.Context, eps []Episode) []Episode {
	if c.lifecycle == nil {
		return eps
	}
	frozen := map[string]bool{}
	out := make([]Episode, 0, len(eps))
	for _, e := range eps {
		f, seen := frozen[e.Cwd]
		if !seen {
			f = c.lifecycle.IsFrozen(ctx, e.Cwd)
			frozen[e.Cwd] = f
		}
		if !f {
			out = append(out, e)
		}
	}
	return out
}

// ensureVectors fills missing / foreign-embedder vectors in place. Stored
// journal vectors (the projectdoc backfill) are reused when the embedder
// matches, so the steady-state sweep embeds ~nothing.
func (c *ExperienceCompiler) ensureVectors(ctx context.Context, eps []Episode) error {
	var missing []int
	for i := range eps {
		if len(eps[i].Embedding) == 0 || eps[i].Embedder != c.emb.Name() {
			missing = append(missing, i)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	texts := make([]string, len(missing))
	for j, i := range missing {
		texts[j] = episodeText(eps[i])
	}
	vecs, err := c.emb.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("experience: embed episodes: %w", err)
	}
	if len(vecs) != len(missing) {
		return fmt.Errorf("experience: embedder returned %d vectors for %d texts", len(vecs), len(missing))
	}
	for j, i := range missing {
		eps[i].Embedding = vecs[j]
		eps[i].Embedder = c.emb.Name()
	}
	return nil
}

// episodeText is what gets embedded / quoted against — title + content.
func episodeText(e Episode) string {
	if e.Content == "" {
		return e.Title
	}
	return e.Title + "\n" + e.Content
}

// clusterEpisodes greedily groups episodes by cosine similarity to the
// cluster centroid. O(n·k) over ≤400 episodes — no ANN machinery needed.
func clusterEpisodes(eps []Episode, threshold float64) []cluster {
	var clusters []cluster
	for _, e := range eps {
		if len(e.Embedding) == 0 {
			continue
		}
		best, bestSim := -1, threshold
		for i := range clusters {
			if len(clusters[i].centroid) != len(e.Embedding) {
				continue
			}
			if sim := cosine(clusters[i].centroid, e.Embedding); sim >= bestSim {
				best, bestSim = i, sim
			}
		}
		if best < 0 {
			clusters = append(clusters, cluster{
				episodes: []Episode{e},
				centroid: toFloat64(e.Embedding),
			})
			continue
		}
		cl := &clusters[best]
		n := float64(len(cl.episodes))
		for d := range cl.centroid {
			cl.centroid[d] = (cl.centroid[d]*n + float64(e.Embedding[d])) / (n + 1)
		}
		cl.episodes = append(cl.episodes, e)
	}
	return clusters
}

// qualify applies the recurrence + success bar and computes the ranking
// stats. Skill-worthiness is decided HERE, mechanically — the LLM never
// sees a cluster that didn't repeat and succeed.
func (c *ExperienceCompiler) qualify(cl cluster) (clusterStats, bool) {
	sessions := map[string]struct{}{}
	projects := map[string]struct{}{}
	var minutes []int
	for _, e := range cl.episodes {
		if !e.Success || e.SessionID == "" {
			continue
		}
		if _, dup := sessions[e.SessionID]; dup {
			continue
		}
		sessions[e.SessionID] = struct{}{}
		projects[e.Cwd] = struct{}{}
		minutes = append(minutes, durationMinutes(e.Duration))
	}
	if len(sessions) < c.minRecurrence {
		return clusterStats{}, false
	}
	st := clusterStats{
		sessions:   sortedKeys(sessions),
		projects:   sortedKeys(projects),
		recurrence: len(sessions),
		estMinutes: median(minutes),
	}
	st.score = float64(st.recurrence * st.estMinutes)
	st.sig = sessionsFingerprint(st.sessions)
	return st, true
}

// durationMinutes converts a session duration into a sane manual-time-cost
// estimate: floor 1 minute (instant sessions still cost a context switch),
// cap 4h (an idle overnight PTY is not a 10-hour procedure).
func durationMinutes(d time.Duration) int {
	m := int(d / time.Minute)
	if m < 1 {
		return 1
	}
	if m > 240 {
		return 240
	}
	return m
}

func median(xs []int) int {
	if len(xs) == 0 {
		return 1
	}
	sort.Ints(xs)
	return xs[len(xs)/2]
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sessionsFingerprint identifies a cluster by its successful session set.
func sessionsFingerprint(sessions []string) string {
	h := sha256.Sum256([]byte(strings.Join(sessions, "\n")))
	return hex.EncodeToString(h[:8])
}

// episodesSignature fingerprints the sweep feedstock for the dirty check.
func episodesSignature(eps []Episode) string {
	var b strings.Builder
	for _, e := range eps {
		b.WriteString(e.SessionID)
		b.WriteByte('|')
		b.WriteString(e.CreatedAt.UTC().Format(time.RFC3339))
		b.WriteByte('\n')
	}
	h := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(h[:8])
}

// consumedBy reports whether an existing candidate already cites enough of
// this cluster's sessions — the procedure is distilled; don't redo it just
// because the cluster grew by one episode.
func consumedBy(existing []Node, sessions []string, minOverlap int) bool {
	set := map[string]struct{}{}
	for _, s := range sessions {
		set[s] = struct{}{}
	}
	for _, n := range existing {
		raw, ok := n.Provenance["sessions"].([]any)
		if !ok {
			continue
		}
		overlap := 0
		for _, v := range raw {
			if s, ok := v.(string); ok {
				if _, hit := set[s]; hit {
					overlap++
				}
			}
		}
		if overlap >= minOverlap {
			return true
		}
	}
	return false
}

// --- compilation (the single LLM call per qualifying cluster) ---------------

// draftExperience is what the compiler LLM returns for one cluster.
type draftExperience struct {
	Title       string   `json:"title"`
	AppliesWhen string   `json:"applies_when"`
	Steps       []string `json:"steps"`
	Pitfalls    []string `json:"pitfalls"`
	// Evidence quotes are tied to a specific episode label (E1..En) so each
	// quote can be verified verbatim against that episode's text.
	Evidence []experienceQuote `json:"evidence"`
	// Script: a complete executable form of the procedure, ONLY when every
	// step is mechanical (commands, no judgment calls). Empty otherwise.
	Script string `json:"script"`
	// Validation: how the script (or a human run) confirms success —
	// required whenever Script is set; the script must end by running it.
	Validation string `json:"validation"`
}

type experienceQuote struct {
	Episode string `json:"episode"`
	Quote   string `json:"quote"`
}

const experienceSystem = `You compile RECURRING work into one reusable procedure ("experience").
You receive a GROUP of work episodes (real session summaries, mechanically clustered as similar, each labeled E1..En with project + outcome + duration). The group is known to contain the same kind of work done successfully at least twice.
Decide whether the episodes truly share ONE repeatable multi-step procedure. If they do not (the clustering is fuzzy), return {"experience":null} — that is the CORRECT output for a noisy group.
Otherwise output ONLY JSON:
{"experience":{"title":"...","applies_when":"...","steps":["..."],"pitfalls":["..."],"evidence":[{"episode":"E2","quote":"..."}],"script":"...","validation":"..."}}
Rules:
- title = short imperative ("Ship an unreleased build to the local Mac"). Never duplicate an existing title (list provided).
- applies_when = the recurring trigger/situation.
- steps = concrete + ordered, reusing the REAL commands / paths / file names from the episodes. Someone must be able to follow them without guessing.
- pitfalls = failure modes actually hit in the episodes; what varies between runs.
- evidence = 2-4 SHORT VERBATIM quotes, each from a DIFFERENT episode, copied EXACTLY from that episode's text (they are checked mechanically — a paraphrased quote discards the draft). They must prove the procedure happened in separate sessions.
- script: ONLY when every step is a command an agent could run unattended — emit a complete bash script reusing the episodes' real commands, ending with the validation step. Any step needing human judgment, review, or interactive input → "".
- validation: the concrete check that the procedure worked (a curl, a status command, a file test). REQUIRED when script is set; recommended otherwise.
- NEVER include secrets (passwords, tokens, keys) — name where a credential lives, never its value.
- Reflect the CURRENT way of doing things — if a later episode shows a tool/path replaced an older one, compile the current way.
- JSON only: no prose, no markdown fences.`

// compileCluster runs the LLM over one qualifying cluster and persists the
// surviving candidate. Returns 0 or 1.
func (c *ExperienceCompiler) compileCluster(ctx context.Context, cl cluster, st clusterStats, titles map[string]struct{}) (int, error) {
	input, labeled := buildExperienceInput(cl, st, titles)
	raw, err := c.llm.Complete(ctx, experienceSystem, input)
	if err != nil {
		return 0, err
	}
	d, ok := parseExperience(raw)
	if !ok {
		c.log.Info("experience: model declined cluster (no shared procedure)", "sig", st.sig)
		return 0, nil
	}
	title := strings.TrimSpace(d.Title)
	if _, dup := titles[strings.ToLower(title)]; dup {
		c.log.Info("experience: duplicate title — skipped", "title", title)
		return 0, nil
	}
	if reason, ok := rejectExperience(d, labeled, c.minRecurrence); !ok {
		c.log.Info("experience: draft rejected by quality gate", "title", title, "reason", reason)
		return 0, nil
	}
	script := strings.TrimSpace(d.Script)
	if script != "" && strings.TrimSpace(d.Validation) == "" {
		// A script without a validation step is not a tested artifact —
		// keep the prose procedure, drop the unverifiable script.
		c.log.Info("experience: script lacked a validation step — kept prose only", "title", title)
		script = ""
	}

	scope, scopeKey := ScopeProject, ""
	if len(st.projects) >= 2 {
		scope = ScopeGlobal
	} else if len(st.projects) == 1 {
		scopeKey = st.projects[0]
	}
	prov := map[string]any{
		"source":       "experience_compiler",
		"cluster_sig":  st.sig,
		"sessions":     st.sessions,
		"projects":     st.projects,
		"recurrence":   st.recurrence,
		"est_minutes":  st.estMinutes,
		"score":        st.score,
		"applies_when": strings.TrimSpace(d.AppliesWhen),
		"summary": fmt.Sprintf("%s — succeeded in %d sessions (~%d min each)",
			strings.TrimSpace(d.AppliesWhen), st.recurrence, st.estMinutes),
	}
	if script != "" {
		prov["script"] = composeRunScript(script)
		prov["validation"] = strings.TrimSpace(d.Validation)
	}
	node, err := c.store.CreateNode(ctx, Node{
		Kind:       KindPlaybook,
		Title:      title,
		Body:       renderExperienceBody(d, st, script != ""),
		Scope:      scope,
		ScopeKey:   scopeKey,
		Maturity:   MaturityPlaybook,
		Provenance: prov,
	})
	if err != nil {
		return 0, fmt.Errorf("experience: create candidate: %w", err)
	}
	for _, cwd := range st.projects {
		// Soft-fail: the project entity may not be anchored yet.
		_ = c.store.CreateEdge(ctx, Edge{SrcID: node.ID, EdgeType: EdgeAbout, DstID: ProjectEntityID(cwd)})
	}
	titles[strings.ToLower(title)] = struct{}{}
	return 1, nil
}

// buildExperienceInput renders the cluster for the LLM and returns the
// episode texts keyed by label (E1..En) for quote verification.
func buildExperienceInput(cl cluster, st clusterStats, titles map[string]struct{}) (string, map[string]string) {
	var b strings.Builder
	labeled := make(map[string]string, len(cl.episodes))
	fmt.Fprintf(&b, "EPISODE GROUP (%d episodes; %d successful sessions across %d projects; median session ~%d min):\n\n",
		len(cl.episodes), st.recurrence, len(st.projects), st.estMinutes)
	for i, e := range cl.episodes {
		label := fmt.Sprintf("E%d", i+1)
		outcome := "FAILED"
		if e.Success {
			outcome = "succeeded"
		}
		text := episodeText(e)
		if len(text) > 900 {
			text = text[:900] + "…"
		}
		labeled[label] = text
		fmt.Fprintf(&b, "[%s] project=%s date=%s outcome=%s duration=%dmin\n%s\n\n",
			label, e.Cwd, e.CreatedAt.Format("2006-01-02"), outcome, durationMinutes(e.Duration), text)
	}
	if len(titles) > 0 {
		b.WriteString("EXISTING TITLES (do not duplicate):\n")
		for t := range titles {
			b.WriteString("- ")
			b.WriteString(t)
			b.WriteByte('\n')
		}
	}
	return b.String(), labeled
}

// rejectExperience layers the recurrence-evidence gate on top of the
// structural floor (validateDraftPlaybook):
//   - every evidence quote must appear VERBATIM (whitespace-normalised)
//     in the episode it names — fabricated evidence kills the draft
//   - verified quotes must span ≥ minRecurrence distinct episodes —
//     a procedure "proven" by one session is not an experience
func rejectExperience(d draftExperience, labeled map[string]string, minRecurrence int) (string, bool) {
	verified := map[string]struct{}{} // episode labels with ≥1 verified quote
	var quotes []string
	for _, ev := range d.Evidence {
		text, ok := labeled[strings.ToUpper(strings.TrimSpace(ev.Episode))]
		if !ok {
			continue
		}
		q := strings.TrimSpace(ev.Quote)
		if len(q) < 15 {
			continue
		}
		if !strings.Contains(normalizeForMatch(text), normalizeForMatch(q)) {
			continue // not a verbatim quote — hallucinated evidence
		}
		verified[strings.ToUpper(strings.TrimSpace(ev.Episode))] = struct{}{}
		quotes = append(quotes, q)
	}
	if len(verified) < minRecurrence {
		return fmt.Sprintf("verified evidence spans %d episodes — need %d", len(verified), minRecurrence), false
	}
	// Structural floor — same gate the old reflector enforced.
	return validateDraftPlaybook(draftPlaybook{
		Title:       d.Title,
		AppliesWhen: d.AppliesWhen,
		Steps:       d.Steps,
		Pitfalls:    d.Pitfalls,
		Evidence:    quotes,
	})
}

// normalizeForMatch lowers and collapses whitespace so a quote survives
// line-wrapping differences but nothing more.
func normalizeForMatch(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// composeRunScript wraps the model's script into the run.sh that Skillify
// materialises next to SKILL.md.
func composeRunScript(script string) string {
	if strings.HasPrefix(script, "#!") {
		return strings.TrimSpace(script) + "\n"
	}
	return "#!/usr/bin/env bash\nset -euo pipefail\n\n" + strings.TrimSpace(script) + "\n"
}

// renderExperienceBody is the human-readable candidate body shown in the
// workbench: provenance header + the playbook floor sections + the
// validation / compiled-form notes.
func renderExperienceBody(d draftExperience, st clusterStats, compiled bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Observed:** succeeded in %d sessions across %d project(s), ~%d min each.\n\n",
		st.recurrence, len(st.projects), st.estMinutes)
	quotes := make([]string, 0, len(d.Evidence))
	for _, ev := range d.Evidence {
		if q := strings.TrimSpace(ev.Quote); q != "" {
			quotes = append(quotes, fmt.Sprintf("%s — %s", strings.ToUpper(strings.TrimSpace(ev.Episode)), q))
		}
	}
	b.WriteString(renderPlaybookBody(draftPlaybook{
		AppliesWhen: d.AppliesWhen,
		Steps:       d.Steps,
		Pitfalls:    d.Pitfalls,
		Evidence:    quotes,
	}))
	if v := strings.TrimSpace(d.Validation); v != "" {
		b.WriteString("\n\n## Verification\n")
		b.WriteString(v)
		b.WriteByte('\n')
	}
	if compiled {
		b.WriteString("\n_Compiled: promotion ships an executable `run.sh` (validation included) alongside SKILL.md._\n")
	}
	return strings.TrimSpace(b.String())
}

// parseExperience extracts the draft from the model output. ok=false means
// the model (correctly) declined or returned garbage.
func parseExperience(raw string) (draftExperience, bool) {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '{'); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndexByte(raw, '}'); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	var parsed struct {
		Experience *draftExperience `json:"experience"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil || parsed.Experience == nil {
		return draftExperience{}, false
	}
	return *parsed.Experience, true
}

// --- small vector helpers ----------------------------------------------------

func toFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(x)
	}
	return out
}

func cosine(a []float64, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		bi := float64(b[i])
		dot += a[i] * bi
		na += a[i] * a[i]
		nb += bi * bi
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
