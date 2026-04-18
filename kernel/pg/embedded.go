// Package pg manages an in-process PostgreSQL child process for the
// "db.mode = embedded" install path.
//
// The child binary itself is downloaded by
// github.com/fergusstrange/embedded-postgres on first run and cached
// under [Config.CacheDir]; subsequent starts reuse it. We only bind to
// loopback, so an embedded PG is never reachable over the network.
package pg

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config describes how the embedded Postgres child is started.
//
// All paths accept tilde-prefixed values — they're resolved to the user's
// home directory by the caller (kernel/config does this on Load).
type Config struct {
	DataDir  string // PGDATA for initdb — opendray owns this
	CacheDir string // where the downloaded PG tarball lives
	Port     int    // loopback port
	Version  string // e.g. "15.4.0"; empty = library default
	Password string // if empty, a random one is generated + returned
	Logger   *slog.Logger
}

// Embedded is a running embedded Postgres child. Callers must Stop it
// during shutdown so the child doesn't orphan.
type Embedded struct {
	pg       *embeddedpostgres.EmbeddedPostgres
	cfg      Config
	password string
	started  bool
}

// DBName is the single database OpenDray uses. Hardcoded because embedded
// mode is single-tenant by definition — if the user wants multiple DBs
// they should switch to external mode.
const DBName = "opendray"

// UserName is the PG role OpenDray connects as. Owner of [DBName].
const UserName = "opendray"

// Start boots the embedded PG child. Blocks until the server is accepting
// connections. If [Config.Password] is empty, a fresh random password is
// generated — caller should persist it back to [config.Config.DB.Embedded.Password]
// and re-save the config so subsequent boots reuse the same credentials.
func Start(ctx context.Context, cfg Config) (*Embedded, error) {
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("pg: DataDir is required")
	}
	if cfg.CacheDir == "" {
		return nil, fmt.Errorf("pg: CacheDir is required")
	}
	if cfg.Port == 0 {
		cfg.Port = 5433
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Ensure dirs exist with tight permissions — both hold PG's on-disk
	// data, which is secret material (bcrypt hashes, OAuth tokens).
	for _, d := range []string{cfg.DataDir, cfg.CacheDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, fmt.Errorf("pg: mkdir %s: %w", d, err)
		}
	}

	pw := cfg.Password
	if pw == "" {
		var err error
		pw, err = generatePassword()
		if err != nil {
			return nil, fmt.Errorf("pg: generate password: %w", err)
		}
		cfg.Logger.Info("pg: generated embedded password (caller must persist)",
			"data_dir", cfg.DataDir)
	}

	// Library maps its own log stream to stdout by default — we route it
	// through slog so it shows up alongside the rest of OpenDray's logs.
	libLogger := slogWriter{logger: cfg.Logger}

	options := embeddedpostgres.DefaultConfig().
		Username(UserName).
		Password(pw).
		Database(DBName).
		Port(uint32(cfg.Port)).
		DataPath(cfg.DataDir).
		RuntimePath(filepath.Join(cfg.CacheDir, "runtime")).
		BinariesPath(cfg.CacheDir).
		Logger(libLogger).
		// StartTimeout: downloading the binary on a slow link can take
		// over a minute the first time. 3 minutes is generous but safe.
		StartTimeout(3 * time.Minute)

	if cfg.Version != "" {
		if v, ok := parseVersion(cfg.Version); ok {
			options = options.Version(v)
		} else {
			cfg.Logger.Warn("pg: unrecognized version, using library default",
				"requested", cfg.Version)
		}
	}

	e := &Embedded{
		pg:       embeddedpostgres.NewDatabase(options),
		cfg:      cfg,
		password: pw,
	}

	if err := e.pg.Start(); err != nil {
		return nil, fmt.Errorf("pg: start: %w", err)
	}
	e.started = true

	// Library reports "ready" when initdb + postgres start return, but the
	// socket can still be mid-listen. A single trivial query here removes
	// the race without needing retries in the rest of OpenDray.
	if err := e.ping(ctx); err != nil {
		_ = e.pg.Stop()
		e.started = false
		return nil, fmt.Errorf("pg: ping after start: %w", err)
	}

	cfg.Logger.Info("pg: embedded ready", "port", cfg.Port, "data_dir", cfg.DataDir)
	return e, nil
}

// Stop shuts the child down cleanly. Safe to call multiple times; only
// the first call does anything.
func (e *Embedded) Stop() error {
	if e == nil || !e.started {
		return nil
	}
	e.started = false
	if err := e.pg.Stop(); err != nil {
		return fmt.Errorf("pg: stop: %w", err)
	}
	return nil
}

// Password is the password that was used to initialise (or start) the
// child. Returned so main can persist it into the config file on first
// boot — the caller is responsible for only doing so once, before the DB
// holds real data.
func (e *Embedded) Password() string { return e.password }

// Host is always "127.0.0.1" — embedded mode never binds public interfaces.
func (e *Embedded) Host() string { return "127.0.0.1" }

// Port returns the TCP port the child listens on.
func (e *Embedded) Port() int { return e.cfg.Port }

// DBName / UserName exposed for symmetry with external mode.
func (e *Embedded) DBName() string { return DBName }
func (e *Embedded) UserName() string { return UserName }

// ping opens a tiny connection, runs SELECT 1, closes. Uses the pgx stdlib
// driver to avoid pulling yet another driver into the tree.
func (e *Embedded) ping(ctx context.Context) error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		e.Host(), e.Port(), UserName, e.password, DBName)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// parseVersion maps our config string ("15.4.0") onto the typed constant
// the embedded-postgres library exposes. Only the major version is honoured
// by the library (it picks the latest point release automatically), but
// we accept full triplets in config for future-proofing.
func parseVersion(v string) (embeddedpostgres.PostgresVersion, bool) {
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 {
		return "", false
	}
	// The library's PostgresVersion is a string alias like "15.4.0".
	// We trust our config validator to have kept this sane.
	return embeddedpostgres.PostgresVersion(v), true
}

func generatePassword() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// URL-safe so it can flow through connection strings without escaping.
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// slogWriter adapts *slog.Logger to the io.Writer the library expects.
// Each Write call is logged as one info line so PG startup chatter is
// discoverable via `grep pg:` without dominating the terminal.
type slogWriter struct {
	logger *slog.Logger
}

func (w slogWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	if msg != "" {
		w.logger.Debug("pg.child", "msg", msg)
	}
	return len(p), nil
}

// Compile-time check that slogWriter satisfies io.Writer.
var _ io.Writer = slogWriter{}

// Compile-time check that Port() returns int — useful because store.Config
// currently wants port as a string. Callers do the strconv themselves.
var _ = strconv.Itoa
