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
	r.Post("/auth/change-credentials", h.changeCredentials)
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
	writeJSON(w, http.StatusOK, map[string]string{
		"username": Username(r.Context()),
		// active_source lets the UI warn when an env var is
		// shadowing whatever is on disk — "you can change your
		// credentials but they won't take effect until you unset
		// $OPENDRAY_ADMIN_PASSWORD."
		"active_source": string(h.svc.ActiveSource()),
	})
}

type changeCredentialsRequest struct {
	CurrentPassword string `json:"current_password"`
	NewUser         string `json:"new_user"`
	NewPassword     string `json:"new_password"`
}

// changeCredentials rotates the operator's username + password.
// Wired under MountProtected so the caller is already
// authenticated; the body's current_password is verified again as
// a defence against a stolen bearer token being used to lock the
// operator out of their own gateway. After success the caller's
// own token is revoked along with everyone else's — the response
// carries a fresh token issued under the new credentials so the
// mobile / web client doesn't have to re-prompt for password
// immediately.
func (h *Handlers) changeCredentials(w http.ResponseWriter, r *http.Request) {
	var req changeCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.ChangeCredentials(req.CurrentPassword, req.NewUser, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Issue a fresh token immediately so the caller stays logged
	// in. We log them in with the new password they just set —
	// no need to re-prompt for it client-side.
	tok, info, lerr := h.svc.Login(h.resolveNewUser(req), req.NewPassword)
	if lerr != nil {
		// Highly unlikely (we just verified the password works)
		// — but if it does happen, surface a clear "logged out,
		// re-authenticate" so the client doesn't spin.
		writeError(w, http.StatusOK, errors.New("credentials updated; please log in again"))
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{
		Token:     tok,
		Username:  info.Username,
		IssuedAt:  info.IssuedAt,
		ExpiresAt: info.ExpiresAt,
	})
}

// resolveNewUser mirrors the same defaulting logic
// auth.Service.ChangeCredentials uses internally — empty new_user
// means "keep current". We re-derive it here to issue the post-
// change login with the right username.
func (h *Handlers) resolveNewUser(req changeCredentialsRequest) string {
	if u := req.NewUser; u != "" {
		return u
	}
	return h.svc.ActiveUser()
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
