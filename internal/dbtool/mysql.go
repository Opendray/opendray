package dbtool

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// mysqlDriver implements Driver over database/sql + go-sql-driver/mysql
// (pure Go, no cgo). It serves both MySQL and MariaDB — the wire protocol
// is identical; only the version string and RETURNING support differ, and
// this driver takes the conservative no-RETURNING path for both so row
// inserts behave the same everywhere.
//
// In MySQL a "schema" IS a database, so Schemas lists databases and every
// table query is qualified `db`.`table`. Every identifier goes through the
// backtick-quoting mysqlDialect; every value is a "?" placeholder. Reads
// run inside a READ ONLY transaction (the second fence behind the
// classifier).
type mysqlDriver struct{}

func (mysqlDriver) dialect() Dialect { return mysqlDialect{returning: false} }

// dsn builds a go-sql-driver DSN via its Config so special characters in
// the password survive without hand-escaping.
func (mysqlDriver) dsn(c Connection, opts DriverOpts) string {
	cfg := mysql.NewConfig()
	cfg.User = c.Username
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	cfg.DBName = c.DBName
	cfg.ParseTime = true // DATETIME/TIMESTAMP → time.Time
	ct := opts.ConnectTimeout
	if ct <= 0 {
		ct = 5 * time.Second
	}
	cfg.Timeout = ct
	switch c.SSLMode {
	case "disable":
		cfg.TLSConfig = "false"
	case "require":
		cfg.TLSConfig = "skip-verify" // encrypt, don't verify the cert
	case "verify-ca", "verify-full":
		cfg.TLSConfig = "true" // encrypt and verify
	default: // prefer / empty
		cfg.TLSConfig = "preferred"
	}
	return cfg.FormatDSN()
}

func (d mysqlDriver) Open(ctx context.Context, c Connection, opts DriverOpts) (Handle, error) {
	db, err := sql.Open("mysql", d.dsn(c, opts))
	if err != nil {
		return nil, fmt.Errorf("dbtool: open mysql: %w", err)
	}
	maxConns := opts.MaxConns
	if maxConns <= 0 {
		maxConns = 3
	}
	db.SetMaxOpenConns(maxConns)
	db.SetConnMaxIdleTime(2 * time.Minute)
	return &sqlHandle{db: db}, nil
}

