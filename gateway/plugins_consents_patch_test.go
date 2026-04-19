package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
)

// memConsentStore is an in-memory consentStore for tests.
type memConsentStore struct {
	mu    sync.Mutex
	rows  map[string]store.PluginConsent
}

func (m *memConsentStore) GetConsent(_ context.Context, name string) (store.PluginConsent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.rows[name]
	if !ok {
		return store.PluginConsent{}, store.ErrConsentNotFound
	}
	return c, nil
}

func (m *memConsentStore) UpdateConsentPerms(_ context.Context, name string, perms json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.rows[name]
	if !ok {
		return store.ErrConsentNotFound
	}
	c.PermsJSON = perms
	m.rows[name] = c
	return nil
}

func (m *memConsentStore) DeleteConsent(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, name)
	return nil
}

// recordingInvalidator captures InvalidateConsent calls.
type recordingInvalidator struct {
	mu    sync.Mutex
	calls [][2]string
}

func (r *recordingInvalidator) InvalidateConsent(plug, cap string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, [2]string{plug, cap})
}

// setupPatchTestServer wires a minimal Server with consentStoreOverride +
// consentBridgeOverride and returns an httptest.Server.
func setupPatchTestServer(t *testing.T, initial map[string]store.PluginConsent) (*httptest.Server, *memConsentStore, *recordingInvalidator) {
	t.Helper()
	store := &memConsentStore{rows: initial}
	inv := &recordingInvalidator{}

	srv := &Server{
		consentStoreOverride:  store,
		consentBridgeOverride: inv,
		logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	// Wire a fresh chi router with just the one route.
	r := chi.NewRouter()
	r.Patch("/api/plugins/{name}/consents", srv.pluginsConsentsPatch)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts, store, inv
}

func TestConsentPatch_HappyPath_MergesFields(t *testing.T) {
	t.Parallel()
	// Plugin starts with storage:true, fs:{read:[abc]}.
	initialPerms := plugin.PermissionsV1{
		Storage: true,
		Fs:      json.RawMessage(`{"read":["/abc/**"]}`),
	}
	initialJSON, _ := json.Marshal(initialPerms)
	ts, store, inv := setupPatchTestServer(t, map[string]store.PluginConsent{
		"test": {PluginName: "test", PermsJSON: initialJSON},
	})

	// Patch: change fs.read allowlist; DO NOT touch storage.
	patchBody := `{"fs":{"read":["/xyz/**"]}}`
	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPatch,
		ts.URL+"/api/plugins/test/consents", patchBody))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}

	// Assert stored perms: fs rewritten, storage untouched.
	final := store.rows["test"]
	var got plugin.PermissionsV1
	if err := json.Unmarshal(final.PermsJSON, &got); err != nil {
		t.Fatalf("unmarshal stored: %v", err)
	}
	if !got.Storage {
		t.Error("storage was cleared, want preserved")
	}
	if string(got.Fs) != `{"read":["/xyz/**"]}` {
		t.Errorf("fs=%s, want new value", got.Fs)
	}

	// Assert invalidate called for the touched cap only.
	if len(inv.calls) != 1 || inv.calls[0][1] != "fs" {
		t.Errorf("invalidate calls=%v, want [[test fs]]", inv.calls)
	}
}

func TestConsentPatch_UnknownCapRejected(t *testing.T) {
	t.Parallel()
	ts, _, _ := setupPatchTestServer(t, map[string]store.PluginConsent{
		"test": {PluginName: "test", PermsJSON: json.RawMessage(`{}`)},
	})

	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPatch,
		ts.URL+"/api/plugins/test/consents", `{"nope":true}`))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
}

func TestConsentPatch_NoConsentRow_Returns404(t *testing.T) {
	t.Parallel()
	ts, _, _ := setupPatchTestServer(t, map[string]store.PluginConsent{})

	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPatch,
		ts.URL+"/api/plugins/ghost/consents", `{"storage":true}`))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d, want 404", resp.StatusCode)
	}
}

func TestConsentPatch_EmptyBody_Returns400(t *testing.T) {
	t.Parallel()
	ts, _, _ := setupPatchTestServer(t, map[string]store.PluginConsent{
		"test": {PluginName: "test", PermsJSON: json.RawMessage(`{}`)},
	})

	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPatch,
		ts.URL+"/api/plugins/test/consents", ``))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
}

func TestConsentPatch_MultipleCapsFireMultipleInvalidates(t *testing.T) {
	t.Parallel()
	initial, _ := json.Marshal(plugin.PermissionsV1{Storage: true})
	ts, _, inv := setupPatchTestServer(t, map[string]store.PluginConsent{
		"test": {PluginName: "test", PermsJSON: initial},
	})

	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPatch,
		ts.URL+"/api/plugins/test/consents",
		`{"storage":false,"events":["session.idle"],"http":["https://api.example.com/*"]}`))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	if len(inv.calls) != 3 {
		t.Errorf("expected 3 invalidate calls, got %d: %v", len(inv.calls), inv.calls)
	}
}

func TestConsentPatch_CorruptStoredRow_Returns500(t *testing.T) {
	t.Parallel()
	ts, _, _ := setupPatchTestServer(t, map[string]store.PluginConsent{
		"test": {PluginName: "test", PermsJSON: json.RawMessage(`not json`)},
	})

	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPatch,
		ts.URL+"/api/plugins/test/consents", `{"storage":true}`))
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status=%d, want 500", resp.StatusCode)
	}
}

func mustReq(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

// compile-time assertion: memConsentStore satisfies consentStore.
var _ consentStore = (*memConsentStore)(nil)

// Same for invalidator.
var _ consentInvalidator = (*recordingInvalidator)(nil)

// silence unused-import lint when errors pkg isn't directly used in the
// file (some tests import it transitively).
var _ = errors.New
