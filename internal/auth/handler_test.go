package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/config"
)

func newRouter(t *testing.T) (*Service, http.Handler) {
	t.Helper()
	svc := New(config.AdminConfig{User: "admin", Password: "secret"}, nil, nil)
	h := NewHandlers(svc, nil)
	r := chi.NewRouter()
	h.MountPublic(r)
	r.Group(func(r chi.Router) {
		r.Use(svc.Middleware)
		h.MountProtected(r)
	})
	return svc, r
}

func TestLogin_OK(t *testing.T) {
	_, r := newRouter(t)
	body := bytes.NewBufferString(`{"username":"admin","password":"secret"}`)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/auth/login", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var resp loginResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Token == "" || resp.Username != "admin" {
		t.Errorf("resp=%+v", resp)
	}
}

func TestLogin_BadCreds(t *testing.T) {
	_, r := newRouter(t)
	body := bytes.NewBufferString(`{"username":"admin","password":"wrong"}`)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/auth/login", body))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestLogin_BadJSON(t *testing.T) {
	_, r := newRouter(t)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/auth/login",
		bytes.NewBufferString(`not json`)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestLogout_RevokesCallerToken(t *testing.T) {
	svc, r := newRouter(t)
	tok, _, _ := svc.Login("admin", "secret")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rr.Code)
	}

	// token now revoked — /me must 401
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("post-logout /me status=%d", rr.Code)
	}
}

func TestMe_ReturnsUsername(t *testing.T) {
	svc, r := newRouter(t)
	tok, _, _ := svc.Login("admin", "secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["username"] != "admin" || resp["role"] != "admin" || resp["primary"] != true {
		t.Errorf("me=%v", resp)
	}
}

func TestMe_RequiresAuth(t *testing.T) {
	_, r := newRouter(t)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}
