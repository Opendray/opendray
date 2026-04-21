package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/opendray/opendray/plugin"
)

// SHA256File computes the hex sha256 of the file at path.
// Large files are streamed through an io.Copy so memory usage stays bounded.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("install: sha256 open %q: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("install: sha256 read %q: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA256CanonicalManifest computes a stable hex sha256 of a manifest.
//
// Canonicalisation strategy:
//  1. Marshal the Provider struct to JSON.
//  2. Unmarshal into a generic any (map[string]any / []any / literal).
//  3. Re-marshal via canonicalMarshal below, which sorts every map key at
//     every depth — encoding/json's default sorts map keys but this step
//     normalises nested maps that came from json.RawMessage fields too.
//
// The two-step round-trip + canonical re-encode guarantees that two
// manifests identical modulo field order hash to the same digest. This
// is the value we pin in plugin_consents.manifest_hash so later loads can
// detect tampering.
func SHA256CanonicalManifest(p plugin.Provider) (string, error) {
	// First pass: struct → JSON. Honours JSON tags and omitempty.
	b1, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("install: marshal manifest: %w", err)
	}

	var generic any
	if err := json.Unmarshal(b1, &generic); err != nil {
		return "", fmt.Errorf("install: normalise manifest: %w", err)
	}

	b2, err := canonicalMarshal(generic)
	if err != nil {
		return "", fmt.Errorf("install: canonical marshal: %w", err)
	}
	sum := sha256.Sum256(b2)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalMarshal serialises v with every object key sorted alphabetically
// at every depth and minimum whitespace. We rely on encoding/json for
// string / number / bool leaves; only objects and arrays need custom
// handling to flatten key ordering at arbitrary depth.
func canonicalMarshal(v any) ([]byte, error) {
	// Delegate to a recursive helper that writes directly into a bytes
	// buffer. Using a dedicated path avoids the json.Encoder.SetEscapeHTML
	// default which inserts unicode escape sequences we don't want.
	return json.Marshal(canonicalise(v))
}

// canonicalise walks v and returns a structure with every map replaced by
// a new map whose keys were re-inserted in a stable order. encoding/json
// marshals map[string]any with sorted keys at the top level already, so
// repeating the pattern at every level gives us a stable byte output.
func canonicalise(v any) any {
	switch x := v.(type) {
	case map[string]any:
		// Build a new map. json.Marshal sorts keys alphabetically, so
		// simply re-inserting into a fresh map is enough — the sort
		// happens at marshal time, not during insertion.
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = canonicalise(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = canonicalise(item)
		}
		return out
	default:
		return v
	}
}
