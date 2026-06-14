package backup

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeFile is a tiny helper: create parent dirs + write a file.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestPackUnpackVault_Roundtrip(t *testing.T) {
	srcRoot := t.TempDir()
	notes := filepath.Join(srcRoot, "notes")
	skills := filepath.Join(srcRoot, "skills")
	writeFile(t, filepath.Join(notes, "a.md"), "alpha")
	writeFile(t, filepath.Join(notes, "sub", "b.md"), "beta")
	writeFile(t, filepath.Join(skills, "s1", "SKILL.md"), "gamma")

	var buf bytes.Buffer
	if err := PackVault(&buf, []VaultSource{
		{Logical: "notes", Dir: notes},
		{Logical: "skills", Dir: skills},
	}); err != nil {
		t.Fatalf("PackVault: %v", err)
	}

	dstRoot := t.TempDir()
	dstNotes := filepath.Join(dstRoot, "notes")
	dstSkills := filepath.Join(dstRoot, "skills")
	n, err := UnpackVault(&buf, func(logical string) (string, bool) {
		switch logical {
		case "notes":
			return dstNotes, true
		case "skills":
			return dstSkills, true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("UnpackVault: %v", err)
	}
	if n != 3 {
		t.Errorf("UnpackVault wrote %d files, want 3", n)
	}

	want := map[string]string{
		filepath.Join(dstNotes, "a.md"):            "alpha",
		filepath.Join(dstNotes, "sub", "b.md"):     "beta",
		filepath.Join(dstSkills, "s1", "SKILL.md"): "gamma",
	}
	for p, body := range want {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if string(got) != body {
			t.Errorf("%s = %q, want %q", p, got, body)
		}
	}
}

func TestPackVault_ExcludesGitAndSymlinks(t *testing.T) {
	srcRoot := t.TempDir()
	notes := filepath.Join(srcRoot, "notes")
	writeFile(t, filepath.Join(notes, "keep.md"), "keep")
	writeFile(t, filepath.Join(notes, ".git", "config"), "[core]")
	writeFile(t, filepath.Join(notes, ".git", "objects", "x"), "obj")

	if runtime.GOOS != "windows" {
		if err := os.Symlink(filepath.Join(notes, "keep.md"), filepath.Join(notes, "link.md")); err != nil {
			t.Fatalf("symlink: %v", err)
		}
	}

	var buf bytes.Buffer
	if err := PackVault(&buf, []VaultSource{{Logical: "notes", Dir: notes}}); err != nil {
		t.Fatalf("PackVault: %v", err)
	}

	names := tarEntryNames(t, buf.Bytes())
	if _, ok := names["notes/keep.md"]; !ok {
		t.Errorf("expected notes/keep.md in archive, got %v", names)
	}
	for n := range names {
		if filepath.Base(filepath.Dir(n)) == ".git" || n == "notes/.git/config" {
			t.Errorf(".git content leaked into archive: %s", n)
		}
		if n == "notes/link.md" {
			t.Errorf("symlink was followed/included: %s", n)
		}
	}
}

func TestPackVault_AbsentDirIsSkipped(t *testing.T) {
	var buf bytes.Buffer
	err := PackVault(&buf, []VaultSource{
		{Logical: "notes", Dir: filepath.Join(t.TempDir(), "does-not-exist")},
	})
	if err != nil {
		t.Fatalf("PackVault with absent dir should not fail: %v", err)
	}
	if len(tarEntryNames(t, buf.Bytes())) != 0 {
		t.Errorf("expected empty archive for absent dir")
	}
}

func TestPackVault_RejectsBadLogical(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "x.md"), "x")
	for _, bad := range []string{"", "..", "a/b", "notes/.."} {
		var buf bytes.Buffer
		if err := PackVault(&buf, []VaultSource{{Logical: bad, Dir: dir}}); err == nil {
			t.Errorf("PackVault with logical %q should fail", bad)
		}
	}
}

func TestUnpackVault_RejectsTraversal(t *testing.T) {
	// Craft a malicious tar by hand — PackVault would never emit this.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	body := []byte("pwned")
	hdr := &tar.Header{Name: "notes/../../escape.txt", Size: int64(len(body)), Mode: 0o600, Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	_, err := UnpackVault(&buf, func(string) (string, bool) {
		return filepath.Join(dst, "notes"), true
	})
	if err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	// The escape target must not have been created.
	if _, statErr := os.Stat(filepath.Join(dst, "escape.txt")); statErr == nil {
		t.Fatal("traversal wrote a file outside the destination")
	}
}

func TestUnpackVault_RejectsAbsolutePath(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{Name: "/etc/evil", Size: 1, Mode: 0o600, Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := UnpackVault(&buf, func(string) (string, bool) { return t.TempDir(), true })
	if err == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}

func TestUnpackVault_SkipsUnknownLogical(t *testing.T) {
	srcRoot := t.TempDir()
	writeFile(t, filepath.Join(srcRoot, "notes", "a.md"), "alpha")
	var buf bytes.Buffer
	if err := PackVault(&buf, []VaultSource{{Logical: "notes", Dir: filepath.Join(srcRoot, "notes")}}); err != nil {
		t.Fatalf("PackVault: %v", err)
	}
	// destFor never recognises "notes" → entry is skipped, no error.
	n, err := UnpackVault(&buf, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("UnpackVault with unknown logical should not fail: %v", err)
	}
	if n != 0 {
		t.Errorf("UnpackVault wrote %d files for unknown logical, want 0", n)
	}
}

// tarEntryNames reads a raw (uncompressed) tar and returns the set of
// regular-file entry names.
func tarEntryNames(t *testing.T, raw []byte) map[string]struct{} {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(raw))
	out := map[string]struct{}{}
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if hdr.Typeflag == tar.TypeReg {
			out[hdr.Name] = struct{}{}
		}
	}
	return out
}
