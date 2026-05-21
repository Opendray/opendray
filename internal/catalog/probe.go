package catalog

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RuntimeInfo is the live, probed state of a provider's CLI — distinct
// from the static Manifest. Populated by Prober at request time, never
// persisted. InstalledVersion is the real `<cli> --version` output (the
// thing the dashboard should show instead of the manifest's schema
// version); LatestVersion/UpdateAvailable come from the npm registry.
type RuntimeInfo struct {
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installedVersion,omitempty"`
	Path             string `json:"path,omitempty"`
	LatestVersion    string `json:"latestVersion,omitempty"`
	UpdateAvailable  bool   `json:"updateAvailable"`
	CheckedAt        string `json:"checkedAt,omitempty"` // RFC3339; when LatestVersion was fetched
}

// Cache TTLs: installed state is cheap (local exec) so a short TTL keeps
// the providers list fresh without re-probing on every poll; the npm
// lookup is a network call, so it's cached much longer and only run by
// the explicit update-check path.
const (
	installedTTL = 60 * time.Second
	latestTTL    = time.Hour
)

type cachedInstalled struct {
	info RuntimeInfo
	at   time.Time
}

type cachedLatest struct {
	version string
	at      time.Time
}

// Prober probes installed CLI versions (local exec) and latest npm
// versions (network), each with its own TTL cache. The exec/lookup
// functions are injectable so tests don't shell out.
type Prober struct {
	mu        sync.Mutex
	installed map[string]cachedInstalled // executable -> info
	latest    map[string]cachedLatest    // npm package -> version

	// updateMu serialises Update() so two concurrent npm installs can't
	// stomp the same global prefix.
	updateMu sync.Mutex

	lookPath   func(string) (string, error)
	runVer     func(ctx context.Context, bin string) (string, error)
	npmView    func(ctx context.Context, pkg string) (string, error)
	npmInstall func(ctx context.Context, pkg string) (string, error)
	now        func() time.Time
}

func NewProber() *Prober {
	return &Prober{
		installed:  map[string]cachedInstalled{},
		latest:     map[string]cachedLatest{},
		lookPath:   exec.LookPath,
		runVer:     defaultCliVersion,
		npmView:    defaultNpmLatest,
		npmInstall: defaultNpmInstall,
		now:        time.Now,
	}
}

// Installed reports whether the manifest's executable is on PATH and its
// `--version` string. Fast (local exec), cached for installedTTL.
func (p *Prober) Installed(ctx context.Context, m Manifest) RuntimeInfo {
	if m.Executable == "" {
		return RuntimeInfo{}
	}
	p.mu.Lock()
	if c, ok := p.installed[m.Executable]; ok && p.now().Sub(c.at) < installedTTL {
		info := c.info
		p.mu.Unlock()
		return info
	}
	p.mu.Unlock()

	var info RuntimeInfo
	if path, err := p.lookPath(m.Executable); err == nil {
		info.Installed = true
		info.Path = path
		if v, err := p.runVer(ctx, m.Executable); err == nil {
			info.InstalledVersion = v
		}
	}

	p.mu.Lock()
	p.installed[m.Executable] = cachedInstalled{info: info, at: p.now()}
	p.mu.Unlock()
	return info
}

// CheckUpdate returns Installed() enriched with the latest published npm
// version and an update-available flag. Network call, cached latestTTL.
func (p *Prober) CheckUpdate(ctx context.Context, m Manifest) RuntimeInfo {
	info := p.Installed(ctx, m)
	if m.NpmPackage == "" {
		return info
	}

	p.mu.Lock()
	c, ok := p.latest[m.NpmPackage]
	fresh := ok && p.now().Sub(c.at) < latestTTL
	latest := c.version
	p.mu.Unlock()

	if !fresh {
		if v, err := p.npmView(ctx, m.NpmPackage); err == nil && v != "" {
			latest = v
			p.mu.Lock()
			p.latest[m.NpmPackage] = cachedLatest{version: v, at: p.now()}
			p.mu.Unlock()
		}
	}

	info.LatestVersion = latest
	info.CheckedAt = p.now().UTC().Format(time.RFC3339)
	info.UpdateAvailable = updateAvailable(info.InstalledVersion, latest)
	return info
}

