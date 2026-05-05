package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/version"
)

// Config bundles the non-secret runtime knobs read from config.toml
// + env. Passphrase + DSN + ConfigPath travel through ServiceDeps so
// this struct stays loggable.
type Config struct {
	Enabled       bool
	LocalDir      string
	ExportDir     string
	PgDumpPath    string
	PgRestorePath string
}

// ServiceDeps groups the secrets + pool + logger required to build
// a Service. Distinct from Config so Config can be JSON-logged.
type ServiceDeps struct {
	Pool       *pgxpool.Pool
	Passphrase string // from OPENDRAY_BACKUP_KEY
	DSN        string // libpq conninfo for pg_dump
	ConfigPath string // optional; cfg.toml path to include in bundles
	Log        *slog.Logger
}

// Service is the entry point for both A (runtime backups) and C
// (export bundles). Methods are split across service.go and
// service_export.go but share this struct.
type Service struct {
	cfg        Config
	pool       *pgxpool.Pool
	store      *store
	cipher     Cipher
	pgdump     *PgDump
	pgrestore  *PgRestore // optional; nil if pg_restore not on PATH
	targets    *targetRegistry
	dsn        string
	configPath string
	log        *slog.Logger
}

// NewService validates config + deps and constructs the Service.
// Call Bootstrap before serving requests so DB-backed targets are
// loaded into the registry.
func NewService(cfg Config, deps ServiceDeps) (*Service, error) {
	if !cfg.Enabled {
		return nil, ErrFeatureDisabled
	}
	if deps.Pool == nil {
		return nil, fmt.Errorf("backup: pool is nil")
	}
	if deps.Passphrase == "" {
		return nil, ErrCipherUnconfigured
	}
	if deps.DSN == "" {
		return nil, fmt.Errorf("backup: dsn is empty")
	}
	if cfg.LocalDir == "" {
		return nil, fmt.Errorf("backup: cfg.local_dir is empty")
	}

	cipher, err := NewCipher(deps.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("backup: %w", err)
	}
	pgdump, err := NewPgDump(cfg.PgDumpPath)
	if err != nil {
		return nil, fmt.Errorf("backup: %w", err)
	}
	// pg_restore is best-effort: a missing binary disables /backups/restore
	// but doesn't kill startup — operators may run with backup-only
	// (no in-place restore) deployments.
	pgrestore, prerr := NewPgRestore(cfg.PgRestorePath)
	if prerr != nil {
		pgrestore = nil
	}

	log := deps.Log
	if log == nil {
		log = slog.Default()
	}

	if err := os.MkdirAll(cfg.LocalDir, 0o700); err != nil {
		return nil, fmt.Errorf("backup: mkdir local_dir: %w", err)
	}

	return &Service{
		cfg:        cfg,
		pool:       deps.Pool,
		store:      newStore(deps.Pool),
		cipher:     cipher,
		pgdump:     pgdump,
		pgrestore:  pgrestore,
		targets:    newTargetRegistry(),
		dsn:        deps.DSN,
		configPath: deps.ConfigPath,
		log:        log,
	}, nil
}

// Bootstrap reads target rows from DB, instantiates BackupTarget
// instances, and seeds a default local target when the table is
// empty. Idempotent — safe to call once at app boot.
func (s *Service) Bootstrap(ctx context.Context) error {
	if err := s.ensureDefaultLocalTarget(ctx); err != nil {
		return fmt.Errorf("backup bootstrap: ensure default: %w", err)
	}
	if err := s.loadTargets(ctx); err != nil {
		return fmt.Errorf("backup bootstrap: load targets: %w", err)
	}
	// Stale-running cleanup: any rows left 'running' from a prior
	// crash get a clear error and stop blocking retention.
	if n, err := s.store.ResetStaleRunning(ctx, 1*time.Hour); err != nil {
		s.log.Warn("backup: reset stale running failed", "err", err)
	} else if n > 0 {
		s.log.Info("backup: reset stale running rows", "count", n)
	}
	return nil
}

