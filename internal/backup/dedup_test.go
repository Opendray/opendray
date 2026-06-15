package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// customHeader builds a minimal pg_dump custom-format header for the
// given archive version + intSize, with each timestamp int's first
// magnitude byte set to tsByte, followed by payload.
func customHeader(vmaj, vmin, intSize int, tsByte byte, payload []byte) []byte {
	var b []byte
	b = append(b, []byte("PGDMP")...)
	b = append(b, byte(vmaj), byte(vmin), 0) // vrev
	b = append(b, byte(intSize), 8, 1)       // intSize, offSize, format
	version := vmaj*100 + vmin
	if version >= 115 {
		b = append(b, 0) // 1-byte compression algorithm
	} else {
		b = append(b, 0)                        // sign byte
		b = append(b, make([]byte, intSize)...) // magnitude
	}
	// 7 timestamp ints: sign byte + intSize magnitude; stamp the first
	// magnitude byte so two headers can differ only here.
	for i := 0; i < 7; i++ {
		b = append(b, 0) // sign
		mag := make([]byte, intSize)
		mag[0] = tsByte
		b = append(b, mag...)
	}
	return append(b, payload...)
}

func TestCustomDumpTSRange(t *testing.T) {
	tests := []struct {
		name             string
		vmaj, vmin, isz  int
		wantStart, wantE int
		wantOK           bool
	}{
		{"v1.14 int4", 1, 14, 4, 16, 16 + 7*5, true},
		{"v1.15 int4", 1, 15, 4, 12, 12 + 7*5, true},
		{"v1.14 int8", 1, 14, 8, 20, 20 + 7*9, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			head := customHeader(tc.vmaj, tc.vmin, tc.isz, 0x11, bytes.Repeat([]byte{0xAB}, 200))
			start, end, ok := customDumpTSRange(head)
			if ok != tc.wantOK || start != tc.wantStart || end != tc.wantE {
				t.Errorf("got (start=%d end=%d ok=%v), want (%d %d %v)",
					start, end, ok, tc.wantStart, tc.wantE, tc.wantOK)
			}
		})
	}
}

func TestCustomDumpTSRange_NotCustom(t *testing.T) {
	if _, _, ok := customDumpTSRange([]byte("not a pg dump at all")); ok {
		t.Error("non-PGDMP header should not parse as custom")
	}
	if _, _, ok := customDumpTSRange([]byte("PG")); ok {
		t.Error("too-short header should not parse")
	}
}

func hashFileToHex(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	parts := []dedupPart{{path: p, isDump: true}}
	h, err := dedupContentHash(parts)
	if err != nil {
		t.Fatalf("dedupContentHash: %v", err)
	}
	return h
}

func TestHashDumpNormalized_IgnoresTimestamp(t *testing.T) {
	dir := t.TempDir()
	payload := bytes.Repeat([]byte{0x42}, 500) // identical "data"
	d1 := customHeader(1, 14, 4, 0x11, payload)
	d2 := customHeader(1, 14, 4, 0x99, payload) // differs ONLY in timestamp bytes

	if bytes.Equal(d1, d2) {
		t.Fatal("test dumps should differ in the timestamp region")
	}

	h1 := hashFileToHex(t, dir, "d1.bin", d1)
	h2 := hashFileToHex(t, dir, "d2.bin", d2)
	if h1 != h2 {
		t.Errorf("identical data differing only in timestamp should hash the same:\n %s\n %s", h1, h2)
	}

	// Different DATA must still hash differently (no false dedup).
	d3 := customHeader(1, 14, 4, 0x11, append(payload[:499:499], 0xFF))
	h3 := hashFileToHex(t, dir, "d3.bin", d3)
	if h3 == h1 {
		t.Error("different data must not collide after timestamp normalisation")
	}
}

func TestPackVault_Deterministic(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes")
	if err := os.MkdirAll(notes, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.md", "b.md", "c.md"} {
		if err := os.WriteFile(filepath.Join(notes, f), []byte("content-"+f), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	src := []VaultSource{{Logical: "notes", Dir: notes}}

	var buf1 bytes.Buffer
	if err := PackVault(&buf1, src); err != nil {
		t.Fatalf("pack1: %v", err)
	}

	// Touch every file to a wildly different mtime — the tar must not change.
	future := time.Now().Add(72 * time.Hour)
	for _, f := range []string{"a.md", "b.md", "c.md"} {
		if err := os.Chtimes(filepath.Join(notes, f), future, future); err != nil {
			t.Fatal(err)
		}
	}
	var buf2 bytes.Buffer
	if err := PackVault(&buf2, src); err != nil {
		t.Fatalf("pack2: %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("PackVault output changed across runs despite identical file contents (mtime not pinned?)")
	}
}
