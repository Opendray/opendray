// Package auth provides admin bearer-token authentication for opendray.
//
// Per design §12, opendray is single-admin. The Service holds an
// in-memory token map; tokens are 32 random bytes, base64url-encoded,
// with absolute TTL (default 24h, configurable via admin.token_ttl).
// No JWT, no clock-sync requirement, no signing key.
//
// Middleware accepts the bearer in `Authorization: Bearer <token>` for
// REST and falls back to a `?token=<bearer>` query parameter for
// WebSocket upgrades, since browsers cannot set custom headers on the
// WS handshake.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/eventbus"
)

const (
	defaultTokenTTL = 24 * time.Hour
	tokenByteLen    = 32
)

// ErrInvalidCredentials is returned by Login when user/password do not
// match the admin config (or when admin is unconfigured).
var ErrInvalidCredentials = errors.New("invalid credentials")

// TokenInfo is the public view of an issued token.
type TokenInfo struct {
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service is the admin auth surface. Construct via New and pass
// Middleware to chi.Router.Use for protected groups.
type Service struct {
	cfg config.AdminConfig
	ttl time.Duration
	bus *eventbus.Hub
	log *slog.Logger

	mu     sync.RWMutex
	tokens map[string]TokenInfo
}

func New(cfg config.AdminConfig, bus *eventbus.Hub, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	ttl := cfg.Duration()
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	s := &Service{
		cfg:    cfg,
		ttl:    ttl,
		bus:    bus,
		log:    log.With("component", "auth"),
		tokens: make(map[string]TokenInfo),
	}
	if cfg.PasswordHash == "" && cfg.Password != "" {
		s.log.Warn("admin password is plaintext in config; generate a hash with `opendray hash-password` and replace [admin].password with [admin].password_hash")
	}
	return s
}

// Login validates user/password and issues a fresh token on success.
// Empty admin config rejects all.
//
// Verification path:
//   - If cfg.PasswordHash is set, password is verified with
//     bcrypt.CompareHashAndPassword (constant-time inside bcrypt).
//   - Else if cfg.Password is set, password is constant-time compared
//     against the plaintext config value (back-compat path; emits a
//     one-time warning at construction time).
//   - Else the admin is unconfigured and Login rejects everything.
//
// Username comparison is always constant-time.
func (s *Service) Login(user, password string) (string, TokenInfo, error) {
	if s.cfg.User == "" || (s.cfg.Password == "" && s.cfg.PasswordHash == "") {
		s.publishLogin(user, false, "admin not configured")
		return "", TokenInfo{}, ErrInvalidCredentials
	}
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(s.cfg.User)) == 1
	passOK := s.verifyPassword(password)
	if !userOK || !passOK {
		s.publishLogin(user, false, "credentials mismatch")
		return "", TokenInfo{}, ErrInvalidCredentials
	}

	tok, err := generateToken()
	if err != nil {
		return "", TokenInfo{}, err
	}
	now := time.Now().UTC()
	info := TokenInfo{
		Username:  s.cfg.User,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.ttl),
	}
	s.mu.Lock()
	s.tokens[tok] = info
	s.mu.Unlock()
	s.publishLogin(s.cfg.User, true, "")
	return tok, info, nil
}

// Validate returns the TokenInfo if the token is known and unexpired.
// Expired tokens are revoked lazily on validation.
func (s *Service) Validate(token string) (TokenInfo, bool) {
	if token == "" {
		return TokenInfo{}, false
	}
	s.mu.RLock()
	info, ok := s.tokens[token]
	s.mu.RUnlock()
	if !ok {
		return TokenInfo{}, false
	}
	if time.Now().After(info.ExpiresAt) {
		s.mu.Lock()
		delete(s.tokens, token)
		s.mu.Unlock()
		return TokenInfo{}, false
	}
	return info, true
}

// Revoke removes a specific token. Used by /auth/logout.
func (s *Service) Revoke(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tokens[token]; !ok {
		return false
	}
	delete(s.tokens, token)
	s.publishLogout()
	return true
}

// Middleware enforces a valid bearer token on the wrapped handler.
// Reads the bearer from Authorization: Bearer <tok> or ?token=<tok>.
// On success the request context carries the username via UsernameKey.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := bearerToken(r)
		info, ok := s.Validate(tok)
		if !ok {
			writeUnauth(w)
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey{}, info.Username)
		ctx = context.WithValue(ctx, tokenCtxKey{}, tok)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AttachContext sets the auth-related context keys if `token` is a
// valid admin bearer; returns (newCtx, true) on success, or
// (origCtx, false) otherwise. Used by combined middleware that
// supports both admin and integration auth on the same route — the
// integration package can attach admin context without poking at
// auth's private keys.
func (s *Service) AttachContext(ctx context.Context, token string) (context.Context, bool) {
	info, ok := s.Validate(token)
	if !ok {
		return ctx, false
	}
	ctx = context.WithValue(ctx, userCtxKey{}, info.Username)
	ctx = context.WithValue(ctx, tokenCtxKey{}, token)
	return ctx, true
}

// Username retrieves the authenticated username from a request context.
func Username(ctx context.Context) string {
	if v, ok := ctx.Value(userCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// TokenFromContext retrieves the bearer string from a request context.
// Used by /auth/logout to revoke the caller's own token.
func TokenFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tokenCtxKey{}).(string); ok {
		return v
	}
	return ""
}

type userCtxKey struct{}
type tokenCtxKey struct{}

// verifyPassword checks the supplied password against either the
// configured bcrypt hash or, as a back-compat path, the plaintext
// password. The bcrypt route is preferred whenever PasswordHash is
// non-empty; the plaintext route is constant-time-compared.
func (s *Service) verifyPassword(password string) bool {
	if s.cfg.PasswordHash != "" {
		err := bcrypt.CompareHashAndPassword(
			[]byte(s.cfg.PasswordHash), []byte(password),
		)
		return err == nil
	}
	if s.cfg.Password == "" {
		return false
	}
	return subtle.ConstantTimeCompare(
		[]byte(password), []byte(s.cfg.Password),
	) == 1
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	return ""
}

func generateToken() (string, error) {
	var b [tokenByteLen]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func writeUnauth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="opendray"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}

func (s *Service) publishLogin(user string, ok bool, reason string) {
	if s.bus == nil {
		return
	}
	topic := "admin.login_success"
	if !ok {
		topic = "admin.login_failed"
	}
	s.bus.Publish(eventbus.Event{
		Topic: topic,
		Data: map[string]any{
			"user":   user,
			"reason": reason,
		},
	})
}

func (s *Service) publishLogout() {
	if s.bus == nil {
		return
	}
	s.bus.Publish(eventbus.Event{Topic: "admin.logout"})
}
