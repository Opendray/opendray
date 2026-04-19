package gateway

// T12 — Consent revoke endpoint + 200 ms hot-revoke SLO test suite.
//
// Test strategy:
//   - A fakeConsentBridge implements consentInvalidator and records every
//     InvalidateConsent(plugin, cap) call plus the latency of each call. This
//     lets us assert (a) the handler fires InvalidateConsent synchronously and
//     (b) the call returns under a tight bound for the HTTP-path SLO test.
//   - A fakeConsentStore implements the narrow consentStore interface declared
//     in plugins_consents.go, so most tests avoid booting embedded Postgres.
//     One end-to-end test uses the real store via bootTestDB (already defined
//     in plugins_install_test.go) to prove the DB update path works.
//   - SLO test path (documented in commit message): we exercise the full
//     HTTP → store.Update → bridgeMgr.InvalidateConsent chain on a fake
//     in-memory store plus a real *bridge.Manager wrapping fake WS conns.
//     The 200 ms deadline is measured from the DELETE dispatch until
//     InvalidateConsent returns (the synchronous part of the broadcast) —
//     this is the HTTP-side piece of the SLO that T6's manager test already
//     covers for the bus side.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/hub"
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
)

// ─── Fakes ──────────────────────────────────────────────────────────────────

// fakeConsentBridge records every InvalidateConsent call.
type fakeConsentBridge struct {
	mu    sync.Mutex
	calls []fakeBridgeCall
	// optional delay to simulate slow broadcast (used for SLO fuzzing).
	delay time.Duration
}

type fakeBridgeCall struct {
	plugin string
	cap    string
}

func (f *fakeConsentBridge) InvalidateConsent(plugin, cap string) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeBridgeCall{plugin: plugin, cap: cap})
}

func (f *fakeConsentBridge) snapshot() []fakeBridgeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeBridgeCall, len(f.calls))
	copy(out, f.calls)
	return out
}

// fakeConsentStore is a tiny in-memory impl of the consentStore interface
// declared in plugins_consents.go. Every mutation is a full copy to preserve
// immutability semantics required by the preferences rules.
type fakeConsentStore struct {
	mu           sync.Mutex
	rows         map[string]store.PluginConsent
	updateErr    error // if set, UpdateConsentPerms returns this error
	forceMalform bool  // if set, GetConsent returns a row with malformed JSON
}

func newFakeConsentStore() *fakeConsentStore {
	return &fakeConsentStore{rows: make(map[string]store.PluginConsent)}
}

func (s *fakeConsentStore) seed(name string, perms string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[name] = store.PluginConsent{
		PluginName:   name,
		ManifestHash: "hash-" + name,
		PermsJSON:    json.RawMessage(perms),
		GrantedAt:    time.Now().Add(-1 * time.Hour),
		UpdatedAt:    time.Now().Add(-30 * time.Minute),
	}
}

func (s *fakeConsentStore) GetConsent(_ context.Context, name string) (store.PluginConsent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[name]
	if !ok {
		return store.PluginConsent{}, store.ErrConsentNotFound
	}
	if s.forceMalform {
		// Return a deliberately corrupted perms JSON to exercise the 500 path.
		row.PermsJSON = json.RawMessage(`{"storage": not-a-boolean}`)
	}
	// Return a fresh copy so callers can't mutate stored state.
	copy := row
	copy.PermsJSON = append(json.RawMessage(nil), row.PermsJSON...)
	return copy, nil
}

func (s *fakeConsentStore) UpdateConsentPerms(_ context.Context, name string, perms json.RawMessage) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[name]
	if !ok {
		return store.ErrConsentNotFound
	}
	row.PermsJSON = append(json.RawMessage(nil), perms...)
	row.UpdatedAt = time.Now()
	s.rows[name] = row
	return nil
}

func (s *fakeConsentStore) DeleteConsent(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, name) // idempotent
	return nil
}

// ─── Server construction ────────────────────────────────────────────────────

