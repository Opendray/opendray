package injector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/opendray/opendray-v2/internal/memory"
)

// MemoryReader is the slice of memory.Service the injector needs.
// Search is required by top_k_relevant + hybrid strategies.
type MemoryReader interface {
	List(ctx context.Context, scope memory.Scope, scopeKey string, limit int) ([]memory.Memory, error)
	Search(ctx context.Context, req memory.SearchRequest) ([]memory.SearchHit, error)
}

// Injector renders a system-prompt prefix from prior memories.
type Injector struct {
	store  *ProfileStore
	memory MemoryReader
	log    *slog.Logger
}

// New constructs an Injector.
func New(store *ProfileStore, mem MemoryReader, log *slog.Logger) *Injector {
	if log == nil {
		log = slog.Default()
	}
	return &Injector{store: store, memory: mem, log: log.With("component", "memory-injector")}
}

// Render decides what (if any) memories to embed in the rendered
// system prompt for a session-about-to-spawn. Returns "" when:
//   - the resolved profile's strategy is "none", or
//   - top_k_recent finds no memories under the project scope, or
//   - any non-fatal error in fetching memories occurs (logged + skipped).
//
// The returned string is markdown — caller (catalog adapter) just
// appends as a system-prompt prefix.
func (i *Injector) Render(ctx context.Context, sessionID, cwd string) (string, error) {
	if i == nil || i.store == nil || i.memory == nil {
		return "", nil
	}
	profile := i.store.Resolve(ctx, sessionID)
	switch profile.StrategyKind {
	case "none", "manual_only", "on_keyword":
		// none: explicit opt-out.
		// manual_only: operator triggers via UI/API only.
		// on_keyword: spawn-time inject not used; keyword hook is
		//   Phase C. Profile still selectable so operators can lock
		//   it in advance.
		return "", nil
	case "top_k_recent":
		return i.renderTopKRecent(ctx, profile, cwd)
	case "top_k_relevant":
		return i.renderTopKRelevant(ctx, profile, cwd)
	case "hybrid":
		return i.renderHybrid(ctx, profile, cwd)
	default:
		return "", fmt.Errorf("injector: unknown strategy %q", profile.StrategyKind)
	}
}

// renderTopKRelevant runs memory.Search using cwd as the query
// (basename of the cwd is more semantically meaningful than the
// full path) and renders the top-K most relevant memories.
func (i *Injector) renderTopKRelevant(ctx context.Context, profile Profile, cwd string) (string, error) {
	if cwd == "" {
		return "", nil
	}
	k := readK(profile.Config, 5, 50)
	// Use the cwd's last segment as a search query — captures the
	// project name without polluting the embedding with the full
	// path's noise.
	query := cwd
	if idx := strings.LastIndex(cwd, "/"); idx >= 0 && idx+1 < len(cwd) {
		query = cwd[idx+1:]
	}
	hits, err := i.memory.Search(ctx, memory.SearchRequest{
		Query:    query,
		Scope:    memory.ScopeProject,
		ScopeKey: cwd,
		TopK:     k,
	})
	if err != nil {
		i.log.Warn("injector: relevant-search failed", "cwd", cwd, "err", err)
		return "", nil
	}
	mems := make([]memory.Memory, 0, len(hits))
	for _, h := range hits {
		mems = append(mems, h.Memory)
	}
	if len(mems) == 0 {
		return "", nil
	}
	return renderTopKPreface(mems), nil
}

// renderHybrid emits a single ultra-short memory line — designed
// for budgets where you can spare ~80 chars but not the multi-line
// banner. Picks the most-recent project memory and truncates.
func (i *Injector) renderHybrid(ctx context.Context, profile Profile, cwd string) (string, error) {
	if cwd == "" {
		return "", nil
	}
	mems, err := i.memory.List(ctx, memory.ScopeProject, cwd, 1)
	if err != nil {
		i.log.Warn("injector: hybrid list failed", "cwd", cwd, "err", err)
		return "", nil
	}
	if len(mems) == 0 {
		return "", nil
	}
	text := strings.TrimSpace(mems[0].Text)
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	const cap = 80
	if len(text) > cap {
		text = text[:cap-1] + "…"
	}
	if text == "" {
		return "", nil
	}
	return "\nProject memory hint: " + text + "\n", nil
}

// readK reads the K (top_k) from profile.Config with bounds
// checking. Empty / missing → fallback; out-of-range → clamp.
func readK(cfg map[string]any, fallback, max int) int {
	k := fallback
	if v, ok := cfg["k"]; ok {
		switch x := v.(type) {
		case float64:
			k = int(x)
		case int:
			k = x
		}
	}
	if k <= 0 {
		k = fallback
	}
	if k > max {
		k = max
	}
	return k
}

// renderTopKRecent fetches memory.List with project scope + cwd.
// K comes from profile.Config["k"] (default 5, max 50). Empty
// list returns "" (no banner is better than a blank one).
func (i *Injector) renderTopKRecent(ctx context.Context, profile Profile, cwd string) (string, error) {
	k := readK(profile.Config, 5, 50)
	var mems []memory.Memory
	if cwd != "" {
		m, err := i.memory.List(ctx, memory.ScopeProject, cwd, k)
		if err != nil {
			i.log.Warn("injector: list memories failed", "cwd", cwd, "err", err)
		} else {
			mems = m
		}
	}
	// Cross-session fallback: a session in a fresh cwd has no project-scoped
	// memories, but global-scope memories (e.g. stored via the memory MCP at
	// OPENDRAY_MEMORY_SCOPE=global) should still surface so "told one
	// session, recalled in another" works regardless of cwd.
	if len(mems) == 0 {
		g, err := i.memory.List(ctx, memory.ScopeGlobal, "", k)
		if err != nil {
			i.log.Warn("injector: global list failed", "err", err)
			return "", nil // non-fatal — skip injection rather than block spawn
		}
		mems = g
	}
	if len(mems) == 0 {
		return "", nil
	}
	return renderTopKPreface(mems), nil
}

// renderTopKPreface produces the markdown shown to the agent.
// Format intentionally minimal: a single H2 header + bullets.
// Each bullet is the memory text verbatim — the summarizer's
// extraction step already produced one-sentence durable claims.
func renderTopKPreface(mems []memory.Memory) string {
	var b strings.Builder
	b.WriteString("\n## Recent project memory\n\n")
	b.WriteString("opendray injected the following durable facts from prior sessions in this project:\n\n")
	for _, m := range mems {
		text := strings.TrimSpace(m.Text)
		if text == "" {
			continue
		}
		// Take only the first line of multi-line memories — keeps
		// the prefix compact even if a memory was a fenced block.
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			text = text[:i]
		}
		b.WriteString("- ")
		b.WriteString(text)
		b.WriteString("\n")
	}
	b.WriteString("\nEnd of memory preface.\n")
	return b.String()
}

// silence unused import
var _ = errors.New
