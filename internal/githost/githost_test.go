package githost

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// devDB returns a pgxpool against the dev DB addressed by
// OPENDRAY_DEV_DB_URL. Tests t.Skip when the env is unset (CI default)
// or when the DB is unreachable. Never hardcode credentials — see
// docs/quickstart.md for spinning up the bundled Postgres.
func devDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("OPENDRAY_DEV_DB_URL")
	if url == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set; export a writable Postgres DSN to run this test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skipf("dev DB unreachable, skipping: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("dev DB ping failed, skipping: %v", err)
	}
	return pool
}

// uniqHost generates a host string unique to this test run so parallel
// CI matrix jobs / leftover rows from earlier failures don't collide
// on the (host) UNIQUE constraint. PID disambiguates concurrent
// processes; nanosecond timestamp disambiguates within a process.
func uniqHost(prefix string) string {
	return prefix + "-test-" +
		strconv.Itoa(os.Getpid()) + "-" +
		time.Now().Format("150405.000000000")
}

func TestService_CRUDFlow(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()

	svc := NewService(pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := uniqHost("github")
	created, err := svc.Create(ctx, CreateRequest{
		Kind:  KindGitHub,
		Host:  host,
		Name:  "test fixture",
		Token: "ghp_smoke_test_token_value_long_enough",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Delete(context.Background(), created.ID)
	})

	if created.ID == "" {
		t.Fatal("Create returned empty ID")
	}
	if created.Token != "" {
		t.Errorf("Create response should redact token, got %q", created.Token)
	}
	// redact() formats as "•••• <last4>"; assert it doesn't leak the
	// full token and ends with the last 4 chars of the input.
	if strings.Contains(created.TokenMask, "smoke_test_token") {
		t.Errorf("TokenMask leaks original token: %q", created.TokenMask)
	}
	if !strings.HasSuffix(created.TokenMask, "ough") {
		t.Errorf("TokenMask should end with last4 of input, got %q",
			created.TokenMask)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Host != host {
		t.Errorf("Host = %q, want %q", got.Host, host)
	}
	if got.Token != "" {
		t.Error("Get should redact token in response")
	}

	// GetByHost returns the unredacted token (used by upstream API calls).
	withTok, err := svc.GetByHost(ctx, host)
	if err != nil {
		t.Fatalf("GetByHost: %v", err)
	}
	if withTok.Token == "" {
		t.Error("GetByHost should return unredacted token")
	}

	all, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, h := range all {
		if h.ID == created.ID {
			found = true
			if h.Token != "" {
				t.Error("List should redact tokens")
			}
			break
		}
	}
	if !found {
		t.Errorf("List did not include the created row %q", created.ID)
	}

	enabled := false
	updated, err := svc.Update(ctx, created.ID, UpdateRequest{Enabled: &enabled})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Enabled {
		t.Error("Update did not toggle Enabled to false")
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); err == nil {
		t.Error("Get after Delete should error")
	}
}

func TestService_ValidationErrors(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	svc := NewService(pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tests := []struct {
		name string
		req  CreateRequest
	}{
		{"missing host", CreateRequest{Kind: KindGitHub, Token: "tok"}},
		{"missing token", CreateRequest{Kind: KindGitHub, Host: "x.example.com"}},
		{"invalid kind", CreateRequest{Kind: "svn", Host: "x", Token: "t"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(ctx, tc.req); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestService_GetMissing(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	svc := NewService(pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := svc.Get(ctx, "doesnotexist-xyz")
	if err == nil {
		t.Fatal("expected ErrNotFound for unknown id")
	}
}
