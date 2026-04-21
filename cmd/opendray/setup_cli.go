// Package main — `opendray setup` — interactive terminal wizard.
//
// Design notes (important, keep when editing):
//
//   - This is the ONLY supported first-run path. OpenDray no longer runs
//     a browser-based setup wizard. If config is missing the server binary
//     refuses to start and directs the operator here.
//
//   - The wizard is driven by a small step machine. Each step reads some
//     input, validates, and returns one of {next, back, quit}. `back`
//     lets the user type `back` at any prompt to revisit the previous
//     step — a usability gain over the old wizard where a typo in step
//     3 meant Ctrl-C and redo.
//
//   - Resume mode: when a complete config already exists, the wizard
//     loads its values as defaults. Typing `enter` at every prompt
//     reproduces the existing config — useful for "I just want to
//     rotate the JWT" flows without hand-editing TOML.
//
//   - Scripted mode: `--yes` with the required flags bypasses prompts
//     entirely for CI / Ansible / cloud-init.
//
//   - Listen address is a first-class question now. Loopback (default,
//     127.0.0.1:8640) is safe for single-host installs. All-interfaces
//     (0.0.0.0:8640) is required for LAN / reverse-proxy deployments
//     but surfaces the server to every network the host is on — the
//     wizard warns about this explicitly.
package main

import (
	"bufio"
	"context"
	"crypto/subtle"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/opendray/opendray/kernel/auth"
	"github.com/opendray/opendray/kernel/config"
	opg "github.com/opendray/opendray/kernel/pg"
	"github.com/opendray/opendray/kernel/store"
	"golang.org/x/term"
)

// ── visual primitives ───────────────────────────────────────────────
//
// Everything here writes to stderr so piping `opendray setup` into a
// log doesn't produce garbled progress output. User-visible text is
// plain ASCII + ANSI colors when stderr is a terminal and NO_COLOR
// is not set.

const (
	divider = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	arrow   = "›"
)

// minPasswordLen is the floor for admin passwords. 8 chars is the
// standard minimum — balances usability against brute-force risk on
// bcrypt-hashed credentials.
const minPasswordLen = 8

// sentinelBack is the string users type to jump back one step. Checked
// at every readLine call.
const sentinelBack = "back"

// ── ANSI styling ────────────────────────────────────────────────────
//
// All colorized output funnels through these helpers so respecting
// NO_COLOR and non-TTY outputs is a single flag flip, not 80 scattered
// \033[ literals.

var colorsEnabled = func() bool {
	// Explicit opt-out per https://no-color.org/
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return false
	}
	// Force-on for pipelines that want colors (tmux outer, log
	// processors that preserve SGR).
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}
	// Default: only when stderr is a real TTY. Piping to `tee` or a
	// file should stay plain ASCII.
	return term.IsTerminal(int(os.Stderr.Fd()))
}()

// sgr wraps s in an ANSI SGR sequence. Multiple codes may be passed;
// they're joined with ';' per the spec (\033[1;36m = bold cyan).
func sgr(s string, codes ...string) string {
	if !colorsEnabled || len(codes) == 0 {
		return s
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + s + "\x1b[0m"
}

// Style helpers — readable names > magic numbers at the call site.
func styleAccent(s string) string  { return sgr(s, "1", "38;5;177") } // bright purple
func styleTitle(s string) string   { return sgr(s, "1") }             // bold
func styleDim(s string) string     { return sgr(s, "2") }             // dim
func styleCyan(s string) string    { return sgr(s, "36") }
func styleBrightCyan(s string) string { return sgr(s, "1", "96") }
func styleGreen(s string) string   { return sgr(s, "32") }
func styleRed(s string) string     { return sgr(s, "31") }
func styleYellow(s string) string  { return sgr(s, "33") }
func styleReverse(s string) string { return sgr(s, "7") } // swapped fg/bg

// prn prints a line to stderr.
func prn(s string) {
	fmt.Fprintln(os.Stderr, s)
}

// prf is prn + Sprintf.
func prf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, args...))
}

// section prints a step header with its 1-based index. The header is
// intentionally big — three blank lines of breathing room on each
// side plus a two-row title block — so the transition between steps
// is unmissable even when the previous step left a wall of text.
func section(idx, total int, label string) {
	prn("")
	prn("")
	prn(styleAccent(divider))
	prn("")
	prf("     %s   %s   %s",
		styleAccent(fmt.Sprintf("STEP %d / %d", idx, total)),
		styleDim("·"),
		styleTitle(strings.ToUpper(label)))
	prn("")
	prn(styleAccent(divider))
	prn("")
	prn("")
}

// markers for progress / summary lines.
func okMark() string    { return styleGreen("✓") }
func failMark() string  { return styleRed("✗") }
func warnMark() string  { return styleYellow("⚠") }

// errf emits an inline validation error line (indented, red mark,
// trailing newline). Used for all "input invalid, try again" prompts.
func errf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "    %s %s\n", failMark(), fmt.Sprintf(format, args...))
}

// ── flags ───────────────────────────────────────────────────────────

type setupFlags struct {
	yes           bool
	db            string // bundled | external
	listenMode    string // loopback | all | <host:port>
	extHost       string
	extPort       int
	extUser       string
	extName       string
	extPassword   string
	extPWFile     string
	extSSLMode    string
	adminUser     string
	adminPassword string
	adminPWFile   string
	jwtSecret     string
	jwtSecretFile string
	overwrite     bool
}

