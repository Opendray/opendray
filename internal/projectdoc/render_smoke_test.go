//go:build smoke

// Smoke test that runs against the live dev database to verify the
// spawn-time banner renders end-to-end. Gated behind the `smoke`
// build tag so `go test ./...` in CI doesn't try to reach the
// development DB at 192.168.3.88.
//
// Run with:
//
//	OPENDRAY_DB_URL="postgres://opd2_user:...@192.168.3.88:5432/opendray_v2?sslmode=disable" \
//	  go test -tags smoke -run TestRenderForSpawn_Live ./internal/projectdoc -v

package projectdoc

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRenderForSpawn_Live(t *testing.T) {
	dsn := os.Getenv("OPENDRAY_DB_URL")
	if dsn == "" {
		t.Skip("OPENDRAY_DB_URL not set; skipping live smoke")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	svc := NewService(pool, nil)
	cwd := "/Users/linivek/Documents/HomeLab/Claude_Workspace/opendray-v2"
	out, err := svc.RenderForSpawn(ctx, cwd, 5)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out == "" {
		t.Skip("project has no goal/plan/journal yet; nothing to inject")
	}
	t.Logf("rendered banner (%d bytes):\n%s", len(out), out)

	for _, want := range []string{
		"Project context (cross-agent shared)",
		"Project goal",
		"Project plan",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered banner missing section %q", want)
		}
	}
}
