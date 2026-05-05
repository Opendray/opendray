package injector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/memory"
)

type fakeMemoryReader struct {
	mems []memory.Memory
	err  error
}

func (f *fakeMemoryReader) List(ctx context.Context, scope memory.Scope, scopeKey string, limit int) ([]memory.Memory, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.mems) > limit {
		return f.mems[:limit], nil
	}
	return f.mems, nil
}

// fakeProfileStore implements *ProfileStore's Resolve via a stub
// since the test doesn't need the rest. We embed a real store on
// nil pool but only use the synthetic-default branch.
type fakeProfileStore struct {
	resolved Profile
}

// Resolve via the embedded ProfileStore would require a DB; we
// build the Injector with a real ProfileStore against a no-op
// pool and override the resolved Profile by injecting it through
// a small adapter.
//
// Pragmatic approach: test the renderer + strategy selection by
// constructing the Injector with the strategy-specific path
// directly.

func TestRender_NoneReturnsEmpty(t *testing.T) {
	mem := &fakeMemoryReader{}
	inj := &Injector{
		store: nil, // not used in the test path
		memory: mem,
		log:    silentLog(),
	}
	// Inject by calling the per-strategy path directly.
	got, err := inj.renderProfile(context.Background(), Profile{StrategyKind: "none"}, "/x")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("none should yield empty, got %q", got)
	}
}

func TestRender_TopKRecent_Renders(t *testing.T) {
	mem := &fakeMemoryReader{
		mems: []memory.Memory{
			{ID: "1", Text: "User prefers pnpm"},
			{ID: "2", Text: "DB at 192.168.3.88"},
			{ID: "3", Text: "first line\nsecond line"},
		},
	}
	inj := &Injector{memory: mem, log: silentLog()}
	out, err := inj.renderTopKRecent(context.Background(),
		Profile{StrategyKind: "top_k_recent", Config: map[string]any{"k": float64(3)}},
		"/proj")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "## Recent project memory") {
		t.Errorf("missing header: %q", out)
	}
	if !strings.Contains(out, "User prefers pnpm") {
		t.Errorf("missing fact1: %q", out)
	}
	if !strings.Contains(out, "DB at 192.168.3.88") {
		t.Errorf("missing fact2: %q", out)
	}
	// Multi-line fact should be truncated to first line.
	if !strings.Contains(out, "first line") {
		t.Errorf("missing fact3 first line")
	}
	if strings.Contains(out, "second line") {
		t.Errorf("multi-line fact not truncated: %q", out)
	}
}

func TestRender_TopKRecent_EmptyMemoriesReturnsEmpty(t *testing.T) {
	mem := &fakeMemoryReader{mems: nil}
	inj := &Injector{memory: mem, log: silentLog()}
	out, err := inj.renderTopKRecent(context.Background(),
		Profile{StrategyKind: "top_k_recent", Config: map[string]any{"k": float64(5)}},
		"/proj")
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("empty memories should yield empty, got %q", out)
	}
}

func TestRender_TopKRecent_DefaultsK(t *testing.T) {
	mems := []memory.Memory{}
	for i := 0; i < 10; i++ {
		mems = append(mems, memory.Memory{ID: "m", Text: "x"})
	}
	mem := &fakeMemoryReader{mems: mems}
	inj := &Injector{memory: mem, log: silentLog()}
	out, _ := inj.renderTopKRecent(context.Background(),
		Profile{StrategyKind: "top_k_recent", Config: map[string]any{}},
		"/proj")
	count := strings.Count(out, "- x")
	if count != 5 {
		t.Errorf("default K should be 5, got %d bullets", count)
	}
}

func TestRender_TopKRecent_ListErrorSkipsGracefully(t *testing.T) {
	mem := &fakeMemoryReader{err: errors.New("db down")}
	inj := &Injector{memory: mem, log: silentLog()}
	out, err := inj.renderTopKRecent(context.Background(),
		Profile{StrategyKind: "top_k_recent", Config: map[string]any{}},
		"/proj")
	if err != nil {
		t.Errorf("renderer should swallow + skip rather than error: %v", err)
	}
	if out != "" {
		t.Errorf("on list error, output should be empty (skip injection), got %q", out)
	}
}

func TestRender_TopKRecent_EmptyCwdReturnsEmpty(t *testing.T) {
	mem := &fakeMemoryReader{mems: []memory.Memory{{ID: "1", Text: "x"}}}
	inj := &Injector{memory: mem, log: silentLog()}
	out, _ := inj.renderTopKRecent(context.Background(),
		Profile{StrategyKind: "top_k_recent"}, "")
	if out != "" {
		t.Errorf("empty cwd should yield empty preface, got %q", out)
	}
}

// silentLog returns a slog.Logger that drops all output.
func silentLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

// renderProfile is the test-only entrypoint that picks per-strategy
// without needing a DB-backed ProfileStore.
func (i *Injector) renderProfile(ctx context.Context, p Profile, cwd string) (string, error) {
	switch p.StrategyKind {
	case "none":
		return "", nil
	case "top_k_recent":
		return i.renderTopKRecent(ctx, p, cwd)
	default:
		return "", errors.New("injector: unknown strategy")
	}
}
