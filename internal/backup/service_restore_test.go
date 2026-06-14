package backup

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sealedFullInstanceBundle builds a full_instance bundle (config +
// vault + secrets + dump) sealed with c, returning the ciphertext.
func sealedFullInstanceBundle(t *testing.T, c Cipher, cfgBody, secBody, notesDir string) []byte {
	t.Helper()
	var vbuf bytes.Buffer
	if err := PackVault(&vbuf, []VaultSource{{Logical: "notes", Dir: notesDir}}); err != nil {
		t.Fatalf("PackVault: %v", err)
	}
	dump := []byte("PGDMP-fake-dump-bytes")
	manifest := BundleManifest{
		BackupID: "bk_dry",
		Encryption: ManifestEncryption{
			Algo:        "aes-256-gcm-chunked",
			Fingerprint: c.Fingerprint(),
		},
	}
	var plain bytes.Buffer
	err := WriteBundle(&plain, manifest, []BundleSource{
		{Name: "config.toml", Body: strings.NewReader(cfgBody), Size: int64(len(cfgBody))},
		{Name: "vault.tar", Body: bytes.NewReader(vbuf.Bytes()), Size: int64(vbuf.Len())},
		{Name: "secrets.env", Body: strings.NewReader(secBody), Size: int64(len(secBody))},
		{Name: "dump.bin", Body: bytes.NewReader(dump), Size: int64(len(dump))},
	})
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	out, err := io.ReadAll(c.Seal(bytes.NewReader(plain.Bytes())))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	return out
}

func TestRestoreBackup_DryRunReportsPlanAndWritesNothing(t *testing.T) {
	c, err := NewCipher("test-passphrase-abc123")
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	notes := filepath.Join(t.TempDir(), "notes")
	writeFile(t, filepath.Join(notes, "a.md"), "alpha")
	writeFile(t, filepath.Join(notes, "sub", "b.md"), "beta")

	sealed := sealedFullInstanceBundle(t, c, "listen='x'", "SECRET=1", notes)

	dst := t.TempDir()
	cfgPath := filepath.Join(dst, "config.toml")
	secPath := filepath.Join(dst, "secrets.env")
	notesDest := filepath.Join(dst, "vault", "notes")

	svc := &Service{
		cipher: c,
		cfg: Config{
			LocalDir:     t.TempDir(),
			VaultSources: []VaultSource{{Logical: "notes", Dir: notesDest}},
			SecretsFile:  secPath,
		},
		configPath: cfgPath,
		log:        slog.Default(),
	}

	res, err := svc.RestoreBackup(context.Background(), RestoreRequest{
		Source: bytes.NewReader(sealed),
		Apply:  false,
	})
	if err != nil {
		t.Fatalf("dry-run RestoreBackup: %v", err)
	}

	if !res.Plan.DryRun {
		t.Error("Plan.DryRun = false, want true")
	}
	if !res.Plan.DumpPresent {
		t.Error("Plan.DumpPresent = false, want true")
	}
	if res.Plan.VaultFiles != 2 {
		t.Errorf("Plan.VaultFiles = %d, want 2", res.Plan.VaultFiles)
	}
	if len(res.Plan.VaultRoots) != 1 || res.Plan.VaultRoots[0] != "notes" {
		t.Errorf("Plan.VaultRoots = %v, want [notes]", res.Plan.VaultRoots)
	}
	if res.Plan.ConfigPath != cfgPath {
		t.Errorf("Plan.ConfigPath = %q, want %q", res.Plan.ConfigPath, cfgPath)
	}
	if res.Plan.SecretsPath != secPath {
		t.Errorf("Plan.SecretsPath = %q, want %q", res.Plan.SecretsPath, secPath)
	}
	if len(res.Plan.Applied) != 0 {
		t.Errorf("dry-run applied %v, want nothing", res.Plan.Applied)
	}
	if !res.FingerprintOK {
		t.Error("FingerprintOK = false, want true (same cipher)")
	}

	// Crucially: a dry-run must not have written any destination file.
	for _, p := range []string{cfgPath, secPath, filepath.Join(notesDest, "a.md")} {
		if _, statErr := os.Stat(p); statErr == nil {
			t.Errorf("dry-run wrote %s — should change nothing", p)
		}
	}
}

func TestRestoreBackup_DryRunRejectsWrongKey(t *testing.T) {
	good, _ := NewCipher("the-right-passphrase-1")
	notes := filepath.Join(t.TempDir(), "notes")
	writeFile(t, filepath.Join(notes, "a.md"), "alpha")
	sealed := sealedFullInstanceBundle(t, good, "x", "y", notes)

	wrong, _ := NewCipher("a-different-passphrase-2")
	svc := &Service{cipher: wrong, cfg: Config{LocalDir: t.TempDir()}, log: slog.Default()}

	_, err := svc.RestoreBackup(context.Background(), RestoreRequest{
		Source: bytes.NewReader(sealed),
		Apply:  false,
	})
	if err == nil {
		t.Fatal("expected wrong-key bundle to fail, got nil")
	}
}

func TestInstallFile_BacksUpExisting(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "config.toml")
	writeFile(t, dst, "OLD")
	src := filepath.Join(dir, "staged")
	writeFile(t, src, "NEW")

	if err := installFile(src, dst); err != nil {
		t.Fatalf("installFile: %v", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "NEW" {
		t.Errorf("dst = %q, want NEW", got)
	}
	if got, _ := os.ReadFile(dst + ".bak"); string(got) != "OLD" {
		t.Errorf("dst.bak = %q, want OLD", got)
	}
}

func TestInstallFile_RollsBackOnFailure(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "config.toml")
	writeFile(t, dst, "OLD")

	// Staged source missing → open fails *after* dst was renamed aside.
	err := installFile(filepath.Join(dir, "does-not-exist"), dst)
	if err == nil {
		t.Fatal("expected installFile to fail on missing source")
	}
	// The original must be rolled back into place, not left missing.
	if got, rerr := os.ReadFile(dst); rerr != nil || string(got) != "OLD" {
		t.Errorf("original not rolled back: got=%q err=%v", got, rerr)
	}
}

func TestInstallFile_PreservesExistingBak(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "secrets.env")
	writeFile(t, dst, "V2")
	writeFile(t, dst+".bak", "V1-last-good")
	src := filepath.Join(dir, "staged")
	writeFile(t, src, "V3")

	if err := installFile(src, dst); err != nil {
		t.Fatalf("installFile: %v", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "V3" {
		t.Errorf("dst = %q, want V3", got)
	}
	// The pre-existing .bak (last good copy) must NOT be clobbered.
	if got, _ := os.ReadFile(dst + ".bak"); string(got) != "V1-last-good" {
		t.Errorf(".bak was clobbered: %q", got)
	}
}

func TestInspectVaultTar(t *testing.T) {
	srcRoot := t.TempDir()
	writeFile(t, filepath.Join(srcRoot, "notes", "a.md"), "a")
	writeFile(t, filepath.Join(srcRoot, "skills", "s", "SKILL.md"), "s")

	tarPath := filepath.Join(t.TempDir(), "v.tar")
	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := PackVault(f, []VaultSource{
		{Logical: "notes", Dir: filepath.Join(srcRoot, "notes")},
		{Logical: "skills", Dir: filepath.Join(srcRoot, "skills")},
	}); err != nil {
		t.Fatalf("PackVault: %v", err)
	}
	f.Close()

	roots, n, err := inspectVaultTar(tarPath)
	if err != nil {
		t.Fatalf("inspectVaultTar: %v", err)
	}
	if n != 2 {
		t.Errorf("files = %d, want 2", n)
	}
	got := map[string]bool{}
	for _, r := range roots {
		got[r] = true
	}
	if !got["notes"] || !got["skills"] {
		t.Errorf("roots = %v, want notes+skills", roots)
	}
}