func parseSetupFlags(args []string) (*setupFlags, error) {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	// Keep flag help off stdout — we emit our own banner and don't
	// want flag's default usage text leaking into the wizard.
	fs.SetOutput(io.Discard)

	out := &setupFlags{
		extSSLMode: "disable",
		extPort:    5432,
		listenMode: "",
	}
	fs.BoolVar(&out.yes, "yes", false, "non-interactive; apply from flags")
	fs.BoolVar(&out.overwrite, "overwrite", false, "overwrite existing config without asking")

	fs.StringVar(&out.db, "db", "", "bundled|external")

	fs.StringVar(&out.listenMode, "listen", "", "loopback|all|<host:port>")

	fs.StringVar(&out.extHost, "db-host", "", "external: host")
	fs.IntVar(&out.extPort, "db-port", 5432, "external: port")
	fs.StringVar(&out.extUser, "db-user", "", "external: user")
	fs.StringVar(&out.extName, "db-name", "", "external: database name")
	fs.StringVar(&out.extPassword, "db-password", "", "external: password (prefer --db-password-file)")
	fs.StringVar(&out.extPWFile, "db-password-file", "", "external: read password from file")
	fs.StringVar(&out.extSSLMode, "db-sslmode", "disable", "external: disable|require|verify-ca|verify-full")

	fs.StringVar(&out.adminUser, "admin-user", "", "admin username")
	fs.StringVar(&out.adminPassword, "admin-password", "", "admin password (prefer --admin-password-file)")
	fs.StringVar(&out.adminPWFile, "admin-password-file", "", "read admin password from file")

	fs.StringVar(&out.jwtSecret, "jwt-secret", "", "JWT signing secret (≥32 chars)")
	fs.StringVar(&out.jwtSecretFile, "jwt-secret-file", "", "read JWT secret from file")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return out, nil
}

// ── entry point ─────────────────────────────────────────────────────

// runSetupCLI is invoked by `opendray setup`. Returns the exit code.
func runSetupCLI() int {
	flags, err := parseSetupFlags(os.Args[2:])
	if err != nil {
		prf("setup: %v", err)
		return 2
	}

	banner(flags.yes)

	// Load prior config if any, so resume-mode has sensible defaults.
	prior, _, _ := config.Load()
	hasPrior := prior.IsComplete()

	if hasPrior && !flags.overwrite && !flags.yes {
		prn("")
		prn("A complete configuration already exists at:")
		paths := config.DefaultPaths()
		if len(paths) > 0 {
			prf("    %s", paths[0])
		}
		prn("")
		prn("Continuing will overwrite it. Existing values are offered as defaults,")
		prn("so pressing Enter at every prompt reproduces the current config.")
		prn("")
		in := bufio.NewReader(os.Stdin)
		if !askYN(in, "Continue?", true) {
			prn("Aborted.")
			return 0
		}
	}

	if flags.yes {
		return runScripted(flags)
	}
	return runInteractive(flags, prior, hasPrior)
}

// asciiLogo is the opening splash. Rendered in the accent color; six
// lines tall so it's unmissable even on a 120-column SSH session.
// Generated from the "ANSI Shadow" figlet font; copied verbatim so
// there's no run-time figlet dependency.
const asciiLogo = `
   ██████╗ ██████╗ ███████╗███╗   ██╗██████╗ ██████╗  █████╗ ██╗   ██╗
  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔══██╗██╔══██╗╚██╗ ██╔╝
  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██████╔╝███████║ ╚████╔╝
  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██╔══██╗██╔══██║  ╚██╔╝
  ╚██████╔╝██║     ███████╗██║ ╚████║██████╔╝██║  ██║██║  ██║   ██║
   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝`

// banner is the opening screen. Keeps prose short — users who got
// here via `curl | sh` have already read install-time messages.
func banner(scripted bool) {
	prn("")
	// Each line of the logo is colored separately so terminals that
	// clip wide output still produce readable partial rows instead of
	// dropping orphaned reset codes.
	for _, line := range strings.Split(strings.TrimLeft(asciiLogo, "\n"), "\n") {
		prn(styleAccent(line))
	}
	prn("")
	prf("   %s  %s  %s",
		styleTitle("SETUP WIZARD"),
		styleDim("·"),
		styleDim(fmt.Sprintf("version %s", version)))
	prn("")
	prn("")
	prn(styleTitle("   This will walk you through:"))
	prn("")
	prn(styleDim("     1.") + "   Write config to   " + styleCyan("~/.opendray/config.toml"))
	prn(styleDim("     2.") + "   Provision database (bundled or external)")
	prn(styleDim("     3.") + "   Create admin account")
	prn(styleDim("     4.") + "   Run migrations")
	prn("")
	prn("")
	if !scripted {
		prn(styleDim("   Press ") +
			styleTitle("⏎") +
			styleDim(" to begin  ·  ") +
			styleTitle("Ctrl-C") +
			styleDim(" to abort  ·  type ") +
			styleTitle("back") +
			styleDim(" at any prompt to revisit the previous step."))
		prn("")
		in := bufio.NewReader(os.Stdin)
		_, _ = in.ReadString('\n')
	}
}

// ── interactive path ────────────────────────────────────────────────

const totalSteps = 4

