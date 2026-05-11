package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handlers is the auth REST surface. Login is public (no auth);
// logout / me require the bearer middleware.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "auth.http")}
}

// MountPublic mounts endpoints that do not require authentication.
// Caller is expected to mount this OUTSIDE the protected group.
//
// Uses direct paths instead of r.Route("/auth", ...) so that
// MountProtected can also reach into /auth on the same parent router
// — chi panics on a second Mount() of the same prefix.
func (h *Handlers) MountPublic(r chi.Router) {
	r.Post("/auth/login", h.login)
	r.Post("/auth/mobile-login", h.mobileLogin)
}

// MountProtected mounts endpoints that require a valid bearer token.
// Caller must already have wrapped this group with Service.Middleware.
func (h *Handlers) MountProtected(r chi.Router) {
	r.Post("/auth/logout", h.logout)
	r.Get("/auth/me", h.me)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	h.handleLogin(w, r, h.svc.Login)
}

// mobileLogin issues a longer-TTL token (default 30d) for the
// Flutter mobile app. Same request/response shape as /auth/login;
// only the token lifetime differs. See ADR 0017 §5 (and ADR 0015 §5
// for the original — superseded — auth design rationale).
func (h *Handlers) mobileLogin(w http.ResponseWriter, r *http.Request) {
	h.handleLogin(w, r, h.svc.LoginMobile)
}

func (h *Handlers) handleLogin(
	w http.ResponseWriter,
	r *http.Request,
	loginFn func(user, password string) (string, TokenInfo, error),
) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tok, info, err := loginFn(req.Username, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{
		Token:     tok,
		Username:  info.Username,
		IssuedAt:  info.IssuedAt,
		ExpiresAt: info.ExpiresAt,
	})
}

func (h *Handlers) logout(w http.ResponseWriter, r *http.Request) {
	tok := TokenFromContext(r.Context())
	h.svc.Revoke(tok)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"username": Username(r.Context())})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
