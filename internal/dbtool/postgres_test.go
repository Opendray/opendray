package dbtool

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration tests against a real Postgres, driven by
// OPENDRAY_DEV_DB_URL. Tests t.Skip when the env is unset (CI default).
// The dev DB doubles as the "external" target: a throwaway schema with a
// fixtures table is created and dropped around each run.

func devPool(t *testing.T) (*pgxpool.Pool, Connection) {
	t.Helper()
	dsn := os.Getenv("OPENDRAY_DEV_DB_URL")
	if dsn == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set; export a writable Postgres DSN to run this test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("dev DB unreachable, skipping: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("dev DB ping failed, skipping: %v", err)
	}
	t.Cleanup(pool.Close)

	u, err := url.Parse(dsn)
	if err != nil {
		t.Skipf("cannot parse OPENDRAY_DEV_DB_URL, skipping: %v", err)
	}
	port := 5432
	if p := u.Port(); p != "" {
		port, _ = strconv.Atoi(p)
	}
	pass, _ := u.User.Password()
	conn := Connection{
		ID:       "dbc_test",
		Driver:   "postgres",
		Host:     u.Hostname(),
		Port:     port,
		DBName:   strings.TrimPrefix(u.Path, "/"),
		Username: u.User.Username(),
		Password: pass,
		SSLMode:  "prefer",
	}
	return pool, conn
}

// Fixtures live as prefixed tables in the public schema — CRUD-only dev
// DB roles (per the lab's database policy) can create tables there but
// not schemas.
const (
	testSchema = "public"
	tblItems   = "dbtool_test_items"
	tblNoPK    = "dbtool_test_no_pk"
)