func runInteractive(flags *setupFlags, prior config.Config, hasPrior bool) int {
	cfg := config.Defaults()
	if hasPrior {
		cfg = prior
	}
	// Always re-set SchemaVersion (Defaults() already does; a prior
	// config from an older binary might have a lower number).
	cfg.SchemaVersion = config.SchemaVersion

	in := bufio.NewReader(os.Stdin)

	// Step machine. Each step returns (next, back, quit).
	steps := []func() stepResult{
		func() stepResult { return stepDatabase(in, &cfg) },
		func() stepResult { return stepListen(in, &cfg) },
		func() stepResult { return stepAdmin(in, &cfg) },
		func() stepResult { return stepJWT(in, &cfg) },
	}
	idx := 0
	for idx < len(steps) {
		section(idx+1, totalSteps, stepLabels[idx])
		switch steps[idx]() {
		case srNext:
			idx++
		case srBack:
			if idx > 0 {
				idx--
			}
		case srQuit:
			prn("")
			prn("Aborted.")
			return 0
		}
	}

	// Summary + confirm.
	prn("")
	prn("")
	prn(styleAccent(divider))
	prn("")
	prf("     %s", styleTitle("SUMMARY"))
	prn("")
	prn(styleAccent(divider))
	prn("")
	prn("")
	writeSummary(cfg)
	prn("")
	prn("")
	if !askYN(in, "Apply this configuration?", true) {
		prn("")
		prn(styleYellow("   Aborted. Nothing written."))
		prn("")
		return 0
	}

	return apply(cfg)
}

var stepLabels = []string{
	"database",
	"listen address",
	"admin account",
	"jwt secret",
}

// ── scripted path ───────────────────────────────────────────────────

func runScripted(flags *setupFlags) int {
	cfg := config.Defaults()
	cfg.SchemaVersion = config.SchemaVersion

	// DATABASE
	switch flags.db {
	case "bundled", "embedded":
		cfg.DB.Mode = "embedded"
		if err := guardRootForEmbedded(); err != nil {
			prf("setup: %v", err)
			return 1
		}
	case "external":
		cfg.DB.Mode = "external"
		if flags.extHost == "" || flags.extUser == "" || flags.extName == "" {
			prn("setup: --db=external requires --db-host, --db-user, --db-name")
			return 2
		}
		pw, err := loadSecretFromFlags(flags.extPassword, flags.extPWFile, "--db-password")
		if err != nil {
			prf("setup: %v", err)
			return 2
		}
		cfg.DB.External = config.ExternalDB{
			Host:     flags.extHost,
			Port:     flags.extPort,
			User:     flags.extUser,
			Password: pw,
			Name:     flags.extName,
			SSLMode:  flags.extSSLMode,
		}
		if err := testExternalDB(cfg.DB.External); err != nil {
			prf("setup: external database unreachable: %v", err)
			return 1
		}
	default:
		prn("setup: --db=bundled|external required with --yes")
		return 2
	}

	// LISTEN
	if flags.listenMode != "" {
		addr, err := resolveListenAddr(flags.listenMode)
		if err != nil {
			prf("setup: %v", err)
			return 2
		}
		cfg.Server.ListenAddr = addr
	}

	// ADMIN
	if flags.adminUser == "" {
		prn("setup: --admin-user required with --yes")
		return 2
	}
	pw, err := loadSecretFromFlags(flags.adminPassword, flags.adminPWFile, "--admin-password")
	if err != nil {
		prf("setup: %v", err)
		return 2
	}
	if len(pw) < minPasswordLen {
		prf("setup: admin password must be at least %d characters", minPasswordLen)
		return 2
	}
	cfg.Auth.AdminBootstrapUsername = flags.adminUser
	cfg.Auth.AdminBootstrapPassword = pw

	// JWT
	jwt, err := loadSecretFromFlags(flags.jwtSecret, flags.jwtSecretFile, "--jwt-secret")
	if err != nil {
		prf("setup: %v", err)
		return 2
	}
	if jwt == "" {
		gen, err := config.GenerateJWTSecret()
		if err != nil {
			prf("setup: cannot generate JWT secret: %v", err)
			return 1
		}
		jwt = gen
	} else if len(jwt) < 32 {
		prn("setup: --jwt-secret must be at least 32 characters")
		return 2
	}
	cfg.Auth.JWTSecret = jwt

	prn("")
	writeSummary(cfg)
	prn("")
	return apply(cfg)
}

// ── step: database ──────────────────────────────────────────────────

func stepDatabase(in *bufio.Reader, cfg *config.Config) stepResult {
	defaultChoice := "bundled"
	if cfg.DB.Mode == "external" {
		defaultChoice = "external"
	}

	items := []menuItem{
		{
			Key:   "bundled",
			Label: "bundled",
			Desc:  "Self-contained. OpenDray manages its own loopback-only\nPostgres child process. First run downloads ~50 MB.\nRecommended for single-host installs.",
		},
		{
			Key:   "external",
			Label: "external",
			Desc:  "Bring your own PostgreSQL 14+. Requires a database,\na role with CRUD privileges, and network reach from\nthis host.",
		},
	}
	choice, r := pickMenu("How should OpenDray get its database?", items, defaultChoice)
	if r != srNext {
		return r
	}
	if choice == "bundled" {
		return stepDatabaseBundled(in, cfg)
	}
	return stepDatabaseExternal(in, cfg)
}

