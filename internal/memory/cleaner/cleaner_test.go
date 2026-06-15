package cleaner

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
)

// archiveSpyMem is a MemoryAdapter that records Archive calls and
// reports whether a merge_into target exists, so we can assert the
// auto-apply path soft-archives instead of hard-deleting.
type archiveSpyMem struct {
	archived     map[string]string // id -> reason
	deleted      []string
	exists       map[string]bool // ids Get should find
	dormantCalls []string        // scope_keys passed to ArchiveDormantStale
}

func (m *archiveSpyMem) List(context.Context, memory.Scope, string, int) ([]memory.Memory, error) {
	return nil, nil
}
func (m *archiveSpyMem) Get(_ context.Context, id string) (memory.Memory, error) {
	if m.exists[id] {
		return memory.Memory{ID: id}, nil
	}
	return memory.Memory{}, memory.ErrNotFound
}
func (m *archiveSpyMem) Delete(_ context.Context, id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *archiveSpyMem) Archive(_ context.Context, id, reason string) error {
	if m.archived == nil {
		m.archived = map[string]string{}
	}
	m.archived[id] = reason
	return nil
}
func (m *archiveSpyMem) PurgeArchived(context.Context, time.Time) (int64, error) { return 0, nil }
func (m *archiveSpyMem) PurgeExpiredQuarantine(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (m *archiveSpyMem) ArchiveDormantStale(_ context.Context, _ memory.Scope, scopeKey string, _, _ time.Time, _ string) (int64, error) {
	m.dormantCalls = append(m.dormantCalls, scopeKey)
	return 0, nil
}

func TestExecute_SoftArchivesNotDeletes(t *testing.T) {
	mem := &archiveSpyMem{exists: map[string]bool{"mem_survivor": true}}
	svc := &Service{mem: mem, cfg: Config{}.applyDefaults(), log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ctx := context.Background()

	// stale → archived, never deleted
	if err := svc.execute(ctx, Decision{MemoryID: "mem_stale", Verdict: VerdictStale, Reason: "old WIP note"}); err != nil {
		t.Fatal(err)
	}
	// duplicate with a live survivor → archived with survivor noted
	if err := svc.execute(ctx, Decision{MemoryID: "mem_dup", Verdict: VerdictDuplicate, MergeInto: "mem_survivor"}); err != nil {
		t.Fatal(err)
	}
	// keep → no-op
	if err := svc.execute(ctx, Decision{MemoryID: "mem_keep", Verdict: VerdictKeep}); err != nil {
		t.Fatal(err)
	}

	if len(mem.deleted) != 0 {
		t.Errorf("nothing should be hard-deleted, got %v", mem.deleted)
	}
	if _, ok := mem.archived["mem_stale"]; !ok {
		t.Error("stale memory should be archived")
	}
	if r, ok := mem.archived["mem_dup"]; !ok || !strings.Contains(r, "mem_survivor") {
		t.Errorf("duplicate should be archived noting the survivor, got %q", r)
	}
	if _, ok := mem.archived["mem_keep"]; ok {
		t.Error("kept memory must not be archived")
	}
}

func TestArchiveDormant_Gating(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	// Disabled (negative days): never calls the store.
	off := &archiveSpyMem{}
	svcOff := &Service{mem: off, log: logger, cfg: Config{LifecycleDormantDays: -1}.applyDefaults()}
	if _, err := svcOff.ArchiveDormant(ctx, "/proj/a"); err != nil {
		t.Fatal(err)
	}
	if len(off.dormantCalls) != 0 {
		t.Errorf("disabled lifecycle must not touch the store, got %v", off.dormantCalls)
	}

	// Enabled (default 90): calls the store for the project.
	on := &archiveSpyMem{}
	svcOn := &Service{mem: on, log: logger, cfg: Config{}.applyDefaults()}
	if _, err := svcOn.ArchiveDormant(ctx, "/proj/a"); err != nil {
		t.Fatal(err)
	}
	if len(on.dormantCalls) != 1 || on.dormantCalls[0] != "/proj/a" {
		t.Errorf("enabled lifecycle should archive-dormant /proj/a, got %v", on.dormantCalls)
	}
	// Empty scope_key is a no-op (global scope has no lifecycle).
	if _, err := svcOn.ArchiveDormant(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if len(on.dormantCalls) != 1 {
		t.Errorf("empty scope_key must be a no-op, got %v", on.dormantCalls)
	}
}

func TestRenderBatch_Shape(t *testing.T) {
	items := []BatchItem{
		{
			ID:        "mem_abc",
			Text:      "User prefers pnpm",
			CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			HitCount:  5,
		},
		{
			ID:        "mem_def",
			Text:      "uses pnpm not npm",
			CreatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			HitCount:  0,
		},
	}
	out := renderBatch(items)
	for _, want := range []string{
		"[1] mem_abc | created 2026-01-15 | hit_count=5",
		"User prefers pnpm",
		"[2] mem_def | created 2026-02-01 | hit_count=0",
		"uses pnpm not npm",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered batch missing %q\n--- batch ---\n%s", want, out)
		}
	}
}

func TestRenderBatch_CollapsesMultilineText(t *testing.T) {
	items := []BatchItem{
		{
			ID:        "mem_x",
			Text:      "line one\nline two\nline three",
			CreatedAt: time.Now().UTC(),
		},
	}
	out := renderBatch(items)
	if strings.Contains(out, "\n    line two") {
		t.Errorf("multi-line memory text should be collapsed, got:\n%s", out)
	}
	if !strings.Contains(out, "line one line two line three") {
		t.Errorf("text should be flattened onto one line, got:\n%s", out)
	}
}

// (TestResponseFormatFor_* removed in M25 — response_format
// enforcement moved from cleaner.responseFormatFor into the
// worker package's SummarizerWorker.Run, which always emits
// json_schema when Request.ResponseFormatJSONSchema is set.
// cleaner.Client just passes DecisionsJSONSchema through.)

func TestValidVerdict(t *testing.T) {
	for _, tc := range []struct {
		v    Verdict
		want bool
	}{
		{VerdictKeep, true},
		{VerdictStale, true},
		{VerdictDuplicate, true},
		{Verdict("delete"), false},
		{Verdict(""), false},
	} {
		if got := ValidVerdict(tc.v); got != tc.want {
			t.Errorf("ValidVerdict(%q) = %v want %v", tc.v, got, tc.want)
		}
	}
}

func TestConfigApplyDefaults(t *testing.T) {
	c := Config{}.applyDefaults()
	if c.BatchSize != 30 {
		t.Errorf("default BatchSize = %d, want 30", c.BatchSize)
	}
	if c.MinAge != 24*time.Hour {
		t.Errorf("default MinAge = %s, want 24h", c.MinAge)
	}
	if c.SkipIfDecidedWithin != 7*24*time.Hour {
		t.Errorf("default SkipIfDecidedWithin = %s, want 168h", c.SkipIfDecidedWithin)
	}
	if c.CallTimeout != 60*time.Second {
		t.Errorf("default CallTimeout = %s, want 60s", c.CallTimeout)
	}
	if c.GracePeriod != 30*24*time.Hour {
		t.Errorf("default GracePeriod = %s, want 720h", c.GracePeriod)
	}
}

func TestTruncate(t *testing.T) {
	for _, tc := range []struct {
		in   string
		max  int
		want string
	}{
		{"short", 100, "short"},
		{"abcdefghij", 5, "abcde…"},
		{"", 10, ""},
	} {
		if got := truncate(tc.in, tc.max); got != tc.want {
			t.Errorf("truncate(%q, %d) = %q want %q", tc.in, tc.max, got, tc.want)
		}
	}
}
