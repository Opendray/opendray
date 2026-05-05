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
type MemoryReader interface {
	List(ctx context.Context, scope memory.Scope, scopeKey string, limit int) ([]memory.Memory, error)
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
	case "none":
		return "", nil
	case "top_k_recent":
		return i.renderTopKRecent(ctx, profile, cwd)
	default:
		return "", fmt.Errorf("injector: unknown strategy %q", profile.StrategyKind)
	}
}

// renderTopKRecent fetches memory.List with project scope + cwd.
// K comes from profile.Config["k"] (default 5, max 50). Empty
// list returns "" (no banner is better than a blank one).
func (i *Injector) renderTopKRecent(ctx context.Context, profile Profile, cwd string) (string, error) {
	k := 5
	if v, ok := profile.Config["k"]; ok {
		switch x := v.(type) {
		case float64:
			k = int(x)
		case int:
			k = x
		}
	}
	if k <= 0 {
		k = 5
	}
	if k > 50 {
		k = 50
	}
	if cwd == "" {
		i.log.Debug("injector: empty cwd, skipping top_k_recent")
		return "", nil
	}
	mems, err := i.memory.List(ctx, memory.ScopeProject, cwd, k)
	if err != nil {
		i.log.Warn("injector: list memories failed", "cwd", cwd, "err", err)
		return "", nil // non-fatal — skip injection rather than block spawn
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