func stepDatabaseBundled(in *bufio.Reader, cfg *config.Config) stepResult {
	if err := guardRootForEmbedded(); err != nil {
		prn("")
		prf(" %s %v", failMark(), err)
		prn("")
		prn(styleTitle(" Fix — create an unprivileged user and re-run as that user:"))
		prn("")
		prn(styleBrightCyan("     useradd -r -m -s /bin/bash -d /home/opendray opendray"))
		prn(styleBrightCyan("     su - opendray"))
		prn(styleBrightCyan("     opendray setup"))
		prn("")
		prn(styleDim(" Or choose `external` to use an existing PostgreSQL."))
		return srBack
	}

	cfg.DB.Mode = "embedded"
	if cfg.DB.Embedded.DataDir == "" {
		cfg.DB.Embedded.DataDir = expandHomeCLI("~/.opendray/pg")
	}
	if cfg.DB.Embedded.CacheDir == "" {
		cfg.DB.Embedded.CacheDir = expandHomeCLI("~/.opendray/pg-cache")
	}
	if cfg.DB.Embedded.Port == 0 {
		cfg.DB.Embedded.Port = 5433
	}

	prn("")
	prn(" Data directory — holds the PostgreSQL cluster files. OpenDray")
	prn(" is the only process that should ever touch it.")
	v, r := readLine(in, "data dir", cfg.DB.Embedded.DataDir)
	if r != srNext {
		return r
	}
	cfg.DB.Embedded.DataDir = expandHomeCLI(v)

	prn("")
	prn(" Binary cache — the downloaded Postgres binary is kept here")
	prn(" so subsequent starts don't re-download the ~50 MB tarball.")
	v, r = readLine(in, "cache dir", cfg.DB.Embedded.CacheDir)
	if r != srNext {
		return r
	}
	cfg.DB.Embedded.CacheDir = expandHomeCLI(v)

	prn("")
	prn(" Postgres port — loopback only. Pick any free port.")
	for {
		v, r = readLine(in, "port", strconv.Itoa(cfg.DB.Embedded.Port))
		if r != srNext {
			return r
		}
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			errf("must be an integer in 1–65535")
			continue
		}
		if !portAvailable("127.0.0.1", p) {
			errf("port %d is already in use on 127.0.0.1", p)
			continue
		}
		cfg.DB.Embedded.Port = p
		break
	}
	return srNext
}

func stepDatabaseExternal(in *bufio.Reader, cfg *config.Config) stepResult {
	cfg.DB.Mode = "external"
	// Wipe bundled Password so we don't round-trip it when the user
	// switches modes mid-wizard.
	cfg.DB.Embedded.Password = ""

	prn("")
	for {
		v, r := readLine(in, "host", cfg.DB.External.Host)
		if r != srNext {
			return r
		}
		if v != "" {
			cfg.DB.External.Host = v
			break
		}
		errf("host is required")
	}

	defPort := cfg.DB.External.Port
	if defPort == 0 {
		defPort = 5432
	}
	for {
		v, r := readLine(in, "port", strconv.Itoa(defPort))
		if r != srNext {
			return r
		}
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			errf("must be an integer in 1–65535")
			continue
		}
		cfg.DB.External.Port = p
		break
	}

	for {
		v, r := readLine(in, "user", cfg.DB.External.User)
		if r != srNext {
			return r
		}
		if v != "" {
			cfg.DB.External.User = v
			break
		}
		errf("user is required")
	}

	for {
		v, r := readSecret(in, "password")
		if r != srNext {
			return r
		}
		if v != "" {
			cfg.DB.External.Password = v
			break
		}
		// Allow reusing an existing password from prior config.
		if cfg.DB.External.Password != "" {
			prn(styleDim("    · reusing password from existing config"))
			break
		}
		errf("password is required")
	}

	for {
		v, r := readLine(in, "database name", cfg.DB.External.Name)
		if r != srNext {
			return r
		}
		if v != "" {
			cfg.DB.External.Name = v
			break
		}
		errf("database name is required")
	}

	defSSL := cfg.DB.External.SSLMode
	if defSSL == "" {
		defSSL = "disable"
	}
	for {
		v, r := readLine(in, "ssl mode", defSSL)
		if r != srNext {
			return r
		}
		switch v {
		case "disable", "require", "verify-ca", "verify-full":
			cfg.DB.External.SSLMode = v
			goto sslOk
		}
		errf("must be one of: disable | require | verify-ca | verify-full")
	}
sslOk:

	// Test connection before advancing. No "save anyway" escape —
	// saving an unreachable config only produces confusion later.
	prn("")
	prf(" %s  Testing connection…", styleAccent("→"))
	start := time.Now()
	if err := testExternalDB(cfg.DB.External); err != nil {
		took := time.Since(start).Round(time.Millisecond)
		prf(" %s  connection failed after %s: %v", failMark(), took, err)
		prn("")
		prn(styleDim(" Fix the connection details and try again."))
		prn(styleDim(" Press Enter to re-enter, or type `back` to return to database choice."))
		_, r := readLine(in, "", "")
		if r == srBack {
			return srBack
		}
		return stepDatabaseExternal(in, cfg) // replay this step
	}
	prf(" %s  connected to %s %s",
		okMark(),
		styleBrightCyan(fmt.Sprintf("%s@%s:%d/%s",
			cfg.DB.External.User, cfg.DB.External.Host,
			cfg.DB.External.Port, cfg.DB.External.Name)),
		styleDim(fmt.Sprintf("(%s)", time.Since(start).Round(time.Millisecond))))
	return srNext
}

// ── step: listen address ────────────────────────────────────────────

