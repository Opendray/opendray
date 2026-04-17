package gateway

import (
	"net"
	"net/http"
	"regexp"
	"strings"
)

// DiscoveredURL is a URL found in a running terminal session's output buffer.
type DiscoveredURL struct {
	URL         string `json:"url"`
	SessionName string `json:"sessionName"`
}

var (
	// ansiRe strips ANSI/VT100 escape sequences from terminal output.
	ansiRe = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[A-Za-z]|\][^\x07]*\x07|[^[\]])`)

	// urlRe matches http/https URLs including query strings (e.g. Flutter DevTools ?uri=ws://...).
	urlRe = regexp.MustCompile(`https?://[^\s\x1b"'<>\x00-\x1f]+`)
)

// previewDiscover scans all running terminal session buffers for URLs and
// returns them after replacing 127.0.0.1/localhost with the server's IP.
func (s *Server) previewDiscover(w http.ResponseWriter, r *http.Request) {
	// Derive the IP the client actually used to reach us (from Host header).
	serverHost, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		serverHost = r.Host // Host header has no port
	}

	ctx := r.Context()
	sessions, err := s.hub.List(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "list sessions: "+err.Error())
		return
	}

	seen := make(map[string]bool)
	var results []DiscoveredURL

	for _, sess := range sessions {
		ts, ok := s.hub.GetTerminalSession(sess.ID)
		if !ok {
			continue // session not currently running
		}

		raw := ansiRe.ReplaceAllString(string(ts.Buffer().Snapshot()), "")

		for _, u := range urlRe.FindAllString(raw, -1) {
			// Trim trailing punctuation that the regex may have swallowed.
			u = strings.TrimRight(u, ".,;:)]}\"'\\")

			// Replace local-only addresses with the server's reachable IP so
			// the mobile client can actually open the URL.
			u = strings.ReplaceAll(u, "127.0.0.1", serverHost)
			u = strings.ReplaceAll(u, "localhost", serverHost)

			if seen[u] {
				continue
			}
			seen[u] = true

			name := sess.Name
			if name == "" {
				name = sess.ID[:8]
			}
			results = append(results, DiscoveredURL{URL: u, SessionName: name})
		}
	}

	if results == nil {
		results = []DiscoveredURL{}
	}
	respondJSON(w, http.StatusOK, results)
}
