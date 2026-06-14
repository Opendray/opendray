package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// store wraps the pgxpool with backup-specific CRUD. Unexported per
// project convention — callers go through Service.
type store struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

// ─── targets ──────────────────────────────────────────────────────

func (s *store) InsertTarget(ctx context.Context, t TargetSpec) error {
	cfgRaw, err := json.Marshal(t.Config)
	if err != nil {
		return fmt.Errorf("marshal target config: %w", err)
	}
	if cfgRaw == nil || string(cfgRaw) == "null" {
		cfgRaw = []byte("{}")
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO backup_targets (id, kind, config, enabled, created_at, updated_at)
		VALUES ($1, $2, $3::jsonb, $4, $5, $5)`,
		t.ID, string(t.Kind), cfgRaw, t.Enabled, t.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert backup target: %w", err)
	}
	return nil
}

func (s *store) GetTarget(ctx context.Context, id string) (TargetSpec, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, kind, config, enabled, created_at, updated_at
		   FROM backup_targets WHERE id=$1`, id)
	t, err := scanTarget(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return TargetSpec{}, ErrTargetNotFound
	}
	return t, err
}

func (s *store) ListTargets(ctx context.Context) ([]TargetSpec, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, kind, config, enabled, created_at, updated_at
		   FROM backup_targets ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list backup targets: %w", err)
	}
	defer rows.Close()
	var out []TargetSpec
	for rows.Next() {
		t, err := scanTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TargetPatch carries optional updates for store.UpdateTarget.
type TargetPatch struct {
	Config  map[string]any
	Enabled *bool
}

func (s *store) UpdateTarget(ctx context.Context, id string, patch TargetPatch) error {
	if patch.Config != nil {
		raw, err := json.Marshal(patch.Config)
		if err != nil {
			return fmt.Errorf("marshal target config: %w", err)
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE backup_targets SET config=$1::jsonb, updated_at=NOW() WHERE id=$2`,
			raw, id); err != nil {
			return fmt.Errorf("update target config: %w", err)
		}
	}
	if patch.Enabled != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE backup_targets SET enabled=$1, updated_at=NOW() WHERE id=$2`,
			*patch.Enabled, id); err != nil {
			return fmt.Errorf("update target enabled: %w", err)
		}
	}
	return nil
}

func (s *store) DeleteTarget(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM backup_targets WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete target: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrTargetNotFound
	}
	return nil
}

func scanTarget(row rowScanner) (TargetSpec, error) {
	var (
		t      TargetSpec
		kind   string
		cfgRaw []byte
	)
	if err := row.Scan(&t.ID, &kind, &cfgRaw, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return TargetSpec{}, err
	}
	t.Kind = TargetKind(kind)
	if len(cfgRaw) > 0 {
		_ = json.Unmarshal(cfgRaw, &t.Config)
	}
	if t.Config == nil {
		t.Config = map[string]any{}
	}
	return t, nil
}

// ─── schedules ────────────────────────────────────────────────────

func (s *store) InsertSchedule(ctx context.Context, sc Schedule) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO backup_schedules
			(id, target_id, interval_sec, retention, enabled, kind, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		sc.ID, sc.TargetID, sc.IntervalSec, sc.Retention, sc.Enabled,
		string(sc.Kind.orDefault()), sc.NextRunAt, sc.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}
	return nil
}

func (s *store) GetSchedule(ctx context.Context, id string) (Schedule, error) {
	row := s.pool.QueryRow(ctx, scheduleSelectStmt+` WHERE id=$1`, id)
	sc, err := scanSchedule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, ErrScheduleNotFound
	}
	return sc, err
}

