package main

import (
	"bufio"
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// runSetupCLI is the headless first-run wizard, invoked by
//
//	opendray setup
//
// It prompts interactively on stdin/stdout for everything the browser
// wizard would ask (DB choice, admin credentials, JWT), writes
// config.toml, spins up the DB to apply migrations + the admin row,
// and exits. The operator then runs `opendray` normally.
//
// Designed for SSH-into-a-Linux-VPS-style installs where no browser is
// available. For local desktop installs the browser wizard is still the
// better UX.
func runSetupCLI() int {
	fmt.Println("")
	fmt.Println("╭─────────────────────────────────────────────╮")
	fmt.Println("│   🚀  OpenDray — interactive setup          │")
	fmt.Println("╰─────────────────────────────────────────────╯")
	fmt.Println("")

	// Single reader shared across every prompt — if each prompt allocated
	// its own bufio.NewReader, lines buffered by the first reader would
	// be invisible to the second (classic shared-stdin bug).
	in := bufio.NewReader(os.Stdin)

	// If a completed config already exists, refuse to overwrite without
	// explicit confirmation — this command is destructive on first run.
	if existing, _, _ := config.Load(); existing.IsComplete() {
		fmt.Println("⚠  A valid config already exists.")
		fmt.Printf("   Paths checked: %v\n", config.DefaultPaths())
		if !promptYesNo(in, "Overwrite it?", false) {
			fmt.Println("Aborted.")
			return 0
		}
	}

	cfg := config.Defaults()

	// ── Step 1: Database ────────────────────────────────────────
	fmt.Println("Step 1/3 — Database")
	fmt.Println("───────────────────")
	fmt.Println("  [E] Embedded  — OpenDray manages a local PostgreSQL (recommended)")
	fmt.Println("  [X] External  — connect to an existing PostgreSQL")
	fmt.Println()

	switch strings.ToLower(promptDefault(in, "Database mode", "E")) {
	case "x", "external":
		cfg.DB.Mode = "external"
		fmt.Println()
		cfg.DB.External.Host = promptRequired(in, "Host (e.g. 10.0.0.5)")
		portStr := promptDefault(in, "Port", "5432")
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.DB.External.Port = p
		} else {
			cfg.DB.External.Port = 5432
		}
		cfg.DB.External.User = promptRequired(in, "User")
		cfg.DB.External.Password = promptSecret(in, "Password")
		cfg.DB.External.Name = promptRequired(in, "Database name")
		cfg.DB.External.SSLMode = promptDefault(in, "SSL mode (disable|require)", "disable")

		// Test the connection before the user invests more typing.
		fmt.Print("\n→ Testing connection… ")
		if err := testExternalDB(cfg.DB.External); err != nil {
			fmt.Printf("✗\n  %v\n", err)
			if !promptYesNo(in, "Save anyway?", false) {
				return 1
			}
		} else {
			fmt.Println("✓")
		}
	default:
		cfg.DB.Mode = "embedded"
		cfg.DB.Embedded.DataDir = expandHomeCLI(
			promptDefault(in, "Data directory", cfg.DB.Embedded.DataDir))
		cfg.DB.Embedded.CacheDir = expandHomeCLI(
			promptDefault(in, "Binary cache directory", cfg.DB.Embedded.CacheDir))
		portStr := promptDefault(in, "Postgres port (loopback-only)", strconv.Itoa(cfg.DB.Embedded.Port))
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.DB.Embedded.Port = p
		}
	}

	// ── Step 2: Admin credentials ───────────────────────────────
	fmt.Println()
	fmt.Println("Step 2/3 — Admin credentials")
	fmt.Println("────────────────────────────")
	cfg.Auth.AdminBootstrapUsername = promptDefault(in, "Admin username", "admin")

	var pass, confirm string
	for {
		pass = promptSecret(in, "Password (min 8 chars)")
		if len(pass) < 8 {
			fmt.Println("  ✗ too short, try again.")
			continue
		}
		confirm = promptSecret(in, "Confirm password")
		if subtle.ConstantTimeCompare([]byte(pass), []byte(confirm)) != 1 {
			fmt.Println("  ✗ passwords don't match, try again.")
			continue
		}
		break
	}
	cfg.Auth.AdminBootstrapPassword = pass

	// ── Step 3: JWT ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("Step 3/3 — JWT secret")
	fmt.Println("─────────────────────")
	fmt.Println("  [A] Auto-generate (recommended)")
	fmt.Println("  [C] Paste a custom value")
	fmt.Println()

	switch strings.ToLower(promptDefault(in, "JWT mode", "A")) {
	case "c", "custom":
		for {
			v := promptSecret(in, "JWT secret (32+ chars)")
			if len(v) >= 32 {
				cfg.Auth.JWTSecret = v
				break
			}
			fmt.Println("  ✗ too short, try again.")
		}
	default:
		s, err := config.GenerateJWTSecret()
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ could not generate JWT secret: %v\n", err)
			return 1
		}
		cfg.Auth.JWTSecret = s
	}

	// ── Summary + confirm ───────────────────────────────────────
	fmt.Println()
	fmt.Println("Summary")
	fmt.Println("───────")
	switch cfg.DB.Mode {
	case "embedded":
		fmt.Printf("  Database        Embedded (port %d, data at %s)\n",
			cfg.DB.Embedded.Port, cfg.DB.Embedded.DataDir)
	default:
		fmt.Printf("  Database        External @ %s:%d/%s\n",
			cfg.DB.External.Host, cfg.DB.External.Port, cfg.DB.External.Name)
	}
	fmt.Printf("  Admin username  %s\n", cfg.Auth.AdminBootstrapUsername)
	fmt.Printf("  JWT secret      %d chars (hidden)\n", len(cfg.Auth.JWTSecret))
	fmt.Println()

	if !promptYesNo(in, "Apply this configuration?", true) {
		fmt.Println("Aborted. Nothing written.")
		return 0
	}

	// ── Apply ───────────────────────────────────────────────────
	cfg.SetupCompletedAt = time.Now().UTC().Format(time.RFC3339)
	fmt.Println()

	step("Writing config.toml", func() error {
		return config.Save(cfg)
	})

	// For embedded mode we start PG here, run migrations, plant the admin
	// row, stop PG. External mode: same, just without the pg.Start detour.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storeCfg store.Config
	var pg *opg.Embedded
	if cfg.DB.Mode == "embedded" {
		fmt.Println("→ Starting embedded PostgreSQL (first run downloads ~50 MB)…")
		p, err := opg.Start(ctx, opg.Config{
			DataDir:  cfg.DB.Embedded.DataDir,
			CacheDir: cfg.DB.Embedded.CacheDir,
			Port:     cfg.DB.Embedded.Port,
			Version:  cfg.DB.Embedded.Version,
			Password: cfg.DB.Embedded.Password,
		})
		if err != nil {
			fmt.Printf("  ✗ %v\n", err)
			return 1
		}
		pg = p
		defer pg.Stop()
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
		fmt.Println("  ✓ PostgreSQL ready")
	} else {
		storeCfg = store.Config{
			Host:     cfg.DB.External.Host,
			Port:     strconv.Itoa(cfg.DB.External.Port),
			User:     cfg.DB.External.User,
			Password: cfg.DB.External.Password,
			DBName:   cfg.DB.External.Name,
		}
	}

	var db *store.DB
	step("Running migrations", func() error {
		d, err := store.New(ctx, storeCfg)
		if err != nil {
			return err
		}
		db = d
		return db.Migrate(ctx)
	})
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	step("Creating admin account", func() error {
		creds := auth.NewCredentialStore(db.Pool)
		return creds.Save(ctx, cfg.Auth.AdminBootstrapUsername, cfg.Auth.AdminBootstrapPassword)
	})

	fmt.Println()
	fmt.Println("✓ Setup complete.")
	fmt.Println()
	fmt.Println("Start the server:")
	fmt.Printf("    %s\n", binaryInvocation())
	if cfg.Server.ListenAddr != "127.0.0.1:8640" {
		fmt.Println()
		fmt.Printf("Default listen address is %s — edit ~/.opendray/config.toml [server] if you want to change it.\n", cfg.Server.ListenAddr)
	}
	fmt.Println()
	return 0
}