// UpdateResult reports the outcome of a provider CLI update.
type UpdateResult struct {
	Package       string `json:"package"`
	BeforeVersion string `json:"beforeVersion,omitempty"`
	AfterVersion  string `json:"afterVersion,omitempty"`
	Changed       bool   `json:"changed"`
	Output        string `json:"output,omitempty"` // tail of the npm output
}

// Update runs `npm install -g <pkg>` for the provider's CLI, then
// re-probes the version. Serialised across calls. The npm package name
// comes from the trusted manifest (never user input) — that is the
// whitelist. Whether the install succeeds depends on the npm global
// prefix being writable by the daemon's user; on a hardened deploy that
// means an opendray-owned prefix, otherwise this returns a permission
// error rather than escalating.
func (p *Prober) Update(ctx context.Context, m Manifest) (UpdateResult, error) {
	if m.NpmPackage == "" {
		return UpdateResult{}, fmt.Errorf("provider %q is not updatable via npm", m.ID)
	}

	p.updateMu.Lock()
	defer p.updateMu.Unlock()

	before := p.Installed(ctx, m).InstalledVersion
	out, err := p.npmInstall(ctx, m.NpmPackage)

	// The install may have changed what's on disk even on partial
	// failure, so always drop the cached install state.
	p.mu.Lock()
	delete(p.installed, m.Executable)
	p.mu.Unlock()

	res := UpdateResult{Package: m.NpmPackage, BeforeVersion: before, Output: tailLines(out, 40)}
	if err != nil {
		return res, fmt.Errorf("npm install -g %s: %w", m.NpmPackage, err)
	}
	res.AfterVersion = p.Installed(ctx, m).InstalledVersion
	res.Changed = res.AfterVersion != before
	return res, nil
}

// tailLines returns the last n lines of s (npm output can be long;
// callers only want the tail for diagnostics).
func tailLines(s string, n int) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// ── version comparison ───────────────────────────────────────────────

var semverRe = regexp.MustCompile(`\d+\.\d+\.\d+`)

// extractSemver pulls the first MAJOR.MINOR.PATCH out of a version
// string. CLIs decorate their --version output ("codex-cli 0.132.0",
// "2.1.146 (Claude Code)"); npm returns a bare "2.1.146".
func extractSemver(s string) string { return semverRe.FindString(s) }

// updateAvailable is true only when a clean latest version is strictly
// greater than the installed one — so a locally-ahead dev build never
// shows a spurious "update available".
func updateAvailable(installed, latest string) bool {
	iv := extractSemver(installed)
	lv := extractSemver(latest)
	if iv == "" || lv == "" {
		return false
	}
	return semverLess(iv, lv)
}

func semverLess(a, b string) bool {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		ai, _ := strconv.Atoi(ap[i])
		bi, _ := strconv.Atoi(bp[i])
		if ai != bi {
			return ai < bi
		}
	}
	return false
}

// ── default probes (shell out) ───────────────────────────────────────

func defaultCliVersion(ctx context.Context, bin string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--version")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	if sc.Scan() {
		return strings.TrimSpace(sc.Text()), nil
	}
	return "", nil
}

func defaultNpmLatest(ctx context.Context, pkg string) (string, error) {
	if _, err := exec.LookPath("npm"); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "npm", "view", pkg, "version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func defaultNpmInstall(ctx context.Context, pkg string) (string, error) {
	if _, err := exec.LookPath("npm"); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	// CombinedOutput so npm's progress/errors (incl. EACCES on a
	// non-writable prefix) come back to the operator.
	out, err := exec.CommandContext(ctx, "npm", "install", "-g", pkg).CombinedOutput()
	return string(out), err
}
