package summarizer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newOllamaHappyPath(t *testing.T, factsContent string, evalIn, evalOut int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
		case "/api/chat":
			// Verify the request shape.
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["format"] != "json" {
				t.Errorf("expected format:json, got %v", body["format"])
			}
			if body["stream"] != false {
				t.Errorf("expected stream:false, got %v", body["stream"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model": "test-model",
				"message": map[string]any{
					"role":    "assistant",
					"content": factsContent,
				},
				"done":              true,
				"prompt_eval_count": evalIn,
				"eval_count":        evalOut,
			})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestOllamaProvider_Summarize_HappyPath(t *testing.T) {
	srv := newOllamaHappyPath(t,
		`{"facts":[{"text":"User prefers pnpm","category":"preference","confidence":0.9}]}`,
		200, 30,
	)
	defer srv.Close()

	p, err := NewOllamaProvider(OllamaConfig{
		Model:   "ollama:test",
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
		t.Errorf("got %d facts, want 1", len(res.Facts))
	}
	if res.Facts[0].Text != "User prefers pnpm" {
		t.Errorf("text mismatch: %q", res.Facts[0].Text)
	}
	if res.InputTokens != 200 {
		t.Errorf("input_tokens = %d, want 200", res.InputTokens)
	}
	if res.OutputTokens != 30 {
		t.Errorf("output_tokens = %d, want 30", res.OutputTokens)
	}
	// ollama:* model has zero pricing
	if res.EstimatedUSD != 0 {
		t.Errorf("local model should cost 0, got %g", res.EstimatedUSD)
	}
}

func TestOllamaProvider_Summarize_EmptyFacts(t *testing.T) {
	srv := newOllamaHappyPath(t, `{"facts":[]}`, 50, 10)
	defer srv.Close()
	p, _ := NewOllamaProvider(OllamaConfig{Model: "x", BaseURL: srv.URL})
	res, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Facts) != 0 {
		t.Errorf("expected empty facts, got %d", len(res.Facts))
	}
}

func TestOllamaProvider_Summarize_MalformedInnerJSON(t *testing.T) {
	srv := newOllamaHappyPath(t, `not-json`, 50, 10)
	defer srv.Close()
	p, _ := NewOllamaProvider(OllamaConfig{Model: "x", BaseURL: srv.URL})
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "hi"}})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Errorf("got %v, want ErrInvalidResponse", err)
	}
}

func TestOllamaProvider_Summarize_ModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()
	p, _ := NewOllamaProvider(OllamaConfig{Model: "missing:1b", BaseURL: srv.URL})
	_, err := p.Summarize(context.Background(), []Message{{Role: RoleUser, Text: "x"}})
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("got %v, want ErrModelNotFound", err)
	}
	if !strings.Contains(err.Error(), "ollama pull") {
		t.Errorf("error should mention `ollama pull` hint, got %v", err)
	}
}

func TestOllamaProvider_Available_OK(t *testing.T) {
	srv := newOllamaHappyPath(t, `{"facts":[]}`, 0, 0)
	defer srv.Close()
	p, _ := NewOllamaProvider(OllamaConfig{Model: "x", BaseURL: srv.URL})
	if err := p.Available(context.Background()); err != nil {
		t.Errorf("Available: %v", err)
	}
}

func TestOllamaProvider_Available_DaemonDown(t *testing.T) {
	p, _ := NewOllamaProvider(OllamaConfig{Model: "x", BaseURL: "http://127.0.0.1:1"})
	err := p.Available(context.Background())
	if !errors.Is(err, ErrUnreachable) {
		t.Errorf("got %v, want ErrUnreachable", err)
	}
}

func TestNewOllamaProvider_RejectsMissingFields(t *testing.T) {
	_, err := NewOllamaProvider(OllamaConfig{BaseURL: "x"})
	if err == nil || !strings.Contains(err.Error(), "Model required") {
		t.Errorf("got %v, want Model required", err)
	}
	_, err = NewOllamaProvider(OllamaConfig{Model: "x"})
	if err == nil || !strings.Contains(err.Error(), "BaseURL required") {
		t.Errorf("got %v, want BaseURL required", err)
	}
}
