// Package backup implements opendray's two backup-shaped surfaces:
//
//   - Operator-facing disaster-recovery backups (A): scheduled or
//     manual full PostgreSQL dumps, encrypted with a passphrase
//     derived AES-256 key, written to a pluggable BackupTarget
//     (local disk, SMB, ...). State persists in the backups,
//     backup_schedules, and backup_targets tables (migration 0014).
//   - Admin-facing data exports (C): one-shot zip bundles of
//     selected logical entities (memories, integrations metadata,
//     custom tasks) downloaded once via a per-export token. State
//     persists in the exports table (migration 0015).
//
// Both surfaces share cipher / target / pgdump primitives but are
// otherwise independent — runtime flow lives in service_runtime.go,
// export flow in service_export.go.
package backup

import (
	"errors"
	"time"
)

// BackupStatus is the lifecycle state of a single Backup row.
type BackupStatus string

const (
	BackupPending   BackupStatus = "pending"
	BackupRunning   BackupStatus = "running"
	BackupSucceeded BackupStatus = "succeeded"
	BackupFailed    BackupStatus = "failed"
	// BackupDeleted means the row is retained for audit but the
	// underlying blob has been removed from its target. Listing
	// endpoints filter these out by default.
	BackupDeleted BackupStatus = "deleted"
)

// TriggeredBy distinguishes how a backup run was started.
type TriggeredBy string

const (
	TriggeredScheduler TriggeredBy = "scheduler"
	TriggeredManual    TriggeredBy = "manual"
	TriggeredAPI       TriggeredBy = "api"
)

// TargetKind identifies the storage backend behind a BackupTarget.
type TargetKind string

const (
	TargetLocal  TargetKind = "local"
	TargetSMB    TargetKind = "smb"
	TargetS3     TargetKind = "s3"     // AWS/R2/B2/阿里/腾讯/MinIO etc. via minio-go
	TargetWebDAV TargetKind = "webdav" // Nextcloud/群晖/坚果云/Box etc.
	TargetSFTP   TargetKind = "sftp"   // any SSH-accessible host
	TargetRclone TargetKind = "rclone" // passthrough to external rclone CLI for 70+ backends
)

// ExportStatus is the lifecycle state of a single Export row.
type ExportStatus string

const (
	ExportPending ExportStatus = "pending"
	ExportRunning ExportStatus = "running"
	ExportReady   ExportStatus = "ready"
	ExportFailed  ExportStatus = "failed"
	ExportExpired ExportStatus = "expired"
)

// IntegrationExportMode controls how integrations are serialised in
// an export bundle. "plaintext" is opt-in and only meaningful for
// system integrations whose plaintext key is cached locally; for
// the rest the manifest records the field as unrecoverable.
type IntegrationExportMode string

const (
	IntegrationExportNone      IntegrationExportMode = "none"
	IntegrationExportMetadata  IntegrationExportMode = "metadata"
	IntegrationExportPlaintext IntegrationExportMode = "plaintext"
)

