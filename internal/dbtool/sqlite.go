package dbtool

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite; registers the "sqlite" driver
)

// sqliteDriver implements Driver over modernc.org/sqlite (pure Go, no cgo,
// so it cross-compiles under CGO_ENABLED=0 like the rest of the binary).
//
// A SQLite "connection" is a FILE PATH on the gateway host, not host/port.
// Two safety fences apply: the file path must resolve inside the
// connection's project cwd (sqliteResolvePath — rejects "../" and
// symlinked-intermediate escapes), and extension loading is never enabled
// (modernc keeps it off; the DSN carries only a busy_timeout pragma). SQLite
// has no schemas, so Schemas reports the single "main" namespace.
type sqliteDriver struct{}

func (sqliteDriver) dialect() Dialect { return sqliteDialect{} }

// sqliteResolvePath validates a SQLite file path against the project cwd.
// A relative path is taken relative to cwd; the result must stay within
// cwd after symlink resolution. This is the SQLite analogue of fs's
// resolveWithinRoot — the per-project isolation fence.
func sqliteResolvePath(dbPath, cwd string) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		return "", errors.New("dbtool: sqlite connection requires a project cwd")
	}
	if strings.TrimSpace(dbPath) == "" {
		return "", errors.New("dbtool: sqlite requires a database file path")
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("dbtool: invalid cwd: %w", err)
	}
	// Resolve the cwd's own symlinks first (e.g. macOS /var → /private/var)
	// so the containment check compares like with like.
	rootResolved := cwdAbs
	if r, rerr := filepath.EvalSymlinks(cwdAbs); rerr == nil {
		rootResolved = r
	}
	p := dbPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(rootResolved, p)
	}
	p = filepath.Clean(p)
	// Resolve symlinks on the path — or, for a not-yet-created file, on its
	// existing parent — so a symlinked intermediate can't point outside cwd.
	resolved := p
	if r, rerr := filepath.EvalSymlinks(p); rerr == nil {
		resolved = r
	} else if r, rerr := filepath.EvalSymlinks(filepath.Dir(p)); rerr == nil {
		resolved = filepath.Join(r, filepath.Base(p))
	}
	rel, err := filepath.Rel(rootResolved, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("dbtool: sqlite path escapes the project directory")
	}
	return resolved, nil
}

// sqliteHandle keeps two pools: a read-write pool (rwc) and a read-only
// pool (mode=ro). Reads go through the ro pool, whose connections SQLite
// refuses writes on at the file layer — that is the read-only fence, since
// modernc does NOT honour database/sql's TxOptions.ReadOnly.
type sqliteHandle struct {
	rw *sql.DB
	ro *sql.DB
}

func (h *sqliteHandle) Close() {
	_ = h.rw.Close()
	_ = h.ro.Close()
}

func sqlitePools(h Handle) (*sqliteHandle, error) {
	sh, ok := h.(*sqliteHandle)
	if !ok || sh.rw == nil || sh.ro == nil {
		return nil, errors.New("dbtool: handle is not a sqlite handle")
	}
	return sh, nil
}

func sqliteDSN(path string, readOnly bool) string {
	// Only a busy_timeout pragma — no extension loading, no writable schema
	// toggles. modernc leaves load_extension disabled by default. mode=ro
	// opens read-only; the default rwc creates the file if needed.
	mode := "rwc"
	if readOnly {
		mode = "ro"
	}
	return fmt.Sprintf("file:%s?mode=%s&_pragma=busy_timeout(5000)", path, mode)
}

func (sqliteDriver) Open(ctx context.Context, c Connection, opts DriverOpts) (Handle, error) {
	path, err := sqliteResolvePath(c.DBName, c.Cwd)
	if err != nil {
		return nil, err
	}
	rw, err := sql.Open("sqlite", sqliteDSN(path, false))
	if err != nil {
		return nil, fmt.Errorf("dbtool: open sqlite: %w", err)
	}
	// Materialize the file up front so the read-only pool can open it.
	if err := rw.PingContext(ctx); err != nil {
		_ = rw.Close()
		return nil, fmt.Errorf("dbtool: open sqlite: %w", err)
	}
	rw.SetMaxOpenConns(1) // SQLite is single-writer; avoids SQLITE_BUSY churn
	rw.SetConnMaxIdleTime(2 * time.Minute)
	ro, err := sql.Open("sqlite", sqliteDSN(path, true))
	if err != nil {
		_ = rw.Close()
		return nil, fmt.Errorf("dbtool: open sqlite (ro): %w", err)
	}
	ro.SetConnMaxIdleTime(2 * time.Minute)
	return &sqliteHandle{rw: rw, ro: ro}, nil
}

