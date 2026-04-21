package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// ─────────────────────────────────────────────
// Mocks
// ─────────────────────────────────────────────

// mockSecretStore is an in-memory SecretStore for bridge-layer unit tests.
type mockSecretStore struct {
	mu       sync.Mutex
	secrets  map[string]mockSecretRow
	wrapped  map[string]mockKEKRow
	getErr   error // if set, GetWrappedDEK returns this
	setErr   error // if set, SecretSet returns this
}

type mockSecretRow struct {
	ct    []byte
	nonce []byte
}

type mockKEKRow struct {
	wrapped []byte
	kid     string
}

func newMockSecretStore() *mockSecretStore {
	return &mockSecretStore{
		secrets: make(map[string]mockSecretRow),
		wrapped: make(map[string]mockKEKRow),
	}
}

func (m *mockSecretStore) key(plugin, k string) string { return plugin + "/" + k }

func (m *mockSecretStore) SecretGet(_ context.Context, plugin, key string) ([]byte, []byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.secrets[m.key(plugin, key)]
	if !ok {
		return nil, nil, false, nil
	}
	return r.ct, r.nonce, true, nil
}

func (m *mockSecretStore) SecretSet(_ context.Context, plugin, key string, ct, nonce []byte) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[m.key(plugin, key)] = mockSecretRow{ct: ct, nonce: nonce}
	return nil
}

func (m *mockSecretStore) SecretDelete(_ context.Context, plugin, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.secrets, m.key(plugin, key))
	return nil
}

func (m *mockSecretStore) SecretList(_ context.Context, plugin string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pfx := plugin + "/"
	var out []string
	for k := range m.secrets {
		if strings.HasPrefix(k, pfx) {
			out = append(out, strings.TrimPrefix(k, pfx))
		}
	}
	// deterministic order
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i] > out[j] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (m *mockSecretStore) EnsureKEKRow(_ context.Context, plugin string, wrapped []byte, kid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wrapped[plugin] = mockKEKRow{wrapped: wrapped, kid: kid}
	return nil
}

func (m *mockSecretStore) GetWrappedDEK(_ context.Context, plugin string) ([]byte, string, error) {
	if m.getErr != nil {
		return nil, "", m.getErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.wrapped[plugin]
	if !ok {
		return nil, "", WrappedDEKNotFound
	}
	return r.wrapped, r.kid, nil
}

// mockKEK returns a fixed 32-byte KEK.
type mockKEK struct {
	material []byte // 32 bytes
	err      error
}

func (m *mockKEK) DeriveKEK(_ context.Context, _ string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]byte, len(m.material))
	copy(out, m.material)
	return out, nil
}

func newMockKEK() *mockKEK {
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(i)
	}
	return &mockKEK{material: kek}
}

// secretGrantedGate returns a Gate that grants the "secret" cap.
func secretGrantedGate() *Gate {
	return NewGate(&secretConsent{granted: true}, nil, nil)
}

// secretDeniedGate returns a Gate that denies secret.
func secretDeniedGate() *Gate {
	return NewGate(&secretConsent{granted: false, found: true}, nil, nil)
}

type secretConsent struct {
	granted bool
	found   bool
}

func (c *secretConsent) Load(_ context.Context, _ string) ([]byte, bool, error) {
	if c.granted {
		return []byte(`{"secret":true}`), true, nil
	}
	if c.found {
		return []byte(`{"secret":false}`), true, nil
	}
	return nil, false, nil
}

// newSecretAPIWithMocks constructs a SecretAPI with default mocks.
func newSecretAPIWithMocks(t *testing.T, gate *Gate) (*SecretAPI, *mockSecretStore, *mockKEK) {
	t.Helper()
	store := newMockSecretStore()
	kek := newMockKEK()
	api := NewSecretAPI(SecretConfig{
		Store: store,
		Gate:  gate,
		KEK:   kek,
	})
	return api, store, kek
}

// argsArr marshals values into the standard [v0, v1, ...] arg array.
func argsArr(values ...any) json.RawMessage {
	raw, _ := json.Marshal(values)
	return raw
}

// ─────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────

// TestSecret_RoundTrip: set(key, value) then get(key) returns the same value.
func TestSecret_Bridge_RoundTrip(t *testing.T) {
	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())
	ctx := context.Background()

	_, err := api.Dispatch(ctx, "p", "set", argsArr("api-key", "sk-secret-value"), "", nil)
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := api.Dispatch(ctx, "p", "get", argsArr("api-key"), "", nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "sk-secret-value" {
		t.Errorf("value: got %v want sk-secret-value", got)
	}
}

