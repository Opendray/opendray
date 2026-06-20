package session

import (
	"database/sql"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo), name "sqlite"
)

// Antigravity (agy) stores each conversation as a standalone SQLite
// database at <conversationsRoot>/<trajectory-uuid>.db. The schema is
// agy-internal and undocumented; the conversation lives in the `steps`
// table, one row per turn/action, with a `step_type` discriminator and
// a protobuf-encoded `step_payload` blob.
//
// We only reconstruct the human-readable conversation (user prompts +
// assistant prose), mirroring the other providers' Turn output — tool
// calls, results, thinking signatures and the encrypted payload fields
// are deliberately dropped. The blob fields we read are plaintext; the
// step_type values we care about, reverse-engineered against live data:
//
//	14 → user message   (top-level field 19 → sub-field 2: text)
//	15 → assistant turn (top-level field 20 → sub-field 3: prose)
//
// Most type-15 steps are silent tool-call turns with no prose (field
// 20→3 empty); only the steps where the model narrates carry text, so
// a reconstructed transcript is intentionally sparser than the raw
// step count. Tool-call rendering is deliberately out of scope here,
// matching the conversation-only Turn contract used by the summariser.
//
// Timestamps live under top-level field 5 → field 1 → {1: unix secs,
// 2: nanos}. All other step types (tool calls 9/17, workflow markers
// 23, metadata 8/21/51/98/132…) are skipped for the transcript.
//
// CAVEAT: agy's database records NO working directory — only ephemeral
// /tmp scratch dirs — so unlike claude/codex we cannot match a
// session to its db by cwd. We match purely by time: the newest .db
// whose mtime falls inside the session's [StartedAt, EndedAt] window.
// Good enough for a single operator; concurrent agy sessions in the
// same window are ambiguous (documented limitation).

// AntigravityHistoryConfig points at agy's per-conversation SQLite store.
// Default: ~/.gemini/antigravity-cli/conversations.
type AntigravityHistoryConfig struct {
	ConversationsRoot string
}

// agy step_type discriminators for conversational turns.
const (
	agyStepUser      = 14
	agyStepAssistant = 15
)

