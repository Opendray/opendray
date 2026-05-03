package mcp

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/zalando/go-keyring"
)

// Secrets is the MCP secrets vault. Two on-disk formats are supported:
//
//   encrypted (preferred): a binary file with header `ODSE\x01` + 12-byte
//     AES-GCM nonce + ciphertext. The 256-bit key lives in the OS
//     keychain (macOS Keychain / Linux secret-service / Windows
//     Credential Manager) under service "opendray", account
//     "mcp-secrets-key".
//
//   plaintext (fallback / legacy): dotenv KEY=VALUE one-per-line. Used
//     when the OS keychain is unavailable (typically headless Linux
//     without gnome-keyring / kwallet running). Migrated to encrypted
//     automatically on first load when the keychain becomes available.
//
// The substitution path used at session spawn time
// (Substitute / SubstituteMap / Resolve) does not care which format
// the file is in — both populate the same in-memory `values` map.
type Secrets struct {
	path       string
	keyringKey []byte // nil = plaintext mode (keychain unavailable)
	values     map[string]string
	log        *slog.Logger
}

// Keychain identifiers. Constants so users can find the entry in
// macOS Keychain Access ("opendray" → "mcp-secrets-key").
const (
	keyringService = "opendray"
	keyringAccount = "mcp-secrets-key"

	encMagic      = "ODSE" // OpenDray Secrets Encrypted
	encVersion    = byte(1)
	encHeaderSize = 5 // magic(4) + version(1)
	nonceSize     = 12
	keySize       = 32 // AES-256
)

// LoadSecrets reads the file at path into a Secrets struct, picking
// the encrypted backend when the OS keychain is available. A missing
// file is treated as an empty vault (no error). When a plaintext file
// is found AND the keychain is available, it is auto-migrated:
// encrypted form is written to `path`, plaintext is preserved at
// `path + ".migrated.bak"` so the user has a one-step rollback.
func LoadSecrets(path string) (*Secrets, error) {
	return LoadSecretsWithLogger(path, slog.Default())
}

// LoadSecretsWithLogger is the variant that takes an explicit logger
// — used by the gateway so migration / fallback events show up in the
// main service log instead of stderr.
func LoadSecretsWithLogger(path string, log *slog.Logger) (*Secrets, error) {
	if log == nil {
		log = slog.Default()
	}
	s := &Secrets{
		path:   path,
		values: map[string]string{},
		log:    log.With("component", "mcp.secrets"),
	}
	if path == "" {
		return s, nil
	}

	// Try the OS keychain. Failure is downgraded to a warning — the
	// vault still works in plaintext mode, just without at-rest
	// encryption.
	key, kerr := loadOrCreateKeychainKey()
	if kerr != nil {
		s.log.Warn("OS keychain unavailable; secrets file will be stored in plaintext",
			"err", kerr, "path", path)
	} else {
		s.keyringKey = key
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s, nil // empty vault
		}
		return nil, fmt.Errorf("read secrets %s: %w", path, err)
	}

	if isEncrypted(data) {
		if s.keyringKey == nil {
			return nil, fmt.Errorf(
				"secrets file at %s is encrypted but the OS keychain is unavailable; "+
					"the encryption key cannot be retrieved", path)
		}
		values, err := decryptValues(data, s.keyringKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt secrets: %w", err)
		}
		s.values = values
		return s, nil
	}

	// Plaintext. Parse first so a bad file fails before we touch the
	// migration path.
	if err := s.parse(data); err != nil {
		return nil, fmt.Errorf("parse secrets %s: %w", path, err)
	}

	// Auto-migrate to encrypted when the keychain is available.
	if s.keyringKey != nil {
		backupPath := path + ".migrated.bak"
		if err := os.WriteFile(backupPath, data, 0o600); err != nil {
			return nil, fmt.Errorf("write secrets migration backup: %w", err)
		}
		if err := s.persist(); err != nil {
			return nil, fmt.Errorf("encrypt secrets during migration: %w", err)
		}
		s.log.Info("migrated plaintext secrets to encrypted storage",
			"path", path, "backup", backupPath, "keys", len(s.values))
	}
	return s, nil
}

// Path returns the file the secrets are persisted to.
func (s *Secrets) Path() string { return s.path }