// ── Prompt helpers ──────────────────────────────────────────────────

func promptDefault(in *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("  %s [%s]: ", label, def)
	} else {
		fmt.Printf("  %s: ", label)
	}
	line, err := in.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return def
		}
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return def
	}
	return v
}

func promptRequired(in *bufio.Reader, label string) string {
	for {
		v := promptDefault(in, label, "")
		if v != "" {
			return v
		}
		fmt.Println("  ✗ required, try again.")
	}
}

func promptYesNo(in *bufio.Reader, label string, def bool) bool {
	defStr := "y/N"
	if def {
		defStr = "Y/n"
	}
	fmt.Printf("  %s [%s]: ", label, defStr)
	line, _ := in.ReadString('\n')
	v := strings.TrimSpace(strings.ToLower(line))
	if v == "" {
		return def
	}
	return v == "y" || v == "yes"
}

// promptSecret reads a password without echoing it. On a real TTY it
// uses term.ReadPassword; on piped stdin (scripted setup) it falls back
// to the shared bufio reader so it doesn't race with lines already
// buffered from earlier prompts.
func promptSecret(in *bufio.Reader, label string) string {
	fmt.Printf("  %s: ", label)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return ""
		}
		return string(b)
	}
	line, _ := in.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

// step runs fn and prints a status line. Exits on failure.
func step(label string, fn func() error) {
	fmt.Printf("→ %s… ", label)
	if err := fn(); err != nil {
		fmt.Printf("✗\n  %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓")
}

// ── External-DB quick ping ──────────────────────────────────────────

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

// binaryInvocation returns the path the user should run to start
// OpenDray after setup completes. Falls back to the basename if we
// can't resolve the absolute path.
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

// expandHomeCLI mirrors the (unexported) helper in kernel/config. We need
// our own copy because the CLI code runs before config.Load().
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

// Compile-time sanity — silence unused imports if later edits trim them.
var _ = exec.Command