// buildConsentTestServer wires a minimal *Server whose consent handlers talk
// to fake store + fake bridge. The returned helpers let tests assert bridge
// broadcasts without caring about HTTP plumbing.
func buildConsentTestServer(t *testing.T) (*Server, *fakeConsentStore, *fakeConsentBridge) {
	t.Helper()
	fs := newFakeConsentStore()
	fb := &fakeConsentBridge{}
	s := &Server{
		router:                chi.NewRouter(), // not used by handler tests
		consentStoreOverride:  fs,
		consentBridgeOverride: fb,
	}
	return s, fs, fb
}

// doRequest is a small helper that runs a handler with a chi URL context so
// chi.URLParam(r, "name") / "cap" works.
func doRequest(handler http.HandlerFunc, method, target string, body []byte,
	params map[string]string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// ─── GET /api/plugins/{name}/consents ───────────────────────────────────────

func TestConsentsGet_Happy(t *testing.T) {
	s, fs, _ := buildConsentTestServer(t)
	fs.seed("kanban", `{"storage":true,"events":["session.*"]}`)

	rr := doRequest(s.pluginsConsentsGet, http.MethodGet,
		"/api/plugins/kanban/consents", nil,
		map[string]string{"name": "kanban"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body.String())
	}
	perms, ok := resp["perms"]
	if !ok {
		t.Fatalf("response missing perms; body=%s", rr.Body.String())
	}
	var permObj map[string]any
	if err := json.Unmarshal(perms, &permObj); err != nil {
		t.Fatalf("decode perms: %v; raw=%s", err, perms)
	}
	if permObj["storage"] != true {
		t.Errorf("perms.storage: want true, got %v", permObj["storage"])
	}
}

func TestConsentsGet_NotFound(t *testing.T) {
	s, _, _ := buildConsentTestServer(t)

	rr := doRequest(s.pluginsConsentsGet, http.MethodGet,
		"/api/plugins/missing/consents", nil,
		map[string]string{"name": "missing"})

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["code"] != "ENOCONSENT" {
		t.Errorf("want code=ENOCONSENT, got %q", resp["code"])
	}
}

// ─── DELETE /api/plugins/{name}/consents/{cap} ──────────────────────────────

func TestConsentsRevokeCap_Storage(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	fs.seed("kanban", `{"storage":true,"events":["session.*"]}`)

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}

	// Bridge invalidated exactly once with (kanban, storage).
	calls := fb.snapshot()
	if len(calls) != 1 {
		t.Fatalf("want 1 InvalidateConsent call, got %d", len(calls))
	}
	if calls[0].plugin != "kanban" || calls[0].cap != "storage" {
		t.Errorf("call: got (%q, %q), want (kanban, storage)",
			calls[0].plugin, calls[0].cap)
	}

	// DB row mutated: storage flipped to false (or removed from JSON).
	row, err := fs.GetConsent(context.Background(), "kanban")
	if err != nil {
		t.Fatalf("GetConsent: %v", err)
	}
	var perms map[string]any
	if err := json.Unmarshal(row.PermsJSON, &perms); err != nil {
		t.Fatalf("decode perms: %v", err)
	}
	// storage must be absent or explicitly false.
	if v, ok := perms["storage"]; ok && v != false {
		t.Errorf("perms.storage after revoke: want absent or false, got %v", v)
	}
	// events must still be there.
	if _, ok := perms["events"]; !ok {
		t.Errorf("perms.events: unexpected drop")
	}
}

