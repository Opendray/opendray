package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	// store sentinel errors are not imported here — bridge is store-agnostic.
	// We define local sentinel mirrors so tests remain independent of kernel/store.
)

// ─────────────────────────────────────────────
// Local sentinel mirrors (mirrors of store sentinels — no kernel/store import)
// ─────────────────────────────────────────────

var (
	errValueTooLarge       = errors.New("store: plugin_kv value exceeds 1 MiB per key")
	errPluginQuotaExceeded = errors.New("store: plugin_kv quota exceeded (100 MiB per plugin)")
)

// ─────────────────────────────────────────────
// mockKV — in-memory KVStore for unit tests
// ─────────────────────────────────────────────

type mockKV struct {
	data   map[string]json.RawMessage // key = "plugin/key"
	setErr error                      // if set, KVSet returns this error
	setCalls []setCall
}

type setCall struct {
	plugin string
	key    string
	value  json.RawMessage
}

func newMockKV() *mockKV {
	return &mockKV{data: make(map[string]json.RawMessage)}
}

func (m *mockKV) KVGet(_ context.Context, plugin, key string) (json.RawMessage, bool, error) {
	v, ok := m.data[plugin+"/"+key]
	return v, ok, nil
}

func (m *mockKV) KVSet(_ context.Context, plugin, key string, value json.RawMessage) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.data[plugin+"/"+key] = value
	m.setCalls = append(m.setCalls, setCall{plugin: plugin, key: key, value: value})
	return nil
}

func (m *mockKV) KVDelete(_ context.Context, plugin, key string) error {
	delete(m.data, plugin+"/"+key)
	return nil
}

func (m *mockKV) KVList(_ context.Context, plugin, prefix string) ([]string, error) {
	var out []string
	for k := range m.data {
		// keys stored as "plugin/key"
		if len(k) <= len(plugin)+1 {
			continue
		}
		pfx := plugin + "/"
		if k[:len(pfx)] != pfx {
			continue
		}
		bare := k[len(pfx):]
		if prefix == "" || (len(bare) >= len(prefix) && bare[:len(prefix)] == prefix) {
			out = append(out, bare)
		}
	}
	// sort for determinism
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i] > out[j] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// ─────────────────────────────────────────────
// mockConsent — always grants the requested cap
// ─────────────────────────────────────────────

type mockConsent struct {
	found bool
	deny  bool
}

func (m *mockConsent) Load(_ context.Context, _ string) ([]byte, bool, error) {
	if !m.found {
		return nil, false, nil
	}
	// Return perms JSON with storage: true so the gate allows it.
	return []byte(`{"storage":true}`), true, nil
}

// grantedGate returns a Gate that always allows the storage cap.
func grantedGate() *Gate {
	return NewGate(&mockConsent{found: true}, nil, nil)
}

// deniedGate returns a Gate that always denies (no consent record).
func deniedGate() *Gate {
	return NewGate(&mockConsent{found: false}, nil, nil)
}

// marshalArgs encodes args as a JSON array.
func marshalArgs(args ...any) json.RawMessage {
	b, _ := json.Marshal(args)
	return json.RawMessage(b)
}

// ─────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────

// TestStorage_GateDeniesReturnsPermError verifies that a denied gate blocks
// any call before touching the KV store.
func TestStorage_GateDeniesReturnsPermError(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, deniedGate())

	_, err := api.Dispatch(context.Background(), "test-plugin", "get",
		marshalArgs("some-key"), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var pe *PermError
	if !errors.As(err, &pe) {
		t.Errorf("want *PermError, got %T: %v", err, err)
	}
	if len(kv.setCalls) != 0 {
		t.Error("KVSet must not be called when gate denies")
	}
}

