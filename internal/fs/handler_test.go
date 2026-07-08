package fs

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newUploadReq(root, dir, relpath, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/fs/upload", strings.NewReader(body))
	q := req.URL.Query()
	q.Set("root", root)
	q.Set("dir", dir)
	q.Set("relpath", relpath)
	req.URL.RawQuery = q.Encode()
	return req
}

func TestUpload_WritesNestedWithinRoot(t *testing.T) {
	root := t.TempDir()
	h := NewHandlers(nil)
	w := httptest.NewRecorder()
	h.upload(w, newUploadReq(root, root, "notes/todo.txt", "hello"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	got, err := os.ReadFile(filepath.Join(root, "notes", "todo.txt"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("content = %q, want %q", got, "hello")
	}
}

func TestUpload_RejectsRelpathEscape(t *testing.T) {
	root := t.TempDir()
	h := NewHandlers(nil)
	for _, rel := range []string{"../escape.txt", "a/../../escape.txt", "/abs.txt"} {
		w := httptest.NewRecorder()
		h.upload(w, newUploadReq(root, root, rel, "x"))
		if w.Code != http.StatusBadRequest && w.Code != http.StatusForbidden {
			t.Fatalf("relpath %q: status = %d, want 400/403", rel, w.Code)
		}
	}
}

func TestUpload_RejectsDirEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	h := NewHandlers(nil)
	w := httptest.NewRecorder()
	h.upload(w, newUploadReq(root, outside, "x.txt", "x"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestUpload_AutoRenamesOnConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "data.csv"), []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(nil)
	w := httptest.NewRecorder()
	h.upload(w, newUploadReq(root, root, "data.csv", "new"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "data-1.csv")); err != nil {
		t.Fatalf("expected data-1.csv to exist: %v", err)
	}
	orig, _ := os.ReadFile(filepath.Join(root, "data.csv"))
	if string(orig) != "orig" {
		t.Fatalf("original was clobbered: %q", orig)
	}
}

func TestUpload_RejectsSymlinkEscapeInRelpath(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	// A symlink inside root pointing outside it.
	if err := os.Symlink(outside, filepath.Join(root, "sub")); err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(nil)
	w := httptest.NewRecorder()
	h.upload(w, newUploadReq(root, root, "sub/evil.txt", "pwned"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(outside, "evil.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file escaped root into the symlink target")
	}
}

func TestUpload_EnforcesSizeCap(t *testing.T) {
	root := t.TempDir()
	h := NewHandlers(nil)
	h.maxUploadBytes = 8
	w := httptest.NewRecorder()
	h.upload(w, newUploadReq(root, root, "big.bin", "123456789")) // 9 bytes
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "big.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial file must not remain after 413")
	}
}
