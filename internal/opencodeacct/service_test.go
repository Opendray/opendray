package opencodeacct

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeRepo is an in-memory repo. It stores exactly what the Service hands
// it, so tests can inspect what would land in the DB.
type fakeRepo struct {
	rows map[string]Account
	n    int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[string]Account{}} }

func (f *fakeRepo) Insert(_ context.Context, a Account) (Account, error) {
	f.n++
	a.ID = fmt.Sprintf("oc_%d", f.n)
	a.CreatedAt, a.UpdatedAt = time.Now(), time.Now()
	f.rows[a.ID] = a
	return a, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (Account, error) {
	a, ok := f.rows[id]
	if !ok {
		return Account{}, ErrNotFound
	}
	return a, nil
}
func (f *fakeRepo) GetByName(_ context.Context, name string) (Account, error) {
	for _, a := range f.rows {
		if a.Name == name {
			return a, nil
		}
	}
	return Account{}, ErrNotFound
}
func (f *fakeRepo) List(_ context.Context) ([]Account, error) {
	out := []Account{}
	for _, a := range f.rows {
		out = append(out, a)
	}
	return out, nil
}
func (f *fakeRepo) Update(_ context.Context, a Account) (Account, error) {
	if _, ok := f.rows[a.ID]; !ok {
		return Account{}, ErrNotFound
	}
	f.rows[a.ID] = a
	return a, nil
}
func (f *fakeRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.rows[id]; !ok {
		return ErrNotFound
	}
	delete(f.rows, id)
	return nil
}
func (f *fakeRepo) sessionLoad(context.Context) (map[string]sessionStats, error) {
	return map[string]sessionStats{}, nil
}

// fakeCipher is a reversible "v1:" envelope (base64) — enough to prove the
// Service only ever hands the repo ciphertext, never the plaintext.
type fakeCipher struct{ armed bool }

func (c fakeCipher) EncryptField(p string) (string, error) {
	if !c.armed {
		return "", errors.New("not armed")
	}
	return "v1:" + base64.StdEncoding.EncodeToString([]byte(p)), nil
}
func (c fakeCipher) DecryptField(e string) (string, error) {
	if !c.armed || !strings.HasPrefix(e, "v1:") {
		return "", errors.New("cannot decrypt")
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(e, "v1:"))
	return string(b), err
}

const testKey = "sk-moonshot-0123456789abcdefWXYZ"

func newTestService(armed bool) (*Service, *fakeRepo) {
	r := newFakeRepo()
	return newService(r, nil, fakeCipher{armed: armed}, nil), r
}

func TestCreate_EncryptsAtRestAndNeverSerializesSecret(t *testing.T) {
	svc, repo := newTestService(true)
	a, err := svc.Create(context.Background(), CreateRequest{
		Name: "work", ProviderID: "moonshotai", APIKey: testKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	stored := repo.rows[a.ID].credentialsEnc
	if !strings.HasPrefix(stored, "v1:") {
		t.Fatalf("stored credentials not an envelope: %q", stored)
	}
	if strings.Contains(stored, testKey) {
		t.Fatal("plaintext key reached the repo")
	}

	for _, v := range []any{a, mustList(t, svc)} {
		raw, _ := json.Marshal(v)
		if strings.Contains(string(raw), testKey) || strings.Contains(string(raw), stored) {
			t.Fatalf("secret or envelope serialized: %s", raw)
		}
	}
	if !a.TokenFilled || len(a.Credentials) != 1 {
		t.Fatalf("want 1 masked credential, got %+v", a.Credentials)
	}
	if c := a.Credentials[0]; c.Provider != "moonshotai" || c.Type != "api" || c.Hint != "sk-…WXYZ" {
		t.Errorf("summary = %+v", c)
	}
	if got := strings.Join(a.Providers, ","); got != "moonshotai" {
		t.Errorf("providers = %q", got)
	}
}

func mustList(t *testing.T, s *Service) []Account {
	t.Helper()
	l, err := s.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestCreate_RefusesWithoutArmedCipher(t *testing.T) {
	for name, svc := range map[string]*Service{
		"unarmed":   func() *Service { s, _ := newTestService(false); return s }(),
		"no cipher": newService(newFakeRepo(), nil, nil, nil),
	} {
		_, err := svc.Create(context.Background(), CreateRequest{Name: "x", ProviderID: "moonshotai", APIKey: testKey})
		if !errors.Is(err, ErrCipherRequired) {
			t.Errorf("%s: err = %v, want ErrCipherRequired", name, err)
		}
	}
}

func TestResolveSpawnAuth_RoundTripsBundle(t *testing.T) {
	svc, _ := newTestService(true)
	creds := map[string]any{
		"moonshotai":     map[string]any{"type": "api", "key": testKey},
		"github-copilot": map[string]any{"type": "oauth", "refresh": "r", "access": "a", "expires": 1},
	}
	a, err := svc.Create(context.Background(), CreateRequest{Name: "multi", Credentials: creds})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.ResolveSpawnAuth(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal([]byte(got), &back); err != nil {
		t.Fatalf("auth content not JSON: %v", err)
	}
	if len(back) != 2 || back["moonshotai"].(map[string]any)["key"] != testKey {
		t.Errorf("bundle did not round-trip: %v", back)
	}
	// oauth entry passes through opaque, no hint leaked.
	for _, c := range a.Credentials {
		if c.Provider == "github-copilot" && c.Hint != "" {
			t.Errorf("oauth entry should have no hint, got %q", c.Hint)
		}
	}
}

func TestResolveSpawnAuth_DisabledRefused(t *testing.T) {
	svc, _ := newTestService(true)
	off := false
	a, err := svc.Create(context.Background(), CreateRequest{Name: "off", ProviderID: "moonshotai", APIKey: testKey, Enabled: &off})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveSpawnAuth(context.Background(), a.ID); !errors.Is(err, ErrDisabled) {
		t.Errorf("err = %v, want ErrDisabled", err)
	}
	if err := svc.CheckUsable(context.Background(), a.ID); !errors.Is(err, ErrDisabled) {
		t.Errorf("CheckUsable err = %v, want ErrDisabled", err)
	}
}

func TestCreate_ValidatesShape(t *testing.T) {
	svc, _ := newTestService(true)
	cases := map[string]CreateRequest{
		"no credentials":       {Name: "a"},
		"api without key":      {Name: "b", Credentials: map[string]any{"moonshotai": map[string]any{"type": "api"}}},
		"entry not object":     {Name: "c", Credentials: map[string]any{"moonshotai": "sk-x"}},
		"missing type":         {Name: "d", Credentials: map[string]any{"moonshotai": map[string]any{"key": "k"}}},
		"bad provider id":      {Name: "e", Credentials: map[string]any{"../etc": map[string]any{"type": "api", "key": "k"}}},
		"key without provider": {Name: "f", APIKey: testKey},
		"both forms":           {Name: "g", ProviderID: "x", APIKey: "y", Credentials: map[string]any{"z": map[string]any{"type": "api", "key": "k"}}},
	}
	for name, req := range cases {
		_, err := svc.Create(context.Background(), req)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("%s: err = %v, want ErrInvalidCredentials", name, err)
			continue
		}
		if strings.Contains(err.Error(), testKey) {
			t.Errorf("%s: error echoes the secret: %v", name, err)
		}
	}
}

func TestCreate_DuplicateName(t *testing.T) {
	svc, _ := newTestService(true)
	req := CreateRequest{Name: "dup", ProviderID: "moonshotai", APIKey: testKey}
	if _, err := svc.Create(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), req); !errors.Is(err, ErrDuplicate) {
		t.Errorf("err = %v, want ErrDuplicate", err)
	}
}

func TestUpdate_ReplacesBundleOnlyWhenGiven(t *testing.T) {
	svc, repo := newTestService(true)
	a, err := svc.Create(context.Background(), CreateRequest{Name: "u", ProviderID: "moonshotai", APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	before := repo.rows[a.ID].credentialsEnc
	desc := "renamed"
	if _, err := svc.Update(context.Background(), a.ID, UpdateRequest{Description: &desc}); err != nil {
		t.Fatal(err)
	}
	if repo.rows[a.ID].credentialsEnc != before {
		t.Error("metadata-only update must keep the stored bundle")
	}
	up, err := svc.Update(context.Background(), a.ID, UpdateRequest{ProviderID: "kimi-for-coding", APIKey: "sk-kimi-abcdefghijkl1234"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(up.Providers, ",") != "kimi-for-coding" {
		t.Errorf("providers after replace = %v", up.Providers)
	}
}

func TestImportLocal_ReadsOnDiskAuth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(`{"moonshotai":{"type":"api","key":"`+testKey+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r := newFakeRepo()
	svc := newService(r, nil, fakeCipher{armed: true}, nil, WithLocalAuthPath(path))
	a, err := svc.ImportLocal(context.Background(), ImportLocalRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "local" || strings.Join(a.Providers, ",") != "moonshotai" {
		t.Errorf("imported = %+v", a)
	}
	if _, err := svc.ImportLocal(context.Background(), ImportLocalRequest{}); !errors.Is(err, ErrDuplicate) {
		t.Errorf("repeat import err = %v, want ErrDuplicate", err)
	}
}

func TestMaskSecret(t *testing.T) {
	for in, want := range map[string]string{
		"sk-abcdefghijWXYZ": "sk-…WXYZ",
		"short":             "••••",
		"abcdefghijklmnop":  "…mnop",
	} {
		if got := maskSecret(in); got != want {
			t.Errorf("maskSecret(%q) = %q, want %q", in, got, want)
		}
	}
}
