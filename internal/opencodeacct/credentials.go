package opencodeacct

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// providerIDPattern bounds provider ids to what opencode uses
// ("moonshotai", "kimi-for-coding", "github-copilot", "amazon-bedrock").
var providerIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// buildBundle turns a request's Credentials (or ProviderID+APIKey
// convenience pair) into a validated auth.json-shaped bundle. Returns
// (nil, nil) when the request carries no credentials at all.
func buildBundle(creds map[string]any, providerID, apiKey string) (map[string]any, error) {
	providerID = strings.TrimSpace(providerID)
	apiKey = strings.TrimSpace(apiKey)
	if len(creds) == 0 && providerID == "" && apiKey == "" {
		return nil, nil
	}
	if len(creds) > 0 && (providerID != "" || apiKey != "") {
		return nil, fmt.Errorf("%w: pass either credentials or provider_id+api_key, not both", ErrInvalidCredentials)
	}
	if len(creds) == 0 {
		if providerID == "" || apiKey == "" {
			return nil, fmt.Errorf("%w: provider_id and api_key are both required", ErrInvalidCredentials)
		}
		creds = map[string]any{providerID: map[string]any{"type": "api", "key": apiKey}}
	}
	if err := validateBundle(creds); err != nil {
		return nil, err
	}
	return creds, nil
}

// validateBundle checks the auth.json shape: a non-empty object of
// provider id → object with a string "type"; "api" entries need a
// non-empty string "key". Other types (oauth, wellknown) pass through
// opaque. Error messages never echo values.
func validateBundle(b map[string]any) error {
	if len(b) == 0 {
		return fmt.Errorf("%w: no providers", ErrInvalidCredentials)
	}
	for id, v := range b {
		if !providerIDPattern.MatchString(id) {
			return fmt.Errorf("%w: bad provider id %q", ErrInvalidCredentials, id)
		}
		entry, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: provider %q must be an object", ErrInvalidCredentials, id)
		}
		typ, _ := entry["type"].(string)
		if typ == "" {
			return fmt.Errorf("%w: provider %q needs a string \"type\"", ErrInvalidCredentials, id)
		}
		if typ == "api" {
			if key, _ := entry["key"].(string); strings.TrimSpace(key) == "" {
				return fmt.Errorf("%w: provider %q (type api) needs a \"key\"", ErrInvalidCredentials, id)
			}
		}
	}
	return nil
}

// providerIDs returns the bundle's provider ids, sorted.
func providerIDs(b map[string]any) []string {
	out := make([]string, 0, len(b))
	for id := range b {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// summarize builds the masked display view of a bundle.
func summarize(b map[string]any) []CredentialSummary {
	out := make([]CredentialSummary, 0, len(b))
	for _, id := range providerIDs(b) {
		entry, _ := b[id].(map[string]any)
		typ, _ := entry["type"].(string)
		s := CredentialSummary{Provider: id, Type: typ}
		if key, _ := entry["key"].(string); key != "" {
			s.Hint = maskSecret(key)
		}
		out = append(out, s)
	}
	return out
}

// maskSecret keeps a short recognizable prefix and the last 4 chars:
// "sk-abcdef…wxyz" → "sk-…wxyz". Short values are fully masked.
func maskSecret(v string) string {
	if len(v) < 12 {
		return "••••"
	}
	prefix := ""
	if i := strings.IndexAny(v, "-_"); i > 0 && i <= 6 {
		prefix = v[:i+1]
	}
	return prefix + "…" + v[len(v)-4:]
}

// localAuthPath is where opencode keeps its on-disk credential set:
// $XDG_DATA_HOME/opencode/auth.json, else ~/.local/share/opencode/auth.json.
func localAuthPath() string {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "opencode", "auth.json")
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "auth.json")
}

// readBundleFile reads + validates an auth.json-shaped file.
func readBundleFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b map[string]any
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, fmt.Errorf("%w: %s is not a JSON object", ErrInvalidCredentials, path)
	}
	if err := validateBundle(b); err != nil {
		return nil, err
	}
	return b, nil
}
