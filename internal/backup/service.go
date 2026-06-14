package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	// VaultSources are the on-disk directories (notes/skills/mcp)
	// captured into full_instance bundles. Empty for a db_only
	// deployment. Restore maps each Logical back to a destination.
	VaultSources []VaultSource
	// SecretsFile is the absolute path to secrets.env (the MCP
	// ${KEY} substitution file). Captured into full_instance
	// bundles; empty disables it.
	SecretsFile string
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
	passphrase string // retained for Recovery Kit export; never logged
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
		passphrase: deps.Passphrase,
		log:        log,
	}, nil
}

// ExportRecoveryKit wraps this instance's backup passphrase under a
// caller-supplied recovery passphrase, returning a serialized kit the
// operator stores out-of-band. See recovery.go for the rationale.
func (s *Service) ExportRecoveryKit(recoveryPassphrase string) ([]byte, error) {
	if s.passphrase == "" {
		return nil, ErrCipherUnconfigured
	}
	// Reuse the already-derived cipher's fingerprint instead of paying
	// a second PBKDF2 inside ExportRecoveryKit.
	return ExportRecoveryKit(s.passphrase, recoveryPassphrase, s.cipher.Fingerprint(), time.Now())
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
		cfg, err := s.decodeS3Config(spec.Config)
		if err != nil {
			return nil, fmt.Errorf("decode s3 config %q: %w", spec.ID, err)
		}
		return NewS3Target(spec.ID, cfg)
	case TargetWebDAV:
		cfg, err := s.decodeWebDAVConfig(spec.Config)
		if err != nil {
			return nil, fmt.Errorf("decode webdav config %q: %w", spec.ID, err)
		}
		return NewWebDAVTarget(spec.ID, cfg)
	case TargetSFTP:
		cfg, err := s.decodeSFTPConfig(spec.Config)
		if err != nil {
			return nil, fmt.Errorf("decode sftp config %q: %w", spec.ID, err)
		}
		return NewSFTPTarget(spec.ID, cfg)
	case TargetRclone:
		cfg := s.decodeRcloneConfig(spec.Config)
		return NewRcloneTarget(spec.ID, cfg)
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

func (s *Service) decodeS3Config(raw map[string]any) (S3Config, error) {
	cfg := S3Config{
		Endpoint:   cfgString(raw, "endpoint"),
		Region:     cfgString(raw, "region"),
		Bucket:     cfgString(raw, "bucket"),
		AccessKey:  cfgString(raw, "access_key"),
		PathPrefix: cfgString(raw, "path_prefix"),
	}
	if v, ok := raw["use_ssl"].(bool); ok {
		cfg.UseSSL = v
	} else {
		cfg.UseSSL = true // safe default
	}
	if v, ok := raw["path_style"].(bool); ok {
		cfg.PathStyle = v
	}
	if env := cfgString(raw, "secret_key"); env != "" {
		plain, err := s.cipher.DecryptField(env)
		if err != nil {
			return cfg, fmt.Errorf("decrypt secret_key: %w", err)
		}
		cfg.SecretKey = plain
	}
	return cfg, nil
}

func (s *Service) decodeWebDAVConfig(raw map[string]any) (WebDAVConfig, error) {
	cfg := WebDAVConfig{
		BaseURL:    cfgString(raw, "base_url"),
		User:       cfgString(raw, "user"),
		PathPrefix: cfgString(raw, "path_prefix"),
	}
	if env := cfgString(raw, "password"); env != "" {
		plain, err := s.cipher.DecryptField(env)
		if err != nil {
			return cfg, fmt.Errorf("decrypt password: %w", err)
		}
		cfg.Password = plain
	}
	return cfg, nil
}

func (s *Service) decodeSFTPConfig(raw map[string]any) (SFTPConfig, error) {
	cfg := SFTPConfig{
		Host:       cfgString(raw, "host"),
		User:       cfgString(raw, "user"),
		HostKey:    cfgString(raw, "host_key"),
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
	if env := cfgString(raw, "private_key"); env != "" {
		plain, err := s.cipher.DecryptField(env)
		if err != nil {
			return cfg, fmt.Errorf("decrypt private_key: %w", err)
		}
		cfg.PrivateKey = plain
	}
	return cfg, nil
}

func (s *Service) decodeRcloneConfig(raw map[string]any) RcloneConfig {
	cfg := RcloneConfig{
		Remote:     cfgString(raw, "remote"),
		PathPrefix: cfgString(raw, "path_prefix"),
		BinaryPath: cfgString(raw, "binary_path"),
		ConfigPath: cfgString(raw, "config_path"),
	}
	if v, ok := raw["extra_args"].([]any); ok {
		for _, a := range v {
			if s, ok := a.(string); ok {
				cfg.ExtraArgs = append(cfg.ExtraArgs, s)
			}
		}
	}
	return cfg
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
	ID      string // optional; auto-generated if empty
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
	switch req.Kind {
	case TargetLocal, TargetSMB, TargetS3, TargetWebDAV, TargetSFTP, TargetRclone:
		// supported
	default:
		return TargetSpec{}, fmt.Errorf("%w: kind=%q", ErrTargetUnsupported, req.Kind)
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
	encryptIfPresent := func(key string) error {
		v, ok := out[key].(string)
		if !ok || v == "" {
			return nil
		}
		if strings.HasPrefix(v, fieldEnvelopePrefix) {
			return nil // already wrapped
		}
		env, err := s.cipher.EncryptField(v)
		if err != nil {
			return fmt.Errorf("encrypt %s: %w", key, err)
		}
		out[key] = env
		return nil
	}

	switch kind {
	case TargetLocal:
		// nothing sensitive
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
		if err := encryptIfPresent("password"); err != nil {
			return nil, err
		}
	case TargetS3:
		if cfgString(out, "endpoint") == "" {
			return nil, fmt.Errorf("s3 config: endpoint required")
		}
		if cfgString(out, "bucket") == "" {
			return nil, fmt.Errorf("s3 config: bucket required")
		}
		if cfgString(out, "access_key") == "" {
			return nil, fmt.Errorf("s3 config: access_key required")
		}
		if err := encryptIfPresent("secret_key"); err != nil {
			return nil, err
		}
	case TargetWebDAV:
		if cfgString(out, "base_url") == "" {
			return nil, fmt.Errorf("webdav config: base_url required")
		}
		if cfgString(out, "user") == "" {
			return nil, fmt.Errorf("webdav config: user required")
		}
		if err := encryptIfPresent("password"); err != nil {
			return nil, err
		}
	case TargetSFTP:
		if cfgString(out, "host") == "" {
			return nil, fmt.Errorf("sftp config: host required")
		}
		if cfgString(out, "user") == "" {
			return nil, fmt.Errorf("sftp config: user required")
		}
		hasPw := cfgString(out, "password") != ""
		hasKey := cfgString(out, "private_key") != ""
		if !hasPw && !hasKey {
			return nil, fmt.Errorf("sftp config: password or private_key required")
		}
		if err := encryptIfPresent("password"); err != nil {
			return nil, err
		}
		if err := encryptIfPresent("private_key"); err != nil {
			return nil, err
		}
	case TargetRclone:
		if cfgString(out, "remote") == "" {
			return nil, fmt.Errorf("rclone config: remote name required")
		}
	default:
		return nil, fmt.Errorf("%w: unknown kind=%q", ErrTargetUnsupported, kind)
	}
	return out, nil
}

// redactTargetSpec returns a copy with sensitive config fields
// replaced by a placeholder so listing endpoints never echo
// ciphertext (or worse, plaintext) back to the UI.
//
// Sensitive keys (any kind): password, secret_key, private_key.
func (s *Service) redactTargetSpec(spec TargetSpec) TargetSpec {
	cfg := map[string]any{}
	for k, v := range spec.Config {
		cfg[k] = v
	}
	for _, key := range []string{"password", "secret_key", "private_key"} {
		if v, ok := cfg[key].(string); ok && v != "" {
			cfg[key] = "********"
		}
	}
	spec.Config = cfg
	return spec
}

// ─── schedule CRUD ────────────────────────────────────────────────

// CreateScheduleRequest is the body for CreateSchedule.
type CreateScheduleRequest struct {
	TargetID    string
	Kind        BackupKind
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
		Kind:        req.Kind.orDefault(),
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
	Kind          BackupKind
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
	kind := req.Kind.orDefault()
	b := Backup{
		ID:          NewBackupID(),
		TargetID:    req.TargetID,
		Status:      BackupPending,
		TriggeredBy: req.TriggeredBy,
		Kind:        kind,
		StartedAt:   time.Now().UTC(),
		Encrypted:   true,
		Metadata: map[string]any{
			"include_config": req.IncludeConfig || kind == KindFullInstance,
			"kind":           string(kind),
		},
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

	// Verify the blob we just wrote is actually restorable. Best-effort:
	// a verify failure is recorded + surfaced, but the backup row stays
	// 'succeeded' (the blob exists; it's the integrity that's in doubt).
	if s.pgrestore != nil {
		if vErr := s.VerifyBackup(ctx, b.ID); vErr != nil {
			log.Warn("backup verification failed", "backup_id", b.ID, "err", vErr)
		} else {
			log.Info("backup verified", "backup_id", b.ID)
		}
	}
}

// VerifyBackup re-reads a stored backup, decrypts it, and confirms the
// dump is a readable archive via `pg_restore --list`, recording the
// outcome (verified_at / verify_error). Returns the verification error
// (if any) after recording it.
func (s *Service) VerifyBackup(ctx context.Context, id string) error {
	vErr := s.verifyBackup(ctx, id)
	msg := ""
	if vErr != nil {
		msg = vErr.Error()
	}
	if mErr := s.store.MarkBackupVerified(ctx, id, msg); mErr != nil {
		// Don't let a DB write failure bury the original diagnosis.
		if vErr != nil {
			s.log.Error("verify: could not record failed verification",
				"backup_id", id, "verify_err", vErr, "mark_err", mErr)
		}
		return mErr
	}
	return vErr
}

// verifyBackup does the actual fetch → decrypt → extract dump.bin →
// pg_restore --list. No DB writes (the caller records the outcome).
func (s *Service) verifyBackup(ctx context.Context, id string) error {
	if s.pgrestore == nil {
		return ErrPgRestoreUnavailable
	}
	b, err := s.store.GetBackup(ctx, id)
	if err != nil {
		return err
	}
	target := s.targets.get(b.TargetID)
	if target == nil {
		return ErrTargetNotFound
	}
	rc, err := target.Get(ctx, TargetRef{
		Target: b.TargetID,
		Path:   b.TargetPath,
		Bytes:  b.Bytes,
		SHA256: b.SHA256,
	})
	if err != nil {
		return fmt.Errorf("verify: fetch blob: %w", err)
	}
	defer rc.Close()

	tmp, err := os.CreateTemp(s.cfg.LocalDir, ".verify-*.bin")
	if err != nil {
		return fmt.Errorf("verify: tmp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	gzr, err := gzip.NewReader(s.cipher.Open(rc))
	if err != nil {
		_ = tmp.Close()
		return fmt.Errorf("verify: gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	found := false
	for {
		hdr, terr := tr.Next()
		if errors.Is(terr, io.EOF) {
			break
		}
		if terr != nil {
			_ = tmp.Close()
			return fmt.Errorf("verify: tar: %w", terr)
		}
		if hdr.Name == "dump.bin" {
			// Buffers the whole dump to LocalDir (peak disk ≈ 2× the
			// dump). If that ever bites on huge DBs, b.Bytes is an upper
			// bound for an io.LimitReader guard here.
			if _, cerr := io.Copy(tmp, tr); cerr != nil {
				_ = tmp.Close()
				return fmt.Errorf("verify: extract dump: %w", cerr)
			}
			found = true
			break
		}
	}
	if cerr := tmp.Close(); cerr != nil {
		return fmt.Errorf("verify: close tmp: %w", cerr)
	}
	if !found {
		return ErrRestoreNoDump
	}
	if _, lerr := s.pgrestore.List(ctx, tmpName); lerr != nil {
		return fmt.Errorf("verify: %w", lerr)
	}
	return nil
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

	// A full_instance bundle always carries config.toml (you can't
	// rebuild an instance without it), regardless of the request flag.
	fullInstance := b.Kind == KindFullInstance
	includeConfig = includeConfig || fullInstance

	sources := []BundleSource{}
	if includeConfig && s.configPath != "" {
		cf, err := os.Open(s.configPath)
		if err != nil {
			s.log.Warn("backup: skip config (open failed)",
				"path", s.configPath, "err", err)
		} else if cfgStat, statErr := cf.Stat(); statErr != nil {
			// A bad size here would write a tar header that doesn't
			// match the body and corrupt the bundle — skip instead.
			_ = cf.Close()
			s.log.Warn("backup: skip config (stat failed)",
				"path", s.configPath, "err", statErr)
		} else {
			defer cf.Close()
			sources = append(sources, BundleSource{
				Name: filepath.Base(s.configPath),
				Body: cf,
				Size: cfgStat.Size(),
			})
		}
	}

	// Full instance: the vault (notes/skills/mcp) and secrets.env, so a
	// restore reconstructs a working instance and not just its DB.
	if fullInstance && len(s.cfg.VaultSources) > 0 {
		vtarName, vtarSize, err := s.packVaultToTemp()
		if err != nil {
			return err
		}
		defer os.Remove(vtarName)
		vf, err := os.Open(vtarName)
		if err != nil {
			return fmt.Errorf("reopen vault tar: %w", err)
		}
		defer vf.Close()
		sources = append(sources, BundleSource{Name: "vault.tar", Body: vf, Size: vtarSize})
	}
	if fullInstance && s.cfg.SecretsFile != "" {
		sf, err := os.Open(s.cfg.SecretsFile)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// No secrets.env on this host — nothing to capture.
		case err != nil:
			s.log.Warn("backup: skip secrets.env (open failed)",
				"path", s.cfg.SecretsFile, "err", err)
		default:
			if sStat, statErr := sf.Stat(); statErr != nil {
				_ = sf.Close()
				s.log.Warn("backup: skip secrets.env (stat failed)",
					"path", s.cfg.SecretsFile, "err", statErr)
			} else {
				defer sf.Close()
				sources = append(sources, BundleSource{
					Name: "secrets.env",
					Body: sf,
					Size: sStat.Size(),
				})
			}
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

	// WriteBundle streams every source body in this goroutine. The
	// synchronous drain chain below (target.Put → cipher.Seal →
	// bundleR) consumes the whole pipe before target.Put returns, so by
	// the time runPipeline's deferred file closes / temp removals fire,
	// the goroutine has finished reading every BundleSource. This
	// ordering is load-bearing: target.Put MUST be synchronous (it is
	// for every BackupTarget impl) — a lazy/early-cancelling Put would
	// let a deferred Close race the still-reading goroutine.
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

// packVaultToTemp writes a vault tar to a sibling temp file in
// LocalDir and returns its path + size. tar headers require a known
// size up front, so (like pg_dump) we buffer to a file first. The
// caller owns removing the returned path.
func (s *Service) packVaultToTemp() (name string, size int64, err error) {
	tmp, err := os.CreateTemp(s.cfg.LocalDir, ".vault-*.tar")
	if err != nil {
		return "", 0, fmt.Errorf("create tmp vault tar: %w", err)
	}
	name = tmp.Name()
	if perr := PackVault(tmp, s.cfg.VaultSources); perr != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return "", 0, fmt.Errorf("pack vault: %w", perr)
	}
	if cerr := tmp.Close(); cerr != nil {
		_ = os.Remove(name)
		return "", 0, fmt.Errorf("close vault tar: %w", cerr)
	}
	st, err := os.Stat(name)
	if err != nil {
		_ = os.Remove(name)
		return "", 0, fmt.Errorf("stat vault tar: %w", err)
	}
	return name, st.Size(), nil
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

// Cipher returns the running AES-GCM cipher derived from
// OPENDRAY_BACKUP_KEY. Exposed so adjacent subsystems (e.g.
// memory/summarizer storing encrypted provider API keys) can
// reuse the same key derivation without each owning their own
// passphrase. nil-safe — callers should treat a nil return as
// "feature disabled, no cipher available".
func (s *Service) Cipher() Cipher { return s.cipher }

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