func stepListen(in *bufio.Reader, cfg *config.Config) stepResult {
	defaultChoice := "loopback"
	isCustom := false
	if strings.HasPrefix(cfg.Server.ListenAddr, "0.0.0.0") {
		defaultChoice = "all"
	} else if cfg.Server.ListenAddr != "" && !strings.HasPrefix(cfg.Server.ListenAddr, "127.") {
		defaultChoice = "custom"
		isCustom = true
	}

	items := []menuItem{
		{
			Key:   "loopback",
			Label: "loopback",
			Desc:  "127.0.0.1:8640 — local only. The UI is reachable from\nthis machine only. Safe default.",
		},
		{
			Key:   "all",
			Label: "all",
			Desc:  "0.0.0.0:8640 — any network interface. Needed for LAN\naccess or a reverse proxy. ⚠  surface-exposes the UI;\nmake sure the JWT secret is strong.",
		},
		{
			Key:   "custom",
			Label: "custom",
			Desc:  "Type a host:port (e.g. 10.0.0.5:8640).",
		},
	}

	choice, r := pickMenu("Where should the OpenDray server listen?", items, defaultChoice)
	if r != srNext {
		return r
	}

	if choice == "custom" {
		defAddr := ""
		if isCustom {
			defAddr = cfg.Server.ListenAddr
		}
		prn("")
		for {
			v, rr := readLine(in, "host:port", defAddr)
			if rr != srNext {
				return rr
			}
			addr, err := resolveListenAddr(v)
			if err != nil {
				prf("    %s %v", failMark(), err)
				continue
			}
			cfg.Server.ListenAddr = addr
			return srNext
		}
	}

	addr, err := resolveListenAddr(choice)
	if err != nil {
		prf("    %s %v", failMark(), err)
		return srBack
	}
	cfg.Server.ListenAddr = addr
	return srNext
}

// resolveListenAddr turns the user's shorthand into a real host:port.
// Accepts:  "loopback" | "all" | "<host:port>"
func resolveListenAddr(s string) (string, error) {
	switch strings.ToLower(s) {
	case "loopback", "local", "lo", "127.0.0.1":
		return "127.0.0.1:8640", nil
	case "all", "any", "0.0.0.0", "public":
		return "0.0.0.0:8640", nil
	}
	// Must look like host:port.
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return "", fmt.Errorf("expected `loopback`, `all`, or `host:port` — got %q", s)
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return "", fmt.Errorf("invalid port %q", port)
	}
	if host == "" {
		return "", fmt.Errorf("host is required in host:port")
	}
	return s, nil
}

// ── step: admin account ─────────────────────────────────────────────

func stepAdmin(in *bufio.Reader, cfg *config.Config) stepResult {
	prn(" The admin account is the first user who can log into the web UI.")
	prn(" You can add more users later from the UI.")
	prn("")

	defUser := cfg.Auth.AdminBootstrapUsername
	if defUser == "" {
		defUser = "admin"
	}
	for {
		v, r := readLine(in, "username", defUser)
		if r != srNext {
			return r
		}
		if v != "" {
			cfg.Auth.AdminBootstrapUsername = v
			break
		}
		errf("username is required")
	}

	prn("")
	prf(" Password — minimum %d characters. Typed input is hidden.", minPasswordLen)
	prn("")

	for {
		pw, r := readSecret(in, "password")
		if r != srNext {
			return r
		}
		if len(pw) < minPasswordLen {
			errf("too short — %d characters, need at least %d", len(pw), minPasswordLen)
			continue
		}
		prf("    %s strength: %s", styleDim("·"), strengthDescription(pw))

		confirm, r := readSecret(in, "confirm")
		if r != srNext {
			return r
		}
		if subtle.ConstantTimeCompare([]byte(pw), []byte(confirm)) != 1 {
			errf("passwords don't match, try again")
			continue
		}
		cfg.Auth.AdminBootstrapPassword = pw
		return srNext
	}
}

// strengthDescription gives a terse qualitative note on password
// strength. No numeric score — that implies more precision than a
// heuristic deserves, and invites users to game the metric instead of
// writing a strong password.
func strengthDescription(pw string) string {
	var lower, upper, digit, symbol bool
	for _, r := range pw {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			symbol = true
		}
	}
	classes := 0
	for _, b := range []bool{lower, upper, digit, symbol} {
		if b {
			classes++
		}
	}
	switch {
	case len(pw) >= 16 && classes >= 3:
		return fmt.Sprintf("strong · %d chars", len(pw))
	case len(pw) >= 12 && classes >= 2:
		return fmt.Sprintf("ok · %d chars", len(pw))
	default:
		return fmt.Sprintf("weak · %d chars — consider longer / more variety", len(pw))
	}
}

// ── step: JWT ───────────────────────────────────────────────────────

func stepJWT(in *bufio.Reader, cfg *config.Config) stepResult {
	hasExisting := cfg.Auth.JWTSecret != ""
	defaultChoice := "auto"
	if hasExisting {
		defaultChoice = "keep"
	}

	items := []menuItem{
		{
			Key:   "auto",
			Label: "auto",
			Desc:  "Generate a 64-character random secret. Recommended.",
		},
		{
			Key:   "custom",
			Label: "custom",
			Desc:  "Paste your own secret (≥ 32 chars).",
		},
	}
	if hasExisting {
		items = append(items, menuItem{
			Key:   "keep",
			Label: "keep",
			Desc:  "Reuse the existing secret from config.",
		})
	}

	choice, r := pickMenu("How should the JWT signing secret be set?", items, defaultChoice)
	if r != srNext {
		return r
	}

	switch choice {
	case "auto":
		s, err := config.GenerateJWTSecret()
		if err != nil {
			prf("    %s cannot generate JWT secret: %v", failMark(), err)
			return srQuit
		}
		cfg.Auth.JWTSecret = s
		return srNext
	case "custom":
		prn("")
		for {
			s, rr := readSecret(in, "secret")
			if rr != srNext {
				return rr
			}
			if len(s) < 32 {
				prf("    %s too short — %d chars, need ≥ 32", failMark(), len(s))
				continue
			}
			cfg.Auth.JWTSecret = s
			return srNext
		}
	case "keep":
		return srNext
	}
	return srNext
}

