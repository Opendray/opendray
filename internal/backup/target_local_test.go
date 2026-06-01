package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newLocalTarget(t *testing.T) *LocalTarget {
	t.Helper()
	dir := t.TempDir()
	tg, err := NewLocalTarget("local-test", dir)
	if err != nil {
		t.Fatalf("NewLocalTarget: %v", err)
	}
	return tg
}

func TestLocalTarget_PutGet_RoundTrip(t *testing.T) {
	tg := newLocalTarget(t)
	ctx := context.Background()
	payload := []byte("hello backup target")

	ref, err := tg.Put(ctx, "subdir/file.bin", bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if ref.Bytes != int64(len(payload)) {
		t.Errorf("Bytes = %d, want %d", ref.Bytes, len(payload))
	}
	wantSum := sha256.Sum256(payload)
	if ref.SHA256 != hex.EncodeToString(wantSum[:]) {
		t.Errorf("SHA256 mismatch")
	}
	if ref.Target != "local-test" {
		t.Errorf("Target = %q, want local-test", ref.Target)
	}

	rc, err := tg.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("payload mismatch")
	}
}

func TestLocalTarget_Put_AtomicNoTempLeftBehind(t *testing.T) {
	tg := newLocalTarget(t)
	ctx := context.Background()
	_, err := tg.Put(ctx, "a/b/c.bin", strings.NewReader("xyz"), 3)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	// no .part file should remain in any subdir.
	err = filepath.Walk(tg.Root(), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".part") {
			t.Errorf("temp .part file left behind: %s", p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

func TestLocalTarget_Put_RejectsPathTraversal(t *testing.T) {
	tg := newLocalTarget(t)
	ctx := context.Background()

	cases := []string{
		"../escape.bin",
		"a/../../escape.bin",
		"/abs/path.bin",
		"",
		"good/../../bad",
	}
	for _, p := range cases {
		_, err := tg.Put(ctx, p, strings.NewReader("x"), 1)
		if !errors.Is(err, ErrTargetRejectedPath) {
			t.Errorf("Put(%q) err = %v, want ErrTargetRejectedPath", p, err)
		}
	}
}

func TestLocalTarget_Put_RejectsNullByte(t *testing.T) {
	tg := newLocalTarget(t)
	_, err := tg.Put(context.Background(), "ok\x00path.bin", strings.NewReader("x"), 1)
	if !errors.Is(err, ErrTargetRejectedPath) {
		t.Fatalf("got %v, want ErrTargetRejectedPath", err)
	}
}

func TestLocalTarget_Get_NotFound(t *testing.T) {
	tg := newLocalTarget(t)
	_, err := tg.Get(context.Background(), TargetRef{Target: "local-test", Path: "missing.bin"})
	if !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("got %v, want ErrBackupNotFound", err)
	}
}

func TestLocalTarget_Delete_Idempotent(t *testing.T) {
	tg := newLocalTarget(t)
	ctx := context.Background()
	_, err := tg.Put(ctx, "del.bin", strings.NewReader("x"), 1)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	ref := TargetRef{Target: "local-test", Path: "del.bin"}
	if err := tg.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete first: %v", err)
	}
	if err := tg.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete second (should be idempotent): %v", err)
	}
}

func TestLocalTarget_HealthCheck_OK(t *testing.T) {
	tg := newLocalTarget(t)
	if err := tg.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	// probe file should be cleaned up.
	if _, err := os.Stat(filepath.Join(tg.Root(), ".healthcheck")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("probe file leaked: %v", err)
	}
}

func TestLocalTarget_HealthCheck_ReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Chmod does not restrict writes on Windows")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(dir, 0o700) // so t.TempDir cleanup works
	tg := &LocalTarget{id: "ro", root: dir}
	if err := tg.HealthCheck(context.Background()); err == nil {
		t.Error("HealthCheck on read-only dir should fail")
	}
}

func TestLocalTarget_NewLocalTarget_RejectsEmpty(t *testing.T) {
	if _, err := NewLocalTarget("", "/tmp"); err == nil {
		t.Error("empty id should fail")
	}
	if _, err := NewLocalTarget("x", ""); err == nil {
		t.Error("empty root should fail")
	}
}

func TestLocalTarget_RoundTrip_LargeStream(t *testing.T) {
	tg := newLocalTarget(t)
	ctx := context.Background()
	// 5 MiB random — ensures chunked I/O paths work.
	payload := randBytes(t, 5*1024*1024)
	ref, err := tg.Put(ctx, "big.bin", bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if ref.Bytes != int64(len(payload)) {
		t.Errorf("Bytes = %d", ref.Bytes)
	}
	rc, _ := tg.Get(ctx, ref)
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if !bytes.Equal(got, payload) {
		t.Error("payload mismatch on large round-trip")
	}
}

func TestLocalTarget_Cipher_E2E(t *testing.T) {
	// End-to-end: cipher seal → target put → target get → cipher
	// open. This is the actual production pipeline shape.
	tg := newLocalTarget(t)
	ctx := context.Background()
	c, _ := NewCipher("e2e-passphrase")

	plain := randBytes(t, chunkPlaintextSize*3+100)
	sealed := c.Seal(bytes.NewReader(plain))

	ref, err := tg.Put(ctx, "e2e.enc", sealed, -1)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, _ := tg.Get(ctx, ref)
	defer rc.Close()
	out, err := io.ReadAll(c.Open(rc))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(out, plain) {
		t.Error("E2E mismatch")
	}
}
