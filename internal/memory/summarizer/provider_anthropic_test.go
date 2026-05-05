package summarizer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newAnthropicHappyPath(t *testing.T, facts []Fact) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "missing x-api-key", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("missing/wrong anthropic-version header")
		}
		// Simulate the tool_use response shape.
		factsArr := make([]map[string]any, 0, len(facts))
		for _, f := range facts {
			factsArr = append(factsArr, map[string]any{
				"text":       f.Text,
				"category":   string(f.Category),
				"confidence": f.Confidence,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-haiku-4-5",
			"content": []map[string]any{
				{
					"type": "tool_use",
					"id":   "tool_test",
					"name": "record_facts",
					"input": map[string]any{
						"facts": factsArr,
					},
				},
			},
			"stop_reason": "tool_use",
			"usage": map[string]any{
				"input_tokens":  150,
				"output_tokens": 50,
			},
		})
	}))
}

func TestAnthropicProvider_Summarize_HappyPath(t *testing.T) {
	wantFacts := []Fact{
		{Text: "User prefers pnpm", Category: CategoryPreference, Confidence: 0.95},
		{Text: "DB is at 192.168.3.88", Category: CategoryIdentifier, Confidence: 0.98},
	}
	srv := newAnthropicHappyPath(t, wantFacts)
	defer srv.Close()

	p, err := NewAnthropicProvider(AnthropicConfig{
		APIKey:  "test-key",
		Model:   "claude-haiku-4-5",
		BaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Summarize(context.Background(), []Message{
		{Role: RoleUser, Text: "I prefer pnpm and the DB is at 192.168.3.88"},
		{Role: RoleAssistant, Text: "Got it."},
	})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if len(res.Facts) != 2 {
		t.Errorf("got %d facts, want 2", len(res.Facts))
	}
	if res.InputTokens != 150 {
		t.Errorf("input_tokens = %d, want 150", res.InputTokens)
	}
	if res.OutputTokens != 50 {
		t.Errorf("output_tokens = %d, want 50", res.OutputTokens)
	}
	// Haiku: 150 * $1/M + 50 * $5/M = 0.00015 + 0.00025 = 0.0004
	if res.EstimatedUSD < 0.0003 || res.EstimatedUSD > 0.0005 {
		t.Errorf("estimated_usd = %g, expected ~0.0004", res.EstimatedUSD)
	}
	if res.Latency <= 0 {
		t.Errorf("latency should be > 0")
	}
	if res.RawResponse == "" {
		t.Errorf("raw response should be captured")
	}
}

func TestAnthropicProvider_Summarize_EmptyFactsAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "tool_use", "name": "record_facts", "input": map[string]any{"facts": []any{}}},
			},
			"usage": map[string]any{"input_tokens": 50, "output_tokens": 10},
		})
	}))
	defer srv.Close()

	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "k", Model: "claude-haiku-4-5", BaseURL: srv.URL})
	res, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "hi"}})
	if err != nil {
		t.Fatalf("empty facts should not error: %v", err)
	}
	if len(res.Facts) != 0 {
		t.Errorf("got %d facts, want 0", len(res.Facts))
	}
}

func TestAnthropicProvider_Summarize_DropsInvalidFacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "tool_use", "name": "record_facts", "input": map[string]any{"facts": []any{
					map[string]any{"text": "valid", "category": "preference", "confidence": 0.9},
					map[string]any{"text": "", "category": "identifier", "confidence": 1.0}, // empty text → drop
					map[string]any{"text": "weird category", "category": "unknown_xyz", "confidence": 0.5},
					map[string]any{"text": "out-of-range", "category": "decision", "confidence": 5.0},
				}}},
			},
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 10},
		})
	}))
	defer srv.Close()

	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "k", Model: "claude-haiku-4-5", BaseURL: srv.URL})
	res, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	// 3 valid: "valid", "weird category" (snapped to other), "out-of-range" (clamped to 1.0)
	if len(res.Facts) != 3 {
		t.Errorf("got %d facts, want 3 (after dropping empty + clamping confidence + snapping category)", len(res.Facts))
	}
	for _, f := range res.Facts {
		if f.Confidence < 0 || f.Confidence > 1 {
			t.Errorf("confidence not clamped: %g", f.Confidence)
		}
	}
}

func TestAnthropicProvider_Summarize_AuthFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "bad", Model: "claude-haiku-4-5", BaseURL: srv.URL})
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("got %v, want ErrAuthFailed", err)
	}
}

func TestAnthropicProvider_Summarize_RetriesOn500(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "tool_use", "name": "record_facts", "input": map[string]any{"facts": []any{}}},
			},
			"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer srv.Close()

	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "k", Model: "claude-haiku-4-5", BaseURL: srv.URL})
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	if err != nil {
		t.Fatalf("retry should have succeeded, got %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("expected 2 calls (1 fail + 1 retry), got %d", calls)
	}
}

func TestAnthropicProvider_Summarize_RateLimitedHonoursBackoff(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "tool_use", "name": "record_facts", "input": map[string]any{"facts": []any{}}},
			},
			"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer srv.Close()

	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "k", Model: "claude-haiku-4-5", BaseURL: srv.URL})
	start := time.Now()
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("rate-limit retry should have succeeded, got %v", err)
	}
	if elapsed < time.Second {
		t.Errorf("expected ≥ 1s backoff, got %v", elapsed)
	}
}

func TestAnthropicProvider_Available_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "k", Model: "claude-haiku-4-5", BaseURL: srv.URL})
	if err := p.Available(context.Background()); err != nil {
		t.Errorf("Available: %v", err)
	}
}

func TestAnthropicProvider_Available_AuthFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer srv.Close()
	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "bad", Model: "claude-haiku-4-5", BaseURL: srv.URL})
	if err := p.Available(context.Background()); !errors.Is(err, ErrAuthFailed) {
		t.Errorf("got %v, want ErrAuthFailed", err)
	}
}

func TestNewAnthropicProvider_RejectsMissingFields(t *testing.T) {
	cases := []struct {
		name string
		cfg  AnthropicConfig
		want string
	}{
		{"no_key", AnthropicConfig{Model: "x"}, "APIKey required"},
		{"no_model", AnthropicConfig{APIKey: "k"}, "Model required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewAnthropicProvider(c.cfg)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want substring %q", err, c.want)
			}
		})
	}
}

func TestAnthropicProvider_EmptyTranscriptRejected(t *testing.T) {
	p, _ := NewAnthropicProvider(AnthropicConfig{APIKey: "k", Model: "claude-haiku-4-5"})
	_, err := p.Summarize(context.Background(), nil)
	if !errors.Is(err, ErrEmptyConversation) {
		t.Errorf("got %v, want ErrEmptyConversation", err)
	}
}
