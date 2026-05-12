package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/config"
)

// TestMain isolates HOME + the keyfile env override for the whole
// test binary so auth.New() doesn't pick up a real
// ~/.opendray/secrets/admin.key from the developer's home dir.
// Without this the config-source creds in newSvc / newRouter get
// shadowed by whatever the dev's UI-created keyfile holds, and
// every Login test that expects "admin / secret" returns 401.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "auth-test-home-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	_ = os.Setenv("HOME", tmp)
	_ = os.Unsetenv(envOverride)
	os.Exit(m.Run())
}

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

func TestLoginMobile_UsesMobileTTL(t *testing.T) {
	cfg := config.AdminConfig{
		User:           "admin",
		Password:       "secret",
		TokenTTL:       "1h",
		MobileTokenTTL: "168h", // 7d
	}
	s := New(cfg, nil, nil)

	_, browserInfo, err := s.Login("admin", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	_, mobileInfo, err := s.LoginMobile("admin", "secret")
	if err != nil {
		t.Fatalf("LoginMobile: %v", err)
	}

	browserSpan := browserInfo.ExpiresAt.Sub(browserInfo.IssuedAt)
	mobileSpan := mobileInfo.ExpiresAt.Sub(mobileInfo.IssuedAt)
	if browserSpan >= mobileSpan {
		t.Errorf("expected mobile TTL > browser TTL; got browser=%s mobile=%s",
			browserSpan, mobileSpan)
	}
	wantBrowser := time.Hour
	wantMobile := 168 * time.Hour
	if browserSpan != wantBrowser {
		t.Errorf("browser TTL = %s, want %s", browserSpan, wantBrowser)
	}
	if mobileSpan != wantMobile {
		t.Errorf("mobile TTL = %s, want %s", mobileSpan, wantMobile)
	}
}

func TestLoginMobile_DefaultsToThirtyDays(t *testing.T) {
	// MobileTokenTTL unset → 30d default; TokenTTL unset → 24h default.
	cfg := config.AdminConfig{User: "admin", Password: "secret"}
	s := New(cfg, nil, nil)
	_, info, err := s.LoginMobile("admin", "secret")
	if err != nil {
		t.Fatalf("LoginMobile: %v", err)
	}
	span := info.ExpiresAt.Sub(info.IssuedAt)
	want := 30 * 24 * time.Hour
	if span != want {
		t.Errorf("default mobile TTL = %s, want %s", span, want)
	}
}

func TestLoginMobile_WrongCredentialsRejected(t *testing.T) {
	s := newSvc(t, 0)
	if _, _, err := s.LoginMobile("admin", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("err=%v, want ErrInvalidCredentials", err)
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