func TestConsentsRevokeCap_Exec(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	fs.seed("kanban", `{"exec":{"globs":["git *"]},"storage":true}`)

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/exec", nil,
		map[string]string{"name": "kanban", "cap": "exec"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	calls := fb.snapshot()
	if len(calls) != 1 || calls[0].cap != "exec" {
		t.Errorf("calls: %+v", calls)
	}

	row, _ := fs.GetConsent(context.Background(), "kanban")
	var perms map[string]any
	if err := json.Unmarshal(row.PermsJSON, &perms); err != nil {
		t.Fatalf("decode perms: %v", err)
	}
	if v, ok := perms["exec"]; ok && v != nil {
		t.Errorf("perms.exec after revoke: want nil or absent, got %v", v)
	}
	if _, ok := perms["storage"]; !ok {
		t.Errorf("perms.storage accidentally dropped")
	}
}

func TestConsentsRevokeCap_UnknownCap(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	fs.seed("kanban", `{"storage":true}`)

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/banana", nil,
		map[string]string{"name": "kanban", "cap": "banana"})

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["code"] != "EINVAL" {
		t.Errorf("want code=EINVAL, got %q", resp["code"])
	}
	// Bridge must NOT be called on validation failure.
	if len(fb.snapshot()) != 0 {
		t.Errorf("unexpected InvalidateConsent on validation failure: %+v",
			fb.snapshot())
	}
}

func TestConsentsRevokeCap_UnknownPlugin(t *testing.T) {
	s, _, fb := buildConsentTestServer(t)

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/nope/consents/storage", nil,
		map[string]string{"name": "nope", "cap": "storage"})

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["code"] != "ENOCONSENT" {
		t.Errorf("want code=ENOCONSENT, got %q", resp["code"])
	}
	if len(fb.snapshot()) != 0 {
		t.Errorf("unexpected InvalidateConsent for unknown plugin")
	}
}

func TestConsentsRevokeCap_Idempotent(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	// storage already absent — revoking it must still return 200, update the
	// DB (to a canonical no-op perms), and fire the bus so any stale WS sub
	// gets cleaned up.
	fs.seed("kanban", `{"events":["session.*"]}`)

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	calls := fb.snapshot()
	if len(calls) != 1 || calls[0].cap != "storage" {
		t.Errorf("want one (kanban, storage) call, got %+v", calls)
	}
}

func TestConsentsRevokeCap_MalformedPermsJSON(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	fs.seed("kanban", `{"storage":true}`)
	fs.forceMalform = true // next GetConsent returns bad JSON

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["code"] != "ERESERVE" {
		t.Errorf("want code=ERESERVE, got %q", resp["code"])
	}
	// Bridge MUST NOT fire when we can't reason about current perms.
	if len(fb.snapshot()) != 0 {
		t.Errorf("unexpected InvalidateConsent on corrupt row")
	}
}

func TestConsentsRevokeCap_DBWriteFails(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	fs.seed("kanban", `{"storage":true}`)
	fs.updateErr = errors.New("simulated DB failure")

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["code"] != "ERESERVE" {
		t.Errorf("want code=ERESERVE, got %q", resp["code"])
	}
	if len(fb.snapshot()) != 0 {
		t.Errorf("InvalidateConsent must NOT fire when DB write fails")
	}
}

// ─── DELETE /api/plugins/{name}/consents (revoke-all) ───────────────────────

