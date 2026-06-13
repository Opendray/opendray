package session

import (
	"database/sql"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── protobuf encode helpers (mirror the wire format the reader
// decodes; kept in the test so the production file stays decode-only) ──

func pbEncTag(field, wire int) []byte {
	return binary.AppendUvarint(nil, uint64(field)<<3|uint64(wire))
}

func pbEncStr(field int, s string) []byte {
	b := pbEncTag(field, wireLen)
	b = binary.AppendUvarint(b, uint64(len(s)))
	return append(b, s...)
}

func pbEncMsg(field int, sub []byte) []byte {
	b := pbEncTag(field, wireLen)
	b = binary.AppendUvarint(b, uint64(len(sub)))
	return append(b, sub...)
}

func pbEncVarint(field int, v uint64) []byte {
	b := pbEncTag(field, wireVarint)
	return binary.AppendUvarint(b, v)
}

// agyMeta builds the field-5 metadata carrying a creation timestamp at
// 5 → 1 → {1: secs, 2: nanos}, matching the real agy step layout.
func agyMeta(ts time.Time) []byte {
	inner := append(pbEncVarint(1, uint64(ts.Unix())), pbEncVarint(2, uint64(ts.Nanosecond()))...)
	return pbEncMsg(5, pbEncMsg(1, inner))
}

func agyUserStep(ts time.Time, text string) []byte {
	// field 19 → field 2: user text
	return append(agyMeta(ts), pbEncMsg(19, pbEncStr(2, text))...)
}

func agyAssistantStep(ts time.Time, text string) []byte {
	// field 20 → field 3: assistant prose
	return append(agyMeta(ts), pbEncMsg(20, pbEncStr(3, text))...)
}

// writeAgyDB creates a <root>/<name>.db with the minimal steps schema
// the reader queries, populated from the given rows, and stamps its
// mtime so time-window matching can find it.
func writeAgyDB(t *testing.T, root, name string, mtime time.Time, rows []struct {
	idx      int
	stepType int
	payload  []byte
}) string {
	t.Helper()
	path := filepath.Join(root, name)
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE steps (idx integer PRIMARY KEY, step_type integer NOT NULL DEFAULT 0, step_payload blob)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO steps(idx, step_type, step_payload) VALUES(?,?,?)`, r.idx, r.stepType, r.payload); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return path
}

func TestAntigravityTranscript(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-time.Minute)
	rows := []struct {
		idx      int
		stepType int
		payload  []byte
	}{
		{0, agyStepUser, agyUserStep(base, "hello world")},
		{1, agyStepAssistant, agyAssistantStep(base.Add(time.Second), "hi there")},
		{2, 9, []byte{0x08, 0x09}},                                       // tool call → skipped
		{3, agyStepAssistant, agyAssistantStep(base.Add(2*time.Second), "   ")}, // blank → skipped
		{4, agyStepUser, agyUserStep(base.Add(3*time.Second), "second question")},
	}
	writeAgyDB(t, root, "conv-a.db", time.Now(), rows)

	cfg := AntigravityHistoryConfig{ConversationsRoot: root}
	turns := antigravityTranscript(cfg, "", time.Now().Add(-2*time.Minute), time.Time{}, 16*1024)

	want := []struct {
		role, text string
	}{
		{"user", "hello world"},
		{"assistant", "hi there"},
		{"user", "second question"},
	}
	if len(turns) != len(want) {
		t.Fatalf("got %d turns, want %d: %+v", len(turns), len(want), turns)
	}
	for i, w := range want {
		if turns[i].Role != w.role || turns[i].Text != w.text {
			t.Errorf("turn %d = (%q,%q), want (%q,%q)", i, turns[i].Role, turns[i].Text, w.role, w.text)
		}
		if turns[i].Ts.IsZero() {
			t.Errorf("turn %d has zero timestamp", i)
		}
	}
}

func TestResolveAntigravityDBPicksNewestInWindow(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	one := []struct {
		idx      int
		stepType int
		payload  []byte
	}{{0, agyStepUser, agyUserStep(now, "x")}}

	writeAgyDB(t, root, "old.db", now.Add(-10*time.Minute), one)   // before window
	writeAgyDB(t, root, "mid.db", now.Add(-30*time.Second), one)   // in window, older
	newest := writeAgyDB(t, root, "new.db", now, one)              // in window, newest

	cfg := AntigravityHistoryConfig{ConversationsRoot: root}
	got := resolveAntigravityDB(cfg, now.Add(-time.Minute), time.Time{})
	if got != newest {
		t.Fatalf("resolveAntigravityDB = %q, want %q", got, newest)
	}
}

func TestPbWireHelpers(t *testing.T) {
	// nested message: field 5 { field 1 { field 1: 42, field 2: 7 } }
	inner := append(pbEncVarint(1, 42), pbEncVarint(2, 7)...)
	msg := pbEncMsg(5, pbEncMsg(1, inner))

	lvl1 := pbBytes(msg, 5)
	lvl2 := pbBytes(lvl1, 1)
	if v, ok := pbVarint(lvl2, 1); !ok || v != 42 {
		t.Errorf("pbVarint field1 = (%d,%v), want (42,true)", v, ok)
	}
	if v, ok := pbVarint(lvl2, 2); !ok || v != 7 {
		t.Errorf("pbVarint field2 = (%d,%v), want (7,true)", v, ok)
	}
	if _, ok := pbVarint(lvl2, 9); ok {
		t.Errorf("pbVarint missing field should report not-found")
	}

	// string field round-trip
	s := pbEncStr(3, "héllo")
	if got := string(pbBytes(s, 3)); got != "héllo" {
		t.Errorf("pbBytes string = %q, want %q", got, "héllo")
	}
	// Adversarial inputs must never panic — the blob is untrusted.
	adversarial := [][]byte{
		{0xFF, 0xFF},                                     // garbage tag tail
		{0x80},                                           // truncated varint (continuation, no next)
		{0x0A, 0x05, 'a', 'b'},                           // wireLen len=5 but only 2 bytes follow
		{0x0A, 0xFF, 0xFF, 0xFF, 0xFF, 0x0F},             // wireLen with huge length
		{0x09, 0x01, 0x02, 0x03},                         // wireI64 with <8 trailing bytes
		{0x0D, 0x01, 0x02},                               // wireI32 with <4 trailing bytes
		{0x08, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // overlong varint value
		{0x1B},                                           // wire type 3 (group) — must stop, not panic
	}
	for i, in := range adversarial {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("adversarial[%d] panicked: %v", i, r)
				}
			}()
			pbScan(in, func(int, int, []byte, uint64) bool { return true })
			_ = pbBytes(in, 1)
			_, _ = pbVarint(in, 1)
		}()
	}

	// wireLen length exactly equal to remaining bytes (boundary, valid).
	exact := []byte{0x12, 0x03, 'x', 'y', 'z'} // field 2, len 3, "xyz"
	if got := string(pbBytes(exact, 2)); got != "xyz" {
		t.Errorf("boundary wireLen = %q, want %q", got, "xyz")
	}
}
