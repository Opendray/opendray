package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"context"

	"github.com/opendray/opendray/plugin/market"
	marketlocal "github.com/opendray/opendray/plugin/market/local"
)

// cachingStub is a Catalog fake that records InvalidateCache calls,
// used to assert the refresh endpoint actually invalidates a
// remote-backed catalog. All other methods are zero-valued.
type cachingStub struct {
	invalidated int
}

func (c *cachingStub) List(_ context.Context) ([]market.Entry, error) {
	return nil, nil
}
func (c *cachingStub) Resolve(_ context.Context, _ market.Ref) (market.Entry, error) {
	return market.Entry{}, nil
}
func (c *cachingStub) BundlePath(_ context.Context, _ market.Ref) (string, bool, error) {
	return "", false, nil
}
func (c *cachingStub) FetchPublisher(_ context.Context, _ string) (market.PublisherRecord, error) {
	return market.PublisherRecord{}, nil
}
func (c *cachingStub) FetchRevocations(_ context.Context) ([]byte, error) {
	return nil, nil
}
func (c *cachingStub) InvalidateCache() { c.invalidated++ }

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

// TestMarketplaceRefresh_InvalidatesCache — POST /refresh calls
// InvalidateCache on catalogs that support it. Fires Settings →
// Marketplace "Refresh cache now" + the revocation poller's
// post-action invalidation.
func TestMarketplaceRefresh_InvalidatesCache(t *testing.T) {
	stub := &cachingStub{}
	s := &Server{marketplace: stub}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/marketplace/refresh", nil)
	s.marketplaceRefresh(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if stub.invalidated != 1 {
		t.Errorf("InvalidateCache called %d times, want 1", stub.invalidated)
	}
}

// TestMarketplaceRefresh_NoCatalogNoop — a nil catalog still
// returns 200 so clients don't have to branch.
func TestMarketplaceRefresh_NoCatalogNoop(t *testing.T) {
	s := &Server{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/marketplace/refresh", nil)
	s.marketplaceRefresh(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want 200 on nil catalog, got %d", rr.Code)
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
	cat, err := marketlocal.Load(dir)
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
