package gateway

// Marketplace HTTP surface.
//
//	GET /api/marketplace/plugins → {"entries":[...]}
//
// The catalog is loaded once at server boot from Server.marketplace; a
// nil catalog degrades to an empty entries list so clients can still
// render the Hub page without crashing on un-seeded hosts.
//
// Install still goes through POST /api/plugins/install with
// `src: "marketplace://<name>"`. The install handler resolves that ref
// against the same catalog, constructs a TrustedSource with the
// resolved bundle dir, and feeds it to the existing Installer — no
// parallel install pipeline.

import (
	"encoding/json"
	"net/http"

	"github.com/opendray/opendray/plugin/marketplace"
)

// marketplaceListResponse is the wire shape for GET /api/marketplace/plugins.
// Kept as a local type so the JSON contract is visible from the handler.
type marketplaceListResponse struct {
	Entries []marketplace.Entry `json:"entries"`
}

// marketplaceList handles GET /api/marketplace/plugins.
//
// Returns every Entry the catalog knows about in a stable order
// (alphabetical by name, then version). An absent or nil catalog
// returns an empty list with 200 so clients don't have to branch on
// 404 — an un-seeded host just has nothing to show.
func (s *Server) marketplaceList(w http.ResponseWriter, r *http.Request) {
	var entries []marketplace.Entry
	if s.marketplace != nil {
		entries = s.marketplace.List()
	}
	if entries == nil {
		entries = []marketplace.Entry{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(marketplaceListResponse{Entries: entries})
}
