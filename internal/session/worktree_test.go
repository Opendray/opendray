package session

import (
	"context"
	"testing"
	"time"
)

func TestCreateRequestValidateIsolation(t *testing.T) {
	base := CreateRequest{ProviderID: "claude", Cwd: "/tmp"}

	tests := []struct {
		name      string
		isolation string
		wantErr   bool
	}{
		{"default empty", "", false},
		{"worktree", IsolationWorktree, false},
		{"unknown mode rejected", "container", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base
			req.Isolation = tt.isolation
			err := req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with isolation=%q: err=%v, wantErr=%v", tt.isolation, err, tt.wantErr)
			}
		})
	}
}

func TestEffectiveWorkDir(t *testing.T) {
	tests := []struct {
		name string
		sess Session
		want string
	}{
		{"no isolation falls back to cwd", Session{Cwd: "/proj"}, "/proj"},
		{"isolated uses work dir", Session{Cwd: "/proj", WorkDir: "/wt/proj-ses_a"}, "/wt/proj-ses_a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sess.EffectiveWorkDir(); got != tt.want {
				t.Errorf("EffectiveWorkDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorkDirContext(t *testing.T) {
	ctx := context.Background()

	if got := WorkDir(ctx); got != "" {
		t.Errorf("empty ctx WorkDir = %q, want \"\"", got)
	}
	// Falls back to the logical cwd when no work dir was set — the
	// pre-worktree behaviour every existing caller relies on.
	ctx2 := WithCwd(ctx, "/proj")
	if got := WorkDir(ctx2); got != "/proj" {
		t.Errorf("WorkDir with only cwd = %q, want /proj", got)
	}
	ctx3 := WithWorkDir(ctx2, "/wt/proj-ses_a")
	if got := WorkDir(ctx3); got != "/wt/proj-ses_a" {
		t.Errorf("WorkDir = %q, want /wt/proj-ses_a", got)
	}
	if got := Cwd(ctx3); got != "/proj" {
		t.Errorf("Cwd = %q, want /proj (must not be shadowed by work dir)", got)
	}
	// Empty work dir is a no-op, preserving the fallback.
	if got := WorkDir(WithWorkDir(ctx2, "")); got != "/proj" {
		t.Errorf("WorkDir after empty WithWorkDir = %q, want /proj", got)
	}
}

// TestStoreWorktreeColumnsRoundTrip pins that work_dir/worktree_branch
// survive Insert → Get. Needs OPENDRAY_DEV_DB_URL (skipped otherwise).
func TestStoreWorktreeColumnsRoundTrip(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	st := newStore(pool)
	ctx := context.Background()

	id := newID()
	in := Session{
		ID: id, Name: "worktree-roundtrip", ProviderID: "claude",
		Cwd:            "/tmp",
		WorkDir:        "/tmp/wt-" + id,
		WorktreeBranch: "opendray/" + id,
		Args:           []string{}, State: StateRunning,
		StartedAt: time.Now().UTC(),
	}
	if err := st.Insert(ctx, in); err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() { _ = st.Delete(ctx, id) })

	got, err := st.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.WorkDir != in.WorkDir {
		t.Errorf("WorkDir = %q, want %q", got.WorkDir, in.WorkDir)
	}
	if got.WorktreeBranch != in.WorktreeBranch {
		t.Errorf("WorktreeBranch = %q, want %q", got.WorktreeBranch, in.WorktreeBranch)
	}
}
