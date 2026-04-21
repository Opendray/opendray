package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// ─── test doubles ────────────────────────────────────────────────

type fakeLLMSource struct {
	rows []LLMEndpoint
	err  error
}

func (f *fakeLLMSource) ListLLMEndpoints(_ context.Context) ([]LLMEndpoint, error) {
	return f.rows, f.err
}

// llmConsent grants or denies the llm cap. Matches the pattern in
// api_storage_test.go / api_secret_test.go so tests stay uniform.
type llmConsent struct {
	found bool
	deny  bool
}

func (m *llmConsent) Load(_ context.Context, _ string) ([]byte, bool, error) {
	if !m.found {
		return nil, false, nil
	}
	if m.deny {
		return []byte(`{"llm":false}`), true, nil
	}
	return []byte(`{"llm":true}`), true, nil
}

func llmGrantedGate() *Gate {
	return NewGate(&llmConsent{found: true}, nil, nil)
}

func llmDeniedGate() *Gate {
	return NewGate(&llmConsent{found: false}, nil, nil)
}

// ─── tests ───────────────────────────────────────────────────────

func TestLLMAPI_ListReturnsOnlyEnabledByDefault(t *testing.T) {
	src := &fakeLLMSource{rows: []LLMEndpoint{
		{ID: "1", Name: "local", BaseURL: "http://127.0.0.1:11434", Enabled: true},
		{ID: "2", Name: "remote", BaseURL: "https://example.com", Enabled: false},
	}}
	api := NewLLMAPI(LLMConfig{Source: src, Gate: llmGrantedGate()})

	got, err := api.Dispatch(context.Background(), "p1", "list", nil, "", nil)
	if err != nil {
		t.Fatalf("Dispatch list: %v", err)
	}
	out, ok := got.([]LLMEndpoint)
	if !ok {
		t.Fatalf("result type = %T, want []LLMEndpoint", got)
	}
	if len(out) != 1 || out[0].ID != "1" {
		t.Errorf("default list must filter disabled rows; got %+v", out)
	}
}

func TestLLMAPI_ListEnabledOnlyFalseReturnsAll(t *testing.T) {
	src := &fakeLLMSource{rows: []LLMEndpoint{
		{ID: "1", Name: "on", Enabled: true},
		{ID: "2", Name: "off", Enabled: false},
	}}
	api := NewLLMAPI(LLMConfig{Source: src, Gate: llmGrantedGate()})

	args := json.RawMessage(`{"enabledOnly":false}`)
	got, err := api.Dispatch(context.Background(), "p1", "list", args, "", nil)
	if err != nil {
		t.Fatalf("Dispatch list: %v", err)
	}
	out := got.([]LLMEndpoint)
	if len(out) != 2 {
		t.Errorf("enabledOnly=false should return all rows, got %+v", out)
	}
}

func TestLLMAPI_ListReturnsEmptySliceWhenNoRows(t *testing.T) {
	api := NewLLMAPI(LLMConfig{
		Source: &fakeLLMSource{rows: nil},
		Gate:   llmGrantedGate(),
	})

	got, err := api.Dispatch(context.Background(), "p1", "list", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	out := got.([]LLMEndpoint)
	if out == nil {
		t.Error("list must return [] not nil when source has no rows")
	}
	if len(out) != 0 {
		t.Errorf("expected empty slice, got %+v", out)
	}
}

func TestLLMAPI_PermissionDeniedWithoutGrant(t *testing.T) {
	api := NewLLMAPI(LLMConfig{
		Source: &fakeLLMSource{rows: []LLMEndpoint{{ID: "1", Enabled: true}}},
		Gate:   llmDeniedGate(),
	})

	_, err := api.Dispatch(context.Background(), "p1", "list", nil, "", nil)
	if err == nil {
		t.Fatal("expected EPERM when llm cap is not granted")
	}
	// Gate.Check wraps the deny message — the error should mention "llm"
	// so ops can trace it. The exact code is EPERM; tests elsewhere
	// (capabilities_test.go) pin the code format.
	if err.Error() == "" {
		t.Error("denied error should carry a message")
	}
}

func TestLLMAPI_UnknownMethodReturnsEUNAVAIL(t *testing.T) {
	api := NewLLMAPI(LLMConfig{
		Source: &fakeLLMSource{},
		Gate:   llmGrantedGate(),
	})

	_, err := api.Dispatch(context.Background(), "p1", "delete", nil, "", nil)
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EUNAVAIL" {
		t.Errorf("want WireError EUNAVAIL, got %v", err)
	}
}

func TestLLMAPI_MalformedArgsReturnsEINVAL(t *testing.T) {
	api := NewLLMAPI(LLMConfig{
		Source: &fakeLLMSource{},
		Gate:   llmGrantedGate(),
	})

	_, err := api.Dispatch(context.Background(), "p1", "list",
		json.RawMessage(`{"enabledOnly":"not-a-bool"}`), "", nil)
	if err == nil {
		t.Fatal("expected EINVAL for malformed args")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINVAL" {
		t.Errorf("want EINVAL, got %v", err)
	}
}

func TestLLMAPI_SourceErrorPropagates(t *testing.T) {
	boom := errors.New("db down")
	api := NewLLMAPI(LLMConfig{
		Source: &fakeLLMSource{err: boom},
		Gate:   llmGrantedGate(),
	})

	_, err := api.Dispatch(context.Background(), "p1", "list", nil, "", nil)
	if err == nil {
		t.Fatal("expected propagated error")
	}
	if !errors.Is(err, boom) {
		t.Errorf("want wrapped boom, got %v", err)
	}
}

func TestLLMAPI_NilSourceReturnsEmpty(t *testing.T) {
	// Defensive: a misconfigured wiring (Source unset) should not
	// panic. Dispatch is still a valid call — it returns an empty
	// slice, mirroring the "no rows yet" behaviour callers already
	// expect.
	api := NewLLMAPI(LLMConfig{Source: nil, Gate: llmGrantedGate()})
	got, err := api.Dispatch(context.Background(), "p1", "list", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out := got.([]LLMEndpoint); len(out) != 0 {
		t.Errorf("nil source should yield [], got %+v", out)
	}
}
