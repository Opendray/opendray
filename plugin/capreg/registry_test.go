package capreg

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/opendray/opendray/plugin/api"
)

// ── Test doubles ───────────────────────────────────────────────────────

type stubProvider struct{ id string }

func (s *stubProvider) ID() string { return s.id }
func (s *stubProvider) Models(ctx context.Context) ([]api.ProviderModel, error) {
	return nil, nil
}
func (s *stubProvider) Stream(ctx context.Context, req api.ProviderRequest) (<-chan api.ProviderChunk, error) {
	ch := make(chan api.ProviderChunk)
	close(ch)
	return ch, nil
}

type stubChannel struct{ id string }

func (s *stubChannel) ID() string                                          { return s.id }
func (s *stubChannel) Start(ctx context.Context) error                     { return nil }
func (s *stubChannel) Stop(ctx context.Context) error                      { return nil }
func (s *stubChannel) Send(ctx context.Context, m api.ChannelMessage) error { return nil }

type stubForge struct{ id string }

func (s *stubForge) ID() string { return s.id }
func (s *stubForge) ListRepositories(ctx context.Context, accountID string) ([]api.ForgeRepository, error) {
	return nil, nil
}
func (s *stubForge) ListPullRequests(ctx context.Context, accountID, repo, state string) ([]api.ForgePullRequest, error) {
	return nil, nil
}

type stubMcp struct{ id string }

func (s *stubMcp) ID() string                                                  { return s.id }
func (s *stubMcp) Start(ctx context.Context) error                             { return nil }
func (s *stubMcp) Stop(ctx context.Context) error                              { return nil }
func (s *stubMcp) Tools(ctx context.Context) ([]api.McpTool, error)            { return nil, nil }
func (s *stubMcp) CallTool(ctx context.Context, name string, args map[string]any) (api.McpToolResult, error) {
	return api.McpToolResult{}, nil
}

// ── Tests ──────────────────────────────────────────────────────────────

func TestRegisterAndLookup(t *testing.T) {
	r := New()

	if err := r.RegisterProvider("anthropic-plugin", &stubProvider{id: "anthropic"}); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	if err := r.RegisterChannel("telegram-plugin", &stubChannel{id: "telegram"}); err != nil {
		t.Fatalf("RegisterChannel: %v", err)
	}
	if err := r.RegisterForge("github-plugin", &stubForge{id: "github"}); err != nil {
		t.Fatalf("RegisterForge: %v", err)
	}
	if err := r.RegisterMcpServer("mcp-fs-plugin", &stubMcp{id: "filesystem"}); err != nil {
		t.Fatalf("RegisterMcpServer: %v", err)
	}

	if p, ok := r.Provider("anthropic"); !ok || p.ID() != "anthropic" {
		t.Errorf("Provider lookup failed: got %v ok=%v", p, ok)
	}
	if c, ok := r.Channel("telegram"); !ok || c.ID() != "telegram" {
		t.Errorf("Channel lookup failed: got %v ok=%v", c, ok)
	}
	if f, ok := r.Forge("github"); !ok || f.ID() != "github" {
		t.Errorf("Forge lookup failed: got %v ok=%v", f, ok)
	}
	if m, ok := r.McpServer("filesystem"); !ok || m.ID() != "filesystem" {
		t.Errorf("McpServer lookup failed: got %v ok=%v", m, ok)
	}

	if _, ok := r.Provider("does-not-exist"); ok {
		t.Errorf("expected miss for unknown provider id")
	}
}

func TestDuplicateRegistrationFails(t *testing.T) {
	r := New()
	if err := r.RegisterProvider("plug-a", &stubProvider{id: "shared"}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := r.RegisterProvider("plug-b", &stubProvider{id: "shared"})
	if err == nil {
		t.Fatal("expected duplicate error, got nil")
	}
	if !errors.Is(err, api.ErrDuplicateCapability) {
		t.Errorf("expected ErrDuplicateCapability, got: %v", err)
	}
}

func TestRejectsEmptyOrNil(t *testing.T) {
	r := New()
	if err := r.RegisterProvider("p", nil); err == nil {
		t.Errorf("nil provider should error")
	}
	if err := r.RegisterProvider("p", &stubProvider{id: ""}); err == nil {
		t.Errorf("empty provider id should error")
	}
	if err := r.RegisterChannel("p", nil); err == nil {
		t.Errorf("nil channel should error")
	}
	if err := r.RegisterForge("p", nil); err == nil {
		t.Errorf("nil forge should error")
	}
	if err := r.RegisterMcpServer("p", nil); err == nil {
		t.Errorf("nil mcp server should error")
	}
}

func TestListSorted(t *testing.T) {
	r := New()
	_ = r.RegisterProvider("p1", &stubProvider{id: "zeta"})
	_ = r.RegisterProvider("p2", &stubProvider{id: "alpha"})
	_ = r.RegisterProvider("p3", &stubProvider{id: "mu"})

	got := r.ListProviders()
	wantOrder := []string{"alpha", "mu", "zeta"}
	if len(got) != len(wantOrder) {
		t.Fatalf("len mismatch: %d vs %d", len(got), len(wantOrder))
	}
	for i, e := range got {
		if e.ID != wantOrder[i] {
			t.Errorf("position %d: want %q got %q", i, wantOrder[i], e.ID)
		}
		if e.Kind != "provider" {
			t.Errorf("position %d: kind = %q, want provider", i, e.Kind)
		}
	}
}

func TestAllSortedByKindThenID(t *testing.T) {
	r := New()
	_ = r.RegisterProvider("a", &stubProvider{id: "anthropic"})
	_ = r.RegisterChannel("a", &stubChannel{id: "telegram"})
	_ = r.RegisterForge("a", &stubForge{id: "github"})
	_ = r.RegisterMcpServer("a", &stubMcp{id: "filesystem"})

	got := r.All()
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	wantKindOrder := []string{"channel", "forge", "mcpServer", "provider"}
	for i, k := range wantKindOrder {
		if got[i].Kind != k {
			t.Errorf("position %d: kind = %q, want %q", i, got[i].Kind, k)
		}
	}
}

func TestRemovePlugin(t *testing.T) {
	r := New()
	_ = r.RegisterProvider("plug-a", &stubProvider{id: "p1"})
	_ = r.RegisterProvider("plug-a", &stubProvider{id: "p2"})
	_ = r.RegisterProvider("plug-b", &stubProvider{id: "p3"})

	r.RemovePlugin("plug-a")

	if _, ok := r.Provider("p1"); ok {
		t.Errorf("p1 should be gone")
	}
	if _, ok := r.Provider("p2"); ok {
		t.Errorf("p2 should be gone")
	}
	if _, ok := r.Provider("p3"); !ok {
		t.Errorf("p3 should remain")
	}

	// Idempotent.
	r.RemovePlugin("plug-a")
	r.RemovePlugin("never-registered")
}

func TestConcurrentRegisterAndRead(t *testing.T) {
	r := New()
	const n = 100
	var wg sync.WaitGroup

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			id := "p-" + string(rune('a'+(i%26))) + "-" + string(rune('0'+(i%10)))
			// Best-effort registration; duplicates expected and ignored.
			_ = r.RegisterProvider("plug", &stubProvider{id: id})
		}(i)
	}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = r.ListProviders()
			_, _ = r.Provider("p-a-0")
		}()
	}
	wg.Wait()
	// No assertion beyond the race detector + no panic.
}
