// Package auth provides identity, scopes, and rate limiting for the gateway.
//
// Responsibilities (per design §8.6):
//   - Admin auth (single human; basic auth or signed cookie).
//   - Integration API key issuance, hashing, verification, scope check.
//   - Per-integration quota counters with periodic DB flush.
//
// Implementation lands alongside M3 (integration registry).
package auth