func (s *store) ListSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := s.pool.Query(ctx, scheduleSelectStmt+` ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		sc, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// SchedulePatch carries partial updates.
type SchedulePatch struct {
	Kind        *BackupKind
	IntervalSec *int
	Retention   *int
	Enabled     *bool
	NextRunAt   *time.Time
}

func (s *store) UpdateSchedule(ctx context.Context, id string, p SchedulePatch) error {
	if p.Kind != nil {
		if _, err := ParseBackupKind(string(*p.Kind)); err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE backup_schedules SET kind=$1, updated_at=NOW() WHERE id=$2`,
			string(p.Kind.orDefault()), id); err != nil {
			return fmt.Errorf("update kind: %w", err)
		}
	}
	if p.IntervalSec != nil {
		if *p.IntervalSec <= 0 {
			return fmt.Errorf("interval_sec must be > 0")
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE backup_schedules SET interval_sec=$1, updated_at=NOW() WHERE id=$2`,
			*p.IntervalSec, id); err != nil {
			return fmt.Errorf("update interval: %w", err)
		}
	}
	if p.Retention != nil {
		if *p.Retention < 0 {
			return fmt.Errorf("retention must be >= 0")
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE backup_schedules SET retention=$1, updated_at=NOW() WHERE id=$2`,
			*p.Retention, id); err != nil {
			return fmt.Errorf("update retention: %w", err)
		}
	}
	if p.Enabled != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE backup_schedules SET enabled=$1, updated_at=NOW() WHERE id=$2`,
			*p.Enabled, id); err != nil {
			return fmt.Errorf("update enabled: %w", err)
		}
	}
	if p.NextRunAt != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE backup_schedules SET next_run_at=$1, updated_at=NOW() WHERE id=$2`,
			*p.NextRunAt, id); err != nil {
			return fmt.Errorf("update next_run_at: %w", err)
		}
	}
	return nil
}

