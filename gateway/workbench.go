package gateway

// T9 — Workbench contributions endpoint.
//
// GET /api/workbench/contributions
//
// Returns the current FlatContributions view from the in-memory contribution
// registry. The handler is pure marshal — all ordering guarantees are
// provided by Registry.Flatten(). No DB access, no pagination.
//
// Defensive: if s.contribReg is nil (Server constructed without Contributions
// config field), the handler returns 503 EREGISTRY instead of panicking.

import (
	"encoding/json"
	"net/http"
)

// workbenchContributions handles GET /api/workbench/contributions.
//
// Response contract:
//   - 200 OK on success, Content-Type: application/json; charset=utf-8
//   - Empty registry → 200 with {"commands":[],"statusBar":[],"keybindings":[],"menus":{}}
//   - nil registry → 503 {"code":"EREGISTRY","msg":"workbench registry not wired"}
func (s *Server) workbenchContributions(w http.ResponseWriter, r *http.Request) {
	if s.contribReg == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "EREGISTRY",
			"workbench registry not wired")
		return
	}

	flat := s.contribReg.Flatten()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(flat) //nolint:errcheck
}