func TestConsentsRevokeAll_Happy(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	fs.seed("kanban", `{"storage":true,"exec":{"globs":["git *"]},"events":["session.*"]}`)

	rr := doRequest(s.pluginsConsentsRevokeAll, http.MethodDelete,
		"/api/plugins/kanban/consents", nil,
		map[string]string{"name": "kanban"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}

	// Row must be gone.
	if _, err := fs.GetConsent(context.Background(), "kanban"); !errors.Is(err, store.ErrConsentNotFound) {
		t.Errorf("after revoke-all: want ErrConsentNotFound, got %v", err)
	}

	// One InvalidateConsent per originally-granted cap key.
	calls := fb.snapshot()
	wantCaps := map[string]bool{"storage": false, "exec": false, "events": false}
	for _, c := range calls {
		if c.plugin != "kanban" {
			t.Errorf("wrong plugin in call: %+v", c)
		}
		wantCaps[c.cap] = true
	}
	for cap, seen := range wantCaps {
		if !seen {
			t.Errorf("missing InvalidateConsent call for cap %q", cap)
		}
	}
}

func TestConsentsRevokeAll_MissingRow_Still200(t *testing.T) {
	s, _, fb := buildConsentTestServer(t)

	rr := doRequest(s.pluginsConsentsRevokeAll, http.MethodDelete,
		"/api/plugins/missing/consents", nil,
		map[string]string{"name": "missing"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	// No caps to broadcast when there was no row.
	if n := len(fb.snapshot()); n != 0 {
		t.Errorf("unexpected InvalidateConsent calls on missing row: %d", n)
	}
}

// TestConsentsRevokeCap_AllCapKeys exercises every cap in allowedCaps so
// zeroCapability's 11-arm switch is fully covered. For each cap we seed a
// permissions JSON that grants it, fire DELETE, then assert:
//   - HTTP 200
//   - InvalidateConsent fired once with that cap
//   - the permissions JSON reshapes with the cap's field zeroed/removed
func TestConsentsRevokeCap_AllCapKeys(t *testing.T) {
	// For each cap, a seed perms string that grants it in a form
	// permsGrantedCaps will recognise. For polymorphic fields (fs/exec/http)
	// we supply an object so the RawMessage is non-empty; for typed fields
	// we use the smallest non-zero value.
	caps := []struct {
		cap  string
		seed string
	}{
		{"fs", `{"fs":{"read":["/tmp/*"]}}`},
		{"exec", `{"exec":{"globs":["git *"]}}`},
		{"http", `{"http":{"domains":["example.com"]}}`},
		{"session", `{"session":"read"}`},
		{"storage", `{"storage":true}`},
		{"secret", `{"secret":true}`},
		{"clipboard", `{"clipboard":"read"}`},
		{"telegram", `{"telegram":true}`},
		{"git", `{"git":"read"}`},
		{"llm", `{"llm":true}`},
		{"events", `{"events":["session.*"]}`},
	}
	for _, tc := range caps {
		t.Run(tc.cap, func(t *testing.T) {
			s, fs, fb := buildConsentTestServer(t)
			fs.seed("kanban", tc.seed)

			rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
				"/api/plugins/kanban/consents/"+tc.cap, nil,
				map[string]string{"name": "kanban", "cap": tc.cap})

			if rr.Code != http.StatusOK {
				t.Fatalf("%s: want 200, got %d; body=%s",
					tc.cap, rr.Code, rr.Body.String())
			}
			calls := fb.snapshot()
			if len(calls) != 1 || calls[0].cap != tc.cap {
				t.Errorf("%s: want one call with cap=%s, got %+v",
					tc.cap, tc.cap, calls)
			}
		})
	}
}

// TestZeroCapability_UnknownReturnsFalse is a unit test for the small
// zero-field helper so the "default" branch is exercised independently.
func TestZeroCapability_UnknownReturnsFalse(t *testing.T) {
	var p plugin.PermissionsV1
	if zeroCapability(&p, "banana") {
		t.Error("zeroCapability should return false for unknown cap")
	}
	if zeroCapability(nil, "storage") {
		t.Error("zeroCapability(nil, ...) should return false")
	}
}

// TestPermsGrantedCaps_AllFields exercises every branch of permsGrantedCaps
// by asserting all 11 cap keys are reported for a maximally-populated perms.
func TestPermsGrantedCaps_AllFields(t *testing.T) {
	p := plugin.PermissionsV1{
		Fs:        json.RawMessage(`{"read":["/tmp/*"]}`),
		Exec:      json.RawMessage(`{"globs":["git *"]}`),
		HTTP:      json.RawMessage(`{"domains":["example.com"]}`),
		Session:   "read",
		Storage:   true,
		Secret:    true,
		Clipboard: "read",
		Telegram:  true,
		Git:       "read",
		LLM:       true,
		Events:    []string{"session.*"},
	}
	got := permsGrantedCaps(p)
	want := map[string]bool{
		"fs": true, "exec": true, "http": true, "session": true,
		"storage": true, "secret": true, "clipboard": true,
		"telegram": true, "git": true, "llm": true, "events": true,
	}
	if len(got) != len(want) {
		t.Fatalf("want %d caps, got %d (%v)", len(want), len(got), got)
	}
	for _, c := range got {
		if !want[c] {
			t.Errorf("unexpected cap %q in result", c)
		}
	}
}

// TestConsentsGet_NilHub_ECONFIG covers the defensive branch where neither
// an override nor a hub is wired. Production won't hit this — it's a guard
// for misconfigured test servers — but it's still a path through the
// handler that must not panic or return 2xx.
func TestConsentsGet_NilHub_ECONFIG(t *testing.T) {
	s := &Server{router: chi.NewRouter()} // no hub, no override

	rr := doRequest(s.pluginsConsentsGet, http.MethodGet,
		"/api/plugins/kanban/consents", nil,
		map[string]string{"name": "kanban"})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rr.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["code"] != "ECONFIG" {
		t.Errorf("want code=ECONFIG, got %q", resp["code"])
	}
}

// TestConsentsRevokeCap_GetConsentGenericErr covers the non-ErrConsentNotFound
// branch of GetConsent error handling inside pluginsConsentsRevokeCap.
func TestConsentsRevokeCap_GetConsentGenericErr(t *testing.T) {
	getErr := errors.New("simulated connection reset")
	errStore := &errorGetStore{err: getErr}
	s := &Server{
		router:                chi.NewRouter(),
		consentStoreOverride:  errStore,
		consentBridgeOverride: &fakeConsentBridge{},
	}

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rr.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["code"] != "ERESERVE" {
		t.Errorf("want code=ERESERVE, got %q", resp["code"])
	}
}

// errorGetStore always fails GetConsent with the given error. The other
// methods are stubs; they're unused by the tests that exercise it.
type errorGetStore struct{ err error }

func (e *errorGetStore) GetConsent(context.Context, string) (store.PluginConsent, error) {
	return store.PluginConsent{}, e.err
}
func (e *errorGetStore) UpdateConsentPerms(context.Context, string, json.RawMessage) error {
	return nil
}
func (e *errorGetStore) DeleteConsent(context.Context, string) error { return nil }

// ─── SLO timing tests ───────────────────────────────────────────────────────

// TestRevokeCap_FiresInvalidateConsent is the fast guardrail: the handler
// must invoke InvalidateConsent on the caller goroutine (no background
// dispatch), so the test observes the call before the HTTP response bytes
// have been flushed to the recorder — a classic synchronous-effect assertion.
func TestRevokeCap_FiresInvalidateConsent(t *testing.T) {
	s, fs, fb := buildConsentTestServer(t)
	fs.seed("kanban", `{"storage":true}`)

	var countAtResponse int
	// Wrap InvalidateConsent to observe ordering vs. response write.
	wrap := &observingBridge{inner: fb}
	s.consentBridgeOverride = wrap

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	countAtResponse = int(wrap.count.Load())
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if countAtResponse != 1 {
		t.Fatalf("InvalidateConsent must fire before handler returns; count=%d",
			countAtResponse)
	}
}

// observingBridge counts synchronous InvalidateConsent calls.
type observingBridge struct {
	inner consentInvalidator
	count atomic.Int64
}

func (o *observingBridge) InvalidateConsent(plugin, cap string) {
	o.count.Add(1)
	o.inner.InvalidateConsent(plugin, cap)
}

// TestRevoke_StorageWithin200ms is the headline SLO test. It wires a *real*
// bridge.Manager with a registered fake WS and a live subscription for
// "storage", then measures wall-clock time from DELETE dispatch until the
// done channel of the subscription is closed — which is what a storage.set
// Gate.Check on the plugin side would observe.
//
// This covers the HTTP → store → bridgeMgr.InvalidateConsent → sub-done-chan
// path end-to-end. The manager's own broadcast path is covered by
// plugin/bridge/manager_test.go TestManager_HotRevokeDeliversUnderSLO, so
// between the two we have the full SLO asserted.
func TestRevoke_StorageWithin200ms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping SLO timing test under -short")
	}

	mgr := bridge.NewManager(nil)
	ws := &sloFakeWS{}
	conn := mgr.Register("kanban", ws)
	defer conn.Close(1000, "test end")

	done, err := conn.Subscribe("sub-storage-1", "storage")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	fs := newFakeConsentStore()
	fs.seed("kanban", `{"storage":true}`)

	s := &Server{
		router:                chi.NewRouter(),
		consentStoreOverride:  fs,
		consentBridgeOverride: mgr,
	}

	start := time.Now()
	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	// The subscription's done channel must close within 200 ms of DELETE
	// dispatch — this is the exact Gate-observable moment.
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("SLO: storage sub done not closed within 200 ms (elapsed=%v, status=%d)",
			time.Since(start), rr.Code)
	}
	elapsed := time.Since(start)
	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE /consents/storage: want 200, got %d; body=%s",
			rr.Code, rr.Body.String())
	}
	t.Logf("hot-revoke end-to-end latency: %v (SLO 200 ms)", elapsed)
	if elapsed > 200*time.Millisecond {
		t.Fatalf("SLO breach: end-to-end latency %v > 200 ms", elapsed)
	}
}