func (s *store) DeleteSchedule(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM backup_schedules WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

// ClaimDueSchedule atomically picks the oldest due, enabled schedule
// (FOR UPDATE SKIP LOCKED so multiple opendray instances cooperate)
// and bumps next_run_at = NOW() + interval. Returns ErrScheduleNotFound
// when nothing's due. Caller invokes RunBackupSync afterwards.
func (s *store) ClaimDueSchedule(ctx context.Context) (Schedule, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Schedule{}, fmt.Errorf("begin claim tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, scheduleSelectStmt+`
		WHERE enabled = TRUE AND next_run_at <= NOW()
		ORDER BY next_run_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1`)
	sc, err := scanSchedule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, ErrScheduleNotFound
	}
	if err != nil {
		return Schedule{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE backup_schedules
		   SET last_run_at = NOW(),
		       next_run_at = NOW() + ($1 || ' seconds')::interval,
		       updated_at  = NOW()
		 WHERE id = $2`,
		fmt.Sprintf("%d", sc.IntervalSec), sc.ID); err != nil {
		return Schedule{}, fmt.Errorf("bump next_run_at: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Schedule{}, fmt.Errorf("commit claim tx: %w", err)
	}
	// Reflect the row we just bumped so the caller sees fresh times.
	now := time.Now().UTC()
	sc.LastRunAt = &now
	sc.NextRunAt = now.Add(time.Duration(sc.IntervalSec) * time.Second)
	return sc, nil
}

const scheduleSelectStmt = `
	SELECT id, target_id, COALESCE(kind, 'db_only'), interval_sec, retention, enabled,
	       last_run_at, next_run_at, created_at, updated_at
	  FROM backup_schedules`

func scanSchedule(row rowScanner) (Schedule, error) {
	var (
		sc        Schedule
		kind      string
		lastRunAt sql.NullTime
	)
	if err := row.Scan(&sc.ID, &sc.TargetID, &kind, &sc.IntervalSec, &sc.Retention,
		&sc.Enabled, &lastRunAt, &sc.NextRunAt, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
		return Schedule{}, err
	}
	sc.Kind = BackupKind(kind)
	if lastRunAt.Valid {
		t := lastRunAt.Time
		sc.LastRunAt = &t
	}
	return sc, nil
}

// ─── backups ──────────────────────────────────────────────────────

func (s *store) InsertBackup(ctx context.Context, b Backup) error {
	metaRaw, err := json.Marshal(b.Metadata)
	if err != nil {
		return fmt.Errorf("marshal backup meta: %w", err)
	}
	if metaRaw == nil || string(metaRaw) == "null" {
		metaRaw = []byte("{}")
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO backups
			(id, schedule_id, target_id, status, triggered_by, kind, started_at,
			 encrypted, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)`,
		b.ID, scheduleIDOrNil(b.ScheduleID), b.TargetID,
		string(b.Status), string(b.TriggeredBy), string(b.Kind.orDefault()),
		b.StartedAt, b.Encrypted, metaRaw)
	if err != nil {
		return fmt.Errorf("insert backup: %w", err)
	}
	return nil
}

func (s *store) GetBackup(ctx context.Context, id string) (Backup, error) {
	row := s.pool.QueryRow(ctx, backupSelectStmt+` WHERE id=$1`, id)
	b, err := scanBackup(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Backup{}, ErrBackupNotFound
	}
	return b, err
}

// BackupListFilter narrows ListBackups results.
type BackupListFilter struct {
	Status         BackupStatus
	TargetID       string
	IncludeDeleted bool
	Limit          int
}

func (s *store) ListBackups(ctx context.Context, f BackupListFilter) ([]Backup, error) {
	q := backupSelectStmt
	args := []any{}
	conds := []string{}
	if f.Status != "" {
		args = append(args, string(f.Status))
		conds = append(conds, fmt.Sprintf("status=$%d", len(args)))
	} else if !f.IncludeDeleted {
		conds = append(conds, "status<>'deleted'")
	}
	if f.TargetID != "" {
		args = append(args, f.TargetID)
		conds = append(conds, fmt.Sprintf("target_id=$%d", len(args)))
	}
	if len(conds) > 0 {
		q += " WHERE " + joinAnd(conds)
	}
	q += " ORDER BY started_at DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		q += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	defer rows.Close()
	var out []Backup
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *store) MarkBackupRunning(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx,
		`UPDATE backups SET status='running' WHERE id=$1 AND status='pending'`, id)
	if err != nil {
		return fmt.Errorf("mark running: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrBackupNotFound
	}
	return nil
}

// BackupResult bundles the success-side write of a backup row.
type BackupResult struct {
	Bytes           int64
	SHA256          string
	KeyFingerprint  string
	TargetPath      string
	PGVersion       string
	OpendrayVersion string
	GitSHA          string
}

func (s *store) MarkBackupSucceeded(ctx context.Context, id string, r BackupResult) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE backups
		   SET status='succeeded',
		       finished_at=NOW(),
		       bytes=$1,
		       sha256=$2,
		       key_fingerprint=$3,
		       target_path=$4,
		       pg_version=$5,
		       opendray_version=$6,
		       git_sha=$7
		 WHERE id=$8`,
		r.Bytes, nullIfEmpty(r.SHA256), nullIfEmpty(r.KeyFingerprint),
		nullIfEmpty(r.TargetPath), nullIfEmpty(r.PGVersion),
		nullIfEmpty(r.OpendrayVersion), nullIfEmpty(r.GitSHA), id)
	if err != nil {
		return fmt.Errorf("mark succeeded: %w", err)
	}
	return nil
}

func (s *store) MarkBackupFailed(ctx context.Context, id string, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE backups
		   SET status='failed', finished_at=NOW(), error=$1
		 WHERE id=$2`,
		errMsg, id)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}

// MarkBackupVerified records the outcome of a post-backup
// verification. On success (empty verifyErr) verified_at is set to now
// and verify_error cleared. On failure only verify_error is set — a
// non-NULL verify_error already signals "last check failed", so we keep
// any prior verified_at rather than destroying a historical success on
// a transient blip (the UI treats verify_error as taking precedence).
func (s *store) MarkBackupVerified(ctx context.Context, id, verifyErr string) error {
	if verifyErr == "" {
		_, err := s.pool.Exec(ctx, `
			UPDATE backups SET verified_at=NOW(), verify_error=NULL WHERE id=$1`,
			id)
		if err != nil {
			return fmt.Errorf("mark verified: %w", err)
		}
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE backups SET verify_error=$1 WHERE id=$2`,
		verifyErr, id)
	if err != nil {
		return fmt.Errorf("mark verify failed: %w", err)
	}
	return nil
}

// MarkBackupDeleted flips status to 'deleted' (soft-delete, kept for
// audit). The blob removal happens out-of-band via Target.Delete.
func (s *store) MarkBackupDeleted(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx,
		`UPDATE backups SET status='deleted' WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("mark deleted: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrBackupNotFound
	}
	return nil
}

// CountSucceededByTarget is consumed by retention to decide if any
// rows need pruning.
func (s *store) CountSucceededByTarget(ctx context.Context, targetID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM backups WHERE target_id=$1 AND status='succeeded'`,
		targetID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count succeeded: %w", err)
	}
	return n, nil
}

