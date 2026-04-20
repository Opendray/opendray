package gateway

// Marketplace HTTP surface.
//
//	GET /api/marketplace/plugins → {"entries":[...]}
//
// The catalog is loaded once at server boot from Server.marketplace
// (a market.Catalog implementation — local in M3, remote in M4). A
// nil catalog degrades to an empty entries list so clients can still
// render the Hub page without crashing on un-seeded hosts.
//
// Install still goes through POST /api/plugins/install with
// `src: "marketplace://..."`. The install handler resolves that ref
// against the same catalog and either copies a local bundle or
// HTTPSSource-downloads the artifact, depending on which backend
// owns the catalog.

import (
	"encoding/json"
	"net/http"

	"github.com/opendray/opendray/plugin/market"
)

// marketplaceListResponse is the wire shape for GET /api/marketplace/plugins.
// Kept as a local type so the JSON contract is visible from the handler.
type marketplaceListResponse struct {
	Entries []market.Entry `json:"entries"`
}

// marketplaceList handles GET /api/marketplace/plugins.
//
// Returns every Entry the catalog knows about in a stable order
// (publisher, name, version). An absent or nil catalog returns an
// empty list with 200 so clients don't have to branch on 404 — an
// un-seeded host just has nothing to show.
func (s *Server) marketplaceList(w http.ResponseWriter, r *http.Request) {
	var entries []market.Entry
	if s.marketplace != nil {
		got, err := s.marketplace.List(r.Context())
		if err != nil {
			// Surface fetch errors (remote backend) as a 503 so
			// clients render a retry banner rather than an empty list
			// they'd mistake for "no plugins published".
			writeJSONError(w, http.StatusServiceUnavailable, "EMARKET",
				"marketplace list failed: "+err.Error())
			return
		}
		entries = got
	}
	if entries == nil {
		entries = []market.Entry{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(marketplaceListResponse{Entries: entries})
}

// cacheInvalidator is any catalog that exposes an explicit cache
// drop — market/remote.Catalog satisfies this via InvalidateCache.
// local.Catalog doesn't need to (no cache) and returns nil for the
// type assertion.
type cacheInvalidator interface {
	InvalidateCache()
}

// marketplaceRefresh handles POST /api/marketplace/refresh.
//
// Drops the in-memory catalog cache so the next List / Resolve
// re-fetches from the registry. Fires the Settings → Marketplace
// "Refresh cache now" button; also called by the revocation
// poller when it wants a fresh sweep outside the tick window.
//
// Always returns 200 — even when the catalog is nil or has no
// cache — so clients don't have to branch on "nothing to refresh".
func (s *Server) marketplaceRefresh(w http.ResponseWriter, _ *http.Request) {
	if ci, ok := s.marketplace.(cacheInvalidator); ok {
		ci.InvalidateCache()
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"refreshed": true})
}
