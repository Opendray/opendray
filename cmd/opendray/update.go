// `opendray update` — check + apply the latest opendray release.
//
//	opendray update              check and apply if newer (interactive confirm)
//	opendray update --check      only print current vs latest; don't apply
//	opendray update --force      re-download + replace even if already on latest
//	opendray update --yes        skip the interactive confirmation prompt
//	opendray update --restart    after replace, restart the systemd unit
//	                             `opendray` (Linux only — no-op on macOS LaunchAgent)
//
// Mechanics: hits GitHub's releases-latest endpoint, downloads the
// goreleaser tarball matching this host's GOOS/GOARCH, verifies the
// SHA-256 against the release's SHA256SUMS, then atomically replaces
// /proc/self/exe via temp+rename. The running process keeps using
// the OLD inode until exec — that's why the unit needs a restart to
// pick up the new code (we don't auto-restart unless asked).
//
// Privileges: the wizard installs the binary at /usr/local/bin/opendray
// (Linux) or ~/.opendray/bin/opendray (macOS). The Linux path is
// root-owned; running `opendray update` as a non-root user there
// fails fast with a "try with sudo" hint rather than silently
// reading old binary.
package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/version"
)

const updateReleasesAPI = "https://api.github.com/repos/Opendray/opendray_v2/releases/latest"

type ghReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string           `json:"tag_name"`
	Assets  []ghReleaseAsset `json:"assets"`
}

func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	checkOnly := fs.Bool("check", false, "only print current vs latest; don't apply")
	force := fs.Bool("force", false, "re-download + replace even when already on the latest")
	yes := fs.Bool("yes", false, "skip the interactive confirmation prompt")
	restart := fs.Bool("restart", false, "after replacing the binary, restart the 'opendray' systemd unit")
	_ = fs.Parse(args)

	current := normaliseVersion(version.Version)
	fmt.Printf("Current opendray:   %s\n", current)

	rel, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch latest release: %v\n", err)
		return 1
	}
	latest := normaliseVersion(rel.TagName)
	fmt.Printf("Latest release:     %s\n", latest)

	if !*force && latest == current {
		fmt.Println("Already on the latest release. (pass --force to re-install anyway)")
		return 0
	}
	if *checkOnly {
		if latest != current {
			fmt.Println("(--check set; not applying — re-run without --check to upgrade)")
		}
		return 0
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH
	tarballName := fmt.Sprintf("opendray_%s_%s_%s.tar.gz", latest, osName, arch)

	var tarballURL, sumsURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case tarballName:
			tarballURL = a.DownloadURL
		case "SHA256SUMS":
			sumsURL = a.DownloadURL
		}
	}
	if tarballURL == "" {
		fmt.Fprintf(os.Stderr, "no release asset matches %s — this host's %s/%s build is not published.\n", tarballName, osName, arch)
		return 1
	}

	selfPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to locate current binary: %v\n", err)
		return 1
	}
	// Resolve symlinks so we don't write the new binary to a symlink
	// while leaving the actual file untouched.
	if resolved, err := filepath.EvalSymlinks(selfPath); err == nil {
		selfPath = resolved
	}

	if err := canWriteSameDir(selfPath); err != nil {
		fmt.Fprintf(os.Stderr, "can't replace %s: %v\n", selfPath, err)
		fmt.Fprintln(os.Stderr, "Re-run with elevated privileges, e.g. `sudo opendray update`.")
		return 1
	}

	fmt.Printf("Will replace:       %s\n", selfPath)
	fmt.Printf("Tarball:            %s\n", tarballURL)
	if !*yes {
		if !askYes(fmt.Sprintf("\nProceed to upgrade %s → %s? [y/N]: ", current, latest)) {
			fmt.Println("Aborted.")
			return 0
		}
	}

	tarballPath, err := downloadToTemp(tarballURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "download failed: %v\n", err)
		return 1
	}
	defer os.Remove(tarballPath)
	fmt.Printf("[✓] downloaded\n")

	if sumsURL != "" {
		if err := verifyChecksum(tarballPath, sumsURL, tarballName); err != nil {
			fmt.Fprintf(os.Stderr, "checksum verification FAILED: %v\n", err)
			return 1
		}
		fmt.Println("[✓] SHA-256 verified")
	} else {
		fmt.Fprintln(os.Stderr, "[!] SHA256SUMS not found on the release — skipping checksum check.")
	}

	binPath, cleanup, err := extractOpendrayBinary(tarballPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract failed: %v\n", err)
		return 1
	}
	defer cleanup()
	fmt.Println("[✓] extracted")

	if err := atomicReplace(binPath, selfPath); err != nil {
		fmt.Fprintf(os.Stderr, "binary replace failed: %v\n", err)
		return 1
	}
	fmt.Printf("[✓] installed:        %s\n", selfPath)

	if *restart && runtime.GOOS == "linux" {
		fmt.Println("Restarting systemd unit 'opendray'...")
		cmd := exec.Command("systemctl", "restart", "opendray")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[!] systemctl restart failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "    Restart manually with: sudo systemctl restart opendray")
			return 1
		}
		fmt.Println("[✓] service restarted")
	} else {
		fmt.Println("\nNew binary is in place but the running service is still on the old inode.")
		fmt.Println("Restart to pick up the new code:")
		switch runtime.GOOS {
		case "linux":
			fmt.Println("  sudo systemctl restart opendray")
		case "darwin":
			fmt.Println("  launchctl kickstart -k gui/$(id -u)/com.opendray.opendray")
		}
		fmt.Println("\nIf this release contains schema migrations, also run:")
		fmt.Println("  sudo -u opendray opendray migrate -config /etc/opendray/config.toml   # Linux")
		fmt.Println("  opendray migrate -config ~/.opendray/config.toml                       # macOS")
	}

	return 0
}

