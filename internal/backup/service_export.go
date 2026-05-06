package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/opendray/opendray-v2/internal/version"
)

// ExportRequest controls a single export bundle invocation.
type ExportRequest struct {
	RequestedBy  string // admin username (for audit)
	Memories     bool
	Integrations IntegrationExportMode // 'none' | 'metadata' | 'plaintext'
	CustomTasks  bool
	// TTL overrides the default 24h expiry. Zero means "use default".
	TTL time.Duration
}

const defaultExportTTL = 24 * time.Hour

// CreateExport synchronously builds a zip bundle of the requested
// scope and stores it under cfg.ExportDir. Returns the export row
// (with download_token populated). Token is the only credential
// required to download — keep it private.
//
// Synchronous because v1 export bodies are small (kilo to a few
// mega bytes); the user is waiting on the HTTP call and we'd
// rather block 1-2s than juggle async jobs.
func (s *Service) CreateExport(ctx context.Context, req ExportRequest) (Export, error) {
	if req.Integrations == "" {
		req.Integrations = IntegrationExportNone
	}
	if req.RequestedBy == "" {
		req.RequestedBy = "admin"
	}
	if !req.Memories && req.Integrations == IntegrationExportNone && !req.CustomTasks {
		return Export{}, fmt.Errorf("export: at least one scope must be selected")
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = defaultExportTTL
	}

	if err := os.MkdirAll(s.cfg.ExportDir, 0o700); err != nil {
		return Export{}, fmt.Errorf("export: mkdir export_dir: %w", err)
	}

	now := time.Now().UTC()
	e := Export{
		ID:            NewExportID(),
		Status:        ExportRunning,
		RequestedBy:   req.RequestedBy,
		Scope:         ExportScope{Memories: req.Memories, Integrations: req.Integrations, CustomTasks: req.CustomTasks},
		StartedAt:     now,
		ExpiresAt:     now.Add(ttl),
		DownloadToken: NewDownloadToken(),
	}
	if err := s.store.InsertExport(ctx, e); err != nil {
		return Export{}, err
	}

	filePath := filepath.Join(s.cfg.ExportDir, e.ID+".zip")
	bytesWritten, sum, err := s.buildExportZip(ctx, e, filePath, req)
	if err != nil {
		_ = os.Remove(filePath)
		_ = s.store.MarkExportFailed(ctx, e.ID, err.Error())
		return e, fmt.Errorf("export build: %w", err)
	}
	if err := s.store.MarkExportReady(ctx, e.ID, ExportResult{
		FilePath: filePath, Bytes: bytesWritten, SHA256: sum,
	}); err != nil {
		return e, err
	}
	// refetch so caller sees finalised row
	return s.store.GetExport(ctx, e.ID)
}

// buildExportZip is the actual zip-writing pipeline. Returns
// (bytesWritten, sha256_hex, error).
func (s *Service) buildExportZip(ctx context.Context, e Export, dest string, req ExportRequest) (int64, string, error) {
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, "", fmt.Errorf("open out: %w", err)
	}
	defer out.Close()

	hasher := sha256.New()
	mw := io.MultiWriter(out, hasher)
	zw := zip.NewWriter(mw)

	manifest := ExportManifest{
		Version:         BundleVersion,
		ExportID:        e.ID,
		CreatedAt:       e.StartedAt,
		OpendrayVersion: version.Current().Version,
		GitSHA:          version.Current().Commit,
		Scope:           e.Scope,
		Encryption: ManifestEncryption{
			Algo:        "aes-256-gcm-fields",
			Fingerprint: s.cipher.Fingerprint(),
		},
		Notes: []string{},
	}

	if req.Memories {
		n, err := s.exportMemories(ctx, zw)
		if err != nil {
			return 0, "", fmt.Errorf("memories: %w", err)
		}
		manifest.Counts.Memories = n
	}
	if req.Integrations != IntegrationExportNone {
		n, recoverable, err := s.exportIntegrations(ctx, zw, req.Integrations)
		if err != nil {
			return 0, "", fmt.Errorf("integrations: %w", err)
		}
		manifest.Counts.Integrations = n
		if req.Integrations == IntegrationExportPlaintext && !recoverable {
			manifest.Notes = append(manifest.Notes,
				"Integrations: plaintext mode requested, but no recoverable plaintext keys exist (all are bcrypt hashes). Bundle contains metadata only.")
		}
	}
	if req.CustomTasks {
		n, err := s.exportCustomTasks(ctx, zw)
		if err != nil {
			return 0, "", fmt.Errorf("custom_tasks: %w", err)
		}
		manifest.Counts.CustomTasks = n
	}

	mfBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return 0, "", fmt.Errorf("marshal manifest: %w", err)
	}
	if err := writeZipFile(zw, "manifest.json", mfBytes); err != nil {
		return 0, "", err
	}

	if err := zw.Close(); err != nil {
		return 0, "", fmt.Errorf("close zip: %w", err)
	}
	stat, err := out.Stat()
	if err != nil {
		return 0, "", fmt.Errorf("stat out: %w", err)
	}
	return stat.Size(), hex.EncodeToString(hasher.Sum(nil)), nil
}