func (d mysqlDriver) Ping(ctx context.Context, c Connection, opts DriverOpts) PingResult {
	start := time.Now()
	db, err := sql.Open("mysql", d.dsn(c, opts))
	if err != nil {
		return PingResult{OK: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	defer func() { _ = db.Close() }()
	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return PingResult{OK: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	return PingResult{
		OK:            true,
		ServerVersion: version,
		IsSuperuser:   mysqlIsSuper(ctx, db),
		LatencyMs:     time.Since(start).Milliseconds(),
	}
}

// mysqlIsSuper powers the superuser warning banner: a role holding
// `ALL PRIVILEGES ON *.*` or `SUPER` is server-admin-grade (operator rule:
// never run project work as such an account).
func mysqlIsSuper(ctx context.Context, db *sql.DB) bool {
	rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var g string
		if rows.Scan(&g) != nil {
			continue
		}
		up := strings.ToUpper(g)
		if strings.Contains(up, "ALL PRIVILEGES ON *.*") || strings.Contains(up, "SUPER") {
			return true
		}
	}
	return false
}

func (mysqlDriver) Schemas(ctx context.Context, h Handle, timeout time.Duration) ([]Schema, error) {
	db, err := sqlDB(h)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rows, err := db.QueryContext(ctx, `
		SELECT schema_name FROM information_schema.schemata
		WHERE schema_name NOT IN ('mysql','performance_schema','information_schema','sys')
		ORDER BY schema_name`)
	if err != nil {
		return nil, fmt.Errorf("dbtool: list schemas: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Schema
	for rows.Next() {
		var s Schema
		if err := rows.Scan(&s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (mysqlDriver) Tables(ctx context.Context, h Handle, schema string, timeout time.Duration) ([]Table, error) {
	db, err := sqlDB(h)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, table_type, COALESCE(table_rows, 0)
		FROM information_schema.tables
		WHERE table_schema = ?
		ORDER BY table_name`, schema)
	if err != nil {
		return nil, fmt.Errorf("dbtool: list tables: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Table
	for rows.Next() {
		var t Table
		var tType string
		if err := rows.Scan(&t.Name, &tType, &t.RowEstimate); err != nil {
			return nil, err
		}
		if strings.EqualFold(tType, "VIEW") {
			t.Kind = "view"
		} else {
			t.Kind = "table"
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (mysqlDriver) TableMeta(ctx context.Context, h Handle, schema, table string, timeout time.Duration) (TableMeta, error) {
	db, err := sqlDB(h)
	if err != nil {
		return TableMeta{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	meta := TableMeta{Schema: schema, Table: table, PrimaryKey: []string{}, Indexes: []Index{}, ForeignKeys: []ForeignKey{}}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return meta, fmt.Errorf("dbtool: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	colRows, err := tx.QueryContext(ctx, `
		SELECT column_name, column_type, is_nullable = 'YES',
		       COALESCE(column_default, ''), ordinal_position
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table columns: %w", err)
	}
	for colRows.Next() {
		var c Column
		if err := colRows.Scan(&c.Name, &c.DataType, &c.Nullable, &c.Default, &c.Position); err != nil {
			_ = colRows.Close()
			return meta, err
		}
		meta.Columns = append(meta.Columns, c)
	}
	_ = colRows.Close()
	if err := colRows.Err(); err != nil {
		return meta, err
	}
	if len(meta.Columns) == 0 {
		return meta, fmt.Errorf("dbtool: table %s.%s not found", schema, table)
	}

	// Primary key columns in key order.
	pkRows, err := tx.QueryContext(ctx, `
		SELECT column_name FROM information_schema.statistics
		WHERE table_schema = ? AND table_name = ? AND index_name = 'PRIMARY'
		ORDER BY seq_in_index`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table primary key: %w", err)
	}
	for pkRows.Next() {
		var col string
		if err := pkRows.Scan(&col); err != nil {
			_ = pkRows.Close()
			return meta, err
		}
		meta.PrimaryKey = append(meta.PrimaryKey, col)
	}
	_ = pkRows.Close()
	if err := pkRows.Err(); err != nil {
		return meta, err
	}

	// Indexes: one row per index, columns concatenated in order.
	idxRows, err := tx.QueryContext(ctx, `
		SELECT index_name, MAX(non_unique) = 0,
		       GROUP_CONCAT(column_name ORDER BY seq_in_index SEPARATOR ', ')
		FROM information_schema.statistics
		WHERE table_schema = ? AND table_name = ?
		GROUP BY index_name
		ORDER BY index_name`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table indexes: %w", err)
	}
	for idxRows.Next() {
		var ix Index
		var cols string
		if err := idxRows.Scan(&ix.Name, &ix.Unique, &cols); err != nil {
			_ = idxRows.Close()
			return meta, err
		}
		ix.Primary = ix.Name == "PRIMARY"
		ix.Definition = fmt.Sprintf("(%s)", cols)
		meta.Indexes = append(meta.Indexes, ix)
	}
	_ = idxRows.Close()
	if err := idxRows.Err(); err != nil {
		return meta, err
	}

	// Foreign keys, grouped by constraint.
	fkRows, err := tx.QueryContext(ctx, `
		SELECT constraint_name, column_name,
		       referenced_table_schema, referenced_table_name, referenced_column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ? AND table_name = ? AND referenced_table_name IS NOT NULL
		ORDER BY constraint_name, ordinal_position`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table foreign keys: %w", err)
	}
	fkByName := map[string]*ForeignKey{}
	var fkOrder []string
	for fkRows.Next() {
		var name, col, refSchema, refTable, refCol string
		if err := fkRows.Scan(&name, &col, &refSchema, &refTable, &refCol); err != nil {
			_ = fkRows.Close()
			return meta, err
		}
		fk, ok := fkByName[name]
		if !ok {
			fk = &ForeignKey{Name: name, RefSchema: refSchema, RefTable: refTable}
			fkByName[name] = fk
			fkOrder = append(fkOrder, name)
		}
		fk.Columns = append(fk.Columns, col)
		fk.RefColumns = append(fk.RefColumns, refCol)
	}
	_ = fkRows.Close()
	if err := fkRows.Err(); err != nil {
		return meta, err
	}
	for _, name := range fkOrder {
		meta.ForeignKeys = append(meta.ForeignKeys, *fkByName[name])
	}
	return meta, nil
}

func (d mysqlDriver) TableData(ctx context.Context, h Handle, req TableDataReq, timeout time.Duration) (*ResultSet, error) {
	db, err := sqlDB(h)
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
	return sqlExecute(ctx, db, sqlText, args, true, true, limit, timeout)
}

// InsertRow takes the no-RETURNING path (MySQL 8 has no RETURNING). It
// reports rows affected; the grid refreshes from the server afterwards.
func (d mysqlDriver) InsertRow(ctx context.Context, h Handle, req RowInsertReq, timeout time.Duration) (*ResultSet, error) {
	db, err := sqlDB(h)
	if err != nil {
		return nil, err
	}
	sqlText, args, err := buildInsertSQL(req, d.dialect()) // returning=false → bare INSERT
	if err != nil {
		return nil, err
	}
	return sqlExecute(ctx, db, sqlText, args, false, false, 1, timeout)
}

func (d mysqlDriver) UpdateRow(ctx context.Context, h Handle, req RowUpdateReq, timeout time.Duration) (int64, error) {
	db, err := sqlDB(h)
	if err != nil {
		return 0, err
	}
	sqlText, args, err := buildUpdateSQL(req, d.dialect())
	if err != nil {
		return 0, err
	}
	return sqlExecuteAffected(ctx, db, sqlText, args, timeout)
}

func (d mysqlDriver) DeleteRows(ctx context.Context, h Handle, req RowDeleteReq, timeout time.Duration) (int64, error) {
	if req.Schema == "" || req.Table == "" {
		return 0, fmt.Errorf("dbtool: schema and table are required")
	}
	if len(req.PKs) == 0 {
		return 0, fmt.Errorf("dbtool: delete requires at least one primary key")
	}
	db, err := sqlDB(h)
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
		n, err := sqlExecuteAffected(ctx, db, sb.String(), args, timeout)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

func (d mysqlDriver) Query(ctx context.Context, h Handle, req QueryReq, class StatementClass, maxRows int, timeout time.Duration) (*ResultSet, error) {
	db, err := sqlDB(h)
	if err != nil {
		return nil, err
	}
	readOnly := class == ClassRead
	return sqlExecute(ctx, db, req.SQL, nil, readOnly, readOnly, maxRows, timeout)
}