// ── helpers ──────────────────────────────────────────────────────────

// normaliseVersion strips a leading "v" so "v2.0.0" and "2.0.0" compare equal.
func normaliseVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func fetchLatestRelease() (*ghRelease, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", updateReleasesAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "opendray-update/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release JSON: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("github returned an empty tag_name (rate-limited? auth needed?)")
	}
	return &rel, nil
}

// canWriteSameDir tries to create + remove a temp file in the directory
// containing the target path, as a proxy for "we can rename onto it".
func canWriteSameDir(targetPath string) error {
	dir := filepath.Dir(targetPath)
	f, err := os.CreateTemp(dir, ".opendray-update-test-")
	if err != nil {
		return fmt.Errorf("no write access to %s (%w)", dir, err)
	}
	name := f.Name()
	f.Close()
	_ = os.Remove(name)
	return nil
}

func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d while fetching %s", resp.StatusCode, url)
	}
	f, err := os.CreateTemp("", "opendray-update-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("tempfile: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("write tempfile: %w", err)
	}
	return f.Name(), nil
}

func verifyChecksum(tarballPath, sumsURL, tarballName string) error {
	resp, err := http.Get(sumsURL)
	if err != nil {
		return fmt.Errorf("fetch SHA256SUMS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("SHA256SUMS HTTP %d", resp.StatusCode)
	}
	var expected string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		// "<sha256>  <name>" — accept either single or double space.
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 && fields[len(fields)-1] == tarballName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("scan SHA256SUMS: %w", err)
	}
	if expected == "" {
		return fmt.Errorf("%s not listed in SHA256SUMS", tarballName)
	}

	f, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("open tarball: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash tarball: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, got)
	}
	return nil
}

func extractOpendrayBinary(tarballPath string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "opendray-update-extract-")
	if err != nil {
		return "", func() {}, fmt.Errorf("tempdir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	f, err := os.Open(tarballPath)
	if err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("open tarball: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var binDest string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			cleanup()
			return "", func() {}, fmt.Errorf("tar: %w", err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		// goreleaser tarball has `opendray` at the root.
		base := filepath.Base(h.Name)
		if base != "opendray" {
			continue
		}
		// Reject suspicious entries to be safe.
		if strings.Contains(h.Name, "..") {
			cleanup()
			return "", func() {}, fmt.Errorf("tarball contains a suspicious path: %q", h.Name)
		}
		binDest = filepath.Join(tmpDir, "opendray")
		dst, err := os.OpenFile(binDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			cleanup()
			return "", func() {}, fmt.Errorf("create dest: %w", err)
		}
		if _, err := io.Copy(dst, tr); err != nil {
			dst.Close()
			cleanup()
			return "", func() {}, fmt.Errorf("copy: %w", err)
		}
		dst.Close()
		break
	}
	if binDest == "" {
		cleanup()
		return "", func() {}, fmt.Errorf("no 'opendray' binary inside tarball")
	}
	return binDest, cleanup, nil
}

// atomicReplace puts newBin at targetPath via temp+rename in the same
// directory, so the swap is atomic even if the running process still
// holds the old inode.
func atomicReplace(newBin, targetPath string) error {
	tmp := targetPath + ".new"
	src, err := os.Open(newBin)
	if err != nil {
		return fmt.Errorf("open new binary: %w", err)
	}
	defer src.Close()
	dst, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create %s: %w", tmp, err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("copy to %s: %w", tmp, err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, targetPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, targetPath, err)
	}
	return nil
}

// askYes prompts on stdin/stderr and returns true on y / Y / yes.
// Always defaults to NO so accidental Enter doesn't trigger destructive
// actions.
func askYes(prompt string) bool {
	fmt.Fprint(os.Stderr, prompt)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(sc.Text()))
	return answer == "y" || answer == "yes"
}
