// Package auth provides JWT authentication for NTC.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Claims represents JWT payload claims.
type Claims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

// Auth handles JWT signing and verification.
type Auth struct {
	secret []byte
	ttl    time.Duration
}

// New creates an Auth with the given secret and token TTL.
func New(secret string, ttl time.Duration) *Auth {
	return &Auth{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Issue creates a signed JWT for the given subject.
func (a *Auth) Issue(subject string) (string, error) {
	header := base64url(mustJSON(map[string]string{"alg": "HS256", "typ": "JWT"}))

	now := time.Now()
	claims := Claims{
		Sub: subject,
		Iat: now.Unix(),
		Exp: now.Add(a.ttl).Unix(),
	}
	payload := base64url(mustJSON(claims))

	sig := a.sign(header + "." + payload)
	return header + "." + payload + "." + sig, nil
}

// Verify parses and validates a JWT, returning the claims.
func (a *Auth) Verify(token string) (Claims, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("auth: malformed token")
	}

	expectedSig := a.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return Claims{}, fmt.Errorf("auth: invalid signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("auth: decode payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Claims{}, fmt.Errorf("auth: parse claims: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return Claims{}, fmt.Errorf("auth: token expired")
	}
	return claims, nil
}

// Middleware returns an HTTP middleware that enforces JWT auth.
// Tokens are read from the Authorization header (Bearer) or the "token" query parameter.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
			return
		}

		claims, err := a.Verify(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		r.Header.Set("X-User", claims.Sub)
		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	// Authorization: Bearer <token>
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Query parameter fallback (for WebSocket connections)
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	return ""
}

func (a *Auth) sign(data string) string {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
