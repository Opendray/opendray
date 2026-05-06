// Package wsutil holds tiny helpers shared across opendray's
// websocket endpoints. The first inhabitant is the origin checker
// used by every Upgrader to mitigate cross-site WS hijacking.
package wsutil

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// SameOriginCheck returns a websocket.Upgrader.CheckOrigin function
// suitable for browser-facing WS endpoints (admin SPA). It accepts:
//
//   - Empty Origin header (non-browser clients: curl, server-to-server,
//     mobile native — these can't be tricked by a malicious page).
//   - Origin host equal to the request's Host header (same-origin).
//   - Origin host that is a loopback or RFC1918 / RFC4193 private
//     address (LAN deployment is the canonical opendray topology).
//   - Origin host present in the optional `extra` allow-list (exact
//     match on host[:port]).
//
// This is defence-in-depth on top of bearer-token auth. Token in the
// `?token=` query string is the primary authentication; this guard
// blocks Cross-Site WebSocket Hijacking *if* the token ever leaks into
// a referrer or document.URL on a third-party page.
//
// For non-browser endpoints (bridge adapters, integration server-to-
// server consumers) prefer AllowAnyOrigin so legitimate non-browser
// clients aren't broken.
func SameOriginCheck(extra ...string) func(*http.Request) bool {
	allow := make(map[string]struct{}, len(extra))
	for _, h := range extra {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			allow[h] = struct{}{}
		}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		host := strings.ToLower(u.Host)
		if host == strings.ToLower(r.Host) {
			return true
		}
		if _, ok := allow[host]; ok {
			return true
		}
		// Strip the port for IP-based checks. Hostnames like
		// "localhost" are handled here too.
		hostname := u.Hostname()
		if hostname == "localhost" {
			return true
		}
		if ip := net.ParseIP(hostname); ip != nil {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
				return true
			}
		}
		return false
	}
}

// AllowAnyOrigin returns a CheckOrigin that always returns true. Use
// for WS endpoints whose primary auth is a per-frame token *and* whose
// legitimate clients are non-browser (e.g. external bridge adapters
// running in Python, or integration apps doing server-to-server WS).
//
// Centralising the bypass through this helper makes the intent
// explicit and greppable — every call site documents *why* it doesn't
// need an origin check.
func AllowAnyOrigin() func(*http.Request) bool {
	return func(*http.Request) bool { return true }
}