// ListSucceededByTargetOldestFirst is consumed by retention. Caller
// keeps the last N rows and Delete-then-MarkBackupDeleted the rest.
func (s *store) ListSucceededByTargetOldestFirst(ctx context.Context, targetID string) ([]Backup, error) {
	rows, err := s.pool.Query(ctx,
		backupSelectStmt+` WHERE target_id=$1 AND status='succeeded' ORDER BY started_at ASC`,
		targetID)
	if err != nil {
		return nil, fmt.Errorf("list for retention: %w", err)
	}
	defer rows.Close()
	var out []Backup
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ResetStaleRunning flips backup rows that have been 'running' for
// longer than the cutoff to 'failed' with a stale-marker error.
// Called once at scheduler startup.
func (s *store) ResetStaleRunning(ctx context.Context, cutoff time.Duration) (int, error) {
	cmd, err := s.pool.Exec(ctx, `
		UPDATE backups
		   SET status='failed', finished_at=NOW(),
		       error='reset by scheduler: still running after restart'
		 WHERE status='running' AND started_at < NOW() - $1::interval`,
		cutoff.String())
	if err != nil {
		return 0, fmt.Errorf("reset stale running: %w", err)
	}
	return int(cmd.RowsAffected()), nil
}

const backupSelectStmt = `
	SELECT id, schedule_id,
	       COALESCE(target_id, '') AS target_id,
	       status, triggered_by,
	       COALESCE(kind, 'db_only'),
	       started_at, finished_at, bytes,
	       COALESCE(sha256, ''),
	       encrypted,
	       COALESCE(key_fingerprint, ''),
	       COALESCE(target_path, ''),
	       COALESCE(pg_version, ''),
	       COALESCE(opendray_version, ''),
	       COALESCE(git_sha, ''),
	       COALESCE(error, ''),
	       verified_at,
	       COALESCE(verify_error, ''),
	       COALESCE(metadata, '{}'::jsonb)
	  FROM backups`

// scanBackup reads a row produced by backupSelectStmt. target_id
// is COALESCE'd to ” so we can scan into a plain string — empty
// string means "this row's target was deleted" (post-migration
// 0017 nullable column).
func scanBackup(row rowScanner) (Backup, error) {
	var (
		b           Backup
		scheduleID  sql.NullString
		finishedAt  sql.NullTime
		verifiedAt  sql.NullTime
		status      string
		triggeredBy string
		kind        string
		metaRaw     []byte
	)
	err := row.Scan(
		&b.ID, &scheduleID, &b.TargetID, &status, &triggeredBy, &kind,
		&b.StartedAt, &finishedAt, &b.Bytes,
		&b.SHA256, &b.Encrypted, &b.KeyFingerprint,
		&b.TargetPath, &b.PGVersion, &b.OpendrayVersion, &b.GitSHA,
		&b.Error, &verifiedAt, &b.VerifyError, &metaRaw,
	)
	if err != nil {
		return Backup{}, err
	}
	if scheduleID.Valid {
		s := scheduleID.String
		b.ScheduleID = &s
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		b.FinishedAt = &t
	}
	if verifiedAt.Valid {
		t := verifiedAt.Time
		b.VerifiedAt = &t
	}
	b.Status = BackupStatus(status)
	b.TriggeredBy = TriggeredBy(triggeredBy)
	b.Kind = BackupKind(kind)
	if len(metaRaw) > 0 {
		_ = json.Unmarshal(metaRaw, &b.Metadata)
	}
	if b.Metadata == nil {
		b.Metadata = map[string]any{}
	}
	return b, nil
}

// ─── exports ──────────────────────────────────────────────────────

func (s *store) InsertExport(ctx context.Context, e Export) error {
	scopeRaw, err := json.Marshal(e.Scope)
	if err != nil {
		return fmt.Errorf("marshal export scope: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO exports
			(id, status, requested_by, scope, started_at, expires_at, download_token)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7)`,
		e.ID, string(e.Status), e.RequestedBy, scopeRaw,
		e.StartedAt, e.ExpiresAt, e.DownloadToken)
	if err != nil {
		return fmt.Errorf("insert export: %w", err)
	}
	return nil
}

func (s *store) GetExport(ctx context.Context, id string) (Export, error) {
	row := s.pool.QueryRow(ctx, exportSelectStmt+` WHERE id=$1`, id)
	e, err := scanExport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Export{}, ErrExportNotFound
	}
	return e, err
}

// GetExportByToken returns the export iff the supplied token
// matches. Used for the unauthenticated download endpoint.
func (s *store) GetExportByToken(ctx context.Context, id, token string) (Export, error) {
	row := s.pool.QueryRow(ctx, exportSelectStmt+` WHERE id=$1 AND download_token=$2`, id, token)
	e, err := scanExport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Export{}, ErrInvalidDownloadToken
	}
	return e, err
}

func (s *store) ListExports(ctx context.Context) ([]Export, error) {
	rows, err := s.pool.Query(ctx, exportSelectStmt+` ORDER BY started_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list exports: %w", err)
	}
	defer rows.Close()
	var out []Export
	for rows.Next() {
		e, err := scanExport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ExportResult bundles the post-build write fields.
type ExportResult struct {
	FilePath string
	Bytes    int64
	SHA256   string
}

func (s *store) MarkExportReady(ctx context.Context, id string, r ExportResult) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE exports
		   SET status='ready', finished_at=NOW(),
		       file_path=$1, bytes=$2, sha256=$3
		 WHERE id=$4`,
		r.FilePath, r.Bytes, nullIfEmpty(r.SHA256), id)
	if err != nil {
		return fmt.Errorf("mark export ready: %w", err)
	}
	return nil
}

func (s *store) MarkExportFailed(ctx context.Context, id, msg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE exports SET status='failed', finished_at=NOW(), error=$1 WHERE id=$2`,
		msg, id)
	if err != nil {
		return fmt.Errorf("mark export failed: %w", err)
	}
	return nil
}

func (s *store) MarkExportExpired(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE exports SET status='expired' WHERE id=$1 AND status<>'expired'`, id)
	if err != nil {
		return fmt.Errorf("mark export expired: %w", err)
	}
	return nil
}

func (s *store) ListExpiredExports(ctx context.Context) ([]Export, error) {
	rows, err := s.pool.Query(ctx,
		exportSelectStmt+` WHERE expires_at < NOW() AND status NOT IN ('expired')`)
	if err != nil {
		return nil, fmt.Errorf("list expired exports: %w", err)
	}
	defer rows.Close()
	var out []Export
	for rows.Next() {
		e, err := scanExport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *store) DeleteExport(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM exports WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete export: %w", err)
	}
	return nil
}

const exportSelectStmt = `
	SELECT id, status, requested_by, scope, started_at, finished_at,
	       expires_at, bytes,
	       COALESCE(sha256, ''),
	       download_token,
	       COALESCE(file_path, ''),
	       COALESCE(error, '')
	  FROM exports`

func scanExport(row rowScanner) (Export, error) {
	var (
		e          Export
		status     string
		scopeRaw   []byte
		finishedAt sql.NullTime
		filePath   string
	)
	if err := row.Scan(
		&e.ID, &status, &e.RequestedBy, &scopeRaw,
		&e.StartedAt, &finishedAt, &e.ExpiresAt, &e.Bytes,
		&e.SHA256, &e.DownloadToken, &filePath, &e.Error,
	); err != nil {
		return Export{}, err
	}
	e.Status = ExportStatus(status)
	if finishedAt.Valid {
		t := finishedAt.Time
		e.FinishedAt = &t
	}
	if len(scopeRaw) > 0 {
		_ = json.Unmarshal(scopeRaw, &e.Scope)
	}
	// file_path is intentionally not in Export — it's an internal
	// detail. Caller wanting it goes through service.
	_ = filePath
	return e, nil
}

// GetExportFilePath returns just the file_path column. Internal:
// used by service to open the bundle for streaming download.
func (s *store) GetExportFilePath(ctx context.Context, id string) (string, error) {
	var p sql.NullString
	err := s.pool.QueryRow(ctx, `SELECT file_path FROM exports WHERE id=$1`, id).Scan(&p)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrExportNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get export file_path: %w", err)
	}
	return p.String, nil
}

// ─── imports ──────────────────────────────────────────────────────

func (s *store) InsertImport(ctx context.Context, imp Import) error {
	countsRaw, _ := json.Marshal(imp.Counts)
	if countsRaw == nil {
		countsRaw = []byte("{}")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO imports
			(id, status, requested_by, started_at, source_filename, source_bytes, counts)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)`,
		imp.ID, string(imp.Status), imp.RequestedBy, imp.StartedAt,
		nullIfEmpty(imp.SourceFilename), imp.SourceBytes, countsRaw)
	if err != nil {
		return fmt.Errorf("insert import: %w", err)
	}
	return nil
}

