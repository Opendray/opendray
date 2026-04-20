package gateway

import (
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// originPolicy decides which browser Origins may issue cross-origin API
// calls and open WebSocket connections. Native clients (iOS / Android /
// curl) never send an Origin header and are always allowed — CSRF is a
// browser-only concern.
//
// Input forms:
//   nil / empty slice   → same-origin only (no Allow-Origin header set,
//                         WebSocket CheckOrigin falls back to Host match)
//   []string{"*"}       → wildcard; any origin. Logged loudly at startup.
//   specific origins    → exact-match allowlist; Allow-Origin echoed back.
type originPolicy struct {
	wildcard bool
	origins  map[string]struct{}
}

func newOriginPolicy(allowed []string) *originPolicy {
	p := &originPolicy{origins: make(map[string]struct{})}
	for _, raw := range allowed {
		o := strings.TrimRight(strings.TrimSpace(raw), "/")
		if o == "" {
			continue
		}
		if o == "*" {
			p.wildcard = true
			continue
		}
		p.origins[o] = struct{}{}
	}
	return p
}

// allowCORS returns the exact string to echo in Access-Control-Allow-Origin,
// or "" for "don't set the header" (browser falls back to same-origin).
func (p *originPolicy) allowCORS(origin string) string {
	if origin == "" {
		return ""
	}
	if p.wildcard {
		return "*"
	}
	if _, ok := p.origins[strings.TrimRight(origin, "/")]; ok {
		return origin
	}
	return ""
}

// allowWS is the gorilla/websocket CheckOrigin implementation. Empty
// Origin (mobile apps) passes; allowlist matches pass; otherwise same-
// origin (Origin's host == request Host) passes. Everything else is
// rejected at Upgrade time.
func (p *originPolicy) allowWS(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if p.wildcard {
		return true
	}
	if _, ok := p.origins[strings.TrimRight(origin, "/")]; ok {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == r.Host
}

// corsMiddleware returns the chi-compatible middleware that stamps CORS
// headers on allowed cross-origin requests. Requests from disallowed
// origins pass through without Allow-Origin so the browser blocks the
// response under the default same-origin rule.
func (p *originPolicy) corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allow := p.allowCORS(r.Header.Get("Origin")); allow != "" {
				w.Header().Set("Access-Control-Allow-Origin", allow)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// logStartup records the effective policy on boot — users need to see
// which origins are live and, crucially, a loud warning when they've
// opted into wildcard.
func (p *originPolicy) logStartup(logger *slog.Logger) {
	switch {
	case p.wildcard:
		logger.Warn("ALLOWED_ORIGINS=* — any website can open cross-origin API calls and WebSockets against this server. Replace with an explicit list, or run OpenDray behind an authenticating reverse proxy.")
	case len(p.origins) == 0:
		logger.Info("cross-origin policy: same-origin only (set ALLOWED_ORIGINS to permit browsers on other hosts)")
	default:
		os := make([]string, 0, len(p.origins))
		for o := range p.origins {
			os = append(os, o)
		}
		sort.Strings(os)
		logger.Info("cross-origin policy: allowlist", "origins", strings.Join(os, ","))
	}
}

