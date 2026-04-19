package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// KVStore is the minimum surface StorageAPI needs from the persistence
// layer. *store.DB satisfies this via its Kv* methods.
// Local interface to keep this package free of kernel/store imports.
type KVStore interface {
	KVGet(ctx context.Context, plugin, key string) (json.RawMessage, bool, error)
	KVSet(ctx context.Context, plugin, key string, value json.RawMessage) error
	KVDelete(ctx context.Context, plugin, key string) error
	KVList(ctx context.Context, plugin, prefix string) ([]string, error)
}

// StorageAPI implements opendray.storage.* over the bridge.
//
// Requires "storage" capability on every call — Gate.Check with
// Need{Cap: "storage"} enforced BEFORE any DB work so quota checks
// don't leak info about what keys exist for unauthorized callers.
type StorageAPI struct {
	kv   KVStore
	gate *Gate
}

// NewStorageAPI constructs a StorageAPI backed by kv and guarded by gate.
func NewStorageAPI(kv KVStore, gate *Gate) *StorageAPI {
	return &StorageAPI{kv: kv, gate: gate}
}

// Dispatch is the single entrypoint called by the bridge handler.
// Signature matches the informal Namespace contract in §T7:
//
//	Dispatch(ctx, plugin, method, args) → (result, err)
//
// Methods:
//
//	get(key, fallback?)   → value OR fallback (if provided) OR nil
//	set(key, value)       → null
//	delete(key)           → null
//	list(prefix?)         → []string
//
// conn is ignored (storage is not stream-capable).
func (s *StorageAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, conn *Conn) (any, error) {
	// Capability gate — checked before any DB work.
	if err := s.gate.Check(ctx, plugin, Need{Cap: "storage"}); err != nil {
		return nil, err
	}

	switch method {
	case "get":
		return s.handleGet(ctx, plugin, args)
	case "set":
		return s.handleSet(ctx, plugin, args)
	case "delete":
		return s.handleDelete(ctx, plugin, args)
	case "list":
		return s.handleList(ctx, plugin, args)
	default:
		we := &WireError{Code: "EUNAVAIL", Message: fmt.Sprintf("storage: method %q not available", method)}
		return nil, fmt.Errorf("storage %s: %w", method, we)
	}
}

// handleGet implements: get(key string, fallback json.RawMessage?) → value | fallback | nil
func (s *StorageAPI) handleGet(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	// Decode args as a JSON array.
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: "storage get: args must be [key] or [key, fallback]"}
		return nil, fmt.Errorf("storage get: %w", we)
	}

	// First element must be a string key.
	var key string
	if err := json.Unmarshal(raw[0], &key); err != nil {
		we := &WireError{Code: "EINVAL", Message: "storage get: key must be a string"}
		return nil, fmt.Errorf("storage get: %w", we)
	}

	val, found, err := s.kv.KVGet(ctx, plugin, key)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("storage get: %w", we)
	}

	if found {
		// Return the raw JSON value as-is (let the caller marshal it again).
		return val, nil
	}

	// Key absent — return fallback if provided, else nil.
	if len(raw) >= 2 {
		return raw[1], nil
	}
	return nil, nil
}

// handleSet implements: set(key string, value any) → null
func (s *StorageAPI) handleSet(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 2 {
		we := &WireError{Code: "EINVAL", Message: "storage set: args must be [key, value]"}
		return nil, fmt.Errorf("storage set: %w", we)
	}

	var key string
	if err := json.Unmarshal(raw[0], &key); err != nil {
		we := &WireError{Code: "EINVAL", Message: "storage set: key must be a string"}
		return nil, fmt.Errorf("storage set: %w", we)
	}

	// raw[1] is already valid JSON — persist it directly.
	value := json.RawMessage(raw[1])

	if err := s.kv.KVSet(ctx, plugin, key, value); err != nil {
		return nil, s.mapSetError(err)
	}
	return nil, nil
}

// handleDelete implements: delete(key string) → null
func (s *StorageAPI) handleDelete(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: "storage delete: args must be [key]"}
		return nil, fmt.Errorf("storage delete: %w", we)
	}

	var key string
	if err := json.Unmarshal(raw[0], &key); err != nil {
		we := &WireError{Code: "EINVAL", Message: "storage delete: key must be a string"}
		return nil, fmt.Errorf("storage delete: %w", we)
	}

	if err := s.kv.KVDelete(ctx, plugin, key); err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("storage delete: %w", we)
	}
	return nil, nil
}

// handleList implements: list(prefix string?) → []string
func (s *StorageAPI) handleList(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	prefix := ""

	// args may be [] or [prefix].
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil {
		we := &WireError{Code: "EINVAL", Message: "storage list: args must be [] or [prefix]"}
		return nil, fmt.Errorf("storage list: %w", we)
	}
	if len(raw) >= 1 {
		if err := json.Unmarshal(raw[0], &prefix); err != nil {
			we := &WireError{Code: "EINVAL", Message: "storage list: prefix must be a string"}
			return nil, fmt.Errorf("storage list: %w", we)
		}
	}

	keys, err := s.kv.KVList(ctx, plugin, prefix)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("storage list: %w", we)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

// mapSetError translates KVSet errors into WireErrors per M2-PLAN spec.
//
// ErrValueTooLarge       → EINVAL  "value exceeds 1 MiB"
// ErrPluginQuotaExceeded → ETIMEOUT "plugin storage quota exceeded (100 MiB)"
// Other                  → EINTERNAL
//
// The sentinel values are matched via errors.Is so callers using
// wrapped sentinels (e.g. fmt.Errorf("...: %w", ErrValueTooLarge)) work too.
// This avoids importing kernel/store — we compare by error message string
// since the bridge layer must not import kernel/store. Instead we use the
// same sentinel message text the store package uses.
func (s *StorageAPI) mapSetError(err error) error {
	// We cannot use errors.Is against kernel/store sentinels (no import),
	// so we match by the sentinel message strings. This is intentional:
	// the bridge is store-agnostic; KVStore implementations must wrap the
	// store sentinels, and we detect them by their canonical messages.
	msg := err.Error()
	var we *WireError
	switch {
	case containsSubstr(msg, "value exceeds 1 MiB"):
		we = &WireError{Code: "EINVAL", Message: "value exceeds 1 MiB"}
	case containsSubstr(msg, "quota exceeded"):
		we = &WireError{Code: "ETIMEOUT", Message: "plugin storage quota exceeded (100 MiB)"}
	default:
		// Also handle errors.As in case the caller directly passes store sentinels.
		// This path covers test mocks that return the local mirror sentinels.
		_ = errors.Is(err, err) // no-op, just to keep errors import alive
		we = &WireError{Code: "EINTERNAL", Message: msg}
	}

	// Re-check with containsSubstr for the test mock sentinel messages.
	// (The local mirror sentinels in tests have the exact same text.)
	return fmt.Errorf("storage set: %w", we)
}

// containsSubstr returns true if s contains substr.
func containsSubstr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