func (s *store) GetImport(ctx context.Context, id string) (Import, error) {
	row := s.pool.QueryRow(ctx, importSelectStmt+` WHERE id=$1`, id)
	imp, err := scanImport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Import{}, ErrImportNotFound
	}
	return imp, err
}

func (s *store) ListImports(ctx context.Context, limit int) ([]Import, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		importSelectStmt+` ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list imports: %w", err)
	}
	defer rows.Close()
	var out []Import
	for rows.Next() {
		imp, err := scanImport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, imp)
	}
	return out, rows.Err()
}

func (s *store) MarkImportSucceeded(ctx context.Context, id string, counts ImportCounts) error {
	raw, _ := json.Marshal(counts)
	_, err := s.pool.Exec(ctx, `
		UPDATE imports
		   SET status='succeeded', finished_at=NOW(), counts=$1::jsonb
		 WHERE id=$2`,
		raw, id)
	if err != nil {
		return fmt.Errorf("mark import succeeded: %w", err)
	}
	return nil
}

func (s *store) MarkImportFailed(ctx context.Context, id, msg string, counts ImportCounts) error {
	raw, _ := json.Marshal(counts)
	_, err := s.pool.Exec(ctx, `
		UPDATE imports
		   SET status='failed', finished_at=NOW(), counts=$1::jsonb, error=$2
		 WHERE id=$3`,
		raw, msg, id)
	if err != nil {
		return fmt.Errorf("mark import failed: %w", err)
	}
	return nil
}

const importSelectStmt = `
	SELECT id, status, requested_by, started_at, finished_at,
	       COALESCE(source_filename, ''),
	       source_bytes,
	       COALESCE(counts, '{}'::jsonb),
	       COALESCE(error, '')
	  FROM imports`

func scanImport(row rowScanner) (Import, error) {
	var (
		imp        Import
		status     string
		finishedAt sql.NullTime
		countsRaw  []byte
	)
	if err := row.Scan(&imp.ID, &status, &imp.RequestedBy,
		&imp.StartedAt, &finishedAt, &imp.SourceFilename,
		&imp.SourceBytes, &countsRaw, &imp.Error); err != nil {
		return Import{}, err
	}
	imp.Status = ImportStatus(status)
	if finishedAt.Valid {
		t := finishedAt.Time
		imp.FinishedAt = &t
	}
	if len(countsRaw) > 0 {
		_ = json.Unmarshal(countsRaw, &imp.Counts)
	}
	return imp, nil
}

// ─── helpers ──────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func scheduleIDOrNil(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}
