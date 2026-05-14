package memhealth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandler_MissingCwd(t *testing.T) {
	h := NewHandlers(&Service{}, nil)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/memory/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cwd") {
		t.Errorf("body should mention cwd: %s", rec.Body.String())
	}
}

func TestHandler_LiveDB(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	svc, err := New(pool)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(svc, nil)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/memory/health?cwd=/__handler_test__", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
	}
	var got Snapshot
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Cwd != "/__handler_test__" {
		t.Errorf("Cwd: got %q", got.Cwd)
	}
	if got.LookbackDays != LookbackDays {
		t.Errorf("LookbackDays: got %d", got.LookbackDays)
	}
}
