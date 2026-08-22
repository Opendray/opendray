package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// --- polarity classification (V1/V2 of the lifecycle design) -------------
//
// Memory held every row as an undifferentiated statement, and the KB
// drafter consumed each one as "evidence to fold into the page". That is
// a category error with a sharp edge: a directive about the docs
// themselves ("docs must not mention X") is feedstock whose text contains
// X, so the drafter read it as evidence ABOUT X and wrote X back — and
// every operator restatement of the instruction fed the loop another
// line. The discriminator is NOT sentence polarity (the conventions page
// is legitimately full of NEVER-rules about work that belong on it); it
// is the TARGET of the directive — the work, or this documentation. That
// judgement needs the full text and an LLM, so it happens here, once at
// classification time, never per-draft against a 120-char title.
//
// Classification is asynchronous and fail-open: rows start NULL, every
// consumer treats NULL as "fact" (the historical behaviour), and a
// classifier outage just leaves the backlog for the next cycle. Nothing
// is ever lost to a misclassification either — "fact" is the
// conservative default, and "meta" only removes a row from feedstock; it
// never deletes or archives anything.

// Polarity values. Kept as loose strings across the wire; validated here.
const (
	PolarityFact       = "fact"       // statement about the world (default)
	PolarityRule       = "rule"       // binding directive about how we WORK
	PolarityMeta       = "meta"       // directive about the docs/knowledge system ITSELF
	PolarityCorrection = "correction" // supersedes an earlier claim
)

// ValidPolarity reports whether p is a value the classifier may assign.
func ValidPolarity(p string) bool {
	switch p {
	case PolarityFact, PolarityRule, PolarityMeta, PolarityCorrection:
		return true
	}
	return false
}

// PolaritySource is the slice of memory the classifier needs. The app
// adapts internal/memory to it (one-way rule: knowledge owns the
// interface, never imports memory).
type PolaritySource interface {
	// ListUnclassified returns live rows whose polarity is not yet set,
	// oldest first, capped at limit.
	ListUnclassified(ctx context.Context, limit int) ([]MemoryRow, error)
	// SetPolarity records the classification for one row.
	SetPolarity(ctx context.Context, id, polarity string) error
}

// Classifier fills memory polarity in batches. One consolidation cycle
// classifies at most maxPerCycle rows — the backlog drains over a few
// cycles instead of front-loading a costly migration.
type Classifier struct {
	mem PolaritySource
	llm LLM
	log *slog.Logger
}

// NewClassifier builds a classifier; nil llm or mem disables it.
func NewClassifier(mem PolaritySource, llm LLM, log *slog.Logger) *Classifier {
	if log == nil {
		log = slog.Default()
	}
	return &Classifier{mem: mem, llm: llm, log: log}
}

const (
	classifyBatch    = 20
	classifyPerCycle = 100
)

const classifySystem = `You classify stored memories for a personal AI infrastructure system.
Each memory is one utterance. Assign exactly one polarity:

- "fact": a statement about the world — systems, code, hosts, events, states.
  ("the dev DB is at 192.168.3.88", "the release pipeline signs binaries")
- "rule": a binding directive about how WORK is done. Rules belong in
  documentation. ("NEVER connect to PostgreSQL as the superuser", "use pnpm only")
- "meta": a directive about the DOCUMENTATION or KNOWLEDGE SYSTEM ITSELF —
  what pages may or may not say, how docs are written or organised.
  ("project docs must not mention <某个废弃系统>", "keep the infrastructure page terse")
  The test: does it govern the docs/knowledge system rather than the work? If yes, "meta".
- "correction": explicitly supersedes or fixes an earlier claim.
  ("actually the port is 8652, not 5173", "that earlier note was wrong — X is deprecated")

When torn between "fact" and anything else, choose "fact" — it is the safe default.
Reply with ONLY a JSON array: [{"id": "...", "polarity": "..."}] — one entry per input, no prose.`

// RunOnce classifies up to classifyPerCycle unclassified rows. Errors are
// contained per batch: a bad LLM reply skips that batch (rows stay NULL
// and are retried next cycle) rather than failing the whole stage.
func (c *Classifier) RunOnce(ctx context.Context) (int, error) {
	if c == nil || c.mem == nil || c.llm == nil {
		return 0, nil
	}
	rows, err := c.mem.ListUnclassified(ctx, classifyPerCycle)
	if err != nil {
		return 0, fmt.Errorf("classify: list unclassified: %w", err)
	}
	done := 0
	for start := 0; start < len(rows); start += classifyBatch {
		if ctx.Err() != nil {
			return done, ctx.Err()
		}
		end := min(start+classifyBatch, len(rows))
		n, berr := c.classifyBatch(ctx, rows[start:end])
		if berr != nil {
			c.log.Warn("classify: batch failed", "err", berr)
			continue
		}
		done += n
	}
	if done > 0 {
		c.log.Info("memory polarity classified", "rows", done)
	}
	return done, nil
}

func (c *Classifier) classifyBatch(ctx context.Context, rows []MemoryRow) (int, error) {
	var b strings.Builder
	b.WriteString("MEMORIES TO CLASSIFY:\n")
	for _, r := range rows {
		text := r.Text
		if len(text) > 600 {
			text = text[:600] + "…"
		}
		fmt.Fprintf(&b, "\n[id: %s]\n%s\n", r.ID, text)
	}
	out, err := c.llm.Complete(ctx, classifySystem, b.String())
	if err != nil {
		return 0, fmt.Errorf("llm: %w", err)
	}
	var verdicts []struct {
		ID       string `json:"id"`
		Polarity string `json:"polarity"`
	}
	if err := json.Unmarshal([]byte(stripFences(strings.TrimSpace(out))), &verdicts); err != nil {
		return 0, fmt.Errorf("parse: %w", err)
	}
	// Only ids we actually sent, only valid polarities — a hallucinated id
	// or value is skipped, not written.
	sent := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		sent[r.ID] = struct{}{}
	}
	n := 0
	for _, v := range verdicts {
		if _, ok := sent[v.ID]; !ok || !ValidPolarity(v.Polarity) {
			continue
		}
		if err := c.mem.SetPolarity(ctx, v.ID, v.Polarity); err != nil {
			c.log.Warn("classify: set polarity failed", "id", v.ID, "err", err)
			continue
		}
		n++
	}
	return n, nil
}
