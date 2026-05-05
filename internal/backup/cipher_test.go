package backup

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"strings"
	"testing"
)

func newTestCipher(t *testing.T) Cipher {
	t.Helper()
	c, err := NewCipher("opendray-test-passphrase-x")
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestNewCipher_EmptyPassphraseRejected(t *testing.T) {
	_, err := NewCipher("")
	if !errors.Is(err, ErrCipherUnconfigured) {
		t.Fatalf("got %v, want ErrCipherUnconfigured", err)
	}
}

func TestNewCipher_FingerprintDeterministic(t *testing.T) {
	a, _ := NewCipher("p1")
	b, _ := NewCipher("p1")
	if a.Fingerprint() != b.Fingerprint() {
		t.Errorf("same passphrase produced different fingerprints: %s vs %s", a.Fingerprint(), b.Fingerprint())
	}
	c, _ := NewCipher("p2")
	if a.Fingerprint() == c.Fingerprint() {
		t.Errorf("different passphrases produced same fingerprint: %s", a.Fingerprint())
	}
	if got := len(a.Fingerprint()); got != 16 {
		t.Errorf("fingerprint len = %d, want 16", got)
	}
}

func TestCipher_RoundTrip_Empty(t *testing.T) {
	c := newTestCipher(t)
	roundTrip(t, c, nil)
}

func TestCipher_RoundTrip_OneByte(t *testing.T) {
	c := newTestCipher(t)
	roundTrip(t, c, []byte{0x42})
}

func TestCipher_RoundTrip_SubChunk(t *testing.T) {
	c := newTestCipher(t)
	roundTrip(t, c, bytes.Repeat([]byte("hello "), 100)) // 600 bytes
}

func TestCipher_RoundTrip_ExactChunk(t *testing.T) {
	c := newTestCipher(t)
	roundTrip(t, c, randBytes(t, chunkPlaintextSize))
}

func TestCipher_RoundTrip_ManyChunks(t *testing.T) {
	c := newTestCipher(t)
	// 10 MiB — exercises ~160 chunks at 64 KiB each.
	roundTrip(t, c, randBytes(t, 10*1024*1024))
}

func TestCipher_RoundTrip_AcrossChunkBoundary(t *testing.T) {
	c := newTestCipher(t)
	// 64 KiB + 13 bytes → one full + one partial frame.
	roundTrip(t, c, randBytes(t, chunkPlaintextSize+13))
}

func roundTrip(t *testing.T, c Cipher, plain []byte) {
	t.Helper()
	var ctBuf bytes.Buffer
	if _, err := io.Copy(&ctBuf, c.Seal(bytes.NewReader(plain))); err != nil {
		t.Fatalf("seal: %v", err)
	}
	var ptBuf bytes.Buffer
	if _, err := io.Copy(&ptBuf, c.Open(bytes.NewReader(ctBuf.Bytes()))); err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(plain, ptBuf.Bytes()) {
		t.Fatalf("round-trip mismatch: plain=%d ct=%d out=%d",
			len(plain), ctBuf.Len(), ptBuf.Len())
	}
}

func TestCipher_Open_WrongKey(t *testing.T) {
	a, _ := NewCipher("alpha")
	b, _ := NewCipher("beta")

	plain := []byte("hello world")
	var ct bytes.Buffer
	_, _ = io.Copy(&ct, a.Seal(bytes.NewReader(plain)))

	out, err := io.ReadAll(b.Open(bytes.NewReader(ct.Bytes())))
	if err == nil {
		t.Fatalf("expected error opening with wrong key, got %d bytes", len(out))
	}
	if !errors.Is(err, ErrCipherWrongKey) {
		t.Errorf("got %v, want ErrCipherWrongKey wrapped", err)
	}
}

func TestCipher_Open_TamperedCiphertext(t *testing.T) {
	c := newTestCipher(t)
	plain := bytes.Repeat([]byte{0xAA}, 200)

	var ct bytes.Buffer
	_, _ = io.Copy(&ct, c.Seal(bytes.NewReader(plain)))
	raw := ct.Bytes()

	// Flip a byte well inside the first frame's ciphertext (after
	// the 8-byte file header + 4-byte ptLen + 12-byte nonce).
	raw[8+4+12+5] ^= 0xFF

	_, err := io.ReadAll(c.Open(bytes.NewReader(raw)))
	if !errors.Is(err, ErrCipherWrongKey) {
		t.Fatalf("tamper: got %v, want ErrCipherWrongKey", err)
	}
}

func TestCipher_Open_TruncatedBeforeTerminator(t *testing.T) {
	c := newTestCipher(t)
	plain := randBytes(t, chunkPlaintextSize+200)

	var ct bytes.Buffer
	_, _ = io.Copy(&ct, c.Seal(bytes.NewReader(plain)))
	raw := ct.Bytes()

	// Drop the last 64 bytes, which strips the terminator frame.
	truncated := raw[:len(raw)-64]

	_, err := io.ReadAll(c.Open(bytes.NewReader(truncated)))
	if err == nil {
		t.Fatalf("truncated stream opened cleanly")
	}
	if !errors.Is(err, ErrCipherCorrupted) && !errors.Is(err, ErrCipherWrongKey) {
		// either is acceptable: truncation may land mid-frame
		// (Corrupted) or just before the terminator (WrongKey when
		// AAD index mismatches). Both indicate failure.
		t.Errorf("got %v, want corrupted/wrongkey", err)
	}
}

func TestCipher_Open_BadMagic(t *testing.T) {
	c := newTestCipher(t)
	junk := []byte("XXXX\x01\x00\x00\x00") // 8 bytes, wrong magic
	_, err := io.ReadAll(c.Open(bytes.NewReader(junk)))
	if !errors.Is(err, ErrCipherFormat) {
		t.Fatalf("got %v, want ErrCipherFormat", err)
	}
}

func TestCipher_Open_BadVersion(t *testing.T) {
	c := newTestCipher(t)
	hdr := []byte("ODBK\x99\x00\x00\x00") // version 0x99
	_, err := io.ReadAll(c.Open(bytes.NewReader(hdr)))
	if !errors.Is(err, ErrCipherFormat) {
		t.Fatalf("got %v, want ErrCipherFormat", err)
	}
}

func TestCipher_Seal_PropagatesReadError(t *testing.T) {
	c := newTestCipher(t)
	want := errors.New("upstream broken")
	r := &errReader{err: want}
	_, err := io.ReadAll(c.Seal(r))
	if err == nil || !strings.Contains(err.Error(), want.Error()) {
		t.Fatalf("got %v, want wrapping %v", err, want)
	}
}

func TestCipher_FieldRoundTrip(t *testing.T) {
	c := newTestCipher(t)
	cases := []string{
		"hunter2",
		"",
		"a longer secret that may span multiple GCM blocks " + strings.Repeat("x", 200),
		"unicode → 中文 ✓ \x00 with NUL",
	}
	for _, plain := range cases {
		env, err := c.EncryptField(plain)
		if err != nil {
			t.Fatalf("EncryptField(%q): %v", plain, err)
		}
		if plain == "" && env != "" {
			t.Fatalf("empty input should yield empty envelope, got %q", env)
		}
		got, err := c.DecryptField(env)
		if err != nil {
			t.Fatalf("DecryptField: %v", err)
		}
		if got != plain {
			t.Errorf("round-trip: got %q want %q", got, plain)
		}
	}
}

func TestCipher_DecryptField_WrongKey(t *testing.T) {
	a, _ := NewCipher("alpha")
	b, _ := NewCipher("beta")
	env, _ := a.EncryptField("secret")
	_, err := b.DecryptField(env)
	if !errors.Is(err, ErrCipherWrongKey) {
		t.Fatalf("got %v, want ErrCipherWrongKey", err)
	}
}

func TestCipher_DecryptField_BadFormat(t *testing.T) {
	c := newTestCipher(t)
	cases := []string{"plain-no-prefix", "v1:!!!notbase64", "v2:abcd"}
	for _, in := range cases {
		_, err := c.DecryptField(in)
		if !errors.Is(err, ErrCipherFormat) {
			t.Errorf("DecryptField(%q) err = %v, want ErrCipherFormat", in, err)
		}
	}
}

func TestCipher_FieldEnvelope_NotInterchangeableWithStream(t *testing.T) {
	// Defence-in-depth: a Field envelope must not decode under the
	// stream Open path (uses different AAD constants). A bug here
	// could let an attacker substitute a field for a frame.
	c := newTestCipher(t)
	env, _ := c.EncryptField("secret")
	body := []byte(env[len(fieldEnvelopePrefix):])
	_, err := c.DecryptField("v1:" + string(body)) // sanity: still works
	if err != nil {
		t.Fatalf("sanity decrypt: %v", err)
	}
	// Now feed envelope bytes into Open as if they were a stream.
	rawCt := make([]byte, len(body))
	copy(rawCt, body)
	_, err = io.ReadAll(c.Open(bytes.NewReader(rawCt)))
	if err == nil {
		t.Error("Open should reject field-shaped bytes")
	}
}

type errReader struct{ err error }

func (e *errReader) Read(p []byte) (int, error) { return 0, e.err }

func randBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return b
}
