// Package auth provides admin bearer-token authentication for opendray.
//
// The primary admin comes from the keyfile / config / env (design §12);
// additional operator logins live in the users table (see users.go) and
// carry a Role. The Service holds an
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
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/eventbus"
)

const (
	defaultTokenTTL       = 24 * time.Hour
	defaultMobileTokenTTL = 30 * 24 * time.Hour
	tokenByteLen          = 32
)

// ErrInvalidCredentials is returned by Login when user/password do not
// match the admin config (or when admin is unconfigured).
var ErrInvalidCredentials = errors.New("invalid credentials")

// TokenInfo is the public view of an issued token.
type TokenInfo struct {
	Username string `json:"username"`
	// Role is admin for the primary admin and for users created with
	// the admin role; member otherwise.
	Role Role `json:"role"`
	// UserID is the users-table id; empty for the primary admin.
	UserID    string    `json:"user_id,omitempty"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service is the admin auth surface. Construct via New and pass
// Middleware to chi.Router.Use for protected groups.
//
// Since PR #53 the active credentials live in an atomic.Pointer so
// the change-credentials handler can hot-swap them without
// restarting the gateway. Reads (login path) are lock-free; the
// only writer is ChangeCredentials, which serialises via credsMu.
type Service struct {
	cfg       config.AdminConfig
	ttl       time.Duration
	mobileTTL time.Duration
	bus       *eventbus.Hub
	log       *slog.Logger

	// creds is the live credentials snapshot. Atomically swapped
	// by ChangeCredentials. Loaded once at startup via LoadCreds;
	// callers read via activeCreds() which is a single atomic
	// load.
	creds   atomic.Pointer[AdminCreds]
	credsMu sync.Mutex

	// users holds the additional operator logins. nil = single-admin
	// mode (tests, or a gateway without a DB-backed store wired).
	users UserStore

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
	mobileTTL := cfg.MobileDuration()
	if mobileTTL <= 0 {
		mobileTTL = defaultMobileTokenTTL
	}
	s := &Service{
		cfg:       cfg,
		ttl:       ttl,
		mobileTTL: mobileTTL,
		bus:       bus,
		log:       log.With("component", "auth"),
		tokens:    make(map[string]TokenInfo),
	}
	// Boot-time credential load. Errors here are loud (corrupt
	// keyfile, etc.) but we degrade rather than refuse to start —
	// callers can re-configure via env or by removing the bad
	// file. activeCreds() returning nil simply rejects all logins
	// until that happens.
	creds, err := LoadCreds(cfg.User, cfg.Password)
	if err != nil {
		s.log.Error("admin credentials load failed; logins disabled until fixed", "err", err)
	} else if creds.User != "" {
		s.creds.Store(&creds)
		s.log.Info("admin credentials loaded", "user", creds.User, "source", string(creds.Source))
	}
	return s
}

// SetUserStore wires the additional-users store. Call once at startup,
// before serving requests.
func (s *Service) SetUserStore(us UserStore) { s.users = us }

// Users returns the wired user store, or nil in single-admin mode.
func (s *Service) Users() UserStore { return s.users }

// activeCreds returns the current credentials snapshot, or nil
// when admin is not configured. Single atomic load.
func (s *Service) activeCreds() *AdminCreds {
	return s.creds.Load()
}

// Login validates user/password using constant-time comparison and
// issues a fresh token on success. Empty admin config rejects all.
// Uses the configured `[admin].token_ttl` (default 24h).
func (s *Service) Login(user, password string) (string, TokenInfo, error) {
	return s.loginWithTTL(user, password, s.ttl)
}

// LoginMobile is identical to Login except the token's absolute
// lifetime is `[admin].mobile_token_ttl` (default 30d). The longer
// TTL is reasonable on mobile because the device gates access via
// biometrics + secure storage; on a desktop browser the shorter
// default applies. See ADR 0015 §5.
func (s *Service) LoginMobile(user, password string) (string, TokenInfo, error) {
	return s.loginWithTTL(user, password, s.mobileTTL)
}

func (s *Service) loginWithTTL(user, password string, ttl time.Duration) (string, TokenInfo, error) {
	var info TokenInfo
	if creds := s.activeCreds(); creds != nil && creds.VerifyPassword(user, password) {
		info = TokenInfo{Username: creds.User, Role: RoleAdmin}
	} else if u, ok := s.verifyUser(user, password); ok {
		info = TokenInfo{Username: u.Username, Role: u.Role, UserID: u.ID}
	} else {
		s.publishLogin(user, false, "credentials mismatch")
		return "", TokenInfo{}, ErrInvalidCredentials
	}
	return s.issue(info, ttl)
}

// issue stores a fresh token for info with the given lifetime.
func (s *Service) issue(info TokenInfo, ttl time.Duration) (string, TokenInfo, error) {
	tok, err := generateToken()
	if err != nil {
		return "", TokenInfo{}, err
	}
	now := time.Now().UTC()
	info.IssuedAt = now
	info.ExpiresAt = now.Add(ttl)
	s.mu.Lock()
	s.tokens[tok] = info
	s.mu.Unlock()
	if info.UserID != "" && s.users != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.users.TouchLogin(ctx, info.UserID); err != nil {
			s.log.Warn("record last login failed", "user", info.Username, "err", err)
		}
		cancel()
	}
	s.publishLogin(info.Username, true, "")
	return tok, info, nil
}

// dummyHash keeps a failed users-table lookup as slow as a real bcrypt
// verify so login timing doesn't reveal which usernames exist.
const dummyHash = "$2a$10$3VPyywpwKPbWoaaIFTpWZ.OE3hrKwgZdvDt7yPMtRSXY7WhgVoP4G"

// verifyUser checks user/password against the users table. Disabled
// users never verify.
func (s *Service) verifyUser(user, password string) (User, bool) {
	if s.users == nil {
		return User{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	u, err := s.users.GetByUsername(ctx, user)
	if err != nil || u.Disabled {
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			s.log.Warn("user lookup failed", "err", err)
		}
		_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return User{}, false
	}
	if bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password)) != nil {
		return User{}, false
	}
	return u, true
}

// ChangeUserPassword rotates a users-table login's own password after
// re-verifying the current one, then revokes that user's tokens.
func (s *Service) ChangeUserPassword(ctx context.Context, userID, currentPassword, newPassword string) (User, error) {
	if s.users == nil {
		return User{}, ErrInvalidCredentials
	}
	u, err := s.users.Get(ctx, userID)
	if err != nil || u.Disabled {
		return User{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(currentPassword)) != nil {
		s.publishLogin(u.Username, false, "change-password: current password mismatch")
		return User{}, ErrInvalidCredentials
	}
	u, err = s.users.Update(ctx, userID, UserPatch{Password: &newPassword})
	if err != nil {
		return User{}, err
	}
	s.RevokeUser(u.Username)
	return u, nil
}

// IssueFor logs u in without a password check. Used right after a
// verified password change so the caller keeps a working session.
func (s *Service) IssueFor(u User) (string, TokenInfo, error) {
	return s.issue(TokenInfo{Username: u.Username, Role: u.Role, UserID: u.ID}, s.ttl)
}

// RevokeUser drops every token issued to username (case-insensitive).
// Called when a user is disabled, deleted, re-roled or has their
// password reset, so the change takes effect immediately.
func (s *Service) RevokeUser(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for tok, info := range s.tokens {
		if strings.EqualFold(info.Username, username) {
			delete(s.tokens, tok)
		}
	}
}

// ChangeCredentials rotates the operator's username + password.
// Verifies currentPassword against the live credentials first
// (defence against a stolen bearer token — without the current
// password an attacker can't lock the operator out of their own
// gateway).
//
// On success: writes a new keyfile, hot-swaps the in-memory creds
// pointer, and revokes ALL existing tokens so everyone has to
// re-authenticate with the new password. The revoke step is what
// makes "change password after a suspected leak" actually
// defensive — otherwise the attacker's token keeps working.
func (s *Service) ChangeCredentials(currentPassword, newUser, newPassword string) error {
	s.credsMu.Lock()
	defer s.credsMu.Unlock()

	active := s.activeCreds()
	if active == nil {
		return ErrInvalidCredentials
	}
	if !active.VerifyPassword(active.User, currentPassword) {
		s.publishLogin(active.User, false, "change-credentials: current password mismatch")
		return ErrInvalidCredentials
	}

	newUser = strings.TrimSpace(newUser)
	if newUser == "" {
		newUser = active.User
	}
	if len(newPassword) < MinPasswordLen {
		return errors.New("new password too short")
	}

	if _, err := WriteKeyFile(newUser, newPassword); err != nil {
		return err
	}
	// Re-load so the in-memory copy matches what was just
	// written, including the bcrypt hash (not the plaintext).
	loaded, err := LoadCreds(s.cfg.User, s.cfg.Password)
	if err != nil {
		return err
	}
	s.creds.Store(&loaded)
	s.revokePrimaryTokens()
	s.publishCredsChanged(newUser)
	return nil
}

// revokePrimaryTokens drops every token issued to the primary admin so a
// credential rotation forces it to re-authenticate. Users-table logins
// keep their tokens — their credentials did not change.
func (s *Service) revokePrimaryTokens() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for tok, info := range s.tokens {
		if info.UserID == "" {
			delete(s.tokens, tok)
		}
	}
}

func (s *Service) publishCredsChanged(user string) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(eventbus.Event{
		Topic: "admin.credentials_changed",
		Data: map[string]any{
			"user": user,
		},
	})
}

// ActiveUser exposes the current admin username for read-only UI
// surfaces (e.g. populating "new username" with the current value).
// Returns "" when admin isn't configured.
func (s *Service) ActiveUser() string {
	c := s.activeCreds()
	if c == nil {
		return ""
	}
	return c.User
}

// ActiveSource lets the UI explain how the operator is currently
// authenticated (env / file / config) — useful for the change-
// credentials screen so it can warn when an env var is going to
// keep shadowing whatever the operator writes via UI.
func (s *Service) ActiveSource() CredSource {
	c := s.activeCreds()
	if c == nil {
		return CredSourceNone
	}
	return c.Source
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
		next.ServeHTTP(w, r.WithContext(withInfo(r.Context(), info, tok)))
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
	return withInfo(ctx, info, token), true
}

func withInfo(ctx context.Context, info TokenInfo, token string) context.Context {
	ctx = context.WithValue(ctx, userCtxKey{}, info.Username)
	ctx = context.WithValue(ctx, infoCtxKey{}, info)
	return context.WithValue(ctx, tokenCtxKey{}, token)
}

// Info retrieves the authenticated token's info from a request context.
func Info(ctx context.Context) (TokenInfo, bool) {
	v, ok := ctx.Value(infoCtxKey{}).(TokenInfo)
	return v, ok
}

// IsAdmin reports whether the request was authenticated with an
// admin-role token.
func IsAdmin(ctx context.Context) bool {
	info, ok := Info(ctx)
	return ok && info.Role == RoleAdmin
}

// RequireAdmin rejects requests whose token is not admin-role with 403.
// Mount it inside a group that already runs Middleware.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r.Context()) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"admin role required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
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
type infoCtxKey struct{}

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
