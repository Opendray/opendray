package dbtool

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresDriver implements Driver over pgx/v5. Every user-supplied
// identifier goes through pgx.Identifier.Sanitize(); every value is a
// positional parameter. Reads run inside an explicit READ ONLY
// transaction with a SET LOCAL statement_timeout — the second fence
// behind the classifier.
type postgresDriver struct{}

type pgHandle struct{ pool *pgxpool.Pool }

func (h *pgHandle) Close() { h.pool.Close() }

// dsn builds a URL-form DSN so passwords with special characters survive
// (operator rule: never pass raw special-char passwords through string
// concatenation that something might later shell out with).
func (postgresDriver) dsn(c Connection, opts DriverOpts) string {
	u := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   "/" + c.DBName,
	}
	if c.Username != "" {
		if c.Password != "" {
			u.User = url.UserPassword(c.Username, c.Password)
		} else {
			u.User = url.User(c.Username)
		}
	}
	q := url.Values{}
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}
	q.Set("sslmode", sslMode)
	q.Set("application_name", "opendray-dbtool")
	ct := opts.ConnectTimeout
	if ct <= 0 {
		ct = 5 * time.Second
	}
	q.Set("connect_timeout", strconv.Itoa(int(ct.Seconds())))
	u.RawQuery = q.Encode()
	return u.String()
}

func (d postgresDriver) Open(ctx context.Context, c Connection, opts DriverOpts) (Handle, error) {
	cfg, err := pgxpool.ParseConfig(d.dsn(c, opts))
	if err != nil {
		return nil, fmt.Errorf("dbtool: parse connection config: %w", err)
	}
	maxConns := opts.MaxConns
	if maxConns <= 0 {
		maxConns = 3
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 2 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("dbtool: open pool: %w", err)
	}
	return &pgHandle{pool: pool}, nil
}

