package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// base64Field is URL-safe (no '+'/'/' that would break in JSON or
// query strings) and unpadded so the envelope is compact.
var base64Field = base64.RawURLEncoding

// File format produced by Cipher.Seal:
//
//	header  : magic "ODBK"(4) || version(1) || reserved(3)        =  8 bytes
//	frame*  : ptLen(uint32 BE, 4) || nonce(12) || ct(ptLen) || tag(16)
//	         AAD = frameIdx (uint64 BE)
//	terminator : a final frame with ptLen=0; same AAD scheme
//
// The terminator is what distinguishes "stream ended" from "stream
// was truncated". Truncating before the terminator surfaces as
// ErrCipherCorrupted on Open. The frameIdx in AAD prevents
// reordering attacks — a permuted frame fails AEAD verification.
//
// Nonces are 12 random bytes. Same-key collision risk for n random
// 96-bit nonces is ~n²/2¹⁰⁰; at 64 KiB plaintext per frame the
// effective birthday bound is comfortably above any plausible
// backup size.
const (
	// kdfSalt is the v1 PBKDF2 salt — intentionally global-static.
	// Don't change this string: every v1 backup file ever written
	// derives its key from it. The migration to per-install random
	// salt is captured in ADR 0016 (Proposed) and lands as a v2
	// header format. Until then v1 stays — including for any new
	// backups produced by this code path.
	kdfSalt       = "opendray-v1-backup"
	kdfIterations = 200_000
	kdfKeyLen     = 32 // AES-256

	// chunkPlaintextSize is the plaintext bytes per frame. 64 KiB
	// gives ~0.04% size overhead from per-frame nonce+tag.
	chunkPlaintextSize = 64 * 1024

	headerMagic   = "ODBK"
	headerVersion = byte(1)
	headerSize    = 8

	frameLenSize = 4
	nonceSize    = 12
	tagSize      = 16
)

// Cipher seals/opens an opendray backup stream.
//
// Both Seal and Open are non-blocking — they return a Reader whose
// underlying goroutine pumps data through the AEAD. Errors surface
// when the caller reads to EOF.
//
// EncryptField / DecryptField are the short-payload counterparts
// used to wrap individual sensitive values inside JSON (e.g. SMB
// passwords stored in backup_targets.config, or plaintext API keys
// inside an export bundle).
type Cipher interface {
	// Seal returns a reader of the ciphertext. Reads from `plain`
	// happen lazily as the caller consumes the returned reader.
	Seal(plain io.Reader) io.Reader
	// Open returns a reader of the plaintext. The first read after
	// a tampered or wrong-key ciphertext returns one of the cipher
	// sentinel errors.
	Open(ciph io.Reader) io.Reader
	// Fingerprint is the first 16 hex chars of SHA-256(derived-key).
	// Stored on every backup row so restore can reject blobs taken
	// under a different passphrase.
	Fingerprint() string
	// EncryptField wraps a short string with AES-GCM and returns
	// "v1:base64url(nonce|ciphertext|tag)". Empty input maps to
	// empty output (no wrapping).
	EncryptField(plain string) (string, error)
	// DecryptField is the reverse. Empty input maps to empty output.
	// Returns ErrCipherWrongKey on tampering / wrong key, or
	// ErrCipherFormat on a malformed envelope.
	DecryptField(envelope string) (string, error)
}

// NewCipher derives an AES-256 key from passphrase via PBKDF2-HMAC-
// SHA256 and returns a chunked AES-GCM cipher.
//
// Returns ErrCipherUnconfigured if passphrase is empty so callers
// can errors.Is it without string-matching.
func NewCipher(passphrase string) (Cipher, error) {
	if passphrase == "" {
		return nil, ErrCipherUnconfigured
	}
	key, err := pbkdf2.Key(sha256.New, passphrase, []byte(kdfSalt), kdfIterations, kdfKeyLen)
	if err != nil {
		return nil, fmt.Errorf("backup cipher: kdf: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("backup cipher: new aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("backup cipher: new gcm: %w", err)
	}
	sum := sha256.Sum256(key)
	return &chunkedAEAD{
		aead:        gcm,
		fingerprint: hex.EncodeToString(sum[:8]),
	}, nil
}

type chunkedAEAD struct {
	aead        cipher.AEAD
	fingerprint string
}

func (c *chunkedAEAD) Fingerprint() string { return c.fingerprint }

// fieldEnvelopePrefix marks the wire format produced by
// EncryptField. Bumping the version lets a future Cipher rotate
// algorithms while keeping old ciphertexts decryptable.
const fieldEnvelopePrefix = "v1:"

func (c *chunkedAEAD) EncryptField(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("backup field: nonce: %w", err)
	}
	// AAD distinguishes field envelopes from stream frames so a
	// stream frame's ciphertext can never be re-pasted as a field.
	aad := []byte("field-v1")
	ct := c.aead.Seal(nil, nonce, []byte(plain), aad)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return fieldEnvelopePrefix + base64Field.EncodeToString(out), nil
}

