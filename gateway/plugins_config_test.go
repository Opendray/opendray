package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/plugin"
)

// ─── Fakes ──────────────────────────────────────────────────────────────────

// fakeKVStore implements the narrow configStore interface. Every row
// is stored as a json.RawMessage keyed by plugin+key.
type fakeKVStore struct {
	mu   sync.Mutex
	rows map[string]map[string]json.RawMessage
}

func newFakeKVStore() *fakeKVStore {
	return &fakeKVStore{rows: make(map[string]map[string]json.RawMessage)}
}

func (s *fakeKVStore) KVGet(_ context.Context, plugin, key string) (json.RawMessage, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.rows[plugin]; ok {
		if v, ok := p[key]; ok {
			return v, true, nil
		}
	}
	return nil, false, nil
}

func (s *fakeKVStore) KVSet(_ context.Context, plugin, key string, v json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rows[plugin]; !ok {
		s.rows[plugin] = make(map[string]json.RawMessage)
	}
	s.rows[plugin][key] = append(json.RawMessage(nil), v...)
	return nil
}

func (s *fakeKVStore) KVDelete(_ context.Context, plugin, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.rows[plugin]; ok {
		delete(p, key)
	}
	return nil
}

// fakeSecrets implements platformSecrets with plaintext storage —
// we're testing the handler's routing, not the crypto.
type fakeSecrets struct {
	mu   sync.Mutex
	rows map[string]map[string]string
}

func newFakeSecrets() *fakeSecrets {
	return &fakeSecrets{rows: make(map[string]map[string]string)}
}

func (s *fakeSecrets) PlatformSet(_ context.Context, plugin, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rows[plugin]; !ok {
		s.rows[plugin] = make(map[string]string)
	}
	s.rows[plugin][key] = value
	return nil
}

func (s *fakeSecrets) PlatformGet(_ context.Context, plugin, key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.rows[plugin]; ok {
		if v, ok := p[key]; ok {
			return v, true, nil
		}
	}
	return "", false, nil
}

func (s *fakeSecrets) PlatformDelete(_ context.Context, plugin, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.rows[plugin]; ok {
		delete(p, key)
	}
	return nil
}

// fakeKiller records every Kill call so tests can assert the sidecar
// restart fired after a PUT.
type fakeKiller struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeKiller) Kill(pluginName, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, pluginName)
	return nil
}

// ─── Test server ────────────────────────────────────────────────────────────