// TestStorage_Get_FoundReturnsValue verifies that a present key returns its value.
func TestStorage_Get_FoundReturnsValue(t *testing.T) {
	kv := newMockKV()
	kv.data["myplugin/greeting"] = json.RawMessage(`"hello"`)
	api := NewStorageAPI(kv, grantedGate())

	result, err := api.Dispatch(context.Background(), "myplugin", "get",
		marshalArgs("greeting"), "", nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	// result should be the raw JSON "hello"
	got, _ := json.Marshal(result)
	if string(got) != `"hello"` {
		t.Errorf("got %s want %q", got, "hello")
	}
}

// TestStorage_Get_MissingReturnsFallback verifies that when a key is absent and
// a fallback is provided, the fallback is returned.
func TestStorage_Get_MissingReturnsFallback(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	fallback := json.RawMessage(`{"default":true}`)
	args, _ := json.Marshal([]json.RawMessage{json.RawMessage(`"absent-key"`), fallback})

	result, err := api.Dispatch(context.Background(), "myplugin", "get",
		json.RawMessage(args), "", nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	got, _ := json.Marshal(result)
	if string(got) != `{"default":true}` {
		t.Errorf("got %s want {\"default\":true}", got)
	}
}

// TestStorage_Get_MissingNoFallbackReturnsNull verifies that when a key is absent
// and no fallback is given, the result is nil (JSON null).
func TestStorage_Get_MissingNoFallbackReturnsNull(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	result, err := api.Dispatch(context.Background(), "myplugin", "get",
		marshalArgs("absent-key"), "", nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if result != nil {
		t.Errorf("want nil, got %v", result)
	}
}

// TestStorage_Set_RoundTrip verifies that set correctly encodes and stores the value.
func TestStorage_Set_RoundTrip(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	args, _ := json.Marshal([]any{"my-key", map[string]any{"x": 1}})
	result, err := api.Dispatch(context.Background(), "myplugin", "set",
		json.RawMessage(args), "", nil)
	if err != nil {
		t.Fatalf("Dispatch set: %v", err)
	}
	if result != nil {
		t.Errorf("set result: want nil, got %v", result)
	}

	// Verify the key is stored in the mock.
	stored, ok := kv.data["myplugin/my-key"]
	if !ok {
		t.Fatal("key not stored in mock KV")
	}
	var v map[string]any
	if err := json.Unmarshal(stored, &v); err != nil {
		t.Fatalf("unmarshal stored: %v", err)
	}
	if v["x"] != float64(1) {
		t.Errorf("stored value x: got %v want 1", v["x"])
	}
}

// TestStorage_Set_ValueTooLarge verifies that ErrValueTooLarge maps to WireError EINVAL.
func TestStorage_Set_ValueTooLarge(t *testing.T) {
	kv := newMockKV()
	kv.setErr = errValueTooLarge
	api := NewStorageAPI(kv, grantedGate())

	args, _ := json.Marshal([]any{"big-key", "some value"})
	_, err := api.Dispatch(context.Background(), "myplugin", "set",
		json.RawMessage(args), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_Set_QuotaExceeded verifies that ErrPluginQuotaExceeded maps to
// WireError ETIMEOUT (per M2-PLAN spec).
func TestStorage_Set_QuotaExceeded(t *testing.T) {
	kv := newMockKV()
	kv.setErr = errPluginQuotaExceeded
	api := NewStorageAPI(kv, grantedGate())

	args, _ := json.Marshal([]any{"any-key", "value"})
	_, err := api.Dispatch(context.Background(), "myplugin", "set",
		json.RawMessage(args), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "ETIMEOUT" {
		t.Errorf("want code ETIMEOUT, got %q", we.Code)
	}
}

// TestStorage_Delete_RoundTrip verifies that delete removes the key.
func TestStorage_Delete_RoundTrip(t *testing.T) {
	kv := newMockKV()
	kv.data["myplugin/bye"] = json.RawMessage(`1`)
	api := NewStorageAPI(kv, grantedGate())

	result, err := api.Dispatch(context.Background(), "myplugin", "delete",
		marshalArgs("bye"), "", nil)
	if err != nil {
		t.Fatalf("Dispatch delete: %v", err)
	}
	if result != nil {
		t.Errorf("delete result: want nil, got %v", result)
	}
	if _, ok := kv.data["myplugin/bye"]; ok {
		t.Error("key still present after delete")
	}
}

// TestStorage_List_DefaultPrefix verifies that list with no args returns all keys.
func TestStorage_List_DefaultPrefix(t *testing.T) {
	kv := newMockKV()
	kv.data["myplugin/alpha"] = json.RawMessage(`1`)
	kv.data["myplugin/beta"] = json.RawMessage(`2`)
	kv.data["myplugin/gamma"] = json.RawMessage(`3`)
	api := NewStorageAPI(kv, grantedGate())

	// No args → empty prefix.
	result, err := api.Dispatch(context.Background(), "myplugin", "list",
		json.RawMessage(`[]`), "", nil)
	if err != nil {
		t.Fatalf("Dispatch list: %v", err)
	}
	keys, ok := result.([]string)
	if !ok {
		t.Fatalf("want []string result, got %T", result)
	}
	if len(keys) != 3 {
		t.Errorf("want 3 keys, got %d: %v", len(keys), keys)
	}
}

// TestStorage_List_WithPrefix verifies that list with a prefix filters correctly.
func TestStorage_List_WithPrefix(t *testing.T) {
	kv := newMockKV()
	kv.data["myplugin/ns.foo"] = json.RawMessage(`1`)
	kv.data["myplugin/ns.bar"] = json.RawMessage(`2`)
	kv.data["myplugin/other"] = json.RawMessage(`3`)
	api := NewStorageAPI(kv, grantedGate())

	result, err := api.Dispatch(context.Background(), "myplugin", "list",
		marshalArgs("ns."), "", nil)
	if err != nil {
		t.Fatalf("Dispatch list: %v", err)
	}
	keys, ok := result.([]string)
	if !ok {
		t.Fatalf("want []string result, got %T", result)
	}
	if len(keys) != 2 {
		t.Errorf("want 2 keys with prefix ns., got %d: %v", len(keys), keys)
	}
}

// TestStorage_UnknownMethod_ReturnsEUNAVAIL verifies unknown methods return EUNAVAIL.
func TestStorage_UnknownMethod_ReturnsEUNAVAIL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "nonexistent",
		json.RawMessage(`[]`), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Errorf("want code EUNAVAIL, got %q", we.Code)
	}
}

// TestStorage_MalformedArgs_ReturnsEINVAL verifies malformed args return EINVAL.
func TestStorage_MalformedArgs_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	// get requires at least one string arg; passing a non-array is invalid.
	_, err := api.Dispatch(context.Background(), "myplugin", "get",
		json.RawMessage(`"not-an-array"`), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_Get_NonStringKey_ReturnsEINVAL verifies that a non-string key argument
// returns EINVAL.
func TestStorage_Get_NonStringKey_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	// key must be a string; 42 is invalid.
	_, err := api.Dispatch(context.Background(), "myplugin", "get",
		marshalArgs(42), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_Set_MalformedArgs_ReturnsEINVAL verifies that set with too few args
// returns EINVAL.
func TestStorage_Set_MalformedArgs_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	// set needs [key, value]; passing only one element is invalid.
	_, err := api.Dispatch(context.Background(), "myplugin", "set",
		marshalArgs("key-only"), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_Set_NonStringKey_ReturnsEINVAL verifies that set with a non-string key
// returns EINVAL.
func TestStorage_Set_NonStringKey_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "set",
		marshalArgs(99, "value"), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_Delete_MalformedArgs_ReturnsEINVAL verifies delete with no args
// returns EINVAL.
func TestStorage_Delete_MalformedArgs_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "delete",
		json.RawMessage(`[]`), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_Delete_NonStringKey_ReturnsEINVAL verifies delete with a non-string key
// returns EINVAL.
func TestStorage_Delete_NonStringKey_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "delete",
		marshalArgs(false), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_List_MalformedJSON_ReturnsEINVAL verifies that list with non-array
// JSON returns EINVAL.
func TestStorage_List_MalformedJSON_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "list",
		json.RawMessage(`"not-an-array"`), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_List_NonStringPrefix_ReturnsEINVAL verifies that a non-string prefix
// returns EINVAL.
func TestStorage_List_NonStringPrefix_ReturnsEINVAL(t *testing.T) {
	kv := newMockKV()
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "list",
		marshalArgs(42), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("want code EINVAL, got %q", we.Code)
	}
}

// TestStorage_Get_InternalError maps an unexpected KVGet error to EINTERNAL.
func TestStorage_Get_InternalError(t *testing.T) {
	kv := &errKV{err: errors.New("db gone")}
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "get",
		marshalArgs("any-key"), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINTERNAL" {
		t.Errorf("want code EINTERNAL, got %q", we.Code)
	}
}

// errKV is a KVStore that always returns an error.
type errKV struct{ err error }

func (e *errKV) KVGet(_ context.Context, _, _ string) (json.RawMessage, bool, error) {
	return nil, false, e.err
}
func (e *errKV) KVSet(_ context.Context, _, _ string, _ json.RawMessage) error { return e.err }
func (e *errKV) KVDelete(_ context.Context, _, _ string) error                 { return e.err }
func (e *errKV) KVList(_ context.Context, _, _ string) ([]string, error)       { return nil, e.err }

// TestStorage_Set_InternalError maps an unexpected KVSet error to EINTERNAL.
func TestStorage_Set_InternalError(t *testing.T) {
	kv := &errKV{err: errors.New("connection lost")}
	api := NewStorageAPI(kv, grantedGate())

	args, _ := json.Marshal([]any{"k", "v"})
	_, err := api.Dispatch(context.Background(), "myplugin", "set",
		json.RawMessage(args), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINTERNAL" {
		t.Errorf("want code EINTERNAL, got %q", we.Code)
	}
}

// TestStorage_Delete_InternalError maps an unexpected KVDelete error to EINTERNAL.
func TestStorage_Delete_InternalError(t *testing.T) {
	kv := &errKV{err: errors.New("connection lost")}
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "delete",
		marshalArgs("k"), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINTERNAL" {
		t.Errorf("want code EINTERNAL, got %q", we.Code)
	}
}

// TestStorage_List_InternalError maps an unexpected KVList error to EINTERNAL.
func TestStorage_List_InternalError(t *testing.T) {
	kv := &errKV{err: errors.New("connection lost")}
	api := NewStorageAPI(kv, grantedGate())

	_, err := api.Dispatch(context.Background(), "myplugin", "list",
		json.RawMessage(`[]`), "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINTERNAL" {
		t.Errorf("want code EINTERNAL, got %q", we.Code)
	}
}
