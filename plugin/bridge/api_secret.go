package bridge

// SecretAPI implements opendray.secret.* over the bridge. Values are
// AES-GCM encrypted at rest; the DEK is wrapped under a host-derived KEK
// (T7). Flow:
//
//   set(key, value):
//     1. Gate.Check secret:true.
//     2. MatchSecretNamespace validates key.
//     3. GetWrappedDEK → if ErrNoRows, generate random DEK + wrap via
//        KEKProvider.DeriveKEK + EnsureKEKRow; else unwrap existing.
//     4. AES-GCM encrypt value under DEK with fresh 12-byte nonce.
//     5. SecretSet(ctx, plugin, key, ciphertext, nonce).
//
//   get(key): unwrap DEK, decrypt ciphertext + nonce, return plaintext.
//
//   delete(key): SecretDelete.
//
//   list(): SecretList → keys only (never values).
//
// Secrets never leak across plugins — all store-layer calls are scoped
// by plugin name, which is passed via Dispatch and trusted (gateway sets
// it from the URL path).

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
)

// ─────────────────────────────────────────────
// Dependency interfaces
// ─────────────────────────────────────────────

// SecretStore is the minimum surface SecretAPI needs from kernel/store.
// *store.DB satisfies this. Local interface keeps the bridge package
// decoupled from kernel/store.
type SecretStore interface {
	SecretGet(ctx context.Context, plugin, key string) (ciphertext, nonce []byte, found bool, err error)
	SecretSet(ctx context.Context, plugin, key string, ciphertext, nonce []byte) error
	SecretDelete(ctx context.Context, plugin, key string) error
	SecretList(ctx context.Context, plugin string) ([]string, error)
	EnsureKEKRow(ctx context.Context, plugin string, wrappedDEK []byte, kid string) error
	GetWrappedDEK(ctx context.Context, plugin string) (wrapped []byte, kid string, err error)
}

// KEKProviderLike is the minimum surface SecretAPI needs from the KEK
// provider. kernel/auth.KEKProvider satisfies this. Having a local
// interface lets tests stub.
type KEKProviderLike interface {
	DeriveKEK(ctx context.Context, kid string) ([]byte, error)
}

// WrappedDEKNotFound is the sentinel returned by SecretStore.GetWrappedDEK
// when the plugin has no KEK row yet. Callers match via errors.Is.
//
// Why this isn't kernel/store's pgx.ErrNoRows: the bridge layer must not
// import kernel/store or pgx. Implementations of SecretStore are expected
// to translate pgx.ErrNoRows into this sentinel.
//
// *store.DB's GetWrappedDEK currently returns pgx.ErrNoRows directly; the
// production wiring wraps the DB with an adapter that performs the
// translation. Tests pass a mock that returns WrappedDEKNotFound directly.
var WrappedDEKNotFound = errors.New("secret: no wrapped DEK row for plugin")

// ─────────────────────────────────────────────
// Config & API struct
// ─────────────────────────────────────────────

// SecretConfig wires the SecretAPI's dependencies.
type SecretConfig struct {
	Store SecretStore
	Gate  *Gate
	KEK   KEKProviderLike
	Log   *slog.Logger

	// Kid is the KEK key-id used on first-write. Defaults to "v1".
	Kid string
}

// SecretAPI implements opendray.secret.* over the bridge. Construct via
// NewSecretAPI.
type SecretAPI struct {
	store SecretStore
	gate  *Gate
	kek   KEKProviderLike
	log   *slog.Logger
	kid   string
}

// NewSecretAPI wires a SecretAPI.
func NewSecretAPI(cfg SecretConfig) *SecretAPI {
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.Kid == "" {
		cfg.Kid = "v1"
	}
	return &SecretAPI{
		store: cfg.Store,
		gate:  cfg.Gate,
		kek:   cfg.KEK,
		log:   cfg.Log,
		kid:   cfg.Kid,
	}
}

