// Package session owns the lifecycle of PTY processes for AI CLI agents.
//
// Responsibilities (per design §8.1):
//   - Spawn / list / terminate PTY-backed CLI sessions.
//   - Maintain a ring buffer of recent terminal output for resume.
//   - Detect idle / exit and publish session.* events on the bus.
//
// Implementation lands in M1. This file declares the package boundary so
// app/ can import it once the constructor exists.
package session