// Encrypted reports whether the on-disk file is AES-GCM encrypted
// (true) or plaintext fallback (false). Surfaced to the API so the UI
// can show the user which mode the vault is operating in.
func (s *Secrets) Encrypted() bool { return s.keyringKey != nil }

// Keys returns the loaded key names (sorted) without values. Used by
// the handler's listing endpoint — the actual values are never
// returned over the API.
func (s *Secrets) Keys() []string {
	out := make([]string, 0, len(s.values))
	for k := range s.values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Get returns the value for key, with `ok=false` when missing.
func (s *Secrets) Get(key string) (string, bool) {
	v, ok := s.values[key]
	return v, ok
}

// Has returns true when the key exists. Cheap shortcut for the UI to
// know whether a PUT will be a create or an update.
func (s *Secrets) Has(key string) bool {
	_, ok := s.values[key]
	return ok
}

// Set stores key=value and atomically rewrites the file in whichever
// format the vault is currently using. Returns an error when the key
// name is invalid or the disk write fails.
func (s *Secrets) Set(key, value string) error {
	if !validSecretKey(key) {
		return fmt.Errorf("invalid secret key %q (must match [A-Za-z_][A-Za-z0-9_]*)", key)
	}
	s.values[key] = value
	if err := s.persist(); err != nil {
		// Roll back the in-memory change so the next read sees the
		// previous state.
		delete(s.values, key)
		return err
	}
	return nil
}

// Delete removes a key. Missing keys return ErrSecretNotFound so the
// handler can surface 404 cleanly.
func (s *Secrets) Delete(key string) error {
	if _, ok := s.values[key]; !ok {
		return ErrSecretNotFound
	}
	prev := s.values[key]
	delete(s.values, key)
	if err := s.persist(); err != nil {
		s.values[key] = prev // roll back
		return err
	}
	return nil
}

// ErrSecretNotFound is returned by Delete when the key doesn't exist.
var ErrSecretNotFound = errors.New("secret key not found")

// placeholder matches `${VAR}` (the only form we support — keeps it
// unambiguous vs. shell-style `$VAR`, which would collide with values
// agents might want to pass through literally).
var placeholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Substitute replaces every ${KEY} in v. Missing keys are left as the
// literal `${KEY}` so the agent sees a clear "credentials not set"
// failure mode rather than a silent empty string. The returned slice
// lists keys that were referenced but absent — caller can log it.
func (s *Secrets) Substitute(v string) (resolved string, missing []string) {
	resolved = placeholder.ReplaceAllStringFunc(v, func(match string) string {
		name := match[2 : len(match)-1] // strip ${ and }
		if val, ok := s.values[name]; ok {
			return val
		}
		missing = append(missing, name)
		return match
	})
	return resolved, missing
}

// SubstituteMap runs Substitute over every value in m. Returns a new
// map (m is not mutated) and the union of missing keys.
func (s *Secrets) SubstituteMap(m map[string]string) (map[string]string, []string) {
	if len(m) == 0 {
		return m, nil
	}
	out := make(map[string]string, len(m))
	seen := map[string]bool{}
	var missing []string
	for k, v := range m {
		next, miss := s.Substitute(v)
		out[k] = next
		for _, mk := range miss {
			if !seen[mk] {
				seen[mk] = true
				missing = append(missing, mk)
			}
		}
	}
	return out, missing
}

// Resolve runs all placeholder-bearing fields of a Server through the
// secrets table. Returns a copy of the server with values substituted,
// plus the missing key list across all fields. The original Server is
// not mutated so callers can keep the registry view intact.
func (s *Secrets) Resolve(srv Server) (Server, []string) {
	out := srv
	var allMissing []string
	seen := map[string]bool{}

	add := func(missing []string) {
		for _, k := range missing {
			if !seen[k] {
				seen[k] = true
				allMissing = append(allMissing, k)
			}
		}
	}

	if len(srv.Env) > 0 {
		next, miss := s.SubstituteMap(srv.Env)
		out.Env = next
		add(miss)
	}
	if len(srv.Headers) > 0 {
		next, miss := s.SubstituteMap(srv.Headers)
		out.Headers = next
		add(miss)
	}
	if srv.URL != "" {
		next, miss := s.Substitute(srv.URL)
		out.URL = next
		add(miss)
	}
	if len(srv.Args) > 0 {
		out.Args = make([]string, len(srv.Args))
		for i, a := range srv.Args {
			next, miss := s.Substitute(a)
			out.Args[i] = next
			add(miss)
		}
	}
	return out, allMissing
}

// persist atomically writes the current in-memory state in whichever
// backend mode the vault is using. Encrypted form when the keychain
// is available, plaintext otherwise.
func (s *Secrets) persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create secrets dir: %w", err)
	}
	var data []byte
	if s.keyringKey != nil {
		enc, err := encryptValues(s.values, s.keyringKey)
		if err != nil {
			return err
		}
		data = enc
	} else {
		data = formatPlaintext(s.values)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, s.path, err)
	}
	return nil
}

