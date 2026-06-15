package backup

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// vaultEpoch is the fixed mod-time stamped on every vault tar entry.
// File mtimes aren't restorable content — UnpackVault forces 0600 mode
// and never restores times — so pinning them makes vault.tar
// byte-stable for identical file contents, which lets content-dedup
// recognise an unchanged vault across backups. A non-zero constant
// keeps tooling that dislikes the zero time happy.
var vaultEpoch = time.Unix(0, 0).UTC()

// VaultSource is one logical directory captured into a vault tarball.
// Logical is the root name recorded in the archive (e.g. "notes",
// "skills", "mcp"); Dir is the absolute on-disk directory to walk.
// Restore maps Logical back to a destination directory, so the two
// ends agree on layout without hard-coding paths in the archive.
type VaultSource struct {
	Logical string
	Dir     string
}

// vaultSkipDirs are directory names never captured: version-control
// metadata is large and recoverable from the git remote (the vault
// already syncs to git), so it only bloats the bundle.
var vaultSkipDirs = map[string]bool{".git": true}

// PackVault writes an uncompressed tar of the given source directories
// to w. Files are stored under "<Logical>/<rel>". Symlinks and other
// non-regular files are skipped (never followed) so the archive can't
// reach outside the source tree. The outer backup bundle gzips this
// stream, so PackVault deliberately does not compress.
//
// A configured-but-absent source directory is skipped rather than
// failing the whole backup — a fresh install may not have every dir.
func PackVault(w io.Writer, sources []VaultSource) error {
	tw := tar.NewWriter(w)
	for _, src := range sources {
		if src.Logical == "" || src.Logical == ".." || strings.ContainsRune(src.Logical, '/') {
			return fmt.Errorf("vaulttar: invalid logical %q", src.Logical)
		}
		if err := packOne(tw, src); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("vaulttar: close tar: %w", err)
	}
	return nil
}

func packOne(tw *tar.Writer, src VaultSource) error {
	root := filepath.Clean(src.Dir)
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // absent dir is not fatal
		}
		return fmt.Errorf("vaulttar: stat %s: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("vaulttar: %s is not a directory", root)
	}

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if vaultSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip anything that isn't a regular file (symlinks, sockets,
		// devices, fifos) — following them could escape the tree.
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		arcName := filepath.ToSlash(filepath.Join(src.Logical, rel))
		hdr := &tar.Header{
			Name: arcName,
			Size: fi.Size(),
			Mode: 0o600,
			// Pinned, not the real mtime — see vaultEpoch. Keeps the tar
			// byte-stable so content-dedup can match an unchanged vault.
			ModTime:  vaultEpoch,
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("vaulttar: header %s: %w", arcName, err)
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		n, copyErr := io.Copy(tw, f)
		_ = f.Close()
		if copyErr != nil {
			return fmt.Errorf("vaulttar: copy %s: %w", arcName, copyErr)
		}
		if n != fi.Size() {
			return fmt.Errorf("vaulttar: short copy %s: %d != %d", arcName, n, fi.Size())
		}
		return nil
	})
}

// UnpackVault extracts a tar produced by PackVault. destFor maps a
// logical root (the first path element of each entry) to its on-disk
// destination directory; returning ok=false skips that entry's root,
// so a bundle carrying roots this build doesn't recognise is tolerated.
//
// Security: entries whose cleaned path would escape the destination
// are rejected (tar-slip / zip-slip guard), as are absolute paths and
// non-regular entries (symlinks/hardlinks/devices are never written).
//
// Returns the number of files written.
func UnpackVault(r io.Reader, destFor func(logical string) (dest string, ok bool)) (int, error) {
	tr := tar.NewReader(r)
	written := 0
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return written, fmt.Errorf("vaulttar: read: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue // only plain files are restored
		}
		name := filepath.ToSlash(hdr.Name)
		if strings.HasPrefix(name, "/") {
			return written, fmt.Errorf("vaulttar: absolute path entry %q", hdr.Name)
		}
		logical, rest, ok := strings.Cut(name, "/")
		if !ok || rest == "" {
			continue // no logical root — nothing to map
		}
		dest, ok := destFor(logical)
		if !ok {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(rest))
		if !withinDir(filepath.Clean(dest), target) {
			return written, fmt.Errorf("vaulttar: entry %q escapes destination", hdr.Name)
		}
		if err := writeFileFromTar(target, tr); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

func writeFileFromTar(target string, tr *tar.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("vaulttar: mkdir: %w", err)
	}
	// Mode is deliberately forced to 0600 rather than honouring the tar
	// header: the vault may hold secrets (mcp manifests, notes) and we
	// never want a permissive mode smuggled in via a crafted archive.
	f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("vaulttar: create %s: %w", target, err)
	}
	_, copyErr := io.Copy(f, tr)
	closeErr := f.Close()
	if copyErr != nil {
		return fmt.Errorf("vaulttar: write %s: %w", target, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("vaulttar: close %s: %w", target, closeErr)
	}
	return nil
}

// withinDir reports whether target resolves to a path inside dir.
func withinDir(dir, target string) bool {
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
