package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// memUsers is an in-memory UserStore for tests.
type memUsers struct {
	mu   sync.Mutex
	rows map[string]User
	n    int
}

func newMemUsers() *memUsers { return &memUsers{rows: map[string]User{}} }

func (m *memUsers) List(context.Context) ([]User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []User{}
	for _, u := range m.rows {
		out = append(out, u)
	}
	return out, nil
}

func (m *memUsers) Get(_ context.Context, id string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.rows[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (m *memUsers) GetByUsername(_ context.Context, name string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.rows {
		if strings.EqualFold(u.Username, name) {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (m *memUsers) Create(ctx context.Context, name, display, pw string, role Role) (User, error) {
	if _, err := m.GetByUsername(ctx, name); err == nil {
		return User{}, ErrUserExists
	}
	hash, err := hashPassword(pw)
	if err != nil {
		return User{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.n++
	u := User{ID: "usr_" + string(rune('a'+m.n)), Username: name, DisplayName: display, Role: role, passwordHash: hash, CreatedAt: time.Now()}
	m.rows[u.ID] = u
	return u, nil
}

func (m *memUsers) Update(_ context.Context, id string, p UserPatch) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.rows[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	if p.DisplayName != nil {
		u.DisplayName = *p.DisplayName
	}
	if p.Role != nil {
		u.Role = *p.Role
	}
	if p.Disabled != nil {
		u.Disabled = *p.Disabled
	}
	if p.Password != nil {
		h, err := hashPassword(*p.Password)
		if err != nil {
			return User{}, err
		}
		u.passwordHash = h
	}
	m.rows[id] = u
	return u, nil
}

func (m *memUsers) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rows[id]; !ok {
		return ErrUserNotFound
	}
	delete(m.rows, id)
	return nil
}

func (m *memUsers) TouchLogin(context.Context, string) error { return nil }

func newMultiSvc(t *testing.T) (*Service, *memUsers) {
	t.Helper()
	// Per-test HOME: a primary-credential rotation writes a keyfile
	// that would otherwise shadow the config creds in later tests.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	s := newSvc(t, 0)
	us := newMemUsers()
	s.SetUserStore(us)
	return s, us
}

func TestUsers_MemberLoginCarriesRole(t *testing.T) {
	s, us := newMultiSvc(t)
	u, err := us.Create(context.Background(), "alice", "Alice", "alice-password", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	tok, info, err := s.Login("ALICE", "alice-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if info.Role != RoleMember || info.UserID != u.ID || info.Username != "alice" {
		t.Fatalf("info=%+v", info)
	}
	if got, ok := s.Validate(tok); !ok || got.Role != RoleMember {
		t.Fatalf("validate=%+v ok=%v", got, ok)
	}
	_, admin, err := s.Login("admin", "secret")
	if err != nil || admin.Role != RoleAdmin || admin.UserID != "" {
		t.Fatalf("primary login: %+v %v", admin, err)
	}
}

func TestUsers_DisabledAndWrongPasswordRejected(t *testing.T) {
	s, us := newMultiSvc(t)
	u, _ := us.Create(context.Background(), "bob", "", "bob-password", RoleMember)
	if _, _, err := s.Login("bob", "nope-nope"); err != ErrInvalidCredentials {
		t.Fatalf("wrong password err=%v", err)
	}
	off := true
	_, _ = us.Update(context.Background(), u.ID, UserPatch{Disabled: &off})
	if _, _, err := s.Login("bob", "bob-password"); err != ErrInvalidCredentials {
		t.Fatalf("disabled err=%v", err)
	}
}

func TestUsers_PrimaryRotationKeepsMemberTokens(t *testing.T) {
	s, us := newMultiSvc(t)
	_, _ = us.Create(context.Background(), "carol", "", "carol-password", RoleMember)
	mtok, _, _ := s.Login("carol", "carol-password")
	atok, _, _ := s.Login("admin", "secret")
	if err := s.ChangeCredentials("secret", "admin", "new-secret-123"); err != nil {
		t.Fatalf("change: %v", err)
	}
	if _, ok := s.Validate(atok); ok {
		t.Fatal("primary token survived rotation")
	}
	if _, ok := s.Validate(mtok); !ok {
		t.Fatal("member token revoked by primary rotation")
	}
}

func TestUsers_ChangeOwnPasswordRevokesOldTokens(t *testing.T) {
	s, us := newMultiSvc(t)
	u, _ := us.Create(context.Background(), "dave", "", "dave-password", RoleMember)
	old, _, _ := s.Login("dave", "dave-password")
	if _, err := s.ChangeUserPassword(context.Background(), u.ID, "wrong-current", "dave-new-password"); err != ErrInvalidCredentials {
		t.Fatalf("wrong current err=%v", err)
	}
	if _, err := s.ChangeUserPassword(context.Background(), u.ID, "dave-password", "dave-new-password"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Validate(old); ok {
		t.Fatal("old token survived password change")
	}
	if _, _, err := s.Login("dave", "dave-new-password"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

// adminRouter mirrors app wiring: Middleware + RequireAdmin around
// MountUsers, plus MountProtected for /auth/*.
func adminRouter(s *Service) http.Handler {
	h := NewHandlers(s, nil)
	r := chi.NewRouter()
	h.MountPublic(r)
	r.Group(func(r chi.Router) {
		r.Use(s.Middleware)
		h.MountProtected(r)
		r.Group(func(r chi.Router) {
			r.Use(RequireAdmin)
			h.MountUsers(r)
		})
	})
	return r
}

func do(t *testing.T, h http.Handler, method, path, tok, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUsersAPI_AdminManagesMemberForbidden(t *testing.T) {
	s, _ := newMultiSvc(t)
	h := adminRouter(s)
	atok, _, _ := s.Login("admin", "secret")

	rec := do(t, h, "POST", "/users", atok, `{"username":"erin","password":"erin-password","role":"member"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", rec.Code, rec.Body)
	}
	var created User
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	if rec := do(t, h, "POST", "/users", atok, `{"username":"Admin","password":"whatever-123"}`); rec.Code != http.StatusConflict {
		t.Fatalf("primary-name clash=%d", rec.Code)
	}

	mtok, _, err := s.Login("erin", "erin-password")
	if err != nil {
		t.Fatal(err)
	}
	if rec := do(t, h, "GET", "/users", mtok, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("member list users=%d", rec.Code)
	}
	if rec := do(t, h, "GET", "/auth/me", mtok, ""); !strings.Contains(rec.Body.String(), `"role":"member"`) {
		t.Fatalf("me=%s", rec.Body)
	}

	// Promoting revokes the member's current token.
	if rec := do(t, h, "PATCH", "/users/"+created.ID, atok, `{"role":"admin"}`); rec.Code != http.StatusOK {
		t.Fatalf("promote=%d %s", rec.Code, rec.Body)
	}
	if _, ok := s.Validate(mtok); ok {
		t.Fatal("token survived role change")
	}

	rec = do(t, h, "GET", "/users", atok, "")
	if !strings.Contains(rec.Body.String(), `"primary":true`) || !strings.Contains(rec.Body.String(), `"erin"`) {
		t.Fatalf("list=%s", rec.Body)
	}
	if rec := do(t, h, "DELETE", "/users/"+created.ID, atok, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete=%d", rec.Code)
	}
}

func TestUsersAPI_AdminCannotDemoteSelf(t *testing.T) {
	s, us := newMultiSvc(t)
	h := adminRouter(s)
	u, _ := us.Create(context.Background(), "frank", "", "frank-password", RoleAdmin)
	tok, _, _ := s.Login("frank", "frank-password")
	if rec := do(t, h, "PATCH", "/users/"+u.ID, tok, `{"role":"member"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("self-demote=%d", rec.Code)
	}
	if rec := do(t, h, "DELETE", "/users/"+u.ID, tok, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("self-delete=%d", rec.Code)
	}
}

func TestChangeCredentials_MemberChangesOnlyOwnPassword(t *testing.T) {
	s, us := newMultiSvc(t)
	h := adminRouter(s)
	_, _ = us.Create(context.Background(), "gina", "", "gina-password", RoleMember)
	tok, _, _ := s.Login("gina", "gina-password")
	if rec := do(t, h, "POST", "/auth/change-credentials", tok,
		`{"current_password":"gina-password","new_user":"admin","new_password":"gina-new-password"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("rename attempt=%d", rec.Code)
	}
	rec := do(t, h, "POST", "/auth/change-credentials", tok,
		`{"current_password":"gina-password","new_password":"gina-new-password"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("change=%d %s", rec.Code, rec.Body)
	}
	if _, _, err := s.Login("admin", "secret"); err != nil {
		t.Fatalf("primary admin affected: %v", err)
	}
}
