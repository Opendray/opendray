package dbtool

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// SQLite runs entirely on a local temp file (pure-Go modernc, no external
// service), so this full round-trip executes in CI — unlike the postgres /
// mysql integration tests that skip without a live server.
func TestSQLiteDriverRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Connection{Driver: "sqlite", Cwd: dir, DBName: "test.db"}
	d := sqliteDriver{}
	ctx := context.Background()
	to := 10 * time.Second

	if pr := d.Ping(ctx, c, DriverOpts{}); !pr.OK {
		t.Fatalf("ping failed: %s", pr.Error)
	}
	h, err := d.Open(ctx, c, DriverOpts{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer h.Close()

	if _, err := d.Query(ctx, h, QueryReq{
		SQL: `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, note TEXT)`,
	}, ClassDDL, 100, to); err != nil {
		t.Fatalf("create table: %v", err)
	}

	rs, err := d.InsertRow(ctx, h, RowInsertReq{
		Schema: "main", Table: "users",
		Values: map[string]any{"id": 1, "name": "alice"},
	}, to)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	// SQLite supports RETURNING, so the inserted row comes back.
	if len(rs.Rows) != 1 {
		t.Fatalf("insert RETURNING gave %d rows, want 1", len(rs.Rows))
	}

	schemas, err := d.Schemas(ctx, h, to)
	if err != nil || len(schemas) != 1 || schemas[0].Name != "main" {
		t.Fatalf("schemas = %#v, err %v", schemas, err)
	}

	tables, err := d.Tables(ctx, h, "main", to)
	if err != nil || len(tables) != 1 || tables[0].Name != "users" {
		t.Fatalf("tables = %#v, err %v", tables, err)
	}

	meta, err := d.TableMeta(ctx, h, "main", "users", to)
	if err != nil {
		t.Fatalf("meta: %v", err)
	}
	if len(meta.PrimaryKey) != 1 || meta.PrimaryKey[0] != "id" {
		t.Fatalf("primary key = %#v, want [id]", meta.PrimaryKey)
	}
	if len(meta.Columns) != 3 {
		t.Fatalf("columns = %d, want 3", len(meta.Columns))
	}

	td, err := d.TableData(ctx, h, TableDataReq{Schema: "main", Table: "users", Limit: 10}, to)
	if err != nil {
		t.Fatalf("table data: %v", err)
	}
	if len(td.Rows) != 1 {
		t.Fatalf("table data rows = %d, want 1", len(td.Rows))
	}

	// Read-only fence: a write forced down the read path (readOnly tx via
	// PRAGMA query_only) must be refused by SQLite, not silently applied.
	if _, err := d.Query(ctx, h, QueryReq{
		SQL: `UPDATE users SET name = 'x' WHERE id = 1`,
	}, ClassRead, 100, to); err == nil {
		t.Fatal("read-only transaction allowed a write")
	}

	n, err := d.UpdateRow(ctx, h, RowUpdateReq{
		Schema: "main", Table: "users",
		PK:     map[string]any{"id": 1},
		Values: map[string]any{"name": "bob"},
	}, to)
	if err != nil || n != 1 {
		t.Fatalf("update affected %d, err %v", n, err)
	}

	del, err := d.DeleteRows(ctx, h, RowDeleteReq{
		Schema: "main", Table: "users",
		PKs: []map[string]any{{"id": 1}},
	}, to)
	if err != nil || del != 1 {
		t.Fatalf("delete affected %d, err %v", del, err)
	}
}

// The cwd fence is the SQLite isolation guard — a path escaping the
// project directory (via "..", an absolute path, or a symlink) must be
// rejected before any file is opened.
func TestSQLiteResolvePath(t *testing.T) {
	dir := t.TempDir()

	if _, err := sqliteResolvePath("../evil.db", dir); err == nil {
		t.Error("`..` escape was allowed")
	}
	if _, err := sqliteResolvePath("/etc/passwd", dir); err == nil {
		t.Error("absolute out-of-cwd path was allowed")
	}
	if _, err := sqliteResolvePath("app.db", ""); err == nil {
		t.Error("empty cwd was allowed")
	}
	if _, err := sqliteResolvePath("", dir); err == nil {
		t.Error("empty path was allowed")
	}

	got, err := sqliteResolvePath("sub/app.db", dir)
	if err != nil {
		t.Fatalf("in-cwd relative path rejected: %v", err)
	}
	want := filepath.Join(dir, "sub", "app.db")
	// dir may itself be a symlink (macOS /var → /private/var); compare the
	// resolved forms.
	if rd, e := filepath.EvalSymlinks(dir); e == nil {
		want = filepath.Join(rd, "sub", "app.db")
	}
	if got != want && !strings.HasSuffix(got, filepath.Join("sub", "app.db")) {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

// TestParams (the "Test" button on an unsaved connection) must carry the
// cwd through to the driver — SQLite's Ping needs it for the file-path
// fence. Regression: TestParams built the Connection without Cwd, so every
// SQLite Test failed with "requires a project cwd".
func TestTestParamsSQLiteCarriesCwd(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(nil, Options{}, nil)
	defer svc.Close()
	res := svc.TestParams(context.Background(), CreateParams{
		Cwd: dir, Name: "probe", Driver: "sqlite", DBName: "probe.db",
	})
	if !res.OK {
		t.Fatalf("sqlite TestParams failed: %s", res.Error)
	}
}