// ExportManifest is serialised as manifest.json inside every export
// zip. Restore tools / future import flows read this first to
// know what to expect.
type ExportManifest struct {
	Version         string             `json:"version"`
	ExportID        string             `json:"export_id"`
	CreatedAt       time.Time          `json:"created_at"`
	OpendrayVersion string             `json:"opendray_version,omitempty"`
	GitSHA          string             `json:"git_sha,omitempty"`
	Scope           ExportScope        `json:"scope"`
	Counts          ExportCounts       `json:"counts"`
	Encryption      ManifestEncryption `json:"encryption"`
	Notes           []string           `json:"notes,omitempty"`
}

type ExportCounts struct {
	Memories     int `json:"memories"`
	Integrations int `json:"integrations"`
	CustomTasks  int `json:"custom_tasks"`
}

// exportMemories streams the memories table into memories.jsonl
// inside zw. The vector column is omitted (unrelated to user
// content + bulky); embedder + dim live on each row's metadata so
// re-embedding on import is possible.
func (s *Service) exportMemories(ctx context.Context, zw *zip.Writer) (int, error) {
	w, err := zw.Create("memories.jsonl")
	if err != nil {
		return 0, fmt.Errorf("create memories.jsonl: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, scope, scope_key, text, embedder,
		       COALESCE(metadata, '{}'::jsonb),
		       created_at, updated_at,
		       COALESCE(hit_count, 0),
		       last_hit_at
		  FROM memories
		 ORDER BY created_at ASC`)
	if err != nil {
		return 0, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	enc := json.NewEncoder(w)
	count := 0
	for rows.Next() {
		var (
			id, scope, scopeKey, text, embedder string
			meta                                []byte
			createdAt, updatedAt                time.Time
			hitCount                            int64
			lastHitAt                           pgxNullableTime
		)
		if err := rows.Scan(&id, &scope, &scopeKey, &text, &embedder,
			&meta, &createdAt, &updatedAt, &hitCount, &lastHitAt); err != nil {
			return count, err
		}
		row := map[string]any{
			"id":         id,
			"scope":      scope,
			"scope_key":  scopeKey,
			"text":       text,
			"embedder":   embedder,
			"created_at": createdAt,
			"updated_at": updatedAt,
			"hit_count":  hitCount,
		}
		if lastHitAt.Valid {
			row["last_hit_at"] = lastHitAt.Time
		}
		var metaObj any
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &metaObj)
			row["metadata"] = metaObj
		}
		if err := enc.Encode(row); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

// exportIntegrations dumps integrations sans api_key_hash. The
// "plaintext" mode is reserved for future plaintext-cache support;
// for v1 we have no recoverable plaintext keys (bcrypt-only) so
// the second return value tells the caller to add a manifest note.
func (s *Service) exportIntegrations(ctx context.Context, zw *zip.Writer, mode IntegrationExportMode) (int, bool, error) {
	w, err := zw.Create("integrations.jsonl")
	if err != nil {
		return 0, false, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, base_url, route_prefix,
		       COALESCE(scopes, '[]'::jsonb),
		       COALESCE(version, ''),
		       enabled, created_at, rotated_at,
		       COALESCE(is_system, FALSE)
		  FROM integrations
		 ORDER BY created_at ASC`)
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()

	enc := json.NewEncoder(w)
	count := 0
	for rows.Next() {
		var (
			id, name, baseURL, routePrefix, ver string
			scopes                              []byte
			enabled, isSystem                   bool
			createdAt                           time.Time
			rotatedAt                           pgxNullableTime
		)
		if err := rows.Scan(&id, &name, &baseURL, &routePrefix, &scopes,
			&ver, &enabled, &createdAt, &rotatedAt, &isSystem); err != nil {
			return count, false, err
		}
		var scopesArr any
		_ = json.Unmarshal(scopes, &scopesArr)
		row := map[string]any{
			"id":           id,
			"name":         name,
			"base_url":     baseURL,
			"route_prefix": routePrefix,
			"scopes":       scopesArr,
			"version":      ver,
			"enabled":      enabled,
			"is_system":    isSystem,
			"created_at":   createdAt,
		}
		if rotatedAt.Valid {
			row["rotated_at"] = rotatedAt.Time
		}
		// We never export api_key_hash. For plaintext mode there's
		// nothing to output (caller will see the "not recoverable"
		// note in the manifest).
		if mode == IntegrationExportPlaintext {
			row["api_key_plaintext"] = nil
			row["api_key_recoverable"] = false
		}
		if err := enc.Encode(row); err != nil {
			return count, false, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return count, false, err
	}
	// Always false for v1 — see comment on exportIntegrations.
	return count, false, nil
}

func (s *Service) exportCustomTasks(ctx context.Context, zw *zip.Writer) (int, error) {
	w, err := zw.Create("custom_tasks.jsonl")
	if err != nil {
		return 0, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, command, description, cwd, created_at, updated_at
		  FROM custom_tasks
		 ORDER BY created_at ASC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	enc := json.NewEncoder(w)
	count := 0
	for rows.Next() {
		var (
			id, name, cmd, desc, cwd string
			createdAt, updatedAt     time.Time
		)
		if err := rows.Scan(&id, &name, &cmd, &desc, &cwd, &createdAt, &updatedAt); err != nil {
			return count, err
		}
		if err := enc.Encode(map[string]any{
			"id":          id,
			"name":        name,
			"command":     cmd,
			"description": desc,
			"cwd":         cwd,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

func writeZipFile(zw *zip.Writer, name string, body []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("zip create %s: %w", name, err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("zip write %s: %w", name, err)
	}
	return nil
}

// pgxNullableTime is a tiny wrapper for nullable timestamp scans
// without dragging database/sql into every function.
type pgxNullableTime struct {
	Time  time.Time
	Valid bool
}

func (n *pgxNullableTime) Scan(src any) error {
	if src == nil {
		n.Valid = false
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		n.Time = v
		n.Valid = true
		return nil
	}
	return fmt.Errorf("pgxNullableTime: unsupported type %T", src)
}

// ─── reads / lifecycle ────────────────────────────────────────────

func (s *Service) GetExport(ctx context.Context, id string) (Export, error) {
	return s.store.GetExport(ctx, id)
}

func (s *Service) ListExports(ctx context.Context) ([]Export, error) {
	return s.store.ListExports(ctx)
}

// DownloadExport opens the zip bundle iff the supplied download
// token matches and the export hasn't expired. Returns the file's
// reader; caller is responsible for closing.
func (s *Service) DownloadExport(ctx context.Context, id, token string) (io.ReadCloser, Export, error) {
	e, err := s.store.GetExportByToken(ctx, id, token)
	if err != nil {
		return nil, Export{}, err
	}
	if e.Status == ExportExpired || time.Now().UTC().After(e.ExpiresAt) {
		return nil, e, ErrExportExpired
	}
	if e.Status != ExportReady {
		return nil, e, fmt.Errorf("export %s status=%s; not ready", id, e.Status)
	}
	path, err := s.store.GetExportFilePath(ctx, id)
	if err != nil {
		return nil, e, err
	}
	if path == "" {
		return nil, e, fmt.Errorf("export %s has no file_path", id)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, e, fmt.Errorf("open export file: %w", err)
	}
	return f, e, nil
}

// DeleteExport removes the zip + flips the row to expired.
func (s *Service) DeleteExport(ctx context.Context, id string) error {
	path, err := s.store.GetExportFilePath(ctx, id)
	if err != nil && !errors.Is(err, ErrExportNotFound) {
		return err
	}
	if path != "" {
		_ = os.Remove(path)
	}
	if err := s.store.DeleteExport(ctx, id); err != nil {
		return err
	}
	return nil
}

// ReapExpiredExports deletes the on-disk zip for any export whose
// expires_at is past, and flips its status to 'expired' (kept for
// audit; row is not removed). Called periodically by the
// scheduler.
func (s *Service) ReapExpiredExports(ctx context.Context) error {
	rows, err := s.store.ListExpiredExports(ctx)
	if err != nil {
		return err
	}
	for _, e := range rows {
		path, _ := s.store.GetExportFilePath(ctx, e.ID)
		if path != "" {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				s.log.Warn("export reap: remove failed", "id", e.ID, "path", path, "err", err)
			}
		}
		if err := s.store.MarkExportExpired(ctx, e.ID); err != nil {
			s.log.Warn("export reap: mark expired failed", "id", e.ID, "err", err)
		}
	}
	return nil
}

// silence unused import if pgx becomes unreferenced later.
var _ = pgx.ErrNoRows
