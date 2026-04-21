package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestStrict_OnDiskExamplesPass asserts every on-disk example manifest
// in plugins/examples/ that is v1 passes ValidateV1Strict — no unknown
// fields slipped in. Examples live on disk (not embedded); walking them
// catches drift between the schema and the reference plugins.
func TestStrict_OnDiskExamplesPass(t *testing.T) {
	root := examplesDir(t)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read examples dir %s: %v", root, err)
	}
	seen := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(root, e.Name())
		p, raw, lErr := LoadManifestWithRaw(path)
		if lErr != nil {
			t.Fatalf("load %s: %v", path, lErr)
		}
		if !p.IsV1() {
			continue
		}
		seen++
		if errs := ValidateV1Strict(p, raw); len(errs) != 0 {
			t.Errorf("%s: strict validator failed:\n%v", e.Name(), errs)
		}
	}
	if seen == 0 {
		t.Fatal("no v1 example plugins found — regression")
	}
}

// examplesDir walks up from this file to locate plugins/examples/.
func examplesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "plugins", "examples")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate plugins/examples/")
	return ""
}

// TestStrict_UnknownTopLevelField rejects a manifest with a bogus
// top-level key.
func TestStrict_UnknownTopLevelField(t *testing.T) {
	raw := []byte(`{
		"name": "plug",
		"version": "1.0.0",
		"publisher": "acme",
		"engines": {"opendray": "^1.0.0"},
		"rocketFuel": "high"
	}`)
	p := mustParseProvider(t, string(raw))
	errs := ValidateV1Strict(p, raw)
	if !hasError(errs, "rocketFuel") {
		t.Errorf("want unknown-field error on rocketFuel, got: %v", errs)
	}
}

// TestStrict_UnknownContributesField rejects a nested unknown key.
func TestStrict_UnknownContributesField(t *testing.T) {
	raw := []byte(`{
		"name": "plug",
		"version": "1.0.0",
		"publisher": "acme",
		"engines": {"opendray": "^1.0.0"},
		"contributes": {
			"commands": [],
			"futureThing": []
		}
	}`)
	p := mustParseProvider(t, string(raw))
	errs := ValidateV1Strict(p, raw)
	if !hasError(errs, "contributes.futureThing") {
		t.Errorf("want unknown contributes field error, got: %v", errs)
	}
}

// TestStrict_LegacySkipsWhitelist — a legacy (non-v1) manifest with
// unknown-to-v1 fields is accepted, because IsV1()==false short-circuits.
func TestStrict_LegacySkipsWhitelist(t *testing.T) {
	raw := []byte(`{
		"name": "legacy",
		"type": "panel",
		"category": "docs",
		"legacyCustomField": true
	}`)
	p := mustParseProvider(t, string(raw))
	if p.IsV1() {
		t.Fatal("test fixture should be non-v1")
	}
	if errs := ValidateV1Strict(p, raw); len(errs) != 0 {
		t.Errorf("legacy should skip strict; got: %v", errs)
	}
}

// TestStrict_AllowsV2ReservedEscapeHatch verifies the v2Reserved field
// accepts any JSON shape without complaint (forward-compat).
func TestStrict_AllowsV2ReservedEscapeHatch(t *testing.T) {
	raw := []byte(`{
		"name": "plug",
		"version": "1.0.0",
		"publisher": "acme",
		"engines": {"opendray": "^1.0.0"},
		"v2Reserved": {"telegramCommands": [{"name":"future"}]}
	}`)
	p := mustParseProvider(t, string(raw))
	if errs := ValidateV1Strict(p, raw); len(errs) != 0 {
		t.Errorf("v2Reserved should be free-form; got: %v", errs)
	}
}

// TestStrict_SchemaAttributeAllowed — the `$schema` JSON Schema hint is
// common tooling output; strict mode must accept it.
func TestStrict_SchemaAttributeAllowed(t *testing.T) {
	raw := []byte(`{
		"$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
		"name": "plug",
		"version": "1.0.0",
		"publisher": "acme",
		"engines": {"opendray": "^1.0.0"}
	}`)
	p := mustParseProvider(t, string(raw))
	if errs := ValidateV1Strict(p, raw); len(errs) != 0 {
		t.Errorf("$schema should be whitelisted; got: %v", errs)
	}
}
