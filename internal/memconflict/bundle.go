package memconflict

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// detectionBundle is the input the worker LLM sees: a snapshot of
// every memory layer the detector is allowed to reason about.
// Kept as a struct (not just a string) so tests can pin which
// layers actually populated.
type detectionBundle struct {
	cwd     string
	plan    string
	goal    string
	facts   []memory.Memory
	journal []projectdoc.LogEntry
}

func (b detectionBundle) empty() bool {
	return strings.TrimSpace(b.plan) == "" &&
		strings.TrimSpace(b.goal) == "" &&
		len(b.facts) == 0 &&
		len(b.journal) == 0
}

// gather runs the read side: pulls plan + goal + top hit facts +
// recent journal. Each read failure degrades to empty for that
// slice — a missing journal table shouldn't stop the detector
// from finding a plan-vs-facts contradiction.
func (s *Service) gather(ctx context.Context, cwd string) (detectionBundle, error) {
	b := detectionBundle{cwd: cwd}

	if plan, err := s.docs.GetDoc(ctx, cwd, projectdoc.KindPlan); err == nil {
		b.plan = strings.TrimSpace(plan.Content)
	}
	if goal, err := s.docs.GetDoc(ctx, cwd, projectdoc.KindGoal); err == nil {
		b.goal = strings.TrimSpace(goal.Content)
	}
	// Top-K by hit_count would ideally be a Store-level helper; for
	// now we List with a generous cap and let the renderer pick the
	// highest hit_count rows.
	mems, err := s.mem.List(ctx, memory.ScopeProject, cwd, 100)
	if err == nil {
		b.facts = pickTopByHits(mems, 20)
	}
	logs, err := s.docs.ListLogs(ctx, cwd, 30)
	if err == nil {
		// Filter to the last 14 days — older entries dilute the prompt
		// without adding signal because they've usually been
		// superseded.
		cutoff := time.Now().UTC().Add(-14 * 24 * time.Hour)
		for _, l := range logs {
			if l.CreatedAt.After(cutoff) {
				b.journal = append(b.journal, l)
			}
		}
	}
	return b, nil
}

// pickTopByHits returns up to limit memories ordered by hit_count
// desc. Ties broken by created_at desc (newer first).
func pickTopByHits(mems []memory.Memory, limit int) []memory.Memory {
	// Sort in place: simple selection sort since N is small (≤ 100).
	out := make([]memory.Memory, len(mems))
	copy(out, mems)
	for i := range out {
		bestIdx := i
		for j := i + 1; j < len(out); j++ {
			if out[j].HitCount > out[bestIdx].HitCount ||
				(out[j].HitCount == out[bestIdx].HitCount && out[j].CreatedAt.After(out[bestIdx].CreatedAt)) {
				bestIdx = j
			}
		}
		if bestIdx != i {
			out[i], out[bestIdx] = out[bestIdx], out[i]
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// render assembles the user-message payload. Each item gets its
// id stamped inline so the LLM can quote ids back in its findings
// (we reject findings with unknown ids on insert).
func (b detectionBundle) render() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Project context for %s\n\n", b.cwd)
	if b.goal != "" {
		sb.WriteString("## Project goal (layer=goal, ref=goal-doc)\n\n")
		sb.WriteString(b.goal)
		sb.WriteString("\n\n")
	}
	if b.plan != "" {
		sb.WriteString("## Project plan (layer=plan, ref=plan-doc)\n\n")
		sb.WriteString(b.plan)
		sb.WriteString("\n\n")
	}
	if len(b.facts) > 0 {
		sb.WriteString("## High-hit facts (layer=fact)\n\n")
		for _, f := range b.facts {
			fmt.Fprintf(&sb, "- ref=`%s` hits=%d: %s\n",
				f.ID, f.HitCount, oneLine(f.Text, 240))
		}
		sb.WriteString("\n")
	}
	if len(b.journal) > 0 {
		sb.WriteString("## Recent journal entries (layer=journal)\n\n")
		// ListLogs returns newest first; show oldest first so chronology reads top-to-bottom.
		for i := len(b.journal) - 1; i >= 0; i-- {
			e := b.journal[i]
			fmt.Fprintf(&sb, "- ref=`%s` (%s): %s — %s\n",
				e.ID, e.CreatedAt.Format("2006-01-02"),
				e.Title, oneLine(e.Content, 240))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func oneLine(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}