// ── summary & apply ─────────────────────────────────────────────────

func writeSummary(cfg config.Config) {
	sumKey := func(s string) string {
		return styleDim(fmt.Sprintf("%-13s", s))
	}
	row := func(key, val string) {
		prf("     %s   %s", sumKey(key), val)
		prn("")
	}
	switch cfg.DB.Mode {
	case "embedded":
		row("database",
			styleBrightCyan(fmt.Sprintf("bundled · 127.0.0.1:%d", cfg.DB.Embedded.Port)))
		row("data dir", styleCyan(cfg.DB.Embedded.DataDir))
	default:
		row("database",
			styleBrightCyan(fmt.Sprintf("external · %s@%s:%d/%s (ssl: %s)",
				cfg.DB.External.User, cfg.DB.External.Host,
				cfg.DB.External.Port, cfg.DB.External.Name, cfg.DB.External.SSLMode)))
	}
	row("listen", styleBrightCyan(cfg.Server.ListenAddr))
	row("admin", styleBrightCyan(cfg.Auth.AdminBootstrapUsername))
	row("jwt", styleCyan(fmt.Sprintf("%d chars", len(cfg.Auth.JWTSecret))))
	paths := config.DefaultPaths()
	if len(paths) > 0 {
		row("config file", styleCyan(paths[0]))
	}
}

// apply writes config, boots DB, runs migrations, plants admin. Returns
// exit code.
func apply(cfg config.Config) int {
	cfg.SetupCompletedAt = time.Now().UTC().Format(time.RFC3339)

	prn("")

	if err := progress("Writing config", func() error { return config.Save(cfg) }); err != nil {
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storeCfg store.Config
	var pg *opg.Embedded

	switch cfg.DB.Mode {
	case "embedded":
		err := progress("Starting bundled PostgreSQL (first run downloads ~50 MB)", func() error {
			p, err := opg.Start(ctx, opg.Config{
				DataDir:  cfg.DB.Embedded.DataDir,
				CacheDir: cfg.DB.Embedded.CacheDir,
				Port:     cfg.DB.Embedded.Port,
				Version:  cfg.DB.Embedded.Version,
				Password: cfg.DB.Embedded.Password,
			})
			if err != nil {
				return err
			}
			pg = p
			return nil
		})
		if err != nil {
			return 1
		}
		defer pg.Stop()

		// The generated password needs to round-trip into config so
		// subsequent boots reuse the same credentials.
		if cfg.DB.Embedded.Password == "" {
			cfg.DB.Embedded.Password = pg.Password()
			_ = config.Save(cfg)
		}

		storeCfg = store.Config{
			Host:     pg.Host(),
			Port:     strconv.Itoa(pg.Port()),
			User:     pg.UserName(),
			Password: pg.Password(),
			DBName:   pg.DBName(),
		}
	default:
		storeCfg = store.Config{
			Host:     cfg.DB.External.Host,
			Port:     strconv.Itoa(cfg.DB.External.Port),
			User:     cfg.DB.External.User,
			Password: cfg.DB.External.Password,
			DBName:   cfg.DB.External.Name,
		}
	}

	var db *store.DB
	err := progress("Running migrations", func() error {
		d, err := store.New(ctx, storeCfg)
		if err != nil {
			return err
		}
		db = d
		return db.Migrate(ctx)
	})
	if err != nil {
		return 1
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	err = progress("Creating admin account", func() error {
		creds := auth.NewCredentialStore(db.Pool)
		return creds.Save(ctx,
			cfg.Auth.AdminBootstrapUsername,
			cfg.Auth.AdminBootstrapPassword)
	})
	if err != nil {
		return 1
	}

	prn("")
	prn("")
	prn(styleGreen(divider))
	prn("")
	prf("     %s   %s", okMark(), styleTitle("SETUP COMPLETE"))
	prn("")
	prn(styleGreen(divider))
	prn("")
	prn("")
	prf("     %s   %s",
		styleDim("Listening at"),
		styleBrightCyan(fmt.Sprintf("http://%s", displayHost(cfg.Server.ListenAddr))))
	prf("     %s   %s",
		styleDim("Admin user  "),
		styleBrightCyan(cfg.Auth.AdminBootstrapUsername))
	prn("")
	prn("")
	prn(styleTitle("   Start the server with:"))
	prn("")
	prf("       %s", styleBrightCyan(binaryInvocation()))
	prn("")
	prn("")
	return 0
}

// displayHost turns "0.0.0.0:8640" into "<this-machine-ip>:8640" for
// the success banner, so the user knows the actual URL they should
// open in a browser.
func displayHost(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "0.0.0.0" || host == "::" {
		if ip, err := primaryLANAddr(); err == nil {
			return fmt.Sprintf("%s:%s (also :%s on all interfaces)", ip, port, port)
		}
		return fmt.Sprintf("<host-ip>:%s  (listening on every interface)", port)
	}
	return addr
}

func primaryLANAddr() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("no non-loopback IPv4 interface")
}

// progress prints "→ label…" then either "✓" or "✗" + error.
func progress(label string, fn func() error) error {
	prf(" %s  %s…", styleAccent("→"), label)
	if err := fn(); err != nil {
		prf(" %s  %v", failMark(), err)
		return err
	}
	prf(" %s", okMark())
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────

type stepResult int

const (
	srNext stepResult = iota
	srBack
	srQuit
)

// readLine prompts with label + default, returns trimmed line and flow
// control (`back` → srBack, Ctrl-D / `quit` → srQuit, else srNext).
func readLine(in *bufio.Reader, label, def string) (string, stepResult) {
	coloredArrow := styleAccent(arrow)
	if label == "" {
		fmt.Fprintf(os.Stderr, "   %s ", coloredArrow)
	} else if def != "" {
		fmt.Fprintf(os.Stderr, "   %s  %s %s ",
			styleTitle(label),
			styleDim(fmt.Sprintf("(default: %s)", def)),
			coloredArrow)
	} else {
		fmt.Fprintf(os.Stderr, "   %s %s ", styleTitle(label), coloredArrow)
	}
	line, err := in.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return "", srQuit
		}
	}
	v := strings.TrimSpace(line)
	switch strings.ToLower(v) {
	case sentinelBack, ":back":
		return "", srBack
	case "quit", ":quit", "exit", ":exit":
		return "", srQuit
	}
	if v == "" {
		return def, srNext
	}
	return v, srNext
}