func setupFixtures(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		DROP TABLE IF EXISTS %[1]s.%[2]s, %[1]s.%[3]s;
		CREATE TABLE %[1]s.%[2]s (
			id   SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			qty  INT NOT NULL DEFAULT 0,
			tags JSONB
		);
		CREATE TABLE %[1]s.%[3]s (v TEXT);
		CREATE INDEX %[2]s_name_idx ON %[1]s.%[2]s (name);
		INSERT INTO %[1]s.%[2]s (name, qty) VALUES ('apple', 3), ('banana', 5), ('cherry', 1);`,
		testSchema, tblItems, tblNoPK))
	if err != nil {
		t.Fatalf("fixtures: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			fmt.Sprintf(`DROP TABLE IF EXISTS %s.%s, %s.%s`,
				testSchema, tblItems, testSchema, tblNoPK))
	})
}

func testDriverHandle(t *testing.T, conn Connection) (postgresDriver, Handle) {
	t.Helper()
	drv := postgresDriver{}
	h, err := drv.Open(context.Background(), conn, DriverOpts{MaxConns: 2})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(h.Close)
	return drv, h
}

func TestPostgresPing(t *testing.T) {
	_, conn := devPool(t)
	drv := postgresDriver{}
	res := drv.Ping(context.Background(), conn, DriverOpts{})
	if !res.OK {
		t.Fatalf("ping failed: %s", res.Error)
	}
	if res.ServerVersion == "" {
		t.Fatal("no server version")
	}
}

func TestPostgresIntrospection(t *testing.T) {
	pool, conn := devPool(t)
	setupFixtures(t, pool)
	drv, h := testDriverHandle(t, conn)
	ctx := context.Background()
	timeout := 10 * time.Second

	schemas, err := drv.Schemas(ctx, h, timeout)
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	found := false
	for _, s := range schemas {
		if s.Name == testSchema {
			found = true
		}
		if strings.HasPrefix(s.Name, "pg_") || s.Name == "information_schema" {
			t.Fatalf("system schema %q leaked into listing", s.Name)
		}
	}
	if !found {
		t.Fatalf("schema %q missing from %v", testSchema, schemas)
	}

	tables, err := drv.Tables(ctx, h, testSchema, timeout)
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	seen := map[string]bool{}
	for _, tb := range tables {
		seen[tb.Name] = true
	}
	if !seen[tblItems] || !seen[tblNoPK] {
		t.Fatalf("fixture tables missing from listing (%d tables)", len(tables))
	}

	meta, err := drv.TableMeta(ctx, h, testSchema, tblItems, timeout)
	if err != nil {
		t.Fatalf("meta: %v", err)
	}
	if len(meta.Columns) != 4 {
		t.Fatalf("columns = %v", meta.Columns)
	}
	if len(meta.PrimaryKey) != 1 || meta.PrimaryKey[0] != "id" {
		t.Fatalf("pk = %v", meta.PrimaryKey)
	}
	if len(meta.Indexes) < 2 { // pkey + name idx
		t.Fatalf("indexes = %v", meta.Indexes)
	}

	noPK, err := drv.TableMeta(ctx, h, testSchema, tblNoPK, timeout)
	if err != nil {
		t.Fatalf("no_pk meta: %v", err)
	}
	if len(noPK.PrimaryKey) != 0 {
		t.Fatalf("no_pk pk = %v", noPK.PrimaryKey)
	}
	// Regression: a PK-less table must return EMPTY (non-nil) slices, not
	// nil — nil JSON-encodes as `null` and the web grid does `.map()` on
	// primary_key/indexes/foreign_keys, crashing with
	// "Cannot read properties of null (reading 'map')".
	if noPK.PrimaryKey == nil || noPK.Indexes == nil || noPK.ForeignKeys == nil {
		t.Fatalf("no_pk meta slices must be non-nil: pk=%#v idx=%#v fk=%#v",
			noPK.PrimaryKey, noPK.Indexes, noPK.ForeignKeys)
	}
}

func TestPostgresTableDataAndCRUD(t *testing.T) {
	pool, conn := devPool(t)
	setupFixtures(t, pool)
	drv, h := testDriverHandle(t, conn)
	ctx := context.Background()
	timeout := 10 * time.Second

	rs, err := drv.TableData(ctx, h, TableDataReq{
		Schema: testSchema, Table: tblItems, Limit: 2,
		Sort: []Sort{{Column: "qty", Desc: true}},
	}, timeout)
	if err != nil {
		t.Fatalf("table data: %v", err)
	}
	if len(rs.Rows) != 2 || !rs.Truncated {
		t.Fatalf("rows = %d truncated = %v", len(rs.Rows), rs.Truncated)
	}
	if rs.Rows[0][1] != "banana" { // highest qty first
		t.Fatalf("sort broken: %v", rs.Rows)
	}

	filtered, err := drv.TableData(ctx, h, TableDataReq{
		Schema: testSchema, Table: tblItems,
		Filters: []Filter{{Column: "name", Op: "ILIKE", Value: "%err%"}},
	}, timeout)
	if err != nil {
		t.Fatalf("filtered: %v", err)
	}
	if len(filtered.Rows) != 1 || filtered.Rows[0][1] != "cherry" {
		t.Fatalf("filter broken: %v", filtered.Rows)
	}

	// Regression: a query that matches zero rows must return EMPTY
	// (non-nil) Rows/Columns so the JSON is `[]`, not `null` (the web
	// grid maps over both).
	empty, err := drv.TableData(ctx, h, TableDataReq{
		Schema: testSchema, Table: tblItems,
		Filters: []Filter{{Column: "name", Op: "=", Value: "__nope__"}},
	}, timeout)
	if err != nil {
		t.Fatalf("empty query: %v", err)
	}
	if empty.Rows == nil || empty.Columns == nil {
		t.Fatalf("empty result slices must be non-nil: rows=%#v cols=%#v",
			empty.Rows, empty.Columns)
	}
	if len(empty.Rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(empty.Rows))
	}

	ins, err := drv.InsertRow(ctx, h, RowInsertReq{
		Schema: testSchema, Table: tblItems,
		Values: map[string]any{"name": "durian", "qty": 9},
	}, timeout)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if len(ins.Rows) != 1 {
		t.Fatalf("insert returned %v", ins.Rows)
	}

	n, err := drv.UpdateRow(ctx, h, RowUpdateReq{
		Schema: testSchema, Table: tblItems,
		PK: map[string]any{"id": 1}, Values: map[string]any{"qty": 42},
	}, timeout)
	if err != nil || n != 1 {
		t.Fatalf("update n=%d err=%v", n, err)
	}

	n, err = drv.DeleteRows(ctx, h, RowDeleteReq{
		Schema: testSchema, Table: tblItems,
		PKs: []map[string]any{{"id": 2}, {"id": 3}},
	}, timeout)
	if err != nil || n != 2 {
		t.Fatalf("delete n=%d err=%v", n, err)
	}
}

// The second fence: a write smuggled through the read path (read-only
// transaction) must fail server-side even if a classifier bug ever let
// it through.
func TestPostgresReadOnlyTransactionFence(t *testing.T) {
	pool, conn := devPool(t)
	setupFixtures(t, pool)
	drv, h := testDriverHandle(t, conn)
	ctx := context.Background()
	timeout := 10 * time.Second

	// Legit read through the read fence works.
	rs, err := drv.Query(ctx, h, QueryReq{
		SQL: fmt.Sprintf("SELECT count(*) FROM %s.%s", testSchema, tblItems),
	}, ClassRead, 100, timeout)
	if err != nil {
		t.Fatalf("read query: %v", err)
	}
	if len(rs.Rows) != 1 {
		t.Fatalf("rows = %v", rs.Rows)
	}

	// Force an UPDATE through the read path (simulating a classifier
	// miss): the READ ONLY transaction must reject it.
	_, err = drv.Query(ctx, h, QueryReq{
		SQL: fmt.Sprintf("UPDATE %s.%s SET qty = 0", testSchema, tblItems),
	}, ClassRead, 100, timeout)
	if err == nil {
		t.Fatal("write executed inside read-only transaction")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("unexpected error: %v", err)
	}

	// The write path executes the same statement fine.
	res, err := drv.Query(ctx, h, QueryReq{
		SQL: fmt.Sprintf("UPDATE %s.%s SET qty = 0", testSchema, tblItems),
	}, ClassWrite, 100, timeout)
	if err != nil {
		t.Fatalf("write query: %v", err)
	}
	if res.RowsAffected != 3 {
		t.Fatalf("rows affected = %d", res.RowsAffected)
	}
}

func TestServiceEndToEnd(t *testing.T) {
	pool, conn := devPool(t)
	setupFixtures(t, pool)

	// The dev DB is also opendray's own store here; the migration must
	// already exist (run via the app) — create the table ad hoc if the
	// dev DB hasn't seen 0072 yet.
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `SELECT 1 FROM db_connections LIMIT 1`); err != nil {
		t.Skip("db_connections table missing on dev DB; run migrations first")
	}

	store := NewStore(pool, fakeCipher{})
	svc := NewService(store, Options{}, nil)
	t.Cleanup(svc.Close)

	created, err := svc.CreateConnection(ctx, CreateParams{
		Cwd: "/tmp/dbtool-e2e", Name: "e2e-" + fmt.Sprint(time.Now().UnixNano()),
		Host: conn.Host, Port: conn.Port, DBName: conn.DBName,
		Username: conn.Username, Password: conn.Password, SSLMode: conn.SSLMode,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = svc.DeleteConnection(context.Background(), created.ID) })

	// Password must round-trip through the cipher: stored encrypted,
	// decrypted on read, never serialized.
	got, err := svc.GetConnection(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if conn.Password != "" {
		if !got.HasPassword || got.Password != conn.Password {
			t.Fatalf("password round-trip broken (has=%v)", got.HasPassword)
		}
		var stored string
		if err := pool.QueryRow(ctx,
			`SELECT password_enc FROM db_connections WHERE id=$1`, created.ID).Scan(&stored); err != nil {
			t.Fatalf("read stored: %v", err)
		}
		if !strings.HasPrefix(stored, "v1:") {
			t.Fatalf("password stored unencrypted: %q", stored)
		}
	}

	// read_only connection rejects writes at the service layer.
	ro := true
	if _, err := svc.UpdateConnection(ctx, created.ID, UpdatePatch{ReadOnly: &ro}); err != nil {
		t.Fatalf("set read_only: %v", err)
	}
	if _, err := svc.Query(ctx, created.ID, QueryReq{
		SQL: fmt.Sprintf("DELETE FROM %s.%s", testSchema, tblItems),
	}, true); err != ErrReadOnlyConnection {
		t.Fatalf("read_only write err = %v, want ErrReadOnlyConnection", err)
	}
	if _, err := svc.InsertRow(ctx, created.ID, RowInsertReq{
		Schema: testSchema, Table: tblItems, Values: map[string]any{"name": "x"},
	}); err != ErrReadOnlyConnection {
		t.Fatalf("read_only insert err = %v", err)
	}
	// Reads still work.
	if _, err := svc.Query(ctx, created.ID, QueryReq{
		SQL: fmt.Sprintf("SELECT * FROM %s.%s", testSchema, tblItems),
	}, false); err != nil {
		t.Fatalf("read on read_only conn: %v", err)
	}

	// Write scope required for non-read statements.
	rw := false
	if _, err := svc.UpdateConnection(ctx, created.ID, UpdatePatch{ReadOnly: &rw}); err != nil {
		t.Fatalf("clear read_only: %v", err)
	}
	if _, err := svc.Query(ctx, created.ID, QueryReq{
		SQL: fmt.Sprintf("DELETE FROM %s.%s WHERE id = 999", testSchema, tblItems),
	}, false); err != ErrWriteScope {
		t.Fatalf("no-scope write err = %v, want ErrWriteScope", err)
	}

	// Editing a PK-less table is refused.
	if _, err := svc.UpdateRow(ctx, created.ID, RowUpdateReq{
		Schema: testSchema, Table: tblNoPK,
		PK: map[string]any{"v": "x"}, Values: map[string]any{"v": "y"},
	}); err != ErrNoPrimaryKey {
		t.Fatalf("no-pk update err = %v, want ErrNoPrimaryKey", err)
	}
}
