package store

import (
	"context"
	"testing"
)

// insertTestSession creates a minimal sessions row and returns the
// generated id so SCBaseline FK writes succeed.
func insertTestSession(t *testing.T, ctx context.Context, db *DB) string {
	t.Helper()
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO sessions (name, session_type, cwd, status, model)
		VALUES ('sc-test', 'claude', '/tmp', 'stopped', '')
		RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insertTestSession: %v", err)
	}
	return id
}

func TestSCBaseline_UpsertAndGet(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()
	sid := insertTestSession(t, ctx, db)

	// Insert.
	b, err := db.SCBaselineUpsert(ctx, sid, "/tmp/repo", "abc123")
	if err != nil {
		t.Fatalf("SCBaselineUpsert: %v", err)
	}
	if b.HeadSHA != "abc123" || b.RepoPath != "/tmp/repo" {
		t.Fatalf("unexpected baseline: %+v", b)
	}
	if b.CreatedAt.IsZero() {
		t.Error("CreatedAt should be populated")
	}

	// Get.
	got, ok, err := db.SCBaselineGet(ctx, sid, "/tmp/repo")
	if err != nil {
		t.Fatalf("SCBaselineGet: %v", err)
	}
	if !ok {
		t.Fatal("SCBaselineGet: found=false")
	}
	if got.HeadSHA != "abc123" {
		t.Errorf("got sha %q, want abc123", got.HeadSHA)
	}
}

func TestSCBaseline_UpsertReplacesExisting(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()
	sid := insertTestSession(t, ctx, db)

	_, err := db.SCBaselineUpsert(ctx, sid, "/tmp/r", "old-sha")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	updated, err := db.SCBaselineUpsert(ctx, sid, "/tmp/r", "new-sha")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if updated.HeadSHA != "new-sha" {
		t.Errorf("expected HEAD to update; got %q", updated.HeadSHA)
	}
}

func TestSCBaseline_GetMissingReturnsFalse(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()
	sid := insertTestSession(t, ctx, db)

	_, ok, err := db.SCBaselineGet(ctx, sid, "/no/such/repo")
	if err != nil {
		t.Fatalf("SCBaselineGet: %v", err)
	}
	if ok {
		t.Error("SCBaselineGet: want found=false")
	}
}

func TestSCBaseline_ListSessionAcrossRepos(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()
	sid := insertTestSession(t, ctx, db)

	for _, repo := range []string{"/a", "/b", "/c"} {
		if _, err := db.SCBaselineUpsert(ctx, sid, repo, "sha-"+repo); err != nil {
			t.Fatalf("upsert %s: %v", repo, err)
		}
	}
	got, err := db.SCBaselineListSession(ctx, sid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d baselines, want 3", len(got))
	}
	// Ordered by repo_path — first must be /a.
	if got[0].RepoPath != "/a" {
		t.Errorf("expected first entry = /a, got %s", got[0].RepoPath)
	}
}

func TestSCBaseline_Delete(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()
	sid := insertTestSession(t, ctx, db)

	if _, err := db.SCBaselineUpsert(ctx, sid, "/tmp/r", "sha1"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	ok, err := db.SCBaselineDelete(ctx, sid, "/tmp/r")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !ok {
		t.Error("expected delete to report true on first run")
	}
	// Second delete: no-op.
	ok, err = db.SCBaselineDelete(ctx, sid, "/tmp/r")
	if err != nil {
		t.Fatalf("delete idempotent: %v", err)
	}
	if ok {
		t.Error("expected false when nothing to delete")
	}
}

func TestSCBaseline_CascadesWithSession(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()
	sid := insertTestSession(t, ctx, db)

	if _, err := db.SCBaselineUpsert(ctx, sid, "/tmp/x", "s1"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sid); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	_, ok, err := db.SCBaselineGet(ctx, sid, "/tmp/x")
	if err != nil {
		t.Fatalf("get after cascade: %v", err)
	}
	if ok {
		t.Error("expected baseline gone after session cascade")
	}
}

func TestSCBaseline_UpsertRejectsEmpty(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	cases := []struct {
		sid, repo, sha string
	}{
		{"", "/r", "s"},
		{"sid", "", "s"},
		{"sid", "/r", ""},
	}
	for _, c := range cases {
		if _, err := db.SCBaselineUpsert(ctx, c.sid, c.repo, c.sha); err == nil {
			t.Errorf("upsert(%q,%q,%q): expected error, got nil", c.sid, c.repo, c.sha)
		}
	}
}
