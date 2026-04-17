// Package llmproxy holds the Anthropic↔OpenAI translation primitives.
//
// The HTTP handler that used to wrap these (proxy.go + route
// registration in gateway/server.go) was removed after the "Claude CLI
// pointed at local model" experiment was rolled back. The translator
// and SSE bridge are kept here because they are pure, DB-free
// functions useful for any future integration that needs the same
// format bridge (e.g. proxying a third-party Anthropic-compatible
// service, or an internal route that maps Anthropic calls to an
// OpenAI-shaped downstream).
package llmproxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// now is a small indirection so the test suite can freeze time.
var now = func() time.Time { return time.Now() }

// randID returns a short hex identifier used for message / tool_use IDs.
// Not a UUID — these IDs only need to be collision-free inside one
// conversation. 16 random bytes is plenty.
func randID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is catastrophic on this platform; fall
		// back to a timestamp-derived ID that's still unique per call.
		return fmt.Sprintf("%x", now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