// ─────────────────────────────────────────────
// Dispatch
// ─────────────────────────────────────────────

// Dispatch routes bridge calls to the right handler. Every method runs
// Gate.Check for "secret" before touching the store.
func (a *SecretAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	_ = envID
	_ = conn
	// Capability gate — secret is a simple bool cap.
	if err := a.gate.Check(ctx, plugin, Need{Cap: "secret"}); err != nil {
		return nil, err
	}

	switch method {
	case "get":
		return a.handleGet(ctx, plugin, args)
	case "set":
		return a.handleSet(ctx, plugin, args)
	case "delete":
		return a.handleDelete(ctx, plugin, args)
	case "list":
		return a.handleList(ctx, plugin, args)
	default:
		we := &WireError{Code: "EUNAVAIL", Message: fmt.Sprintf("secret.%s: method not available", method)}
		return nil, fmt.Errorf("secret %s: %w", method, we)
	}
}

// ─────────────────────────────────────────────
// get
// ─────────────────────────────────────────────

// handleGet returns the plaintext string for key, or nil if absent.
func (a *SecretAPI) handleGet(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	key, err := parseKeyArg(args, "get")
	if err != nil {
		return nil, err
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}

	ct, nonce, found, err := a.store.SecretGet(ctx, plugin, key)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("secret get: %w", we)
	}
	if !found {
		return nil, nil
	}

	// Unwrap DEK and decrypt.
	dek, _, err := a.loadDEK(ctx, plugin)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("secret get: %w", we)
	}
	defer zeroKeyBytes(dek)

	plaintext, err := aesGCMOpen(dek, nonce, ct)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: "secret get: decrypt failed"}
		return nil, fmt.Errorf("secret get: %w", we)
	}
	return string(plaintext), nil
}

// ─────────────────────────────────────────────
// set
// ─────────────────────────────────────────────

// handleSet encrypts value under the plugin's DEK and persists ciphertext
// + nonce. Lazily generates a DEK + KEK row on first set.
func (a *SecretAPI) handleSet(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 2 {
		we := &WireError{Code: "EINVAL", Message: "secret set: args must be [key, value]"}
		return nil, fmt.Errorf("secret set: %w", we)
	}
	var key, value string
	if err := json.Unmarshal(raw[0], &key); err != nil {
		we := &WireError{Code: "EINVAL", Message: "secret set: key must be a string"}
		return nil, fmt.Errorf("secret set: %w", we)
	}
	if err := json.Unmarshal(raw[1], &value); err != nil {
		we := &WireError{Code: "EINVAL", Message: "secret set: value must be a string"}
		return nil, fmt.Errorf("secret set: %w", we)
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}

	// Load-or-generate DEK.
	dek, _, err := a.loadOrGenerateDEK(ctx, plugin)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("secret set: load DEK: %v", err)}
		return nil, fmt.Errorf("secret set: %w", we)
	}
	defer zeroKeyBytes(dek)

	// AES-GCM encrypt the value.
	ct, nonce, err := aesGCMSeal(dek, []byte(value))
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("secret set: encrypt: %v", err)}
		return nil, fmt.Errorf("secret set: %w", we)
	}

	if err := a.store.SecretSet(ctx, plugin, key, ct, nonce); err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("secret set: %w", we)
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// delete
// ─────────────────────────────────────────────