// antigravityTranscript reconstructs the user + assistant turns of the
// agy conversation whose database best matches the session's time
// window. cwd is accepted for signature symmetry with the other
// providers but unused — agy does not persist a working directory.
func antigravityTranscript(cfg AntigravityHistoryConfig, _ string, startedAt, endedAt time.Time, maxBytes int) []Turn {
	path := resolveAntigravityDB(cfg, startedAt, endedAt)
	if path == "" {
		return nil
	}
	// immutable=1: read a consistent snapshot without taking any lock,
	// so we never contend with a live agy process holding the db open.
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&immutable=1")
	if err != nil {
		return nil
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query("SELECT step_type, step_payload FROM steps ORDER BY idx")
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var turns []Turn
	bytesUsed := 0
	for rows.Next() {
		var stepType int
		var payload []byte
		if err := rows.Scan(&stepType, &payload); err != nil {
			continue
		}
		var role, text string
		switch stepType {
		case agyStepUser:
			role = "user"
			msg := pbBytes(payload, 19)
			text = string(pbBytes(msg, 2))
			if text == "" {
				text = string(pbBytes(pbBytes(msg, 3), 1))
			}
		case agyStepAssistant:
			role = "assistant"
			text = string(pbBytes(pbBytes(payload, 20), 3))
		default:
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		bytesUsed += len(text) + len(role) + 4
		if bytesUsed > maxBytes && len(turns) > 0 {
			turns = trimTurnsHead(turns, &bytesUsed, maxBytes)
		}
		turns = append(turns, Turn{Role: role, Text: text, Ts: agyStepTimestamp(payload)})
	}
	if err := rows.Err(); err != nil {
		return turns
	}
	return turns
}

// resolveAntigravityDB returns the path of the conversation database
// most likely to belong to this session: the newest .db whose mtime is
// inside [startedAt-margin, endedAt+margin]. A 2-minute margin absorbs
// clock skew and the gap between spawn and the first persisted step.
func resolveAntigravityDB(cfg AntigravityHistoryConfig, startedAt, endedAt time.Time) string {
	root := cfg.ConversationsRoot
	if root == "" {
		if home := os.Getenv("HOME"); home != "" {
			root = filepath.Join(home, ".gemini", "antigravity-cli", "conversations")
		}
	}
	if root == "" {
		return ""
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	var winStart, winEnd time.Time
	if !startedAt.IsZero() {
		winStart = startedAt.Add(-2 * time.Minute)
	}
	if !endedAt.IsZero() {
		winEnd = endedAt.Add(2 * time.Minute)
	}
	var bestPath string
	var bestMod time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		mt := info.ModTime()
		if !winStart.IsZero() && mt.Before(winStart) {
			continue
		}
		if !winEnd.IsZero() && mt.After(winEnd) {
			continue
		}
		if bestPath == "" || mt.After(bestMod) {
			bestPath = filepath.Join(root, e.Name())
			bestMod = mt
		}
	}
	return bestPath
}

// agyStepTimestamp pulls the step's creation time from the protobuf
// metadata at field 5 → field 1 → {1: unix secs, 2: nanos}. Returns
// the zero time when the field is absent or malformed.
func agyStepTimestamp(payload []byte) time.Time {
	tsMsg := pbBytes(pbBytes(payload, 5), 1)
	secs, ok := pbVarint(tsMsg, 1)
	if !ok {
		return time.Time{}
	}
	nanos, _ := pbVarint(tsMsg, 2)
	// The blob is untrusted; clamp an out-of-range sub-second value so a
	// garbage varint can't produce a wildly wrong (pre-epoch) timestamp.
	if nanos > 999_999_999 {
		nanos = 0
	}
	return time.Unix(int64(secs), int64(nanos))
}

// ── minimal protobuf wire reader ──────────────────────────────────
//
// agy's payloads are protobuf with no available schema, so we parse by
// raw field number. Only the two wire types we need are decoded
// (varint, length-delimited); 64-bit and 32-bit fields are skipped,
// and any malformed tail stops the scan rather than panicking.

const (
	wireVarint = 0
	wireI64    = 1
	wireLen    = 2
	wireI32    = 5
)

// pbScan walks the top-level fields of a protobuf message, invoking fn
// for each varint and length-delimited field. fn returns false to stop
// early. For wireLen fields `data` is the payload; for wireVarint `v`
// holds the value.
func pbScan(b []byte, fn func(field, wire int, data []byte, v uint64) bool) {
	for len(b) > 0 {
		tag, n := binary.Uvarint(b)
		if n <= 0 {
			return
		}
		b = b[n:]
		field := int(tag >> 3)
		wire := int(tag & 7)
		switch wire {
		case wireVarint:
			v, m := binary.Uvarint(b)
			if m <= 0 {
				return
			}
			b = b[m:]
			if !fn(field, wire, nil, v) {
				return
			}
		case wireI64:
			if len(b) < 8 {
				return
			}
			b = b[8:]
		case wireLen:
			l, m := binary.Uvarint(b)
			if m <= 0 {
				return
			}
			b = b[m:]
			if l > uint64(len(b)) {
				return
			}
			data := b[:l]
			b = b[l:]
			if !fn(field, wire, data, 0) {
				return
			}
		case wireI32:
			if len(b) < 4 {
				return
			}
			b = b[4:]
		default:
			// Wire types 3/4 (deprecated groups) are variable-length and
			// can't be skipped without recursive parsing; 6/7 are
			// reserved. Any of them means we can no longer locate field
			// boundaries safely, so stop the scan rather than risk
			// misreading. (None appear in agy payloads today.)
			return
		}
	}
}

// pbBytes returns the payload of the first length-delimited field with
// the given number, or nil. Callers cast to string for text fields or
// recurse into it for nested messages.
func pbBytes(b []byte, field int) []byte {
	var out []byte
	pbScan(b, func(f, w int, data []byte, _ uint64) bool {
		if f == field && w == wireLen {
			out = data
			return false
		}
		return true
	})
	return out
}

// pbVarint returns the value of the first varint field with the given
// number, and whether it was present.
func pbVarint(b []byte, field int) (uint64, bool) {
	var out uint64
	var found bool
	pbScan(b, func(f, w int, _ []byte, v uint64) bool {
		if f == field && w == wireVarint {
			out = v
			found = true
			return false
		}
		return true
	})
	return out, found
}
