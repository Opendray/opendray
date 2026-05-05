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
)

// newOpenAIHappyPath returns an httptest.Server speaking the OpenAI /
// LM Studio chat-completions wire format. factsContent is stuffed into
// the assistant message body verbatim (so tests can pass invalid JSON
// to exercise the parse-error path).
func newOpenAIHappyPath(t *testing.T, factsContent string, in, out int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		case "/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "chatcmpl-test",
				"choices": []map[string]any{
					{
						"message": map[string]any{
							"role":    "assistant",
							"content": factsContent,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     in,
					"completion_tokens": out,
					"total_tokens":      in + out,
				},
			})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestOpenAICompat_OpenAI_HappyPath(t *testing.T) {
	srv := newOpenAIHappyPath(t,
		`{"facts":[{"text":"User prefers pnpm","category":"preference","confidence":0.9}]}`,
		120, 30,
	)
	defer srv.Close()

	p, err := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind:    "openai",
		APIKey:  "sk-test",
		Model:   "gpt-4o-mini",
		BaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "I prefer pnpm"}})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if len(res.Facts) != 1 {
		t.Fatalf("got %d facts, want 1", len(res.Facts))
	}
	if res.InputTokens != 120 || res.OutputTokens != 30 {
		t.Errorf("token counts: %d/%d, want 120/30", res.InputTokens, res.OutputTokens)
	}
	// gpt-4o-mini: $0.15/$0.60 per MTok → 120 * 0.15e-6 + 30 * 0.6e-6 = 0.000018 + 0.000018 = 0.000036
	if res.EstimatedUSD < 0.00003 || res.EstimatedUSD > 0.00004 {
		t.Errorf("estimated_usd = %g, want ~0.000036", res.EstimatedUSD)
	}
}

func TestOpenAICompat_LMStudio_NoAuthRequired(t *testing.T) {
	srv := newOpenAIHappyPath(t, `{"facts":[]}`, 50, 10)
	defer srv.Close()

	p, err := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind:    "lmstudio",
		Model:   "qwen2.5-7b-instruct",
		BaseURL: srv.URL,
		// No APIKey — must not error.
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "hi"}}); err != nil {
		t.Fatalf("Summarize: %v", err)
	}
}

func TestOpenAICompat_OpenAI_RequiresAPIKey(t *testing.T) {
	_, err := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind:  "openai",
		Model: "gpt-4o-mini",
	})
	if err == nil || !strings.Contains(err.Error(), "APIKey required") {
		t.Errorf("got %v, want APIKey required", err)
	}
}

func TestOpenAICompat_AuthFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer srv.Close()
	p, _ := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind: "openai", APIKey: "bad", Model: "gpt-4o-mini", BaseURL: srv.URL,
	})
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	if !errors.Is(err, ErrAuthFailed) {
		t.Errorf("got %v, want ErrAuthFailed", err)
	}
}

func TestOpenAICompat_ModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()
	p, _ := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind: "lmstudio", Model: "missing", BaseURL: srv.URL,
	})
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("got %v, want ErrModelNotFound", err)
	}
}

func TestOpenAICompat_RetriesOn500(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": `{"facts":[]}`}}},
			"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer srv.Close()
	p, _ := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind: "openai", APIKey: "k", Model: "gpt-4o-mini", BaseURL: srv.URL,
	})
	if _, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}}); err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestOpenAICompat_DefaultBaseURLs(t *testing.T) {
	openai, _ := NewOpenAICompatProvider(OpenAICompatConfig{Kind: "openai", APIKey: "k", Model: "m"})
	if openai.baseURL != "https://api.openai.com/v1" {
		t.Errorf("openai default base = %q", openai.baseURL)
	}
	lms, _ := NewOpenAICompatProvider(OpenAICompatConfig{Kind: "lmstudio", Model: "m"})
	if lms.baseURL != "http://localhost:1234/v1" {
		t.Errorf("lmstudio default base = %q", lms.baseURL)
	}
}

func TestOpenAICompat_InvalidInnerJSON(t *testing.T) {
	srv := newOpenAIHappyPath(t, `not json at all`, 5, 5)
	defer srv.Close()
	p, _ := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind: "lmstudio", Model: "x", BaseURL: srv.URL,
	})
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Errorf("got %v, want ErrInvalidResponse", err)
	}
}

func TestOpenAICompat_Available_HitsModelsEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("expected /models, got %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()
	p, _ := NewOpenAICompatProvider(OpenAICompatConfig{
		Kind: "openai", APIKey: "k", Model: "gpt-4o-mini", BaseURL: srv.URL,
	})
	if err := p.Available(context.Background()); err != nil {
		t.Errorf("Available: %v", err)
	}
}