func (a *SecretAPI) handleDelete(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	key, err := parseKeyArg(args, "delete")
	if err != nil {
		return nil, err
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := a.store.SecretDelete(ctx, plugin, key); err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("secret delete: %w", we)
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// list
// ─────────────────────────────────────────────

func (a *SecretAPI) handleList(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	// Accept [] or [] (trailing args ignored).
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil {
		we := &WireError{Code: "EINVAL", Message: "secret list: args must be []"}
		return nil, fmt.Errorf("secret list: %w", we)
	}
	keys, err := a.store.SecretList(ctx, plugin)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: err.Error()}
		return nil, fmt.Errorf("secret list: %w", we)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

// ─────────────────────────────────────────────
// DEK management
// ─────────────────────────────────────────────

// loadDEK unwraps the plugin's existing DEK. Returns WrappedDEKNotFound
// when the plugin has no KEK row.
func (a *SecretAPI) loadDEK(ctx context.Context, plugin string) (dek []byte, kid string, err error) {
	wrapped, kid, err := a.store.GetWrappedDEK(ctx, plugin)
	if err != nil {
		return nil, "", err
	}
	kek, err := a.kek.DeriveKEK(ctx, kid)
	if err != nil {
		return nil, "", fmt.Errorf("secret: derive KEK: %w", err)
	}
	defer zeroKeyBytes(kek)

	dek, err = aesGCMOpen(kek, wrapped[:12], wrapped[12:])
	if err != nil {
		return nil, "", fmt.Errorf("secret: unwrap DEK: %w", err)
	}
	if len(dek) != 32 {
		return nil, "", fmt.Errorf("secret: unwrapped DEK has wrong size %d", len(dek))
	}
	return dek, kid, nil
}

// loadOrGenerateDEK returns the existing DEK or generates one on first use.
func (a *SecretAPI) loadOrGenerateDEK(ctx context.Context, plugin string) (dek []byte, kid string, err error) {
	dek, kid, err = a.loadDEK(ctx, plugin)
	if err == nil {
		return dek, kid, nil
	}
	if !errors.Is(err, WrappedDEKNotFound) {
		return nil, "", err
	}

	// First write: generate a fresh 32-byte DEK, wrap under the current KEK,
	// persist, and return.
	dek = make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, "", fmt.Errorf("secret: generate DEK: %w", err)
	}
	kek, err := a.kek.DeriveKEK(ctx, a.kid)
	if err != nil {
		return nil, "", fmt.Errorf("secret: derive KEK for first-write: %w", err)
	}
	defer zeroKeyBytes(kek)

	wrapped, err := aesGCMSealConcat(kek, dek)
	if err != nil {
		return nil, "", fmt.Errorf("secret: wrap DEK: %w", err)
	}
	if err := a.store.EnsureKEKRow(ctx, plugin, wrapped, a.kid); err != nil {
		return nil, "", fmt.Errorf("secret: ensure KEK row: %w", err)
	}
	return dek, a.kid, nil
}

// ─────────────────────────────────────────────
// AES-GCM helpers (local — avoid importing kernel/auth just for wrap)
// ─────────────────────────────────────────────

// aesGCMSeal encrypts plaintext under key. Returns (ciphertext, nonce).
func aesGCMSeal(key, plaintext []byte) (ct, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ct = gcm.Seal(nil, nonce, plaintext, nil)
	return ct, nonce, nil
}

// aesGCMSealConcat is aesGCMSeal with output nonce||ciphertext, matching
// the kernel/auth wrap format.
func aesGCMSealConcat(key, plaintext []byte) ([]byte, error) {
	ct, nonce, err := aesGCMSeal(key, plaintext)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// aesGCMOpen decrypts ciphertext with nonce under key. Returns plaintext
// or an error on authentication failure.
func aesGCMOpen(key, nonce, ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, nil)
}

// zeroKeyBytes overwrites b with zeros (best-effort defence).
func zeroKeyBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ─────────────────────────────────────────────
// Key validation
// ─────────────────────────────────────────────

// secretKeyRegex enforces the [a-zA-Z0-9._-]{1,128} shape. Rejects "/"
// and ".." implicitly; we add an explicit ".." guard for defence-in-depth.
var secretKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,128}$`)

// PlatformSet writes an encrypted secret on behalf of the platform
// (not the plugin). Used by the gateway config endpoints — the
// capability gate's plugin→host direction does not apply because the
// write is initiated by an authenticated admin request, not a
// sidecar RPC. The encryption path is identical to handleSet: DEK is
// loaded or generated, AES-GCM seals the value with a fresh nonce,
// ciphertext + nonce land in plugin_secret.
//
// Returns nil on success. Callers should surface errors as 500 in
// HTTP responses — the underlying store / crypto failures aren't
// recoverable by retry.
func (a *SecretAPI) PlatformSet(ctx context.Context, plugin, key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	dek, _, err := a.loadOrGenerateDEK(ctx, plugin)
	if err != nil {
		return fmt.Errorf("secret: load DEK: %w", err)
	}
	defer zeroKeyBytes(dek)
	ct, nonce, err := aesGCMSeal(dek, []byte(value))
	if err != nil {
		return fmt.Errorf("secret: encrypt: %w", err)
	}
	if err := a.store.SecretSet(ctx, plugin, key, ct, nonce); err != nil {
		return fmt.Errorf("secret: persist: %w", err)
	}
	return nil
}

// PlatformGet reads and decrypts a secret on behalf of the platform.
// Returns ("", false, nil) when the key is absent. Errors indicate a
// store or crypto failure, not "not found".
func (a *SecretAPI) PlatformGet(ctx context.Context, plugin, key string) (string, bool, error) {
	if err := validateKey(key); err != nil {
		return "", false, err
	}
	ct, nonce, found, err := a.store.SecretGet(ctx, plugin, key)
	if err != nil {
		return "", false, fmt.Errorf("secret: load: %w", err)
	}
	if !found {
		return "", false, nil
	}
	dek, _, err := a.loadDEK(ctx, plugin)
	if err != nil {
		return "", false, fmt.Errorf("secret: load DEK: %w", err)
	}
	defer zeroKeyBytes(dek)
	plaintext, err := aesGCMOpen(dek, nonce, ct)
	if err != nil {
		return "", false, fmt.Errorf("secret: decrypt: %w", err)
	}
	return string(plaintext), true, nil
}

// PlatformDelete removes a secret on behalf of the platform.
// Idempotent — missing key returns nil.
func (a *SecretAPI) PlatformDelete(ctx context.Context, plugin, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := a.store.SecretDelete(ctx, plugin, key); err != nil {
		return fmt.Errorf("secret: delete: %w", err)
	}
	return nil
}

// MatchSecretNamespace returns true if key is a syntactically valid secret
// key: 1–128 chars of [a-zA-Z0-9._-] with no "/" and no "..". Exposed so
// the gateway/admin surface can share validation.
func MatchSecretNamespace(key string) bool {
	if key == "" {
		return false
	}
	if !secretKeyRegex.MatchString(key) {
		return false
	}
	// Defence-in-depth: "." + "." is a path-traversal primitive; even
	// though the regex above excludes "/" so the key can never be a path,
	// we reject ".." verbatim as well.
	for i := 0; i < len(key)-1; i++ {
		if key[i] == '.' && key[i+1] == '.' {
			return false
		}
	}
	return true
}

// validateKey wraps MatchSecretNamespace with a WireError-shaped return for
// handler callers.
func validateKey(key string) error {
	if !MatchSecretNamespace(key) {
		we := &WireError{Code: "EINVAL", Message: "secret: key must match [a-zA-Z0-9._-]{1,128} and contain no '..'"}
		return fmt.Errorf("secret: %w", we)
	}
	return nil
}

// parseKeyArg extracts a single key from [key]-shaped args. Returns a
// WireError-shaped error on malformed input.
func parseKeyArg(args json.RawMessage, method string) (string, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("secret %s: args must be [key]", method)}
		return "", fmt.Errorf("secret %s: %w", method, we)
	}
	var key string
	if err := json.Unmarshal(raw[0], &key); err != nil {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("secret %s: key must be a string", method)}
		return "", fmt.Errorf("secret %s: %w", method, we)
	}
	return key, nil
}
