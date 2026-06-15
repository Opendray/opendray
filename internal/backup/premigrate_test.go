package backup

import (
	"context"
	"testing"
)

func TestGuardPreMigrate_NoPendingIsNoop(t *testing.T) {
	// Empty pending → returns nil without attempting a snapshot, even
	// though the options are invalid (empty DSN would otherwise fail).
	err := GuardPreMigrate(context.Background(), nil, PreMigrateOptions{})
	if err != nil {
		t.Fatalf("no-pending guard should be a no-op, got %v", err)
	}
}

func TestGuardPreMigrate_SkipEnvBypasses(t *testing.T) {
	t.Setenv(SkipPreMigrateEnv, "1")
	// Pending + invalid options, but the skip env short-circuits before
	// any snapshot is attempted.
	err := GuardPreMigrate(context.Background(), []string{"0001"}, PreMigrateOptions{})
	if err != nil {
		t.Fatalf("skip-env guard should be a no-op, got %v", err)
	}
}

func TestGuardPreMigrate_FailClosedOnSnapshotError(t *testing.T) {
	t.Setenv(SkipPreMigrateEnv, "") // ensure not skipped
	// Pending + empty DSN → SnapshotBeforeMigrate errors → guard must
	// surface it (fail-closed) so the caller aborts the migration.
	err := GuardPreMigrate(context.Background(), []string{"0001"}, PreMigrateOptions{Dir: t.TempDir()})
	if err == nil {
		t.Fatal("expected fail-closed error when snapshot fails, got nil")
	}
}

func TestSnapshotBeforeMigrate_ValidatesInputs(t *testing.T) {
	if _, err := SnapshotBeforeMigrate(context.Background(), PreMigrateOptions{Dir: t.TempDir()}); err == nil {
		t.Error("expected error for empty DSN")
	}
	if _, err := SnapshotBeforeMigrate(context.Background(), PreMigrateOptions{DSN: "postgres://x/y"}); err == nil {
		t.Error("expected error for empty Dir")
	}
}