func (d sqliteDriver) Ping(ctx context.Context, c Connection, opts DriverOpts) PingResult {
	start := time.Now()
	path, err := sqliteResolvePath(c.DBName, c.Cwd)
	if err != nil {
		return PingResult{OK: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	db, err := sql.Open("sqlite", sqliteDSN(path, false))
	if err != nil {
		return PingResult{OK: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	defer func() { _ = db.Close() }()
	var version string
	if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err != nil {
		return PingResult{OK: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	return PingResult{
		OK:            true,
		ServerVersion: "SQLite " + version,
		IsSuperuser:   false, // no role concept; the cwd fence is the guard
		LatencyMs:     time.Since(start).Milliseconds(),
	}
}

func (sqliteDriver) Schemas(ctx context.Context, h Handle, timeout time.Duration) ([]Schema, error) {
	// SQLite has no schemas; report the single default namespace so the
	// tree/UI has a node to expand.
	return []Schema{{Name: "main"}}, nil
}

func (sqliteDriver) Tables(ctx context.Context, h Handle, schema string, timeout time.Duration) ([]Table, error) {
	sh, err := sqlitePools(h)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rows, err := sh.ro.QueryContext(ctx, `
		SELECT name, type FROM sqlite_master
		WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%'
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("dbtool: list tables: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Table
	for rows.Next() {
		var t Table
		var tType string
		if err := rows.Scan(&t.Name, &tType); err != nil {
			return nil, err
		}
		if tType == "view" {
			t.Kind = "view"
		} else {
			t.Kind = "table"
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (d sqliteDriver) TableMeta(ctx context.Context, h Handle, schema, table string, timeout time.Duration) (TableMeta, error) {
	sh, err := sqlitePools(h)
	if err != nil {
		return TableMeta{}, err
	}
	db := sh.ro
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	meta := TableMeta{Schema: schema, Table: table, PrimaryKey: []string{}, Indexes: []Index{}, ForeignKeys: []ForeignKey{}}
	q := sqliteDialect{}.QuoteIdent(table) // "t" — PRAGMA args can't be bound

	// Columns + PK order via table_info: pk>0 gives the key ordinal.
	colRows, err := db.QueryContext(ctx, "PRAGMA table_info("+q+")")
	if err != nil {
		return meta, fmt.Errorf("dbtool: table columns: %w", err)
	}
	type pkCol struct {
		name string
		ord  int
	}
	var pks []pkCol
	for colRows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := colRows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			_ = colRows.Close()
			return meta, err
		}
		meta.Columns = append(meta.Columns, Column{
			Name: name, DataType: ctype, Nullable: notnull == 0,
			Default: dflt.String, Position: cid + 1,
		})
		if pk > 0 {
			pks = append(pks, pkCol{name: name, ord: pk})
		}
	}
	_ = colRows.Close()
	if err := colRows.Err(); err != nil {
		return meta, err
	}
	if len(meta.Columns) == 0 {
		return meta, fmt.Errorf("dbtool: table %s not found", table)
	}
	for i := 1; i <= len(pks); i++ {
		for _, p := range pks {
			if p.ord == i {
				meta.PrimaryKey = append(meta.PrimaryKey, p.name)
			}
		}
	}

	// Indexes via index_list, columns via index_info per index.
	idxRows, err := db.QueryContext(ctx, "PRAGMA index_list("+q+")")
	if err != nil {
		return meta, fmt.Errorf("dbtool: table indexes: %w", err)
	}
	type idxInfo struct {
		name   string
		unique bool
	}
	var idxList []idxInfo
	for idxRows.Next() {
		var seq, unique, partial int
		var name, origin string
		if err := idxRows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			_ = idxRows.Close()
			return meta, err
		}
		idxList = append(idxList, idxInfo{name: name, unique: unique == 1})
	}
	_ = idxRows.Close()
	if err := idxRows.Err(); err != nil {
		return meta, err
	}
	for _, ii := range idxList {
		cols, err := sqliteIndexColumns(ctx, db, ii.name)
		if err != nil {
			return meta, err
		}
		meta.Indexes = append(meta.Indexes, Index{
			Name: ii.name, Unique: ii.unique, Primary: false,
			Definition: fmt.Sprintf("(%s)", strings.Join(cols, ", ")),
		})
	}

	// Foreign keys via foreign_key_list, grouped by id.
	fkRows, err := db.QueryContext(ctx, "PRAGMA foreign_key_list("+q+")")
	if err != nil {
		return meta, fmt.Errorf("dbtool: table foreign keys: %w", err)
	}
	fkByID := map[int]*ForeignKey{}
	var fkOrder []int
	for fkRows.Next() {
		var id, seq int
		var refTable, from, to, onUpdate, onDelete, match string
		if err := fkRows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			_ = fkRows.Close()
			return meta, err
		}
		fk, ok := fkByID[id]
		if !ok {
			fk = &ForeignKey{Name: fmt.Sprintf("fk_%d", id), RefSchema: "main", RefTable: refTable}
			fkByID[id] = fk
			fkOrder = append(fkOrder, id)
		}
		fk.Columns = append(fk.Columns, from)
		fk.RefColumns = append(fk.RefColumns, to)
	}
	_ = fkRows.Close()
	if err := fkRows.Err(); err != nil {
		return meta, err
	}
	for _, id := range fkOrder {
		meta.ForeignKeys = append(meta.ForeignKeys, *fkByID[id])
	}
	return meta, nil
}

func sqliteIndexColumns(ctx context.Context, db *sql.DB, index string) ([]string, error) {
	q := sqliteDialect{}.QuoteIdent(index)
	rows, err := db.QueryContext(ctx, "PRAGMA index_info("+q+")")
	if err != nil {
		return nil, fmt.Errorf("dbtool: index columns: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var cols []string
	for rows.Next() {
		var seqno, cid int
		var name sql.NullString
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, err
		}
		if name.Valid {
			cols = append(cols, name.String)
		}
	}
	return cols, rows.Err()
}

func (d sqliteDriver) TableData(ctx context.Context, h Handle, req TableDataReq, timeout time.Duration) (*ResultSet, error) {
	sh, err := sqlitePools(h)
	if err != nil {
		return nil, err
	}
	sqlText, args, err := buildTableDataSQL(req, d.dialect())
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	return sqlExecute(ctx, sh.ro, sqlText, args, true, true, limit, timeout)
}

func (d sqliteDriver) InsertRow(ctx context.Context, h Handle, req RowInsertReq, timeout time.Duration) (*ResultSet, error) {
	sh, err := sqlitePools(h)
	if err != nil {
		return nil, err
	}
	sqlText, args, err := buildInsertSQL(req, d.dialect()) // RETURNING * (SQLite 3.35+)
	if err != nil {
		return nil, err
	}
	return sqlExecute(ctx, sh.rw, sqlText, args, false, true, 10, timeout)
}

func (d sqliteDriver) UpdateRow(ctx context.Context, h Handle, req RowUpdateReq, timeout time.Duration) (int64, error) {
	sh, err := sqlitePools(h)
	if err != nil {
		return 0, err
	}
	sqlText, args, err := buildUpdateSQL(req, d.dialect())
	if err != nil {
		return 0, err
	}
	return sqlExecuteAffected(ctx, sh.rw, sqlText, args, timeout)
}

func (d sqliteDriver) DeleteRows(ctx context.Context, h Handle, req RowDeleteReq, timeout time.Duration) (int64, error) {
	if req.Schema == "" || req.Table == "" {
		return 0, fmt.Errorf("dbtool: schema and table are required")
	}
	if len(req.PKs) == 0 {
		return 0, fmt.Errorf("dbtool: delete requires at least one primary key")
	}
	sh, err := sqlitePools(h)
	if err != nil {
		return 0, err
	}
	dl := d.dialect()
	var total int64
	for _, pk := range req.PKs {
		if len(pk) == 0 {
			return 0, fmt.Errorf("dbtool: delete requires the row's primary key")
		}
		var sb strings.Builder
		var args []any
		sb.WriteString("DELETE FROM ")
		sb.WriteString(dl.QuoteIdent(req.Schema, req.Table))
		sb.WriteString(" WHERE ")
		writePKWhere(&sb, &args, pk, dl)
		n, err := sqlExecuteAffected(ctx, sh.rw, sb.String(), args, timeout)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

func (d sqliteDriver) Query(ctx context.Context, h Handle, req QueryReq, class StatementClass, maxRows int, timeout time.Duration) (*ResultSet, error) {
	sh, err := sqlitePools(h)
	if err != nil {
		return nil, err
	}
	readOnly := class == ClassRead
	// Reads go through the ro pool (write-refusing at the file layer);
	// writes/DDL through the rw pool.
	db := sh.rw
	if readOnly {
		db = sh.ro
	}
	return sqlExecute(ctx, db, req.SQL, nil, readOnly, readOnly, maxRows, timeout)
}
