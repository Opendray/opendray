package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/opendray/opendray-v2/internal/config"
)

func newSvc(t *testing.T, ttl time.Duration) *Service {
	t.Helper()
	cfg := config.AdminConfig{User: "admin", Password: "secret"}
	if ttl > 0 {
		cfg.TokenTTL = ttl.String()
	}
	return New(cfg, nil, nil)
}

func TestLogin_Success(t *testing.T) {
	s := newSvc(t, 0)
	tok, info, err := s.Login("admin", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok == "" {
		t.Fatal("token is empty")
	}
	if info.Username != "admin" {
		t.Errorf("username=%q", info.Username)
	}
	if info.ExpiresAt.Before(time.Now()) {
		t.Errorf("expires_at in past: %v", info.ExpiresAt)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	s := newSvc(t, 0)
	if _, _, err := s.Login("admin", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("err=%v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_WrongUsername(t *testing.T) {
	s := newSvc(t, 0)
	if _, _, err := s.Login("root", "secret"); err != ErrInvalidCredentials {
		t.Fatalf("err=%v", err)
	}
}

func TestLogin_AdminNotConfigured(t *testing.T) {
	s := New(config.AdminConfig{}, nil, nil)
	if _, _, err := s.Login("admin", "secret"); err != ErrInvalidCredentials {
		t.Fatalf("expected reject when admin unconfigured")
	}
}

// bcryptHash returns a bcrypt hash of plaintext at MinCost, suitable
// for tests that need a real hash without being slow. Production code
// should never use MinCost — operators get bcrypt.DefaultCost through
// the hash-password subcommand.
func bcryptHash(t *testing.T, plaintext string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt hash: %v", err)
	}
	return string(hash)
}

func TestLogin_PasswordHash_Success(t *testing.T) {
	cfg := config.AdminConfig{User: "admin", PasswordHash: bcryptHash(t, "secret")}
	s := New(cfg, nil, nil)
	tok, info, err := s.Login("admin", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok == "" {
		t.Fatal("token empty")
	}
	if info.Username != "admin" {
		t.Errorf("username=%q", info.Username)
	}
}

func TestLogin_PasswordHash_WrongPassword(t *testing.T) {
	cfg := config.AdminConfig{User: "admin", PasswordHash: bcryptHash(t, "secret")}
	s := New(cfg, nil, nil)
	if _, _, err := s.Login("admin", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("err=%v, want ErrInvalidCredentials", err)
	}
}

// PasswordHash takes precedence — supplying the right plaintext for
// the legacy field while the hash is set must NOT authenticate.
func TestLogin_PasswordHashWinsOverPlaintext(t *testing.T) {
	cfg := config.AdminConfig{
		User:         "admin",
		Password:     "old-plaintext",                   // would auth under back-compat…
		PasswordHash: bcryptHash(t, "secret"),           // …but hash is set, so this is ignored.
	}
	s := New(cfg, nil, nil)
	if _, _, err := s.Login("admin", "old-plaintext"); err != ErrInvalidCredentials {
		t.Fatalf("plaintext should be ignored when PasswordHash is set; got err=%v", err)
	}
	if _, _, err := s.Login("admin", "secret"); err != nil {
		t.Fatalf("hash path should authenticate; got err=%v", err)
	}
}

func TestLogin_BackCompatPlaintext(t *testing.T) {
	// Existing installs with [admin].password set and no
	// password_hash must continue to authenticate.
	cfg := config.AdminConfig{User: "admin", Password: "still-works"}
	s := New(cfg, nil, nil)
	if _, _, err := s.Login("admin", "still-works"); err != nil {
		t.Fatalf("legacy plaintext path broken: %v", err)
	}
	if _, _, err := s.Login("admin", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("legacy plaintext should still reject wrong password: %v", err)
	}
}

func TestValidate_OK(t *testing.T) {
	s := newSvc(t, 0)
	tok, _, _ := s.Login("admin", "secret")
	if _, ok := s.Validate(tok); !ok {
		t.Fatal("fresh token failed validation")
	}
}

func TestValidate_Expired(t *testing.T) {
	s := newSvc(t, 10*time.Millisecond)
	tok, _, _ := s.Login("admin", "secret")
	time.Sleep(20 * time.Millisecond)
	if _, ok := s.Validate(tok); ok {
		t.Fatal("expected expired token to fail validation")
	}
	// expired token is revoked lazily — second Validate also fails
	if _, ok := s.Validate(tok); ok {
		t.Fatal("expired token should remain invalid")
	}
}

func TestValidate_EmptyToken(t *testing.T) {
	s := newSvc(t, 0)
	if _, ok := s.Validate(""); ok {
		t.Fatal("empty token must fail")
	}
}

func TestRevoke(t *testing.T) {
	s := newSvc(t, 0)
	tok, _, _ := s.Login("admin", "secret")
	if !s.Revoke(tok) {
		t.Fatal("revoke returned false on first call")
	}
	if _, ok := s.Validate(tok); ok {
		t.Fatal("revoked token still valid")
	}
	if s.Revoke(tok) {
		t.Fatal("revoke returned true on second call")
	}
}

func TestMiddleware_Unauthorized(t *testing.T) {
	s := newSvc(t, 0)
	called := false
	h := s.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
	if called {
		t.Fatal("downstream handler was called without auth")
	}
}

func TestMiddleware_HeaderAuth(t *testing.T) {
	s := newSvc(t, 0)
	tok, _, _ := s.Login("admin", "secret")

	var gotUser, gotToken string
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = Username(r.Context())
		gotToken = TokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	if gotUser != "admin" {
		t.Errorf("user=%q", gotUser)
	}
	if gotToken != tok {
		t.Errorf("token mismatch")
	}
}

func TestMiddleware_QueryAuthForWS(t *testing.T) {
	s := newSvc(t, 0)
	tok, _, _ := s.Login("admin", "secret")

	called := false
	h := s.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x?token="+tok, nil)
	h.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected query-token to authorize WS-style requests")
	}
}

func TestMiddleware_BadToken(t *testing.T) {
	s := newSvc(t, 0)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	s.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		tok, err := generateToken()
		if err != nil {
			t.Fatal(err)
		}
		if seen[tok] {
			t.Fatalf("collision at %d", i)
		}
		seen[tok] = true
	}
}
