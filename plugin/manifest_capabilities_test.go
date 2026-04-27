package plugin

import (
	"encoding/json"
	"testing"
)

// TestContributesV1_CapabilityRoundTrip verifies the M6 capability
// fields survive JSON marshal/unmarshal with their full shape intact.
func TestContributesV1_CapabilityRoundTrip(t *testing.T) {
	in := ContributesV1{
		Providers: []ProviderContributionV1{
			{
				ID:          "anthropic",
				DisplayName: "Anthropic",
				Description: "Claude family of models",
				Icon:        "anthropic.svg",
				Categories:  []string{"cloud", "vision"},
			},
		},
		Channels: []ChannelContributionV1{
			{ID: "telegram", DisplayName: "Telegram"},
		},
		Forges: []ForgeContributionV1{
			{ID: "github", DisplayName: "GitHub", Kind: "github"},
		},
		McpServers: []McpServerContributionV1{
			{ID: "filesystem", DisplayName: "Filesystem"},
		},
	}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out ContributesV1
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(out.Providers) != 1 || out.Providers[0].ID != "anthropic" {
		t.Errorf("providers lost: %+v", out.Providers)
	}
	if got := out.Providers[0].Categories; len(got) != 2 || got[0] != "cloud" {
		t.Errorf("provider categories lost: %+v", got)
	}
	if len(out.Channels) != 1 || out.Channels[0].ID != "telegram" {
		t.Errorf("channels lost: %+v", out.Channels)
	}
	if len(out.Forges) != 1 || out.Forges[0].Kind != "github" {
		t.Errorf("forges lost: %+v", out.Forges)
	}
	if len(out.McpServers) != 1 || out.McpServers[0].ID != "filesystem" {
		t.Errorf("mcpServers lost: %+v", out.McpServers)
	}
}

// TestContributesV1_OmitEmpty verifies that empty capability slices do
// not show up in the marshalled JSON — the M6 fields are additive and
// must not pollute existing manifests' on-disk representation.
func TestContributesV1_OmitEmpty(t *testing.T) {
	in := ContributesV1{
		// All capability fields zero.
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)
	for _, key := range []string{"providers", "channels", "forges", "mcpServers"} {
		if contains(got, "\""+key+"\"") {
			t.Errorf("expected omitempty to suppress %q in output: %s", key, got)
		}
	}
}

// TestStrictAcceptsCapabilityFields verifies the M6 additions are now
// in the unknown-field whitelist so manifests carrying them don't get
// rejected at install time.
func TestStrictAcceptsCapabilityFields(t *testing.T) {
	manifest := []byte(`{
		"name": "anthropic",
		"version": "0.1.0",
		"publisher": "opendray",
		"engines": {"opendray": "^1.0.0"},
		"contributes": {
			"providers": [{"id": "anthropic"}],
			"channels": [{"id": "telegram"}],
			"forges": [{"id": "github"}],
			"mcpServers": [{"id": "filesystem"}]
		}
	}`)
	var p Provider
	if err := json.Unmarshal(manifest, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	errs := ValidateV1Strict(p, manifest)
	for _, e := range errs {
		// We only care that the M6 keys are not flagged. Other rule-
		// based errors (missing fields, bad versions) may still appear
		// but should not mention our new keys.
		for _, key := range []string{"providers", "channels", "forges", "mcpServers"} {
			if e.Path == "contributes."+key {
				t.Errorf("M6 key %q rejected by strict validator: %s", key, e.Msg)
			}
		}
	}
}

// TestStrictRejectsUnknownContributesField is a sanity guard — if it
// breaks, the whitelist mechanism itself regressed and the M6 ACCEPT
// test above proves nothing.
func TestStrictRejectsUnknownContributesField(t *testing.T) {
	manifest := []byte(`{
		"name": "x",
		"version": "0.1.0",
		"publisher": "opendray",
		"engines": {"opendray": "^1.0.0"},
		"contributes": {
			"thisFieldShouldNeverExist": []
		}
	}`)
	var p Provider
	_ = json.Unmarshal(manifest, &p)
	errs := ValidateV1Strict(p, manifest)
	found := false
	for _, e := range errs {
		if e.Path == "contributes.thisFieldShouldNeverExist" {
			found = true
		}
	}
	if !found {
		t.Errorf("strict validator failed to flag unknown contributes field; errs=%+v", errs)
	}
}

// helper
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