// parse reads dotenv-style content into s.values. Format:
//
//	# comment
//	KEY=value
//	KEY="value with spaces"
//	KEY='single quoted'
//	export KEY=value
func (s *Secrets) parse(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	lineno := 0
	for scanner.Scan() {
		lineno++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("line %d: missing '='", lineno)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if !validSecretKey(key) {
			return fmt.Errorf("line %d: invalid key %q", lineno, key)
		}
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		s.values[key] = val
	}
	return scanner.Err()
}

// formatPlaintext renders the values map back to dotenv form. Keys
// are sorted so successive writes produce stable, diffable output.
func formatPlaintext(values map[string]string) []byte {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	b.WriteString("# opendray MCP secrets — do not commit\n")
	for _, k := range keys {
		v := values[k]
		// Quote values that contain whitespace or `#` so they parse
		// back unambiguously.
		if needsQuoting(v) {
			fmt.Fprintf(&b, "%s=%q\n", k, v)
		} else {
			fmt.Fprintf(&b, "%s=%s\n", k, v)
		}
	}
	return b.Bytes()
}

func needsQuoting(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '#' || r == '"' || r == '\n' {
			return true
		}
	}
	return false
}

func validSecretKey(k string) bool {
	if k == "" {
		return false
	}
	for i, r := range k {
		ok := r == '_' ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(i > 0 && r >= '0' && r <= '9')
		if !ok {
			return false
		}
	}
	return true
}

// ── encryption ──────────────────────────────────────────────────────

func loadOrCreateKeychainKey() ([]byte, error) {
	encoded, err := keyring.Get(keyringService, keyringAccount)
	if err == nil && encoded != "" {
		decoded, derr := base64.StdEncoding.DecodeString(encoded)
		if derr == nil && len(decoded) == keySize {
			return decoded, nil
		}
		// Stored value exists but is malformed — wipe it so we can
		// regenerate cleanly.
		_ = keyring.Delete(keyringService, keyringAccount)
	} else if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		// Hard keychain failure (no daemon, permission denied, …).
		return nil, fmt.Errorf("read keychain: %w", err)
	}

	// Generate a fresh 256-bit key and store it.
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	if err := keyring.Set(keyringService, keyringAccount, base64.StdEncoding.EncodeToString(key)); err != nil {
		return nil, fmt.Errorf("write keychain: %w", err)
	}
	return key, nil
}

func isEncrypted(data []byte) bool {
	if len(data) < encHeaderSize+nonceSize {
		return false
	}
	return string(data[:4]) == encMagic && data[4] == encVersion
}

func encryptValues(values map[string]string, key []byte) ([]byte, error) {
	plaintext, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal values: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, encHeaderSize+nonceSize+len(ciphertext))
	out = append(out, encMagic...)
	out = append(out, encVersion)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func decryptValues(data, key []byte) (map[string]string, error) {
	if !isEncrypted(data) {
		return nil, errors.New("not an encrypted secrets file")
	}
	nonce := data[encHeaderSize : encHeaderSize+nonceSize]
	ciphertext := data[encHeaderSize+nonceSize:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed (key may have changed): %w", err)
	}
	var values map[string]string
	if err := json.Unmarshal(plaintext, &values); err != nil {
		return nil, fmt.Errorf("parse decrypted: %w", err)
	}
	if values == nil {
		values = map[string]string{}
	}
	return values, nil
}
