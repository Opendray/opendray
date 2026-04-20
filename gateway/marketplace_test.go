package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/opendray/opendray/plugin/marketplace"
)

func writeTestCatalog(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestMarketplaceList_Empty — a nil-catalog server still answers with
// 200 + an empty entries list so the Flutter Hub page doesn't need a
// 404 branch for un-seeded deployments.
func TestMarketplaceList_Empty(t *testing.T) {
	s := &Server{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/marketplace/plugins", nil)
	s.marketplaceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp marketplaceListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body.String())
	}
	if len(resp.Entries) != 0 {
		t.Errorf("want empty entries, got %d", len(resp.Entries))
	}
}

// TestMarketplaceList_Happy — a loaded catalog is surfaced verbatim
// through the handler. Verifies field-level fidelity including the
// permissions passthrough the Hub card uses for its consent preview.
func TestMarketplaceList_Happy(t *testing.T) {
	dir := writeTestCatalog(t, `{
		"entries": [
			{
				"name": "fs-readme",
				"version": "1.0.0",
				"publisher": "opendray-examples",
				"displayName": "FS Readme",
				"description": "reads README",
				"icon": "📖",
				"form": "host",
				"tags": ["reference"],
				"permissions": {"fs": {"read": ["${home}/**"]}}
			}
		]
	}`)
	cat, err := marketplace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{marketplace: cat}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/marketplace/plugins", nil)
	s.marketplaceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	var resp marketplaceListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body.String())
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(resp.Entries))
	}
	got := resp.Entries[0]
	if got.Name != "fs-readme" || got.Version != "1.0.0" {
		t.Errorf("entry name/version = %q/%q", got.Name, got.Version)
	}
	if got.Form != "host" {
		t.Errorf("form = %q, want host", got.Form)
	}
	if len(got.Permissions) == 0 {
		t.Errorf("permissions: expected passthrough, got empty")
	}
}