func (d postgresDriver) Ping(ctx context.Context, c Connection, opts DriverOpts) PingResult {
	start := time.Now()
	conn, err := pgx.Connect(ctx, d.dsn(c, opts))
	if err != nil {
		return PingResult{OK: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	defer func() { _ = conn.Close(ctx) }()
	var version string
	var super bool
	if err := conn.QueryRow(ctx,
		`SELECT current_setting('server_version'),
		        COALESCE((SELECT rolsuper FROM pg_roles WHERE rolname = current_user), FALSE)`).
		Scan(&version, &super); err != nil {
		return PingResult{OK: false, Error: err.Error(), LatencyMs: time.Since(start).Milliseconds()}
	}
	return PingResult{
		OK:            true,
		ServerVersion: version,
		IsSuperuser:   super,
		LatencyMs:     time.Since(start).Milliseconds(),
	}
}

func pgPool(h Handle) (*pgxpool.Pool, error) {
	ph, ok := h.(*pgHandle)
	if !ok || ph.pool == nil {
		return nil, errors.New("dbtool: handle is not a postgres pool")
	}
	return ph.pool, nil
}

func (postgresDriver) Schemas(ctx context.Context, h Handle, timeout time.Duration) ([]Schema, error) {
	pool, err := pgPool(h)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rows, err := pool.Query(ctx, `
		SELECT nspname FROM pg_namespace
		WHERE nspname NOT LIKE 'pg\_%' AND nspname <> 'information_schema'
		ORDER BY nspname`)
	if err != nil {
		return nil, fmt.Errorf("dbtool: list schemas: %w", err)
	}
	defer rows.Close()
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

func (postgresDriver) Tables(ctx context.Context, h Handle, schema string, timeout time.Duration) ([]Table, error) {
	pool, err := pgPool(h)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rows, err := pool.Query(ctx, `
		SELECT c.relname, c.relkind::text, GREATEST(c.reltuples, 0)::bigint
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind IN ('r','p','v','m','f')
		ORDER BY c.relname`, schema)
	if err != nil {
		return nil, fmt.Errorf("dbtool: list tables: %w", err)
	}
	defer rows.Close()
	kinds := map[string]string{
		"r": "table", "p": "table", "v": "view", "m": "view", "f": "foreign",
	}
	var out []Table
	for rows.Next() {
		var t Table
		var relkind string
		if err := rows.Scan(&t.Name, &relkind, &t.RowEstimate); err != nil {
			return nil, err
		}
		t.Kind = kinds[relkind]
		out = append(out, t)
	}
	return out, rows.Err()
}

func (postgresDriver) TableMeta(ctx context.Context, h Handle, schema, table string, timeout time.Duration) (TableMeta, error) {
	pool, err := pgPool(h)
	if err != nil {
		return TableMeta{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	meta := TableMeta{Schema: schema, Table: table}

	// One read-only snapshot so the four catalog queries (columns / PK /
	// indexes / FKs) see a consistent view even if DDL runs concurrently.
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return meta, fmt.Errorf("dbtool: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT column_name, data_type, is_nullable = 'YES',
		       COALESCE(column_default, ''), ordinal_position
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table columns: %w", err)
	}
	for rows.Next() {
		var c Column
		if err := rows.Scan(&c.Name, &c.DataType, &c.Nullable, &c.Default, &c.Position); err != nil {
			rows.Close()
			return meta, err
		}
		meta.Columns = append(meta.Columns, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return meta, err
	}
	if len(meta.Columns) == 0 {
		return meta, fmt.Errorf("dbtool: table %s.%s not found", schema, table)
	}

	pkRows, err := tx.Query(ctx, `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY (i.indkey)
		WHERE i.indisprimary AND n.nspname = $1 AND c.relname = $2
		ORDER BY array_position(i.indkey, a.attnum)`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table primary key: %w", err)
	}
	for pkRows.Next() {
		var col string
		if err := pkRows.Scan(&col); err != nil {
			pkRows.Close()
			return meta, err
		}
		meta.PrimaryKey = append(meta.PrimaryKey, col)
	}
	pkRows.Close()
	if err := pkRows.Err(); err != nil {
		return meta, err
	}

	idxRows, err := tx.Query(ctx, `
		SELECT ci.relname, pg_get_indexdef(i.indexrelid), i.indisunique, i.indisprimary
		FROM pg_index i
		JOIN pg_class c  ON c.oid  = i.indrelid
		JOIN pg_class ci ON ci.oid = i.indexrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2
		ORDER BY ci.relname`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table indexes: %w", err)
	}
	for idxRows.Next() {
		var ix Index
		if err := idxRows.Scan(&ix.Name, &ix.Definition, &ix.Unique, &ix.Primary); err != nil {
			idxRows.Close()
			return meta, err
		}
		meta.Indexes = append(meta.Indexes, ix)
	}
	idxRows.Close()
	if err := idxRows.Err(); err != nil {
		return meta, err
	}

	fkRows, err := tx.Query(ctx, `
		SELECT con.conname,
		       (SELECT array_agg(a.attname ORDER BY x.ord)
		        FROM unnest(con.conkey) WITH ORDINALITY AS x(attnum, ord)
		        JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = x.attnum),
		       rn.nspname, rc.relname,
		       (SELECT array_agg(a.attname ORDER BY x.ord)
		        FROM unnest(con.confkey) WITH ORDINALITY AS x(attnum, ord)
		        JOIN pg_attribute a ON a.attrelid = con.confrelid AND a.attnum = x.attnum)
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_class rc ON rc.oid = con.confrelid
		JOIN pg_namespace rn ON rn.oid = rc.relnamespace
		WHERE con.contype = 'f' AND n.nspname = $1 AND c.relname = $2
		ORDER BY con.conname`, schema, table)
	if err != nil {
		return meta, fmt.Errorf("dbtool: table foreign keys: %w", err)
	}
	for fkRows.Next() {
		var fk ForeignKey
		if err := fkRows.Scan(&fk.Name, &fk.Columns, &fk.RefSchema, &fk.RefTable, &fk.RefColumns); err != nil {
			fkRows.Close()
			return meta, err
		}
		meta.ForeignKeys = append(meta.ForeignKeys, fk)
	}
	fkRows.Close()
	if err := fkRows.Err(); err != nil {
		return meta, err
	}
	// Never hand back nil slices: they JSON-encode as `null`, and the
	// web grid does `.map()` on each of these — a PK-less table (or one
	// with no indexes / FKs) would crash the UI otherwise.
	if meta.PrimaryKey == nil {
		meta.PrimaryKey = []string{}
	}
	if meta.Indexes == nil {
		meta.Indexes = []Index{}
	}
	if meta.ForeignKeys == nil {
		meta.ForeignKeys = []ForeignKey{}
	}
	return meta, nil
}

func (d postgresDriver) TableData(ctx context.Context, h Handle, req TableDataReq, timeout time.Duration) (*ResultSet, error) {
	sqlText, args, err := buildTableDataSQL(req, pgDialect{})
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	return d.execute(ctx, h, sqlText, args, true, limit, timeout)
}

func (d postgresDriver) InsertRow(ctx context.Context, h Handle, req RowInsertReq, timeout time.Duration) (*ResultSet, error) {
	sqlText, args, err := buildInsertSQL(req, pgDialect{})
	if err != nil {
		return nil, err
	}
	return d.execute(ctx, h, sqlText, args, false, 10, timeout)
}

func (d postgresDriver) UpdateRow(ctx context.Context, h Handle, req RowUpdateReq, timeout time.Duration) (int64, error) {
	sqlText, args, err := buildUpdateSQL(req, pgDialect{})
	if err != nil {
		return 0, err
	}
	return d.executeAffected(ctx, h, sqlText, args, timeout)
}

func (d postgresDriver) DeleteRows(ctx context.Context, h Handle, req RowDeleteReq, timeout time.Duration) (int64, error) {
	if req.Schema == "" || req.Table == "" {
		return 0, errors.New("dbtool: schema and table are required")
	}
	if len(req.PKs) == 0 {
		return 0, errors.New("dbtool: delete requires at least one primary key")
	}
	pool, err := pgPool(h)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("dbtool: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := setLocalTimeout(ctx, tx, timeout); err != nil {
		return 0, err
	}
	var total int64
	for _, pk := range req.PKs {
		if len(pk) == 0 {
			return 0, errors.New("dbtool: delete requires the row's primary key")
		}
		var sb strings.Builder
		var args []any
		sb.WriteString("DELETE FROM ")
		sb.WriteString(pgx.Identifier{req.Schema, req.Table}.Sanitize())
		sb.WriteString(" WHERE ")
		writePKWhere(&sb, &args, pk, pgDialect{})
		tag, err := tx.Exec(ctx, sb.String(), args...)
		if err != nil {
			return 0, fmt.Errorf("dbtool: delete row: %w", err)
		}
		total += tag.RowsAffected()
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("dbtool: commit: %w", err)
	}
	return total, nil
}

func (d postgresDriver) Query(ctx context.Context, h Handle, req QueryReq, class StatementClass, maxRows int, timeout time.Duration) (*ResultSet, error) {
	return d.execute(ctx, h, req.SQL, nil, class == ClassRead, maxRows, timeout)
}

func setLocalTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error {
	ms := timeout.Milliseconds()
	if ms <= 0 {
		ms = 30_000
	}
	// SET LOCAL doesn't take bind parameters; the value is an integer we
	// computed ourselves, never user input.
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", ms)); err != nil {
		return fmt.Errorf("dbtool: set statement_timeout: %w", err)
	}
	return nil
}

// execute runs one statement inside a transaction (READ ONLY when
// readOnly) with a statement timeout, collecting at most maxRows rows
// (+1 probe row to set Truncated) rendered JSON-safe.
func (postgresDriver) execute(ctx context.Context, h Handle, sqlText string, args []any, readOnly bool, maxRows int, timeout time.Duration) (*ResultSet, error) {
	pool, err := pgPool(h)
	if err != nil {
		return nil, err
	}
	if maxRows <= 0 {
		maxRows = 500
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("dbtool: acquire connection: %w", err)
	}
	defer conn.Release()

	txOpts := pgx.TxOptions{}
	if readOnly {
		txOpts.AccessMode = pgx.ReadOnly
	}
	tx, err := conn.BeginTx(ctx, txOpts)
	if err != nil {
		return nil, fmt.Errorf("dbtool: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := setLocalTimeout(ctx, tx, timeout); err != nil {
		return nil, err
	}

	start := time.Now()
	rows, err := tx.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, normalizePGError(err)
	}

	rs := &ResultSet{}
	typeMap := conn.Conn().TypeMap()
	for _, fd := range rows.FieldDescriptions() {
		typeName := "unknown"
		if t, ok := typeMap.TypeForOID(fd.DataTypeOID); ok {
			typeName = t.Name
		}
		rs.Columns = append(rs.Columns, ColumnMeta{Name: fd.Name, Type: typeName})
	}
	for rows.Next() {
		if len(rs.Rows) >= maxRows {
			rs.Truncated = true
			break
		}
		vals, err := rows.Values()
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("dbtool: read row: %w", err)
		}
		row := make([]any, len(vals))
		for i, v := range vals {
			row[i] = jsonSafe(v)
		}
		rs.Rows = append(rs.Rows, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, normalizePGError(err)
	}
	tag := rows.CommandTag()
	rs.Command = strings.SplitN(tag.String(), " ", 2)[0]
	rs.RowsAffected = tag.RowsAffected()
	if err := tx.Commit(ctx); err != nil {
		return nil, normalizePGError(err)
	}
	rs.DurationMs = time.Since(start).Milliseconds()
	// nil slices JSON-encode as `null`; the web results table maps over
	// both — an empty table (0 rows) or a no-column statement must come
	// back as `[]`, not `null`.
	if rs.Columns == nil {
		rs.Columns = []ColumnMeta{}
	}
	if rs.Rows == nil {
		rs.Rows = [][]any{}
	}
	return rs, nil
}

// executeAffected runs a write statement and returns rows affected.
func (postgresDriver) executeAffected(ctx context.Context, h Handle, sqlText string, args []any, timeout time.Duration) (int64, error) {
	pool, err := pgPool(h)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("dbtool: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := setLocalTimeout(ctx, tx, timeout); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, sqlText, args...)
	if err != nil {
		return 0, normalizePGError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, normalizePGError(err)
	}
	return tag.RowsAffected(), nil
}

// normalizePGError keeps postgres errors readable for the console
// (position/detail preserved) without leaking Go wrapping noise.
func normalizePGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		msg := pgErr.Message
		if pgErr.Detail != "" {
			msg += " — " + pgErr.Detail
		}
		if pgErr.Hint != "" {
			msg += " (hint: " + pgErr.Hint + ")"
		}
		return fmt.Errorf("%s: %s", pgErr.Code, msg)
	}
	return err
}

// jsonSafe renders a pgx cell value into something json.Marshal handles
// deterministically: bytea → base64, timestamps → RFC 3339, exotic
// driver types → their string form.
func jsonSafe(v any) any {
	switch x := v.(type) {
	case nil, bool, string,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return x
	case []byte:
		return base64.StdEncoding.EncodeToString(x)
	case time.Time:
		return x.Format(time.RFC3339Nano)
	case [16]byte: // uuid
		return fmt.Sprintf("%x-%x-%x-%x-%x", x[0:4], x[4:6], x[6:8], x[8:10], x[10:16])
	case map[string]any, []any:
		return x
	default:
		if s, ok := v.(fmt.Stringer); ok {
			return s.String()
		}
		return fmt.Sprint(v)
	}
}
