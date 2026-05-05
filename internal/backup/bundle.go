package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// BundleVersion is the on-disk bundle format version. Restore tools
// inspect this before attempting to read entries — bumping the
// number signals an incompatible layout change.
const BundleVersion = "1"

// BundleManifest is serialised as manifest.json (the first tar
// entry) of every backup bundle. Restore tools read this to learn
// what to expect (file names, sizes, encryption fingerprint, etc.)
// without scanning the rest.
type BundleManifest struct {
	Version         string             `json:"version"`
	BackupID        string             `json:"backup_id"`
	CreatedAt       time.Time          `json:"created_at"`
	OpendrayVersion string             `json:"opendray_version,omitempty"`
	GitSHA          string             `json:"git_sha,omitempty"`
	PGVersion       string             `json:"pg_version,omitempty"`
	Encryption      ManifestEncryption `json:"encryption"`
	Entries         []ManifestEntry    `json:"entries"`
}

// ManifestEncryption documents how the bundle is encrypted (or not).
type ManifestEncryption struct {
	Algo        string `json:"algo"`        // "aes-256-gcm-chunked" or "none"
	Fingerprint string `json:"fingerprint"` // 16 hex chars; matches Cipher.Fingerprint
}

// ManifestEntry is one logical file inside the bundle.
type ManifestEntry struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// BundleSource is one input file to WriteBundle. Body must be a
// *fully-readable* stream of exactly Size bytes; tar requires
// knowing the size up front. Callers stream pg_dump to a temp file
// first to obtain a stable size, then open it as Body.
type BundleSource struct {
	Name string
	Body io.Reader
	Size int64
}

// WriteBundle streams a gzip-compressed tar containing manifest.json
// followed by each source. The tar layout is fixed:
//
//	manifest.json       — small, JSON, always first
//	<entry.Name>        — verbatim
//
// On any error the caller's writer state is undefined; treat the
// output as junk and discard.
func WriteBundle(w io.Writer, manifest BundleManifest, sources []BundleSource) error {
	// Backfill manifest.Entries from sources for self-description.
	manifest.Version = BundleVersion
	manifest.Entries = manifest.Entries[:0]
	for _, s := range sources {
		manifest.Entries = append(manifest.Entries, ManifestEntry{Path: s.Name, Bytes: s.Size})
	}
	manifestBody, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("bundle: marshal manifest: %w", err)
	}

	gzw := gzip.NewWriter(w)
	tw := tar.NewWriter(gzw)

	if err := writeTarEntry(tw, "manifest.json", bytes.NewReader(manifestBody), int64(len(manifestBody))); err != nil {
		return err
	}
	for _, s := range sources {
		if err := writeTarEntry(tw, s.Name, s.Body, s.Size); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("bundle: close tar: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return fmt.Errorf("bundle: close gzip: %w", err)
	}
	return nil
}

func writeTarEntry(tw *tar.Writer, name string, r io.Reader, size int64) error {
	hdr := &tar.Header{
		Name:     name,
		Size:     size,
		Mode:     0o600,
		ModTime:  time.Now().UTC(),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("bundle: tar header %s: %w", name, err)
	}
	n, err := io.Copy(tw, r)
	if err != nil {
		return fmt.Errorf("bundle: tar body %s: %w", name, err)
	}
	if n != size {
		return fmt.Errorf("bundle: tar body %s: copied %d != header %d", name, n, size)
	}
	return nil
}