func (s *Service) ensureDefaultLocalTarget(ctx context.Context) error {
	targets, err := s.store.ListTargets(ctx)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if t.Kind == TargetLocal {
			return nil
		}
	}
	now := time.Now().UTC()
	return s.store.InsertTarget(ctx, TargetSpec{
		ID:        "local",
		Kind:      TargetLocal,
		Config:    map[string]any{"root": s.cfg.LocalDir},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *Service) loadTargets(ctx context.Context) error {
	specs, err := s.store.ListTargets(ctx)
	if err != nil {
		return err
	}
	for _, spec := range specs {
		if !spec.Enabled {
			continue
		}
		impl, err := s.instantiateTarget(spec)
		if err != nil {
			s.log.Warn("backup: skipping target (cannot instantiate)",
				"id", spec.ID, "kind", spec.Kind, "err", err)
			continue
		}
		s.targets.set(impl)
	}
	return nil
}

func (s *Service) instantiateTarget(spec TargetSpec) (BackupTarget, error) {
	switch spec.Kind {
	case TargetLocal:
		root, _ := spec.Config["root"].(string)
		if root == "" {
			root = s.cfg.LocalDir
		}
		return NewLocalTarget(spec.ID, root)
	case TargetSMB:
		cfg, err := s.decodeSMBConfig(spec.Config)
		if err != nil {
			return nil, fmt.Errorf("decode smb config %q: %w", spec.ID, err)
		}
		return NewSMBTarget(spec.ID, cfg)
	case TargetS3:
		return nil, fmt.Errorf("%w: kind=s3 reserved for v1.1+", ErrTargetUnsupported)
	default:
		return nil, fmt.Errorf("%w: unknown kind=%q", ErrTargetUnsupported, spec.Kind)
	}
}

// decodeSMBConfig pulls SMB connection params out of the JSONB
// config map, decrypting the password envelope.
func (s *Service) decodeSMBConfig(raw map[string]any) (SMBConfig, error) {
	cfg := SMBConfig{
		Host:       cfgString(raw, "host"),
		Share:      cfgString(raw, "share"),
		User:       cfgString(raw, "user"),
		PathPrefix: cfgString(raw, "path_prefix"),
	}
	cfg.Port = cfgInt(raw, "port")
	if env := cfgString(raw, "password"); env != "" {
		plain, err := s.cipher.DecryptField(env)
		if err != nil {
			return cfg, fmt.Errorf("decrypt password: %w", err)
		}
		cfg.Password = plain
	}
	return cfg, nil
}

func cfgString(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

func cfgInt(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// ─── target CRUD ──────────────────────────────────────────────────

// CreateTargetRequest is the body for CreateTarget. Sensitive
// values (e.g. SMB password) MUST arrive in plaintext — the
// service encrypts them before persisting.
type CreateTargetRequest struct {
	ID      string         // optional; auto-generated if empty
	Kind    TargetKind
	Config  map[string]any // raw, plaintext
	Enabled bool
}

// CreateTarget validates, instantiates, and persists a new target.
// On success the new instance is added to the live registry so
// subsequent backups can immediately use it.
func (s *Service) CreateTarget(ctx context.Context, req CreateTargetRequest) (TargetSpec, error) {
	if req.Kind == "" {
		return TargetSpec{}, fmt.Errorf("kind is required")
	}
	if req.Kind != TargetLocal && req.Kind != TargetSMB {
		return TargetSpec{}, fmt.Errorf("%w: kind=%q (only local + smb supported in v1)",
			ErrTargetUnsupported, req.Kind)
	}
	id := req.ID
	if id == "" {
		id = NewTargetID()
	}

	persisted, err := s.encodeTargetConfig(req.Kind, req.Config)
	if err != nil {
		return TargetSpec{}, err
	}

	now := time.Now().UTC()
	spec := TargetSpec{
		ID:        id,
		Kind:      req.Kind,
		Config:    persisted,
		Enabled:   req.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate by instantiating before we persist — refuses bad
	// configs (e.g. nonexistent local root, missing SMB host) at
	// create time rather than first-backup time.
	impl, err := s.instantiateTarget(spec)
	if err != nil {
		return TargetSpec{}, err
	}

	if err := s.store.InsertTarget(ctx, spec); err != nil {
		return TargetSpec{}, err
	}
	if spec.Enabled {
		s.targets.set(impl)
	}
	return s.redactTargetSpec(spec), nil
}

// UpdateTargetRequest carries optional updates.
type UpdateTargetRequest struct {
	Config  map[string]any // partial; merged with existing; sensitive keys re-encrypted
	Enabled *bool
}

func (s *Service) UpdateTarget(ctx context.Context, id string, req UpdateTargetRequest) (TargetSpec, error) {
	cur, err := s.store.GetTarget(ctx, id)
	if err != nil {
		return TargetSpec{}, err
	}

	patch := TargetPatch{}
	if req.Config != nil {
		merged := map[string]any{}
		for k, v := range cur.Config {
			merged[k] = v
		}
		for k, v := range req.Config {
			merged[k] = v
		}
		// Re-encode (re-encrypts any plaintext sensitive fields the
		// caller just supplied; existing encrypted fields stay as-is).
		persisted, err := s.encodeTargetConfig(cur.Kind, merged)
		if err != nil {
			return TargetSpec{}, err
		}
		patch.Config = persisted
	}
	if req.Enabled != nil {
		patch.Enabled = req.Enabled
	}
	if err := s.store.UpdateTarget(ctx, id, patch); err != nil {
		return TargetSpec{}, err
	}

	// Re-instantiate so the registry reflects the new config.
	updated, err := s.store.GetTarget(ctx, id)
	if err != nil {
		return TargetSpec{}, err
	}
	if updated.Enabled {
		impl, err := s.instantiateTarget(updated)
		if err != nil {
			s.log.Warn("update-target: re-instantiate failed; keeping old impl",
				"id", id, "err", err)
		} else {
			s.targets.set(impl)
		}
	}
	return s.redactTargetSpec(updated), nil
}

func (s *Service) DeleteTarget(ctx context.Context, id string) error {
	// We rely on backup_schedules.target_id ON DELETE RESTRICT to
	// stop deletion when a schedule still references the target.
	// pgx surfaces this as an error string the caller can show.
	if err := s.store.DeleteTarget(ctx, id); err != nil {
		return err
	}
	s.targets.remove(id)
	return nil
}

// TestTarget runs HealthCheck against the target with a tight
// timeout so the UI's "Test connection" button can give immediate
// feedback. Does not require the target to be in the live registry
// — instantiates a one-off from current DB state.
func (s *Service) TestTarget(ctx context.Context, id string) error {
	spec, err := s.store.GetTarget(ctx, id)
	if err != nil {
		return err
	}
	impl, err := s.instantiateTarget(spec)
	if err != nil {
		return err
	}
	tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return impl.HealthCheck(tctx)
}

// ListTargets returns targets with sensitive fields redacted.
func (s *Service) ListTargets(ctx context.Context) ([]TargetSpec, error) {
	list, err := s.store.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i] = s.redactTargetSpec(list[i])
	}
	return list, nil
}

// encodeTargetConfig validates kind-specific shape + encrypts
// sensitive fields. Returns a fresh map, never mutates the input.
func (s *Service) encodeTargetConfig(kind TargetKind, raw map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for k, v := range raw {
		out[k] = v
	}
	switch kind {
	case TargetLocal:
		if root, _ := out["root"].(string); root != "" {
			out["root"] = root // pass-through
		}
	case TargetSMB:
		if cfgString(out, "host") == "" {
			return nil, fmt.Errorf("smb config: host required")
		}
		if cfgString(out, "share") == "" {
			return nil, fmt.Errorf("smb config: share required")
		}
		if cfgString(out, "user") == "" {
			return nil, fmt.Errorf("smb config: user required")
		}
		if pw, ok := out["password"].(string); ok && pw != "" {
			// If it already looks like our envelope, don't re-wrap.
			if !strings.HasPrefix(pw, fieldEnvelopePrefix) {
				env, err := s.cipher.EncryptField(pw)
				if err != nil {
					return nil, fmt.Errorf("encrypt password: %w", err)
				}
				out["password"] = env
			}
		}
	}
	return out, nil
}

// redactTargetSpec returns a copy with sensitive config fields
// replaced by a placeholder so listing endpoints never echo
// ciphertext (or worse, plaintext) back to the UI.
func (s *Service) redactTargetSpec(spec TargetSpec) TargetSpec {
	cfg := map[string]any{}
	for k, v := range spec.Config {
		cfg[k] = v
	}
	if _, ok := cfg["password"]; ok {
		cfg["password"] = "********"
	}
	spec.Config = cfg
	return spec
}

// ─── schedule CRUD ────────────────────────────────────────────────

// CreateScheduleRequest is the body for CreateSchedule.
type CreateScheduleRequest struct {
	TargetID    string
	IntervalSec int
	Retention   int
	Enabled     bool
	// FirstRunAt sets the inaugural next_run_at. Zero value means
	// "right now" — the next scheduler tick will pick it up.
	FirstRunAt time.Time
}

func (s *Service) CreateSchedule(ctx context.Context, req CreateScheduleRequest) (Schedule, error) {
	if req.TargetID == "" {
		return Schedule{}, fmt.Errorf("target_id required")
	}
	if s.targets.get(req.TargetID) == nil {
		return Schedule{}, ErrTargetNotFound
	}
	if req.IntervalSec <= 0 {
		return Schedule{}, fmt.Errorf("interval_sec must be > 0")
	}
	if req.Retention < 0 {
		return Schedule{}, fmt.Errorf("retention must be >= 0")
	}
	now := time.Now().UTC()
	if req.FirstRunAt.IsZero() {
		req.FirstRunAt = now
	}
	sc := Schedule{
		ID:          NewScheduleID(),
		TargetID:    req.TargetID,
		IntervalSec: req.IntervalSec,
		Retention:   req.Retention,
		Enabled:     req.Enabled,
		NextRunAt:   req.FirstRunAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.InsertSchedule(ctx, sc); err != nil {
		return Schedule{}, err
	}
	return sc, nil
}

func (s *Service) GetSchedule(ctx context.Context, id string) (Schedule, error) {
	return s.store.GetSchedule(ctx, id)
}

func (s *Service) ListSchedules(ctx context.Context) ([]Schedule, error) {
	return s.store.ListSchedules(ctx)
}

func (s *Service) UpdateSchedule(ctx context.Context, id string, p SchedulePatch) error {
	return s.store.UpdateSchedule(ctx, id, p)
}

func (s *Service) DeleteSchedule(ctx context.Context, id string) error {
	return s.store.DeleteSchedule(ctx, id)
}

// ─── backup runtime (A) ───────────────────────────────────────────

// RunBackupRequest controls a single backup invocation.
type RunBackupRequest struct {
	TargetID      string
	TriggeredBy   TriggeredBy
	IncludeConfig bool
	// ScheduleID, when non-empty, is written to the backup row so
	// the UI can group scheduler-driven runs under their parent
	// schedule. Set by the Scheduler; left empty for manual / API
	// triggers.
	ScheduleID string
}

// RunBackupNow inserts a backup row, kicks off the backup pipeline
// in a goroutine, and returns the freshly-inserted row immediately.
// The HTTP layer responds 202 + ID; clients poll GetBackup for status.
func (s *Service) RunBackupNow(ctx context.Context, req RunBackupRequest) (Backup, error) {
	target := s.targets.get(req.TargetID)
	if target == nil {
		return Backup{}, ErrTargetNotFound
	}
	backup := s.newBackupRow(req)
	if err := s.store.InsertBackup(ctx, backup); err != nil {
		return Backup{}, err
	}
	go s.doRunBackup(context.Background(), backup, target, req.IncludeConfig)
	return backup, nil
}

// RunBackupSync is the synchronous variant used by the scheduler:
// it returns only after the pipeline finishes (success or failure)
// so retention + last_run_at can be updated atomically afterwards.
func (s *Service) RunBackupSync(ctx context.Context, req RunBackupRequest) (Backup, error) {
	target := s.targets.get(req.TargetID)
	if target == nil {
		return Backup{}, ErrTargetNotFound
	}
	backup := s.newBackupRow(req)
	if err := s.store.InsertBackup(ctx, backup); err != nil {
		return Backup{}, err
	}
	s.doRunBackup(ctx, backup, target, req.IncludeConfig)
	return s.store.GetBackup(ctx, backup.ID)
}

func (s *Service) newBackupRow(req RunBackupRequest) Backup {
	if req.TriggeredBy == "" {
		req.TriggeredBy = TriggeredManual
	}
	b := Backup{
		ID:          NewBackupID(),
		TargetID:    req.TargetID,
		Status:      BackupPending,
		TriggeredBy: req.TriggeredBy,
		StartedAt:   time.Now().UTC(),
		Encrypted:   true,
		Metadata:    map[string]any{"include_config": req.IncludeConfig},
	}
	if req.ScheduleID != "" {
		id := req.ScheduleID
		b.ScheduleID = &id
	}
	return b
}

func (s *Service) doRunBackup(ctx context.Context, b Backup, target BackupTarget, includeConfig bool) {
	log := s.log.With("backup_id", b.ID, "target", b.TargetID)
	if err := s.store.MarkBackupRunning(ctx, b.ID); err != nil {
		log.Error("mark running", "err", err)
		return
	}
	if err := s.runPipeline(ctx, b, target, includeConfig); err != nil {
		log.Error("backup pipeline failed", "err", err)
		if mErr := s.store.MarkBackupFailed(ctx, b.ID, err.Error()); mErr != nil {
			log.Error("mark failed", "err", mErr)
		}
		return
	}
	log.Info("backup succeeded")
}

// runPipeline is the actual end-to-end work:
//
//	pg_dump → temp file → tar(manifest, [config], dump) → cipher.Seal → target.Put
//
// pg_dump is buffered to a sibling temp file because tar headers
// require knowing entry sizes up front. The temp file lives in
// cfg.LocalDir (which we already enforce as opendray-writable).
func (s *Service) runPipeline(ctx context.Context, b Backup, target BackupTarget, includeConfig bool) error {
	pgVerStr, err := s.pgdump.Version(ctx)
	if err != nil {
		return fmt.Errorf("pg_dump --version: %w", err)
	}

	tmpDump, err := os.CreateTemp(s.cfg.LocalDir, ".dump-*.bin")
	if err != nil {
		return fmt.Errorf("create tmp dump: %w", err)
	}
	tmpName := tmpDump.Name()
	defer os.Remove(tmpName)

	res, err := s.pgdump.Dump(ctx, s.dsn)
	if err != nil {
		_ = tmpDump.Close()
		return fmt.Errorf("pg_dump start: %w", err)
	}
	if _, copyErr := io.Copy(tmpDump, res.Reader); copyErr != nil {
		_ = tmpDump.Close()
		return fmt.Errorf("pg_dump copy: %w", copyErr)
	}
	if waitErr := res.Wait(); waitErr != nil {
		_ = tmpDump.Close()
		return waitErr
	}
	if err := tmpDump.Close(); err != nil {
		return fmt.Errorf("close tmp dump: %w", err)
	}

	dumpStat, err := os.Stat(tmpName)
	if err != nil {
		return fmt.Errorf("stat dump: %w", err)
	}
	dumpFile, err := os.Open(tmpName)
	if err != nil {
		return fmt.Errorf("reopen dump: %w", err)
	}
	defer dumpFile.Close()

	sources := []BundleSource{}
	var cfgFile *os.File
	if includeConfig && s.configPath != "" {
		cf, err := os.Open(s.configPath)
		if err != nil {
			s.log.Warn("backup: skip config (open failed)",
				"path", s.configPath, "err", err)
		} else {
			cfgFile = cf
			defer cfgFile.Close()
			cfgStat, _ := cfgFile.Stat()
			sources = append(sources, BundleSource{
				Name: filepath.Base(s.configPath),
				Body: cfgFile,
				Size: cfgStat.Size(),
			})
		}
	}
	sources = append(sources, BundleSource{
		Name: "dump.bin",
		Body: dumpFile,
		Size: dumpStat.Size(),
	})

	info := version.Current()
	manifest := BundleManifest{
		BackupID:        b.ID,
		CreatedAt:       b.StartedAt,
		OpendrayVersion: info.Version,
		GitSHA:          info.Commit,
		PGVersion:       ParsePGMajorMinor(pgVerStr),
		Encryption: ManifestEncryption{
			Algo:        "aes-256-gcm-chunked",
			Fingerprint: s.cipher.Fingerprint(),
		},
	}

	bundleR, bundleW := io.Pipe()
	go func() {
		err := WriteBundle(bundleW, manifest, sources)
		_ = bundleW.CloseWithError(err)
	}()
	sealed := s.cipher.Seal(bundleR)

	targetPath := fmt.Sprintf("%s/%s.tar.gz.enc", b.StartedAt.Format("2006/01"), b.ID)
	ref, err := target.Put(ctx, targetPath, sealed, -1)
	if err != nil {
		return fmt.Errorf("target.Put: %w", err)
	}

	return s.store.MarkBackupSucceeded(ctx, b.ID, BackupResult{
		Bytes:           ref.Bytes,
		SHA256:          ref.SHA256,
		KeyFingerprint:  s.cipher.Fingerprint(),
		TargetPath:      ref.Path,
		PGVersion:       ParsePGMajorMinor(pgVerStr),
		OpendrayVersion: info.Version,
		GitSHA:          info.Commit,
	})
}

// ─── reads / lifecycle ────────────────────────────────────────────

func (s *Service) GetBackup(ctx context.Context, id string) (Backup, error) {
	return s.store.GetBackup(ctx, id)
}

func (s *Service) ListBackups(ctx context.Context, f BackupListFilter) ([]Backup, error) {
	return s.store.ListBackups(ctx, f)
}

// DownloadBackup opens the blob via the backup's target. The caller
// owns the returned ReadCloser. Returns the backup row alongside so
// the HTTP layer can set Content-Length / filename headers.
func (s *Service) DownloadBackup(ctx context.Context, id string) (io.ReadCloser, Backup, error) {
	b, err := s.store.GetBackup(ctx, id)
	if err != nil {
		return nil, Backup{}, err
	}
	if b.Status != BackupSucceeded {
		return nil, b, fmt.Errorf("backup %s status=%s; not downloadable", id, b.Status)
	}
	target := s.targets.get(b.TargetID)
	if target == nil {
		return nil, b, ErrTargetNotFound
	}
	rc, err := target.Get(ctx, TargetRef{
		Target: b.TargetID,
		Path:   b.TargetPath,
		Bytes:  b.Bytes,
		SHA256: b.SHA256,
	})
	if err != nil {
		return nil, b, err
	}
	return rc, b, nil
}

// DeleteBackup removes the blob from its target then soft-deletes
// the row (status='deleted'). Target failures are logged but the
// row still flips so retention isn't blocked by a stuck target.
func (s *Service) DeleteBackup(ctx context.Context, id string) error {
	b, err := s.store.GetBackup(ctx, id)
	if err != nil {
		return err
	}
	target := s.targets.get(b.TargetID)
	if target != nil && b.TargetPath != "" {
		ref := TargetRef{Target: b.TargetID, Path: b.TargetPath}
		if err := target.Delete(ctx, ref); err != nil {
			s.log.Warn("backup: target.Delete failed; flipping row anyway",
				"id", id, "err", err)
		}
	}
	return s.store.MarkBackupDeleted(ctx, id)
}

// CipherFingerprint exposes the active cipher's fingerprint for the
// UI banner that shows "current key: <fp>". A mismatch with a
// backup's stored fingerprint means restore needs the prior key.
func (s *Service) CipherFingerprint() string { return s.cipher.Fingerprint() }

// PGVersion returns the pg_dump --version string (cached at boot).
// The UI shows this so the operator can warn on mismatch with the
// server version when restoring.
func (s *Service) PGVersion(ctx context.Context) (string, error) {
	return s.pgdump.Version(ctx)
}

// ─── target registry (in-memory, BackupTarget instances) ──────────

type targetRegistry struct {
	mu sync.RWMutex
	by map[string]BackupTarget
}

func newTargetRegistry() *targetRegistry {
	return &targetRegistry{by: map[string]BackupTarget{}}
}

func (r *targetRegistry) get(id string) BackupTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.by[id]
}

func (r *targetRegistry) set(t BackupTarget) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.by[t.Name()] = t
}

func (r *targetRegistry) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.by, id)
}