// TestSecret_Bridge_GateDenies: a denied gate blocks all methods with EPERM.
func TestSecret_Bridge_GateDenies(t *testing.T) {
	api, store, _ := newSecretAPIWithMocks(t, secretDeniedGate())

	_, err := api.Dispatch(context.Background(), "p", "set", argsArr("k", "v"), "", nil)
	if err == nil {
		t.Fatal("want EPERM, got nil")
	}
	var pe *PermError
	if !errors.As(err, &pe) {
		t.Fatalf("want *PermError, got %T: %v", err, err)
	}

	// Store MUST NOT have been touched.
	if len(store.secrets) != 0 {
		t.Error("SecretSet was called despite deny gate")
	}
}

// TestSecret_Bridge_GetMissing returns nil, not an error.
func TestSecret_Bridge_GetMissing(t *testing.T) {
	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())

	got, err := api.Dispatch(context.Background(), "p", "get", argsArr("never-set"), "", nil)
	if err != nil {
		t.Fatalf("get missing: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

// TestSecret_Bridge_DeleteThenGet: after delete, get returns nil.
func TestSecret_Bridge_DeleteThenGet(t *testing.T) {
	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())
	ctx := context.Background()

	_, _ = api.Dispatch(ctx, "p", "set", argsArr("k", "v"), "", nil)
	_, err := api.Dispatch(ctx, "p", "delete", argsArr("k"), "", nil)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ := api.Dispatch(ctx, "p", "get", argsArr("k"), "", nil)
	if got != nil {
		t.Errorf("want nil after delete, got %v", got)
	}
}

// TestSecret_Bridge_List returns every key set for the plugin.
func TestSecret_Bridge_List(t *testing.T) {
	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())
	ctx := context.Background()

	for _, k := range []string{"zeta", "alpha", "mu"} {
		if _, err := api.Dispatch(ctx, "p", "set", argsArr(k, "val"), "", nil); err != nil {
			t.Fatalf("set %q: %v", k, err)
		}
	}
	got, err := api.Dispatch(ctx, "p", "list", argsArr(), "", nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	keys, ok := got.([]string)
	if !ok {
		t.Fatalf("want []string, got %T", got)
	}
	if len(keys) != 3 {
		t.Errorf("len: got %d want 3 (%v)", len(keys), keys)
	}
}

// TestSecret_Bridge_KeyValidation_Rejects: slashes, "..", empty, too long.
func TestSecret_Bridge_KeyValidation_Rejects(t *testing.T) {
	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())

	badKeys := []string{
		"",
		"../etc/passwd",
		"slash/key",
		"key..danger",
		strings.Repeat("a", 129),
		"spaces are bad",
		"key\x00null",
	}
	for _, k := range badKeys {
		_, err := api.Dispatch(context.Background(), "p", "set", argsArr(k, "v"), "", nil)
		if err == nil {
			t.Errorf("key %q: want rejection, got nil", k)
			continue
		}
		var we *WireError
		if !errors.As(err, &we) {
			t.Errorf("key %q: want *WireError, got %T", k, err)
			continue
		}
		if we.Code != "EINVAL" {
			t.Errorf("key %q: want EINVAL, got %q", k, we.Code)
		}
	}
}

// TestSecret_Bridge_KeyValidation_Accepts covers valid keys.
func TestSecret_Bridge_KeyValidation_Accepts(t *testing.T) {
	good := []string{
		"a",
		"api-key",
		"api.token.v1",
		"ID_42",
		strings.Repeat("x", 128),
	}
	for _, k := range good {
		if !MatchSecretNamespace(k) {
			t.Errorf("MatchSecretNamespace(%q) = false, want true", k)
		}
	}
}