// readSecret prompts without echoing. Supports `back` by falling back
// to line mode when stdin isn't a TTY (scripted input).
func readSecret(in *bufio.Reader, label string) (string, stepResult) {
	fmt.Fprintf(os.Stderr, "   %s %s ", styleTitle(label), styleAccent(arrow))
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", srQuit
		}
		v := string(b)
		if strings.ToLower(strings.TrimSpace(v)) == sentinelBack {
			return "", srBack
		}
		return v, srNext
	}
	line, err := in.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return "", srQuit
		}
	}
	v := strings.TrimRight(line, "\r\n")
	if strings.ToLower(strings.TrimSpace(v)) == sentinelBack {
		return "", srBack
	}
	return v, srNext
}

// askYN returns true for yes. Hidden prompt so the wizard's look stays
// consistent (defaults shown inline).
func askYN(in *bufio.Reader, label string, defaultYes bool) bool {
	defStr := "y/N"
	if defaultYes {
		defStr = "Y/n"
	}
	fmt.Fprintf(os.Stderr, "   %s  %s %s ",
		styleTitle(label),
		styleDim(fmt.Sprintf("(%s)", defStr)),
		styleAccent(arrow))
	line, _ := in.ReadString('\n')
	v := strings.TrimSpace(strings.ToLower(line))
	if v == "" {
		return defaultYes
	}
	return v == "y" || v == "yes"
}

// guardRootForEmbedded is Postgres's own hard rule: initdb refuses
// uid 0. Fail before we've downloaded the tarball / walked the user
// through more prompts.
func guardRootForEmbedded() error {
	if os.Geteuid() == 0 {
		return errors.New("bundled PostgreSQL cannot run as root (initdb refuses uid 0)")
	}
	return nil
}

// portAvailable returns true if we can bind host:port right now. Best-
// effort — a race is always possible before opg.Start() actually binds.
func portAvailable(host string, port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// testExternalDB opens a connection with a short timeout to verify the
// user's external-DB inputs. Any error short-circuits the wizard —
// "save anyway" was removed because it consistently bit users who
// wound up with a working config file pointing at an unreachable DB.
func testExternalDB(ext config.ExternalDB) error {
	storeCfg := store.Config{
		Host:     ext.Host,
		Port:     strconv.Itoa(ext.Port),
		User:     ext.User,
		Password: ext.Password,
		DBName:   ext.Name,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := store.New(ctx, storeCfg)
	if err != nil {
		return err
	}
	db.Close()
	return nil
}

func binaryInvocation() string {
	exe, err := os.Executable()
	if err != nil {
		return "opendray"
	}
	if abs, err := filepath.Abs(exe); err == nil {
		return abs
	}
	return exe
}

func expandHomeCLI(p string) string {
	if p == "" || !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~/"))
}

// loadSecretFromFlags resolves a secret from either an inline flag or
// a file path. File path wins when both are set. Empty return + nil
// error means "not provided" — caller decides what to do.
func loadSecretFromFlags(inline, path, flagName string) (string, error) {
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("%s file: %w", flagName, err)
		}
		return strings.TrimRight(string(b), "\r\n"), nil
	}
	return inline, nil
}

// ── pickMenu: arrow-key selection UI ────────────────────────────────
//
// Raw-mode terminal read loop that lets users ↑/↓ through a list and
// press Enter to confirm. Much nicer than "type bundled or external"
// for enum choices. Falls back to a text prompt when stdin isn't a
// real TTY (scripted input, redirected files, Docker build logs).
//
// Key map:
//   ↑ / k         move selection up
//   ↓ / j         move selection down
//   Enter         confirm current selection
//   b / B         return srBack to the step machine
//   q / Q / Ctrl-C  return srQuit

type menuItem struct {
	Key   string // returned when item is chosen
	Label string // short name (bundled, external, loopback, …)
	Desc  string // multi-line description, "\n" separated
}

