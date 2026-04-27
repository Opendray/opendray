// Package audit subscribes to selected event-bus topics and writes them to
// the audit_log table.
//
// Responsibilities (per design §6 ordering, §8.5 + §10 schema):
//   - Subscribe to admin actions, integration registrations, key rotations.
//   - Persist as structured rows; do not store PII / message bodies.
//
// Implementation lands alongside M3.
package audit
