package knowledge

import (
	"context"
	"time"
)

// JournalEntry is one project work-trace (a session journal entry) — the real
// record of what was done / fixed / learned. The Overview drafter reads these
// per project; the experience compiler consumes the richer Episode shape.
type JournalEntry struct {
	Title     string
	Content   string
	Kind      string
	CreatedAt time.Time
}

// JournalSource is the read-only view of the project journal. The app wires a
// projectdoc-backed adapter (one-way dependency rule).
type JournalSource interface {
	ListJournal(ctx context.Context, scopeKey string, limit int) ([]JournalEntry, error)
}

// LifecycleFilter reports whether a project (cwd) is frozen (paused/archived).
// P-D: a frozen project is excluded from distillation feedstock. Optional —
// a nil filter treats every project as active. The app adapts projectdoc's
// GetStatus to this; knowledge keeps no projectdoc import.
type LifecycleFilter interface {
	IsFrozen(ctx context.Context, cwd string) bool
}