// pickMenu blocks until the user picks an item or quits. defaultKey
// seeds the initial highlight — useful for resume mode where a prior
// config value should round-trip.
func pickMenu(title string, items []menuItem, defaultKey string) (string, stepResult) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		// No TTY — fall back to text mode so scripted / piped input
		// still works (CI test harnesses, Docker build RUN).
		return pickMenuTextFallback(title, items, defaultKey)
	}

	// Find default index.
	idx := 0
	for i, it := range items {
		if it.Key == defaultKey {
			idx = i
			break
		}
	}

	// Render once to establish baseline, remember how many terminal
	// lines we drew so subsequent renders can scroll back up exactly
	// that many and overwrite.
	lines := renderMenu(title, items, idx)

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return pickMenuTextFallback(title, items, defaultKey)
	}
	defer term.Restore(fd, oldState)

	// Buffered reader over raw stdin. Escape sequences for arrows are
	// multi-byte (ESC '[' 'A' = up), so we read byte-by-byte and parse
	// inline.
	in := bufio.NewReader(os.Stdin)
	for {
		b, err := in.ReadByte()
		if err != nil {
			return "", srQuit
		}
		handled := false
		switch b {
		case '\r', '\n':
			// Clear menu, print the choice as a normal line so the
			// transcript makes sense after the interactive portion.
			clearMenu(lines)
			fmt.Fprintf(os.Stderr, "   %s  %s  %s%s\r\n",
				strings.ToLower(title),
				styleDim("›"),
				styleBrightCyan(items[idx].Label),
				"")
			return items[idx].Key, srNext
		case 0x03: // Ctrl-C
			clearMenu(lines)
			return "", srQuit
		case 'b', 'B':
			clearMenu(lines)
			return "", srBack
		case 'q', 'Q':
			clearMenu(lines)
			return "", srQuit
		case 'k':
			idx = (idx - 1 + len(items)) % len(items)
			handled = true
		case 'j':
			idx = (idx + 1) % len(items)
			handled = true
		case 0x1b: // ESC — possibly start of arrow sequence
			// Peek for '['; anything else means a bare ESC (user
			// pressed Escape → quit).
			b2, err := in.ReadByte()
			if err != nil || b2 != '[' {
				clearMenu(lines)
				return "", srQuit
			}
			b3, err := in.ReadByte()
			if err != nil {
				clearMenu(lines)
				return "", srQuit
			}
			switch b3 {
			case 'A':
				idx = (idx - 1 + len(items)) % len(items)
				handled = true
			case 'B':
				idx = (idx + 1) % len(items)
				handled = true
			}
		}
		if handled {
			clearMenu(lines)
			lines = renderMenu(title, items, idx)
		}
	}
}

// renderMenu writes the title + items + hint line to stderr and
// returns the number of terminal lines used so the caller can rewind
// on the next frame. In raw mode CR isn't implicit with LF; we emit
// "\r\n" explicitly so each line reliably starts at column 1.
//
// Layout: vertical sidebar (▌) on the selected item, an uppercased
// label header, a dim rule, and the description indented underneath.
// Non-selected items collapse to a single grey label + one-line
// description so the choice is obvious at a glance.
func renderMenu(title string, items []menuItem, selected int) int {
	lines := 0
	write := func(s string) {
		fmt.Fprint(os.Stderr, s+"\r\n")
		lines++
	}

	write("")
	write("   " + styleTitle(title))
	write("")
	write("")

	for i, it := range items {
		descLines := strings.Split(it.Desc, "\n")
		if i == selected {
			bar := styleAccent("▌")
			labelBlock := styleReverse(styleTitle(fmt.Sprintf("  %s  ", strings.ToUpper(it.Label))))
			write(fmt.Sprintf("    %s  %s", bar, labelBlock))
			write(fmt.Sprintf("    %s", bar))
			for _, ln := range descLines {
				write(fmt.Sprintf("    %s  %s", bar, ln))
			}
		} else {
			write(fmt.Sprintf("       %s", styleDim(strings.ToLower(it.Label))))
			write("")
			for _, ln := range descLines {
				write(fmt.Sprintf("       %s", styleDim(ln)))
			}
		}
		write("")
		write("")
	}

	write(styleDim("   ↑/↓ move   Enter confirm   b back   Ctrl-C quit"))
	write("")
	return lines
}

// clearMenu walks the cursor back up N lines and clears from there to
// end-of-display so the next render starts from the same anchor.
func clearMenu(n int) {
	if !colorsEnabled {
		// Without ANSI support we can't rewind — just print a newline
		// and accept the scrollback history.
		fmt.Fprint(os.Stderr, "\r\n")
		return
	}
	fmt.Fprintf(os.Stderr, "\x1b[%dA\x1b[0J", n)
}

// pickMenuTextFallback is used when stdin isn't a TTY (scripted input).
// Prints options in a short list and reads a typed answer.
func pickMenuTextFallback(title string, items []menuItem, defaultKey string) (string, stepResult) {
	prn("")
	prn(" " + styleTitle(title))
	prn("")
	for _, it := range items {
		lines := strings.Split(it.Desc, "\n")
		prf("   %s   %s", styleBrightCyan(fmt.Sprintf("%-10s", it.Label)), styleDim(lines[0]))
		for _, l := range lines[1:] {
			prf("              %s", styleDim(l))
		}
		prn("")
	}

	in := bufio.NewReader(os.Stdin)
	for {
		v, r := readLine(in, "choice", defaultKey)
		if r != srNext {
			return "", r
		}
		target := strings.ToLower(v)
		for _, it := range items {
			if strings.ToLower(it.Key) == target ||
				strings.ToLower(it.Label) == target {
				return it.Key, srNext
			}
		}
		prf("    %s unknown choice %q — expected one of:", failMark(), v)
		for _, it := range items {
			prf("        %s", it.Label)
		}
	}
}