// buildConfigTestServer wires a Server whose config handlers use the
// fakes above. The bridgePluginsOverride is set so the schema resolver
// bypasses the real plugin.Runtime.
//
// We can't populate Server.secretAPI / Server.hostSupervisor from the
// test package (both need concrete types we don't want to stand up).
// Instead the handler uses its configSecrets() / configSupervisor()
// methods, which we override via small reflective-ish swaps — here,
// simpler: temporarily swap the methods by injecting test-friendly
// resolvers. We add test-only fields on Server that the handler reads
// first.
func buildConfigTestServer(t *testing.T, schema []plugin.ConfigField) (
	*Server, *fakeKVStore, *fakeSecrets, *fakeKiller,
) {
	t.Helper()
	kv := newFakeKVStore()
	sec := newFakeSecrets()
	killer := &fakeKiller{}
	s := &Server{
		router: chi.NewRouter(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		bridgePluginsOverride: func(name string) (plugin.Provider, bool) {
			if name != "test" {
				return plugin.Provider{}, false
			}
			return plugin.Provider{Name: "test", ConfigSchema: schema}, true
		},
	}
	s.configKVTestOverride = kv
	s.configSecretsTestOverride = sec
	s.configKillerTestOverride = killer
	return s, kv, sec, killer
}

func doConfigRequest(h http.HandlerFunc, method, target, body, name string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", name)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

// ─── GET tests ──────────────────────────────────────────────────────────────

func TestConfigGet_Empty(t *testing.T) {
	schema := []plugin.ConfigField{
		{Key: "host", Label: "Host", Type: "string"},
		{Key: "password", Label: "Password", Type: "secret"},
	}
	s, _, _, _ := buildConfigTestServer(t, schema)
	rr := doConfigRequest(s.pluginsConfigGet, http.MethodGet, "/cfg", "", "test")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var got struct {
		Schema []plugin.ConfigField `json:"schema"`
		Values map[string]string    `json:"values"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Schema) != 2 {
		t.Errorf("schema len = %d, want 2", len(got.Schema))
	}
	if got.Values["host"] != "" {
		t.Errorf("host before any PUT: want empty, got %q", got.Values["host"])
	}
	if got.Values["password"] != "" {
		t.Errorf("password before any PUT: want empty, got %q", got.Values["password"])
	}
}

func TestConfigGet_SecretMasked(t *testing.T) {
	schema := []plugin.ConfigField{
		{Key: "password", Label: "Password", Type: "secret"},
	}
	s, _, sec, _ := buildConfigTestServer(t, schema)
	_ = sec.PlatformSet(context.Background(), "test", "__config.password", "hunter2")

	rr := doConfigRequest(s.pluginsConfigGet, http.MethodGet, "/cfg", "", "test")
	var got struct {
		Values map[string]string `json:"values"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got.Values["password"] != secretSentinel {
		t.Errorf("password: want %q, got %q", secretSentinel, got.Values["password"])
	}
	// Sentinel must never be the raw value.
	if strings.Contains(rr.Body.String(), "hunter2") {
		t.Errorf("raw secret leaked in response: %s", rr.Body.String())
	}
}

func TestConfigGet_PluginNotFound(t *testing.T) {
	s, _, _, _ := buildConfigTestServer(t, nil)
	rr := doConfigRequest(s.pluginsConfigGet, http.MethodGet, "/cfg", "", "missing")
	if rr.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d; body=%s", rr.Code, rr.Body.String())
	}
}

// ─── PUT tests ──────────────────────────────────────────────────────────────

func TestConfigPut_HappyPath(t *testing.T) {
	schema := []plugin.ConfigField{
		{Key: "host", Label: "Host", Type: "string"},
		{Key: "port", Label: "Port", Type: "number"},
		{Key: "enabled", Label: "Enabled", Type: "bool"},
		{Key: "password", Label: "Password", Type: "secret"},
	}
	s, kv, sec, killer := buildConfigTestServer(t, schema)

	body := `{"values":{"host":"db.example.com","port":5432,"enabled":true,"password":"hunter2"}}`
	rr := doConfigRequest(s.pluginsConfigPut, http.MethodPut, "/cfg", body, "test")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}

	// KV store: three non-secret fields persisted as JSON strings.
	tests := map[string]string{
		"__config.host":    `"db.example.com"`,
		"__config.port":    `"5432"`,
		"__config.enabled": `"true"`,
	}
	for k, want := range tests {
		got := string(kv.rows["test"][k])
		if got != want {
			t.Errorf("kv[%s] = %s, want %s", k, got, want)
		}
	}
	// Secret store: password persisted plaintext (fake).
	if sec.rows["test"]["__config.password"] != "hunter2" {
		t.Errorf("secret password = %q, want hunter2", sec.rows["test"]["__config.password"])
	}
	// Supervisor kill fired exactly once for this plugin.
	if got := killer.calls; len(got) != 1 || got[0] != "test" {
		t.Errorf("killer.calls = %v, want [test]", got)
	}
}

func TestConfigPut_SecretSentinelDoesNotOverwrite(t *testing.T) {
	schema := []plugin.ConfigField{
		{Key: "password", Label: "Password", Type: "secret"},
	}
	s, _, sec, _ := buildConfigTestServer(t, schema)
	_ = sec.PlatformSet(context.Background(), "test", "__config.password", "original")

	body := `{"values":{"password":"` + secretSentinel + `"}}`
	rr := doConfigRequest(s.pluginsConfigPut, http.MethodPut, "/cfg", body, "test")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	if got := sec.rows["test"]["__config.password"]; got != "original" {
		t.Errorf("sentinel should not overwrite; got %q", got)
	}
}

func TestConfigPut_RejectsUnknownKey(t *testing.T) {
	schema := []plugin.ConfigField{
		{Key: "host", Label: "Host", Type: "string"},
	}
	s, _, _, _ := buildConfigTestServer(t, schema)
	body := `{"values":{"nonsense":"x"}}`
	rr := doConfigRequest(s.pluginsConfigPut, http.MethodPut, "/cfg", body, "test")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d; body=%s", rr.Code, rr.Body.String())
	}
}

func TestConfigPut_EmptySchemaRejected(t *testing.T) {
	s, _, _, _ := buildConfigTestServer(t, nil)
	s.bridgePluginsOverride = func(name string) (plugin.Provider, bool) {
		return plugin.Provider{Name: "test"}, true // no ConfigSchema
	}
	rr := doConfigRequest(s.pluginsConfigPut, http.MethodPut, "/cfg", `{"values":{}}`, "test")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d; body=%s", rr.Code, rr.Body.String())
	}
}