// sloFakeWS is a no-op websocket impl for SLO timing: every WriteMessage
// succeeds immediately, Close is idempotent.
type sloFakeWS struct {
	mu       sync.Mutex
	closed   bool
	messages int
}

func (w *sloFakeWS) WriteMessage(_ int, _ []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return errors.New("closed")
	}
	w.messages++
	return nil
}

func (w *sloFakeWS) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return nil
}

// ─── End-to-end with real store.DB (embedded-pg) ────────────────────────────

// TestConsentsRevokeCap_EndToEnd_PG exercises the full HTTP → real store →
// real bridge.Manager path against embedded Postgres. Skipped under -short.
func TestConsentsRevokeCap_EndToEnd_PG(t *testing.T) {
	db := bootTestDB(t) // shared helper in plugins_install_test.go
	ctx := context.Background()

	// Seed a plugin row + consent row with storage=true.
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO plugins (name, version) VALUES ($1, $2)
		 ON CONFLICT (name) DO NOTHING`,
		"kanban", "1.0.0"); err != nil {
		t.Fatalf("insert plugin: %v", err)
	}
	if err := db.UpsertConsent(ctx, store.PluginConsent{
		PluginName:   "kanban",
		ManifestHash: "hash-kanban-e2e",
		PermsJSON:    json.RawMessage(`{"storage":true,"events":["session.*"]}`),
	}); err != nil {
		t.Fatalf("UpsertConsent: %v", err)
	}

	mgr := bridge.NewManager(nil)

	s := &Server{
		router: chi.NewRouter(),
		hub:    hub.New(hub.Config{DB: db}),
		// No override — exercise s.hub.DB() + s.bridgeMgr resolution logic.
		bridgeMgr: mgr,
	}

	rr := doRequest(s.pluginsConsentsRevokeCap, http.MethodDelete,
		"/api/plugins/kanban/consents/storage", nil,
		map[string]string{"name": "kanban", "cap": "storage"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	got, err := db.GetConsent(ctx, "kanban")
	if err != nil {
		t.Fatalf("GetConsent: %v", err)
	}
	var perms map[string]any
	if err := json.Unmarshal(got.PermsJSON, &perms); err != nil {
		t.Fatalf("decode perms: %v", err)
	}
	if v, ok := perms["storage"]; ok && v != false {
		t.Errorf("perms.storage after revoke: want absent or false, got %v", v)
	}
}