// Backup is the public view of one backup row.
type Backup struct {
	ID              string         `json:"id"`
	ScheduleID      *string        `json:"schedule_id,omitempty"`
	TargetID        string         `json:"target_id"`
	Status          BackupStatus   `json:"status"`
	TriggeredBy     TriggeredBy    `json:"triggered_by"`
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      *time.Time     `json:"finished_at,omitempty"`
	Bytes           int64          `json:"bytes"`
	SHA256          string         `json:"sha256,omitempty"`
	Encrypted       bool           `json:"encrypted"`
	KeyFingerprint  string         `json:"key_fingerprint,omitempty"`
	TargetPath      string         `json:"target_path,omitempty"`
	PGVersion       string         `json:"pg_version,omitempty"`
	OpendrayVersion string         `json:"opendray_version,omitempty"`
	GitSHA          string         `json:"git_sha,omitempty"`
	Error           string         `json:"error,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// Schedule is a recurring backup spec.
type Schedule struct {
	ID          string     `json:"id"`
	TargetID    string     `json:"target_id"`
	IntervalSec int        `json:"interval_sec"`
	Retention   int        `json:"retention"`
	Enabled     bool       `json:"enabled"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	NextRunAt   time.Time  `json:"next_run_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TargetSpec is the public view of a stored BackupTarget config.
// Sensitive fields inside Config (e.g. SMB password) are returned
// redacted; the raw form is only used internally.
type TargetSpec struct {
	ID        string         `json:"id"`
	Kind      TargetKind     `json:"kind"`
	Config    map[string]any `json:"config"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// Export is the public view of one export row.
type Export struct {
	ID            string         `json:"id"`
	Status        ExportStatus   `json:"status"`
	RequestedBy   string         `json:"requested_by"`
	Scope         ExportScope    `json:"scope"`
	StartedAt     time.Time      `json:"started_at"`
	FinishedAt    *time.Time     `json:"finished_at,omitempty"`
	ExpiresAt     time.Time      `json:"expires_at"`
	Bytes         int64          `json:"bytes"`
	SHA256        string         `json:"sha256,omitempty"`
	DownloadToken string         `json:"download_token,omitempty"` // omitted in list responses
	Error         string         `json:"error,omitempty"`
}

// ExportScope captures what the operator asked to be included in a
// bundle. Stored verbatim in exports.scope (JSONB).
type ExportScope struct {
	Memories     bool                  `json:"memories"`
	Integrations IntegrationExportMode `json:"integrations"`
	CustomTasks  bool                  `json:"custom_tasks"`
}

// ImportStatus is the lifecycle state of a single Import row.
type ImportStatus string

const (
	ImportPending   ImportStatus = "pending"
	ImportRunning   ImportStatus = "running"
	ImportSucceeded ImportStatus = "succeeded"
	ImportFailed    ImportStatus = "failed"
)

// EntityCounts tracks per-entity import outcomes. Each kind reports
// how many rows were created (newly inserted), skipped (already
// existed by id or unique key), and failed (e.g. constraint violation
// not covered by skip path).
type EntityCounts struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// ImportCounts is the aggregate result of one ImportBundle run.
type ImportCounts struct {
	Memories     EntityCounts `json:"memories"`
	Integrations EntityCounts `json:"integrations"`
	CustomTasks  EntityCounts `json:"custom_tasks"`
}

// Import is the public view of one import row.
type Import struct {
	ID             string       `json:"id"`
	Status         ImportStatus `json:"status"`
	RequestedBy    string       `json:"requested_by"`
	StartedAt      time.Time    `json:"started_at"`
	FinishedAt     *time.Time   `json:"finished_at,omitempty"`
	SourceFilename string       `json:"source_filename,omitempty"`
	SourceBytes    int64        `json:"source_bytes"`
	Counts         ImportCounts `json:"counts"`
	Error          string       `json:"error,omitempty"`
}

// RestoreResult is what the /backups/restore endpoint returns. It
// is NOT persisted — restore is a one-shot operator operation whose
// outcome is the database itself. Audit-logged via slog.
type RestoreResult struct {
	Manifest        BundleManifest `json:"manifest"`
	BytesRead       int64          `json:"bytes_read"`
	TargetDSNUsed   string         `json:"target_dsn_used"`   // redacted (host/db only)
	FingerprintOK   bool           `json:"fingerprint_ok"`    // matched server cipher
	PGRestoreOutput string         `json:"pg_restore_output"` // tail of stderr/stdout
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      time.Time      `json:"finished_at"`
}

// Sentinel errors. All errors returned across package boundaries
// wrap one of these so callers can errors.Is them.
var (
	ErrCipherUnconfigured = errors.New("backup: OPENDRAY_BACKUP_KEY not set")
	ErrCipherCorrupted    = errors.New("backup: ciphertext corrupted or truncated")
	ErrCipherWrongKey     = errors.New("backup: wrong key or tampered ciphertext")
	ErrCipherFormat       = errors.New("backup: unrecognised cipher format/version")

	ErrTargetNotFound      = errors.New("backup: target not found")
	ErrTargetUnsupported   = errors.New("backup: target kind not supported in this build")
	ErrBackupNotFound      = errors.New("backup: backup not found")
	ErrScheduleNotFound    = errors.New("backup: schedule not found")
	ErrExportNotFound      = errors.New("backup: export not found")
	ErrExportExpired       = errors.New("backup: export expired")
	ErrInvalidDownloadToken = errors.New("backup: invalid download token")

	ErrPgDumpUnavailable    = errors.New("backup: pg_dump binary not found on PATH")
	ErrPgRestoreUnavailable = errors.New("backup: pg_restore binary not found on PATH")
	ErrFeatureDisabled      = errors.New("backup: feature disabled (cfg.backup.enabled=false)")

	ErrImportNotFound          = errors.New("backup: import not found")
	ErrRestoreFingerprintMismatch = errors.New("backup: restore: bundle fingerprint does not match running cipher")
	ErrRestoreNoDump              = errors.New("backup: restore: no dump.bin in bundle")
	ErrImportBadBundle            = errors.New("backup: import: malformed bundle")
)
