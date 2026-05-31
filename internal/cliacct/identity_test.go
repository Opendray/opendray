package cliacct

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestIdentityStore_FirstObservationWins(t *testing.T) {
	dir := t.TempDir()
	is := newIdentityStore(dir)

	if _, ok := is.Known("cla_x"); ok {
		t.Error("expected no record before first Record")
	}
	if err := is.Record("cla_x", "a@example.com"); err != nil {
		t.Fatal(err)
	}
	if e, ok := is.Known("cla_x"); !ok || e != "a@example.com" {
		t.Errorf("first record: got (%q,%v)", e, ok)
	}
	// Second Record on the same id must be a no-op (drift, not replacement).
	if err := is.Record("cla_x", "b@example.com"); err != nil {
		t.Fatal(err)
	}
	if e, _ := is.Known("cla_x"); e != "a@example.com" {
		t.Errorf("second record should be ignored; got %q", e)
	}
}

func TestIdentityStore_AcceptReplaces(t *testing.T) {
	dir := t.TempDir()
	is := newIdentityStore(dir)
	if err := is.Accept("missing", "x@y.z"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on missing id, got %v", err)
	}
	_ = is.Record("cla_x", "a@example.com")
	if err := is.Accept("cla_x", "b@example.com"); err != nil {
		t.Fatal(err)
	}
	if e, _ := is.Known("cla_x"); e != "b@example.com" {
		t.Errorf("expected accepted email; got %q", e)
	}
}

func TestIdentityStore_PersistsToDisk(t *testing.T) {
	dir := t.TempDir()
	{
		is := newIdentityStore(dir)
		if err := is.Record("cla_x", "a@example.com"); err != nil {
			t.Fatal(err)
		}
	}
	is2 := newIdentityStore(dir)
	if e, ok := is2.Known("cla_x"); !ok || e != "a@example.com" {
		t.Errorf("reload: got (%q,%v)", e, ok)
	}
	// Mode bits — the file holds identity metadata; keep it 0600.
	fi, err := osStatPerms(filepath.Join(dir, "cliacct-identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi != 0o600 {
		t.Errorf("perm = %o, want 0600", fi)
	}
}

func TestIdentityStore_Forget(t *testing.T) {
	dir := t.TempDir()
	is := newIdentityStore(dir)
	_ = is.Record("cla_x", "a@example.com")
	_ = is.Record("cla_y", "b@example.com")
	if err := is.Forget("cla_x"); err != nil {
		t.Fatal(err)
	}
	if _, ok := is.Known("cla_x"); ok {
		t.Error("cla_x should be forgotten")
	}
	if _, ok := is.Known("cla_y"); !ok {
		t.Error("cla_y should still be present")
	}
}

// osStatPerms returns the mode bits of a file. Local helper so the
// other tests in this file stay readable.
func osStatPerms(path string) (uint32, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return uint32(fi.Mode().Perm()), nil
}