// TestSecret_Bridge_UnknownMethod returns EUNAVAIL.
func TestSecret_Bridge_UnknownMethod(t *testing.T) {
	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())
	_, err := api.Dispatch(context.Background(), "p", "dump-all", argsArr(), "", nil)
	if err == nil {
		t.Fatal("want EUNAVAIL, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EUNAVAIL" {
		t.Errorf("want EUNAVAIL, got %T: %v", err, err)
	}
}

// TestSecret_Bridge_Set_GeneratesDEKOnFirstCall asserts that the first set
// for a plugin creates a wrapped DEK row.
func TestSecret_Bridge_Set_GeneratesDEKOnFirstCall(t *testing.T) {
	api, store, _ := newSecretAPIWithMocks(t, secretGrantedGate())

	if _, ok := store.wrapped["p"]; ok {
		t.Fatal("pre-existing KEK row")
	}
	_, err := api.Dispatch(context.Background(), "p", "set", argsArr("k", "v"), "", nil)
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, ok := store.wrapped["p"]; !ok {
		t.Error("KEK row was not created on first set")
	}
}

// TestSecret_Bridge_PerPluginIsolation: plugins cannot read each others secrets.
// (The store mock is per-plugin keyed; this verifies the Dispatch path respects it.)
func TestSecret_Bridge_PerPluginIsolation(t *testing.T) {
	store := newMockSecretStore()
	kek := newMockKEK()

	apiA := NewSecretAPI(SecretConfig{Store: store, Gate: secretGrantedGate(), KEK: kek})
	apiB := NewSecretAPI(SecretConfig{Store: store, Gate: secretGrantedGate(), KEK: kek})

	if _, err := apiA.Dispatch(context.Background(), "plugin-a", "set", argsArr("shared", "secret-a"), "", nil); err != nil {
		t.Fatalf("set a: %v", err)
	}

	got, err := apiB.Dispatch(context.Background(), "plugin-b", "get", argsArr("shared"), "", nil)
	if err != nil {
		t.Fatalf("get b: %v", err)
	}
	if got != nil {
		t.Errorf("plugin-b saw plugin-a's secret: %v", got)
	}
}

// TestSecret_Bridge_NeverLogged captures slog output during a 128-char
// set/get round-trip and asserts the plaintext value does not appear.
func TestSecret_Bridge_NeverLogged(t *testing.T) {
	plaintext := strings.Repeat("X", 128)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(slog.Default()) })

	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())
	api.log = logger // belt-and-braces: wire into SecretAPI's own log too

	ctx := context.Background()
	if _, err := api.Dispatch(ctx, "p", "set", argsArr("k", plaintext), "", nil); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := api.Dispatch(ctx, "p", "get", argsArr("k"), "", nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != plaintext {
		t.Errorf("round-trip: got %q want %q", got, plaintext)
	}

	logStr := buf.String()
	if strings.Contains(logStr, plaintext) {
		t.Errorf("log output leaks secret plaintext: %q", logStr)
	}
}

// TestSecret_Bridge_TamperedCiphertext returns EINTERNAL (GCM authentication failure).
func TestSecret_Bridge_TamperedCiphertext(t *testing.T) {
	api, store, _ := newSecretAPIWithMocks(t, secretGrantedGate())
	ctx := context.Background()

	if _, err := api.Dispatch(ctx, "p", "set", argsArr("k", "value"), "", nil); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Tamper the stored ciphertext.
	store.mu.Lock()
	row := store.secrets["p/k"]
	if len(row.ct) == 0 {
		store.mu.Unlock()
		t.Fatal("row missing or empty ciphertext")
	}
	row.ct[0] ^= 0xFF
	store.secrets["p/k"] = row
	store.mu.Unlock()

	_, err := api.Dispatch(ctx, "p", "get", argsArr("k"), "", nil)
	if err == nil {
		t.Fatal("want error on tampered ciphertext")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINTERNAL" {
		t.Errorf("want EINTERNAL, got %T: %v", err, err)
	}
}

// TestSecret_Bridge_Set_MalformedArgs returns EINVAL.
func TestSecret_Bridge_Set_MalformedArgs(t *testing.T) {
	api, _, _ := newSecretAPIWithMocks(t, secretGrantedGate())

	cases := []json.RawMessage{
		json.RawMessage(`[]`),
		json.RawMessage(`["key-only"]`),
		json.RawMessage(`[123, "value"]`),
		json.RawMessage(`["key", 123]`),
		json.RawMessage(`"not-an-array"`),
	}
	for i, args := range cases {
		_, err := api.Dispatch(context.Background(), "p", "set", args, "", nil)
		if err == nil {
			t.Errorf("case %d: want EINVAL, got nil", i)
			continue
		}
		var we *WireError
		if !errors.As(err, &we) || we.Code != "EINVAL" {
			t.Errorf("case %d: want EINVAL, got %v", i, err)
		}
	}
}

// TestSecret_Bridge_KEKErrorPropagates: a KEK provider failure surfaces
// as EINTERNAL, not an unrelated error.
func TestSecret_Bridge_KEKErrorPropagates(t *testing.T) {
	store := newMockSecretStore()
	kek := &mockKEK{err: fmt.Errorf("kek: simulated derivation failure")}

	api := NewSecretAPI(SecretConfig{Store: store, Gate: secretGrantedGate(), KEK: kek})

	_, err := api.Dispatch(context.Background(), "p", "set", argsArr("k", "v"), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINTERNAL" {
		t.Errorf("want EINTERNAL, got %v", err)
	}
}
