package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies any embedded SQL migrations not yet recorded in
// schema_migrations. Migrations run in lexical filename order, each in its
// own transaction. Hand-rolled per design §19.4 — no goose/golang-migrate
// dependency until the migration set grows past ~20 files.
func (s *Store) Migrate(ctx context.Context, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	if err := s.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	files, err := loadMigrations()
	if err != nil {
		return err
	}

	applied, err := s.appliedVersions(ctx)
	if err != nil {
		return err
	}

	for _, f := range files {
		if applied[f.version] {
			continue
		}
		log.Info("applying migration", "version", f.version)
		if err := s.applyOne(ctx, f); err != nil {
			return fmt.Errorf("store: apply %s: %w", f.version, err)
		}
	}
	return nil
}

func (s *Store) ensureMigrationsTable(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		return fmt.Errorf("store: ensure schema_migrations: %w", err)
	}
	return nil
}

// PendingMigrations returns the versions of embedded migrations not yet
// applied, in apply order. Used to decide whether a pre-migration
// snapshot is warranted before Migrate mutates the schema.
func (s *Store) PendingMigrations(ctx context.Context) ([]string, error) {
	if err := s.ensureMigrationsTable(ctx); err != nil {
		return nil, err
	}
	files, err := loadMigrations()
	if err != nil {
		return nil, err
	}
	applied, err := s.appliedVersions(ctx)
	if err != nil {
		return nil, err
	}
	var pending []string
	for _, f := range files {
		if !applied[f.version] {
			pending = append(pending, f.version)
		}
	}
	return pending, nil
}

type migrationFile struct {
	version string
	body    string
}

func loadMigrations() ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("store: read migrations dir: %w", err)
	}
	var out []migrationFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		body, err := fs.ReadFile(migrationsFS, "migrations/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("store: read %s: %w", e.Name(), err)
		}
		version := strings.TrimSuffix(e.Name(), ".sql")
		out = append(out, migrationFile{version: version, body: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func (s *Store) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("store: list applied migrations: %w", err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func (s *Store) applyOne(ctx context.Context, f migrationFile) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, f.body); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, f.version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