func (c *chunkedAEAD) DecryptField(envelope string) (string, error) {
	if envelope == "" {
		return "", nil
	}
	if len(envelope) < len(fieldEnvelopePrefix) || envelope[:len(fieldEnvelopePrefix)] != fieldEnvelopePrefix {
		return "", fmt.Errorf("%w: bad field prefix", ErrCipherFormat)
	}
	body, err := base64Field.DecodeString(envelope[len(fieldEnvelopePrefix):])
	if err != nil {
		return "", fmt.Errorf("%w: base64: %v", ErrCipherFormat, err)
	}
	if len(body) < nonceSize+tagSize {
		return "", fmt.Errorf("%w: short envelope", ErrCipherFormat)
	}
	nonce := body[:nonceSize]
	ct := body[nonceSize:]
	plain, err := c.aead.Open(nil, nonce, ct, []byte("field-v1"))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCipherWrongKey, err)
	}
	return string(plain), nil
}

func (c *chunkedAEAD) Seal(plain io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		err := c.sealStream(plain, pw)
		_ = pw.CloseWithError(err)
	}()
	return pr
}

func (c *chunkedAEAD) Open(ciph io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		err := c.openStream(ciph, pw)
		_ = pw.CloseWithError(err)
	}()
	return pr
}

func (c *chunkedAEAD) sealStream(plain io.Reader, w io.Writer) error {
	header := make([]byte, headerSize)
	copy(header[:4], headerMagic)
	header[4] = headerVersion
	if _, err := w.Write(header); err != nil {
		return err
	}

	buf := make([]byte, chunkPlaintextSize)
	frameIdx := uint64(0)
	for {
		n, err := io.ReadFull(plain, buf)
		switch {
		case err == nil:
			// full chunk
			if err := c.writeFrame(w, buf[:n], frameIdx); err != nil {
				return err
			}
			frameIdx++
		case errors.Is(err, io.EOF):
			// stream ended cleanly between chunks; write terminator
			return c.writeFrame(w, nil, frameIdx)
		case errors.Is(err, io.ErrUnexpectedEOF):
			// final partial chunk
			if n > 0 {
				if err := c.writeFrame(w, buf[:n], frameIdx); err != nil {
					return err
				}
				frameIdx++
			}
			return c.writeFrame(w, nil, frameIdx)
		default:
			return fmt.Errorf("backup cipher: read plain: %w", err)
		}
	}
}

func (c *chunkedAEAD) writeFrame(w io.Writer, plain []byte, idx uint64) error {
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("backup cipher: nonce: %w", err)
	}
	aad := make([]byte, 8)
	binary.BigEndian.PutUint64(aad, idx)

	ct := c.aead.Seal(nil, nonce, plain, aad)

	hdr := make([]byte, frameLenSize+nonceSize)
	binary.BigEndian.PutUint32(hdr[:frameLenSize], uint32(len(plain)))
	copy(hdr[frameLenSize:], nonce)

	if _, err := w.Write(hdr); err != nil {
		return err
	}
	if _, err := w.Write(ct); err != nil {
		return err
	}
	return nil
}

func (c *chunkedAEAD) openStream(ciph io.Reader, w io.Writer) error {
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(ciph, header); err != nil {
		return fmt.Errorf("%w: read header: %v", ErrCipherCorrupted, err)
	}
	if string(header[:4]) != headerMagic {
		return fmt.Errorf("%w: bad magic %q", ErrCipherFormat, header[:4])
	}
	if header[4] != headerVersion {
		return fmt.Errorf("%w: version %d", ErrCipherFormat, header[4])
	}

	frameIdx := uint64(0)
	for {
		hdr := make([]byte, frameLenSize+nonceSize)
		if _, err := io.ReadFull(ciph, hdr); err != nil {
			return fmt.Errorf("%w: read frame %d hdr: %v", ErrCipherCorrupted, frameIdx, err)
		}
		ptLen := binary.BigEndian.Uint32(hdr[:frameLenSize])
		if ptLen > chunkPlaintextSize {
			return fmt.Errorf("%w: frame %d ptLen %d > max", ErrCipherCorrupted, frameIdx, ptLen)
		}
		nonce := hdr[frameLenSize:]

		ct := make([]byte, int(ptLen)+tagSize)
		if _, err := io.ReadFull(ciph, ct); err != nil {
			return fmt.Errorf("%w: read frame %d ct: %v", ErrCipherCorrupted, frameIdx, err)
		}

		aad := make([]byte, 8)
		binary.BigEndian.PutUint64(aad, frameIdx)
		plain, err := c.aead.Open(nil, nonce, ct, aad)
		if err != nil {
			return fmt.Errorf("%w: frame %d: %v", ErrCipherWrongKey, frameIdx, err)
		}

		if ptLen == 0 {
			// terminator: ensure caller didn't pad bytes after.
			var bb [1]byte
			if n, _ := ciph.Read(bb[:]); n != 0 {
				return fmt.Errorf("%w: trailing data after terminator", ErrCipherCorrupted)
			}
			return nil
		}
		if _, err := w.Write(plain); err != nil {
			return err
		}
		frameIdx++
	}
}
