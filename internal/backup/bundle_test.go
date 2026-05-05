package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestWriteBundle_HappyPath(t *testing.T) {
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	manifest := BundleManifest{
		BackupID:        "bk_test",
		CreatedAt:       now,
		OpendrayVersion: "0.1.0",
		PGVersion:       "14.19",
		Encryption: ManifestEncryption{
			Algo:        "aes-256-gcm-chunked",
			Fingerprint: "deadbeefdeadbeef",
		},
	}
	dump := []byte("pretend pg_dump output")
	cfg := []byte("pretend config.toml")
	var buf bytes.Buffer
	err := WriteBundle(&buf, manifest, []BundleSource{
		{Name: "config.toml", Body: bytes.NewReader(cfg), Size: int64(len(cfg))},
		{Name: "dump.bin", Body: bytes.NewReader(dump), Size: int64(len(dump))},
	})
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}

	gzr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	tr := tar.NewReader(gzr)

	got := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("tar.Next: %v", err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		got[hdr.Name] = body
	}

	mfBody, ok := got["manifest.json"]
	if !ok {
		t.Fatal("missing manifest.json")
	}
	var mf BundleManifest
	if err := json.Unmarshal(mfBody, &mf); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if mf.Version != BundleVersion {
		t.Errorf("manifest version = %q, want %q", mf.Version, BundleVersion)
	}
	if mf.BackupID != "bk_test" {
		t.Errorf("manifest backup_id = %q", mf.BackupID)
	}
	if len(mf.Entries) != 2 {
		t.Errorf("manifest entries len = %d, want 2", len(mf.Entries))
	}

	if !bytes.Equal(got["config.toml"], cfg) {
		t.Error("config.toml body mismatch")
	}
	if !bytes.Equal(got["dump.bin"], dump) {
		t.Error("dump.bin body mismatch")
	}
}

func TestWriteBundle_ManifestFirst(t *testing.T) {
	var buf bytes.Buffer
	err := WriteBundle(&buf, BundleManifest{BackupID: "bk_x"}, []BundleSource{
		{Name: "z.bin", Body: strings.NewReader("z"), Size: 1},
		{Name: "a.bin", Body: strings.NewReader("a"), Size: 1},
	})
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	gzr, _ := gzip.NewReader(&buf)
	tr := tar.NewReader(gzr)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("tar.Next: %v", err)
	}
	if hdr.Name != "manifest.json" {
		t.Errorf("first entry = %q, want manifest.json", hdr.Name)
	}
}

func TestWriteBundle_SizeMismatch(t *testing.T) {
	var buf bytes.Buffer
	err := WriteBundle(&buf, BundleManifest{}, []BundleSource{
		{Name: "lying.bin", Body: strings.NewReader("ab"), Size: 99},
	})
	if err == nil {
		t.Fatal("expected size-mismatch error")
	}
}

func TestWriteBundle_CipherE2E(t *testing.T) {
	// Final shape: bundle → cipher.Seal → cipher.Open → bundle reads back.
	c, _ := NewCipher("e2e-bundle")
	manifest := BundleManifest{
		BackupID:   "bk_e2e",
		CreatedAt:  time.Now().UTC(),
		Encryption: ManifestEncryption{Algo: "aes-256-gcm-chunked", Fingerprint: c.Fingerprint()},
	}
	payload := bytes.Repeat([]byte("x"), 200_000) // 200 KB across multiple cipher chunks

	var bundleBuf bytes.Buffer
	err := WriteBundle(&bundleBuf, manifest, []BundleSource{
		{Name: "payload.bin", Body: bytes.NewReader(payload), Size: int64(len(payload))},
	})
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}

	var encBuf bytes.Buffer
	if _, err := io.Copy(&encBuf, c.Seal(&bundleBuf)); err != nil {
		t.Fatalf("seal: %v", err)
	}

	pt, err := io.ReadAll(c.Open(&encBuf))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	gzr, err := gzip.NewReader(bytes.NewReader(pt))
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	tr := tar.NewReader(gzr)
	saw := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("tar.Next: %v", err)
		}
		saw[hdr.Name] = true
		if hdr.Name == "payload.bin" {
			body, _ := io.ReadAll(tr)
			if !bytes.Equal(body, payload) {
				t.Error("payload mismatch through cipher round-trip")
			}
		}
	}
	if !saw["manifest.json"] || !saw["payload.bin"] {
		t.Errorf("missing entries: %v", saw)
	}
}
