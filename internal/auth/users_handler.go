package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// MountUsers mounts admin-only user management. Caller must wrap the
// group with Middleware + RequireAdmin.
//
//	GET    /users        list (primary admin first, read-only)
//	POST   /users        create {username, password, role, display_name}
//	PATCH  /users/{id}   update {display_name?, role?, disabled?, password?}
//	DELETE /users/{id}
func (h *Handlers) MountUsers(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.listUsers)
		r.Post("/", h.createUser)
		r.Patch("/{id}", h.updateUser)
		r.Delete("/{id}", h.deleteUser)
	})
}

// userView adds the read-only primary admin row to the list so the UI
// shows every login in one table.
type userView struct {
	User
	Primary bool `json:"primary"`
}

func (h *Handlers) listUsers(w http.ResponseWriter, r *http.Request) {
	out := []userView{}
	if u := h.svc.ActiveUser(); u != "" {
		out = append(out, userView{User: User{ID: "primary", Username: u, Role: RoleAdmin}, Primary: true})
	}
	if us := h.svc.Users(); us != nil {
		list, err := us.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		for _, u := range list {
			out = append(out, userView{User: u})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

type createUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        Role   `json:"role"`
}

func (h *Handlers) createUser(w http.ResponseWriter, r *http.Request) {
	us := h.svc.Users()
	if us == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("user store not configured"))
		return
	}
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Role == "" {
		req.Role = RoleMember
	}
	switch {
	case !validUsername(req.Username):
		writeError(w, http.StatusBadRequest, errors.New("username must be 2-64 characters: letters, digits, . _ - @"))
		return
	case strings.EqualFold(req.Username, h.svc.ActiveUser()):
		writeError(w, http.StatusConflict, ErrUserExists)
		return
	case !req.Role.Valid():
		writeError(w, http.StatusBadRequest, errors.New(`role must be "admin" or "member"`))
		return
	}
	u, err := us.Create(r.Context(), req.Username, strings.TrimSpace(req.DisplayName), req.Password, req.Role)
	if errors.Is(err, ErrUserExists) {
		writeError(w, http.StatusConflict, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.log.Info("user created", "user", u.Username, "role", u.Role, "by", Username(r.Context()))
	writeJSON(w, http.StatusCreated, u)
}

type updateUserRequest struct {
	DisplayName *string `json:"display_name"`
	Role        *Role   `json:"role"`
	Disabled    *bool   `json:"disabled"`
	Password    *string `json:"password"`
}

func (h *Handlers) updateUser(w http.ResponseWriter, r *http.Request) {
	us := h.svc.Users()
	if us == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("user store not configured"))
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Role != nil && !req.Role.Valid() {
		writeError(w, http.StatusBadRequest, errors.New(`role must be "admin" or "member"`))
		return
	}
	id := chi.URLParam(r, "id")
	// An admin may not demote or disable their own users-table login:
	// that would lock them out mid-request with no way back except the
	// primary admin.
	if info, _ := Info(r.Context()); info.UserID == id &&
		((req.Role != nil && *req.Role != RoleAdmin) || (req.Disabled != nil && *req.Disabled)) {
		writeError(w, http.StatusBadRequest, errors.New("you cannot demote or disable your own login"))
		return
	}
	u, err := us.Update(r.Context(), id, UserPatch{
		DisplayName: req.DisplayName,
		Role:        req.Role,
		Disabled:    req.Disabled,
		Password:    req.Password,
	})
	if errors.Is(err, ErrUserNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Role, disable and password changes must apply now, not at token
	// expiry: drop the user's tokens so they re-authenticate.
	if req.Role != nil || req.Disabled != nil || req.Password != nil {
		h.svc.RevokeUser(u.Username)
	}
	h.log.Info("user updated", "user", u.Username, "by", Username(r.Context()))
	writeJSON(w, http.StatusOK, u)
}

func (h *Handlers) deleteUser(w http.ResponseWriter, r *http.Request) {
	us := h.svc.Users()
	if us == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("user store not configured"))
		return
	}
	id := chi.URLParam(r, "id")
	if info, _ := Info(r.Context()); info.UserID == id {
		writeError(w, http.StatusBadRequest, errors.New("you cannot delete your own login"))
		return
	}
	u, err := us.Get(r.Context(), id)
	if errors.Is(err, ErrUserNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := us.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	h.svc.RevokeUser(u.Username)
	h.log.Info("user deleted", "user", u.Username, "by", Username(r.Context()))
	w.WriteHeader(http.StatusNoContent)
}
