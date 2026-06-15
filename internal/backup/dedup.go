package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// Content-dedup fingerprinting.
//
// pg_dump has no incremental mode, so opendray dedups by content: a new
// backup whose fingerprint matches an existing one on the same target
// reuses that blob instead of re-uploading. The catch is that a backup
// bundle embeds several per-run timestamps that would otherwise make
// every run unique even when the data is identical:
//
//   - the bundle manifest's CreatedAt,
//   - the pg_dump custom-archive header's creation timestamp (empirically
//     the ONLY difference between two dumps of unchanged data — a handful
//     of header bytes),
//   - per-file mtimes in the vault tar (now pinned, see vaultEpoch).
//
// So the fingerprint is computed over the restorable *content* only —
// the dump (with its header timestamp zeroed) plus any full-instance
// sources — and deliberately excludes the manifest. Every other byte of
// every source is hashed, so different content can never collide; the
// only things neutralised are provably-non-content framing.

// customDumpTSRange locates the creation-timestamp region inside a
// pg_dump custom-format archive header. Layout (pg_backup_custom
// WriteHead):
//
//	"PGDMP"(5) vmaj(1) vmin(1) vrev(1) intSize(1) offSize(1) format(1)
//	compression: 1 byte if archive version >= 1.15, else a WriteInt
//	timestamp:   7 × WriteInt  (sec,min,hour,mday,mon,year,isdst)
//
// WriteInt = 1 sign byte + intSize magnitude bytes. Returns ok=false
// (caller hashes verbatim) when the header is too short or isn't a custom
// archive — a safe fallback that merely forgoes dedup.
func customDumpTSRange(head []byte) (start, end int, ok bool) {
	if len(head) < 11 || string(head[:5]) != "PGDMP" {
		return 0, 0, false
	}
	vmaj, vmin := int(head[5]), int(head[6])
	intSize := int(head[8])
	if intSize <= 0 || intSize > 8 {
		return 0, 0, false
	}
	version := vmaj*100 + vmin
	pos := 11
	if version >= 115 { // K_VERS_1_15: single-byte compression algorithm
		pos++
	} else { // older: WriteInt(compression level)
		pos += 1 + intSize
	}
	intLen := 1 + intSize
	start = pos
	end = pos + 7*intLen
	if end > len(head) {
		return 0, 0, false
	}
	return start, end, true
}

// hashDumpNormalized streams the dump at path into h with its custom-
// format header timestamp zeroed, so two dumps of identical data taken
// at different times hash the same. Falls back to a verbatim hash when
// the header can't be parsed.
func hashDumpNormalized(h hash.Hash, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// The timestamp lives within the first few dozen header bytes; read a
	// small window, neutralise it, hash it, then stream the rest.
	const window = 64
	head := make([]byte, window)
	n, rerr := io.ReadFull(f, head)
	switch rerr {
	case nil:
		// full window read
	case io.EOF, io.ErrUnexpectedEOF:
		head = head[:n] // dump smaller than the window
	default:
		return rerr
	}
	if start, end, ok := customDumpTSRange(head); ok {
		for i := start; i < end; i++ {
			head[i] = 0
		}
	}
	if _, err := h.Write(head); err != nil {
		return err
	}
	_, err = io.Copy(h, f)
	return err
}

// hashFile streams a file verbatim into h.
func hashFile(h hash.Hash, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(h, f)
	return err
}

// dedupPart is one hashed component of a backup's content fingerprint.
type dedupPart struct {
	path   string
	isDump bool
}

// dedupContentHash fingerprints the restorable content of a backup from
// its on-disk source files in a stable order (manifest excluded; dump
// timestamp normalised). Identical content yields the same hash.
func dedupContentHash(parts []dedupPart) (string, error) {
	h := sha256.New()
	for _, p := range parts {
		var err error
		if p.isDump {
			err = hashDumpNormalized(h, p.path)
		} else {
			err = hashFile(h, p.path)
		}
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
