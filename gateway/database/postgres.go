// Package database provides read-only PostgreSQL browsing for panel plugins.
//
// It exposes schema introspection (databases, schemas, tables, columns) and
// a constrained SELECT executor. DDL and DML statements are rejected before
// the query reaches the server. Results are capped by row count and query
// timeout to keep this safe for casual browsing from the UI.
package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PGConfig holds settings read from plugin config for a PostgreSQL connection.
type PGConfig struct {
	Host         string
	Port         int
	Database     string
	Username     string
	Password     string
	SSLMode      string // disable | require | verify-ca | verify-full
	QueryTimeout time.Duration
	MaxRows      int
}

// DSN builds a pgx-compatible connection string.
func (c PGConfig) DSN() string {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Database, c.Username, c.Password, c.SSLMode,
	)
}

// ── Schema types ────────────────────────────────────────────────

type DatabaseEntry struct {
	Name    string `json:"name"`
	Owner   string `json:"owner,omitempty"`
	Size    string `json:"size,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type SchemaEntry struct {
	Name  string `json:"name"`
	Owner string `json:"owner,omitempty"`
}

type TableEntry struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Kind   string `json:"kind"` // table | view | matview
	Rows   int64  `json:"rows,omitempty"`
	Size   string `json:"size,omitempty"`
}

type ColumnEntry struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
	Position int    `json:"position"`
	IsPK     bool   `json:"isPk,omitempty"`
}

type QueryResult struct {
	Columns   []string `json:"columns"`
	Types     []string `json:"types"`
	Rows      [][]any  `json:"rows"`
	RowCount  int      `json:"rowCount"`
	Truncated bool     `json:"truncated,omitempty"`
	Duration  string   `json:"duration"`
}

// ── Connection helper ───────────────────────────────────────────

func connect(ctx context.Context, cfg PGConfig) (*pgx.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: connect: %w", sanitizeErr(err, cfg))
	}
	return conn, nil
}

// sanitizeErr strips the password from any pgconn error message.
func sanitizeErr(err error, cfg PGConfig) error {
	if err == nil || cfg.Password == "" {
		return err
	}
	msg := strings.ReplaceAll(err.Error(), cfg.Password, "***")
	return errors.New(msg)
}

// ── Introspection ───────────────────────────────────────────────

// ListDatabases returns all non-template, connectable databases on the server.
// It connects to cfg.Database (usually "postgres") just to run the query; the
// returned names can then be used to override Database on subsequent calls.
func ListDatabases(ctx context.Context, cfg PGConfig) ([]DatabaseEntry, error) {
	conn, err := connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT
			d.datname,
			COALESCE(r.rolname, ''),
			pg_size_pretty(pg_database_size(d.datname)),
			pg_encoding_to_char(d.encoding)
		FROM pg_database d
		LEFT JOIN pg_roles r ON r.oid = d.datdba
		WHERE NOT d.datistemplate AND d.datallowconn
		ORDER BY d.datname
	`)
	if err != nil {
		return nil, fmt.Errorf("database: list databases: %w", sanitizeErr(err, cfg))
	}
	defer rows.Close()

	var result []DatabaseEntry
	for rows.Next() {
		var d DatabaseEntry
		if err := rows.Scan(&d.Name, &d.Owner, &d.Size, &d.Encoding); err != nil {
			return nil, fmt.Errorf("database: scan db: %w", err)
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func ListSchemas(ctx context.Context, cfg PGConfig) ([]SchemaEntry, error) {
	conn, err := connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT n.nspname, COALESCE(r.rolname, '')
		FROM pg_namespace n
		LEFT JOIN pg_roles r ON r.oid = n.nspowner
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		  AND n.nspname NOT LIKE 'pg_temp_%'
		  AND n.nspname NOT LIKE 'pg_toast_temp_%'
		ORDER BY n.nspname
	`)
	if err != nil {
		return nil, fmt.Errorf("database: list schemas: %w", err)
	}
	defer rows.Close()

	var result []SchemaEntry
	for rows.Next() {
		var e SchemaEntry
		if err := rows.Scan(&e.Name, &e.Owner); err != nil {
			return nil, fmt.Errorf("database: scan schema: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func ListTables(ctx context.Context, cfg PGConfig, schema string) ([]TableEntry, error) {
	if schema == "" {
		schema = "public"
	}
	conn, err := connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname, c.relname,
			CASE c.relkind
				WHEN 'r' THEN 'table'
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'matview'
				WHEN 'p' THEN 'table'
				ELSE c.relkind::text
			END AS kind,
			COALESCE(c.reltuples::bigint, 0) AS approx_rows,
			pg_size_pretty(pg_total_relation_size(c.oid)) AS size
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		  AND c.relkind IN ('r', 'v', 'm', 'p')
		ORDER BY c.relname
	`, schema)
	if err != nil {
		return nil, fmt.Errorf("database: list tables: %w", err)
	}
	defer rows.Close()

	var result []TableEntry
	for rows.Next() {
		var t TableEntry
		if err := rows.Scan(&t.Schema, &t.Name, &t.Kind, &t.Rows, &t.Size); err != nil {
			return nil, fmt.Errorf("database: scan table: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func ListColumns(ctx context.Context, cfg PGConfig, schema, table string) ([]ColumnEntry, error) {
	if schema == "" || table == "" {
		return nil, errors.New("database: schema and table are required")
	}
	conn, err := connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT
			a.attname,
			format_type(a.atttypid, a.atttypmod),
			NOT a.attnotnull,
			COALESCE(pg_get_expr(d.adbin, d.adrelid), ''),
			a.attnum,
			COALESCE(pk.is_pk, false) AS is_pk
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		LEFT JOIN (
			SELECT i.indrelid, unnest(i.indkey) AS attnum, true AS is_pk
			FROM pg_index i
			WHERE i.indisprimary
		) pk ON pk.indrelid = a.attrelid AND pk.attnum = a.attnum
		WHERE n.nspname = $1 AND c.relname = $2
		  AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum
	`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("database: list columns: %w", err)
	}
	defer rows.Close()

	var result []ColumnEntry
	for rows.Next() {
		var c ColumnEntry
		if err := rows.Scan(&c.Name, &c.Type, &c.Nullable, &c.Default, &c.Position, &c.IsPK); err != nil {
			return nil, fmt.Errorf("database: scan column: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// PreviewTable runs a safe SELECT * FROM schema.table LIMIT N.
func PreviewTable(ctx context.Context, cfg PGConfig, schema, table string, limit int) (QueryResult, error) {
	if schema == "" || table == "" {
		return QueryResult{}, errors.New("database: schema and table are required")
	}
	if !safeIdentifier(schema) || !safeIdentifier(table) {
		return QueryResult{}, errors.New("database: invalid schema or table identifier")
	}
	if limit <= 0 || limit > cfg.MaxRows {
		limit = cfg.MaxRows
	}
	query := fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT %d`, schema, table, limit)
	return RunQuery(ctx, cfg, query)
}

// ── Query executor (SELECT-only) ────────────────────────────────

var readOnlyStartRe = regexp.MustCompile(`(?is)^\s*(WITH\b.*?\)\s*)?(SELECT|SHOW|EXPLAIN|VALUES|TABLE)\b`)

// blockedKeywords are rejected anywhere in the query (outside strings/comments).
// This is a defence-in-depth check — the primary control is read-only transaction mode.
var blockedKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "TRUNCATE", "DROP", "ALTER",
	"CREATE", "GRANT", "REVOKE", "REINDEX", "VACUUM", "ANALYZE",
	"CLUSTER", "COPY", "CALL", "DO", "MERGE", "LOCK", "COMMIT",
	"ROLLBACK", "SAVEPOINT", "SET",
}

// RunQuery executes a single read-only SQL statement and returns rows.
func RunQuery(ctx context.Context, cfg PGConfig, sql string) (QueryResult, error) {
	stripped := stripSQLComments(sql)
	trimmed := strings.TrimSpace(stripped)
	trimmed = strings.TrimSuffix(trimmed, ";")
	if trimmed == "" {
		return QueryResult{}, errors.New("database: empty query")
	}
	if strings.Contains(trimmed, ";") {
		return QueryResult{}, errors.New("database: multiple statements are not allowed")
	}
	if !readOnlyStartRe.MatchString(trimmed) {
		return QueryResult{}, errors.New("database: only SELECT / SHOW / EXPLAIN / VALUES / TABLE statements are allowed")
	}
	if kw := findBlockedKeyword(trimmed); kw != "" {
		return QueryResult{}, fmt.Errorf("database: keyword %q is not allowed in read-only queries", kw)
	}

	timeout := cfg.QueryTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	maxRows := cfg.MaxRows
	if maxRows <= 0 {
		maxRows = 500
	}

	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := connect(qctx, cfg)
	if err != nil {
		return QueryResult{}, err
	}
	defer conn.Close(qctx)

	// Enforce read-only at the transaction level. BEGIN READ ONLY will refuse
	// any INSERT/UPDATE/DELETE even if the keyword-scan somehow let it through.
	tx, err := conn.BeginTx(qctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return QueryResult{}, fmt.Errorf("database: begin read-only: %w", sanitizeErr(err, cfg))
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	start := time.Now()
	rows, err := tx.Query(qctx, trimmed)
	if err != nil {
		return QueryResult{}, fmt.Errorf("database: query: %w", sanitizeErr(err, cfg))
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	cols := make([]string, len(fields))
	types := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = string(f.Name)
		types[i] = oidTypeName(conn, f.DataTypeOID)
	}

	resultRows := make([][]any, 0)
	truncated := false
	for rows.Next() {
		if len(resultRows) >= maxRows {
			truncated = true
			break
		}
		values, err := rows.Values()
		if err != nil {
			return QueryResult{}, fmt.Errorf("database: scan row: %w", sanitizeErr(err, cfg))
		}
		for i, v := range values {
			values[i] = marshalValue(v)
		}
		resultRows = append(resultRows, values)
	}
	if err := rows.Err(); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return QueryResult{}, fmt.Errorf("database: %s: %s", pgErr.Code, pgErr.Message)
		}
		return QueryResult{}, fmt.Errorf("database: rows: %w", sanitizeErr(err, cfg))
	}

	return QueryResult{
		Columns:   cols,
		Types:     types,
		Rows:      resultRows,
		RowCount:  len(resultRows),
		Truncated: truncated,
		Duration:  time.Since(start).String(),
	}, nil
}

// ── Helpers ─────────────────────────────────────────────────────

var identRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func safeIdentifier(s string) bool {
	return identRe.MatchString(s)
}

// stripSQLComments removes -- line comments and /* */ block comments so that
// keyword checks don't false-match on commented-out text. String literals are
// left intact.
func stripSQLComments(s string) string {
	var b strings.Builder
	inSingle, inDouble, inLine, inBlock := false, false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inLine:
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
		case inBlock:
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				inBlock = false
				i++
			}
		case inSingle:
			b.WriteByte(c)
			if c == '\'' {
				inSingle = false
			}
		case inDouble:
			b.WriteByte(c)
			if c == '"' {
				inDouble = false
			}
		default:
			if c == '-' && i+1 < len(s) && s[i+1] == '-' {
				inLine = true
				i++
				continue
			}
			if c == '/' && i+1 < len(s) && s[i+1] == '*' {
				inBlock = true
				i++
				continue
			}
			if c == '\'' {
				inSingle = true
			}
			if c == '"' {
				inDouble = true
			}
			b.WriteByte(c)
		}
	}
	return b.String()
}

func findBlockedKeyword(sql string) string {
	upper := strings.ToUpper(sql)
	for _, kw := range blockedKeywords {
		re := regexp.MustCompile(`\b` + kw + `\b`)
		if re.MatchString(upper) {
			return kw
		}
	}
	return ""
}

func oidTypeName(conn *pgx.Conn, oid uint32) string {
	if dt, ok := conn.TypeMap().TypeForOID(oid); ok {
		return dt.Name
	}
	return fmt.Sprintf("oid=%d", oid)
}

// marshalValue normalises values for JSON encoding. pgx can return byte slices,
// times, numerics etc. that don't encode cleanly by default.
func marshalValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		return fmt.Sprintf("\\x%x", x)
	case time.Time:
		return x.Format(time.RFC3339Nano)
	default:
		return v
	}
}
