package dbtool

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opendray/opendray-v2/internal/integration"
)

// A bigint primary key above 2^53 must survive JSON decoding exactly —
// the default decoder rounds through float64, corrupting the WHERE that
// addresses a row. decodeJSON uses UseNumber and normalizeNumber keeps it
// an int64.
func TestNormalizeNumber(t *testing.T) {
	// 2^53 + 1 — the first integer float64 cannot represent exactly.
	if got := normalizeNumber(json.Number("9007199254740993")); got != int64(9007199254740993) {
		t.Fatalf("2^53+1 = %#v, want int64 9007199254740993", got)
	}
	// int64 max, nested in a map (the row-CRUD value path).
	m := normalizeNumberMap(map[string]any{"id": json.Number("9223372036854775807")})
	if m["id"] != int64(9223372036854775807) {
		t.Fatalf("int64-max map value = %#v", m["id"])
	}
	// Fractional stays float64.
	if got := normalizeNumber(json.Number("3.5")); got != 3.5 {
		t.Fatalf("3.5 = %#v", got)
	}
	// Beyond int64 (e.g. numeric) falls back to the exact string.
	if got := normalizeNumber(json.Number("99999999999999999999999999")); got != "99999999999999999999999999" {
		t.Fatalf("huge = %#v", got)
	}
	// Non-numbers pass through untouched.
	if got := normalizeNumber("alice"); got != "alice" {
		t.Fatalf("string = %#v", got)
	}
}

// The scope matrix: which principal passes which gate. The middlewares
// run before any service code, so a nil Service is fine here.
func TestScopeGates(t *testing.T) {
	h := NewHandlers(nil, nil, nil)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	admin := integration.Principal{Kind: integration.KindAdmin, ID: "admin"}
	reader := integration.Principal{Kind: integration.KindIntegration, ID: "int_r", Scopes: []string{ScopeDBRead}}
	writer := integration.Principal{Kind: integration.KindIntegration, ID: "int_w", Scopes: []string{ScopeDBRead, ScopeDBWrite}}
	stranger := integration.Principal{Kind: integration.KindIntegration, ID: "int_x", Scopes: []string{"memory:read"}}

	tests := []struct {
		name string
		gate http.Handler
		p    *integration.Principal
		want int
	}{
		{"unauthenticated read gate", h.requireScope(ScopeDBRead)(ok), nil, http.StatusUnauthorized},
		{"admin read gate", h.requireScope(ScopeDBRead)(ok), &admin, http.StatusOK},
		{"admin write gate", h.requireScope(ScopeDBWrite)(ok), &admin, http.StatusOK},
		{"admin admin gate", h.requireAdmin(ok), &admin, http.StatusOK},
		{"reader read gate", h.requireScope(ScopeDBRead)(ok), &reader, http.StatusOK},
		{"reader write gate", h.requireScope(ScopeDBWrite)(ok), &reader, http.StatusForbidden},
		{"reader admin gate", h.requireAdmin(ok), &reader, http.StatusForbidden},
		{"writer write gate", h.requireScope(ScopeDBWrite)(ok), &writer, http.StatusOK},
		{"writer admin gate", h.requireAdmin(ok), &writer, http.StatusForbidden},
		{"unrelated scopes read gate", h.requireScope(ScopeDBRead)(ok), &stranger, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/dbtool/connections", nil)
			if tt.p != nil {
				req = req.WithContext(integration.WithPrincipal(req.Context(), *tt.p))
			}
			rec := httptest.NewRecorder()
			tt.gate.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("%s: status = %d, want %d", tt.name, rec.Code, tt.want)
			}
		})
	}
}

// requireConnCwd binds a non-admin caller to its own project. These
// cases resolve before the service is touched, so a nil svc is safe:
// admin bypasses, and a non-admin without a cwd param is rejected up
// front (project-enumeration prevention).
func TestRequireConnCwd(t *testing.T) {
	h := NewHandlers(nil, nil, nil)

	tests := []struct {
		name    string
		p       *integration.Principal
		query   string
		allowed bool
	}{
		{
			"admin bypasses cwd binding",
			&integration.Principal{Kind: integration.KindAdmin},
			"", true,
		},
		{
			"non-admin without cwd rejected",
			&integration.Principal{Kind: integration.KindIntegration, Scopes: []string{ScopeDBRead}},
			"", false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x"+tt.query, nil)
			if tt.p != nil {
				req = req.WithContext(integration.WithPrincipal(req.Context(), *tt.p))
			}
			rec := httptest.NewRecorder()
			got := h.requireConnCwd(rec, req)
			if got != tt.allowed {
				t.Fatalf("requireConnCwd = %v, want %v (status %d)", got, tt.allowed, rec.Code)
			}
		})
	}
}

// The HMAC cwd proof: only a signature computed under the server secret
// for that exact cwd verifies. This is what closes the "extract key +
// forge cwd" residual for signed-key providers.
func TestVerifyCwdSig(t *testing.T) {
	secret := []byte("test-secret-0123456789abcdef")
	cwd := "/Users/x/project"
	sig := signCwd(secret, cwd)

	if !verifyCwdSig(secret, cwd, sig) {
		t.Fatal("valid signature rejected")
	}
	for name, bad := range map[string]string{
		"empty":        "",
		"tampered":     sig + "00",
		"wrong length": sig[:len(sig)-2],
		"not hex":      "zzzz",
	} {
		if verifyCwdSig(secret, cwd, bad) {
			t.Fatalf("%s signature accepted", name)
		}
	}
	if verifyCwdSig(secret, "/other/cwd", sig) {
		t.Fatal("signature for a different cwd accepted")
	}
	if verifyCwdSig([]byte("a-different-secret-value"), cwd, sig) {
		t.Fatal("signature under a different secret accepted")
	}
}

// A db:signed key (per-session-config provider) MUST present a valid
// signature; a missing or wrong one is rejected before the service is
// touched (so a nil svc is safe here).
func TestRequireConnCwdSignatureEnforced(t *testing.T) {
	secret := []byte("sign-secret-abcdefghijklmnop")
	h := NewHandlers(nil, secret, nil)
	signed := integration.Principal{
		Kind:   integration.KindIntegration,
		Scopes: []string{ScopeDBRead, ScopeDBSigned},
	}

	cases := map[string]string{
		"no signature":    "",
		"wrong signature": "deadbeef",
	}
	for name, sig := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x?cwd=/proj", nil)
			if sig != "" {
				req.Header.Set(cwdSigHeader, sig)
			}
			req = req.WithContext(integration.WithPrincipal(req.Context(), signed))
			rec := httptest.NewRecorder()
			if h.requireConnCwd(rec, req) {
				t.Fatalf("%s: signed key was allowed", name)
			}
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s: status = %d, want 403", name, rec.Code)
			}
		})
	}
}

// listConnections must also enforce the signature for db:signed keys —
// otherwise a signed key could enumerate another project's connection
// metadata with just a guessed cwd. Rejection happens before the service
// is touched, so a nil svc is safe.
func TestListConnectionsSignatureEnforced(t *testing.T) {
	secret := []byte("sign-secret-abcdefghijklmnop")
	h := NewHandlers(nil, secret, nil)
	signed := integration.Principal{
		Kind:   integration.KindIntegration,
		Scopes: []string{ScopeDBRead, ScopeDBSigned},
	}
	req := httptest.NewRequest(http.MethodGet, "/dbtool/connections?cwd=/proj", nil)
	req = req.WithContext(integration.WithPrincipal(req.Context(), signed))
	rec := httptest.NewRecorder()
	h.listConnections(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("signed list without signature: status = %d, want 403", rec.Code)
	}
}
