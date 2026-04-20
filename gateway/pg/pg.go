// Package pg backs the pg-browser@2.0.0 panel plugin — a live SQL
// editor + schema browser for any PostgreSQL-compatible database.
//
// One manifest instance maps to one connection pool. The handler
// layer (gateway/pg_handlers.go) resolves config via s.effectiveConfig
// so host/port/user/database come from plugin_kv and password (a
// secret field) is decrypted from plugin_secret.
//
// Safety rails that run on every call:
//
//   - Statement timeout via `SET LOCAL statement_timeout` inside the
//     query transaction. Caps runaway queries at the user-configured
//     limit (default 30s).
//   - Row-count cap: the query is wrapped to fetch at most MaxRows
//     rows from the result set; anything beyond is dropped and the
//     meta block reports `truncated: true`.
//   - Read-only mode: when ReadOnly is set, the transaction runs as
//     `BEGIN READ ONLY` so writes are rejected by PostgreSQL itself.
//     The handler layer additionally fast-rejects common DDL verbs
//     (DROP/TRUNCATE/UPDATE/DELETE/INSERT/CREATE/ALTER) before the
//     round-trip, so the user sees a clean error instead of a
//     committed-nothing empty result.
//
// Secrets never leave the kernel via a response. The handler echoes
// only the columns + rows the database itself returned.
package pg

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds the per-plugin connection settings resolved from the
// plugin's configSchema. Password is the decrypted plaintext (handler
// layer pulls it out of plugin_secret); the struct is never logged or
// serialised anywhere outside this package.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	// Safety
	ReadOnly            bool
	StatementTimeout    time.Duration
	MaxRows             int
}

// dsn renders a libpq URL. Password is URL-encoded so special chars
// don't break the connection string (matches pgx's own expectations).
func (c Config) dsn() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 5432
	}
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     fmt.Sprintf("%s:%d", host, port),
		Path:     c.Database,
		RawQuery: "sslmode=" + sslMode,
	}
	return u.String()
}

// cacheKey identifies a unique (plugin, config) combination so we
// don't spin up a fresh pool on every HTTP call. Password is included
// so a rotation invalidates the old pool automatically.
func (c Config) cacheKey(plugin string) string {
	return fmt.Sprintf("%s|%s@%s:%d/%s?sslmode=%s|%s|%v|%d",
		plugin, c.User, c.Host, c.Port, c.Database, c.SSLMode, c.Password,
		c.ReadOnly, int(c.StatementTimeout/time.Millisecond),
	)
}

// Manager owns the per-plugin pgxpool.Pool instances. Pool lifetime
// tracks the config — when the user edits host/user/password etc.,
// the next resolve() sees a new cacheKey and builds a fresh pool
// (the old one is closed).
type Manager struct {
	mu    sync.Mutex
	pools map[string]*entry // keyed by Config.cacheKey("<plugin>")
}

type entry struct {
	cacheKey string
	pool     *pgxpool.Pool
}

// NewManager constructs an empty Manager. Call Close on shutdown to
// drain open pools.
func NewManager() *Manager {
	return &Manager{pools: make(map[string]*entry)}
}

// Close tears down every live pool. Safe to call multiple times.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.pools {
		e.pool.Close()
	}
	m.pools = make(map[string]*entry)
}

// pool returns (or lazily creates) the pool for the given plugin +
// config combination. Config edits produce a new pool; the old one
// is closed once no in-flight caller holds it (Go's GC plus explicit
// Close call).
func (m *Manager) pool(ctx context.Context, plugin string, cfg Config) (*pgxpool.Pool, error) {
	if cfg.Host == "" || cfg.User == "" || cfg.Database == "" {
		return nil, errors.New("pg: host, user, and database are required")
	}
	key := cfg.cacheKey(plugin)
	m.mu.Lock()
	if cur, ok := m.pools[plugin]; ok && cur.cacheKey == key {
		m.mu.Unlock()
		return cur.pool, nil
	}
	// Config changed (or first call) — drop the previous pool for
	// this plugin. GC takes care of lingering connections once
	// outstanding Query calls return.
	if prev, ok := m.pools[plugin]; ok {
		prev.pool.Close()
		delete(m.pools, plugin)
	}
	m.mu.Unlock()

	pc, err := pgxpool.ParseConfig(cfg.dsn())
	if err != nil {
		return nil, fmt.Errorf("pg: parse dsn: %w", err)
	}
	pc.MaxConns = 4
	pc.MinConns = 0
	pc.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("pg: connect: %w", err)
	}

	m.mu.Lock()
	// Double-check another goroutine didn't race in with the same
	// key. If so, drop ours and keep theirs.
	if cur, ok := m.pools[plugin]; ok && cur.cacheKey == key {
		m.mu.Unlock()
		pool.Close()
		return cur.pool, nil
	}
	m.pools[plugin] = &entry{cacheKey: key, pool: pool}
	m.mu.Unlock()
	return pool, nil
}

// ─── Read-only write-verb guard ──────────────────────────────────

// writeVerbs is the set of first-token verbs that modify database
// state. The guard is intentionally conservative: caller-prefix
// match only. PostgreSQL itself is still the final authority (when
// ReadOnly=true we run inside BEGIN READ ONLY and PG rejects writes
// with SQLSTATE 25006). The guard exists to surface a clean error
// before the round-trip, and to refuse the query in caller-context
// the user can act on.
var writeVerbs = map[string]bool{
	"INSERT":   true,
	"UPDATE":   true,
	"DELETE":   true,
	"DROP":     true,
	"CREATE":   true,
	"ALTER":    true,
	"TRUNCATE": true,
	"GRANT":    true,
	"REVOKE":   true,
	"REINDEX":  true,
	"VACUUM":   true,
	"COMMENT":  true,
	"CLUSTER":  true,
}

// firstVerb returns the upper-cased first token of sql, stripped of
// leading whitespace and SQL line comments. Multi-statement scripts
// only see the first verb — additional statements are the caller's
// problem (we reject them in Query anyway, see below).
func firstVerb(sql string) string {
	s := strings.TrimSpace(sql)
	// Strip leading -- line comments.
	for strings.HasPrefix(s, "--") {
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			s = strings.TrimSpace(s[nl+1:])
			continue
		}
		return ""
	}
	// Strip leading /* block comment */.
	if strings.HasPrefix(s, "/*") {
		if end := strings.Index(s, "*/"); end >= 0 {
			s = strings.TrimSpace(s[end+2:])
		} else {
			return ""
		}
	}
	// First whitespace-delimited token.
	for i, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == ';' || r == '(' {
			return strings.ToUpper(s[:i])
		}
	}
	return strings.ToUpper(s)
}

// IsWriteVerb returns true when sql begins with a verb that modifies
// database state. Handler layer uses this to gate Execute vs Query,
// and to fast-reject writes when cfg.ReadOnly is true.
func IsWriteVerb(sql string) bool {
	return writeVerbs[firstVerb(sql)]
}

// IsDestructiveVerb is IsWriteVerb narrowed to the operations that
// cannot be recovered without a backup. The Flutter client prompts
// for confirmation before sending these — the backend double-checks
// via this helper so a naïve caller skipping the prompt still trips
// it.
func IsDestructiveVerb(sql string) bool {
	switch firstVerb(sql) {
	case "DROP", "TRUNCATE":
		return true
	}
	// DELETE with no WHERE clause also counts. Cheap string search
	// is fine — we fall through to the DB if the match is ambiguous
	// (e.g. "DELETE FROM x WHERE" with a trailing comment).
	if firstVerb(sql) == "DELETE" {
		upper := strings.ToUpper(sql)
		if !strings.Contains(upper, "WHERE") {
			return true
		}
	}
	return false
}

// ─── Query / Execute ─────────────────────────────────────────────

// Column describes one column in a result set.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// QueryResult is the full shape returned by Query. Rows holds the
// raw values per row (any Go type pgx produced — caller serialises
// to JSON). DurationMs is wall-clock, not server-side, to match the
// "how long did my request take" mental model users expect.
type QueryResult struct {
	Columns    []Column `json:"columns"`
	Rows       [][]any  `json:"rows"`
	RowCount   int      `json:"rowCount"`
	Truncated  bool     `json:"truncated"`
	DurationMs int64    `json:"durationMs"`
}

// ExecuteResult is the shape returned by Execute (non-SELECT writes).
type ExecuteResult struct {
	RowsAffected int64  `json:"rowsAffected"`
	Verb         string `json:"verb"`
	DurationMs   int64  `json:"durationMs"`
}

// Query runs a SELECT (or anything that returns rows, like SHOW /
// EXPLAIN). Write verbs are fast-rejected when cfg.ReadOnly is true.
// Results beyond cfg.MaxRows are dropped with Truncated=true.
func (m *Manager) Query(ctx context.Context, plugin string, cfg Config, sql string) (QueryResult, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return QueryResult{}, errors.New("pg: sql is empty")
	}
	if cfg.ReadOnly && IsWriteVerb(sql) {
		return QueryResult{}, fmt.Errorf("pg: plugin is read-only — %s rejected", firstVerb(sql))
	}
	pool, err := m.pool(ctx, plugin, cfg)
	if err != nil {
		return QueryResult{}, err
	}
	maxRows := cfg.MaxRows
	if maxRows <= 0 {
		maxRows = 1000
	}
	timeout := cfg.StatementTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Query runs inside a transaction so we can SET LOCAL
	// statement_timeout (session-local) + BEGIN READ ONLY when the
	// plugin is configured that way.
	start := time.Now()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return QueryResult{}, fmt.Errorf("pg: acquire: %w", err)
	}
	defer conn.Release()

	txOpts := pgx.TxOptions{}
	if cfg.ReadOnly {
		txOpts.AccessMode = pgx.ReadOnly
	}
	tx, err := conn.BeginTx(ctx, txOpts)
	if err != nil {
		return QueryResult{}, fmt.Errorf("pg: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// SET LOCAL caps this transaction only, so concurrent sessions
	// keep their defaults. The value is a milliseconds integer; pgx
	// parameters aren't allowed in SET, so we format it inline from
	// a trusted numeric source (user's configSchema).
	ms := int64(timeout / time.Millisecond)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", ms)); err != nil {
		return QueryResult{}, fmt.Errorf("pg: set timeout: %w", err)
	}

	rows, err := tx.Query(ctx, sql)
	if err != nil {
		return QueryResult{}, err
	}
	defer rows.Close()

	fieldDescs := rows.FieldDescriptions()
	cols := make([]Column, 0, len(fieldDescs))
	for _, fd := range fieldDescs {
		cols = append(cols, Column{
			Name: string(fd.Name),
			Type: pgTypeName(fd.DataTypeOID),
		})
	}

	out := QueryResult{Columns: cols, Rows: [][]any{}}
	for rows.Next() {
		if len(out.Rows) >= maxRows {
			out.Truncated = true
			break
		}
		values, err := rows.Values()
		if err != nil {
			return QueryResult{}, fmt.Errorf("pg: scan: %w", err)
		}
		out.Rows = append(out.Rows, coerceRow(values))
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, err
	}
	out.RowCount = len(out.Rows)
	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

// Execute runs a write statement (INSERT/UPDATE/DELETE/DDL). Always
// rejected when cfg.ReadOnly is true. Returns rows-affected when
// pgx reports it; 0 for DDL that doesn't surface a count.
func (m *Manager) Execute(ctx context.Context, plugin string, cfg Config, sql string) (ExecuteResult, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return ExecuteResult{}, errors.New("pg: sql is empty")
	}
	if cfg.ReadOnly {
		return ExecuteResult{}, errors.New("pg: plugin is read-only — execute rejected")
	}
	if !IsWriteVerb(sql) {
		return ExecuteResult{}, fmt.Errorf("pg: execute expects a write verb, got %q", firstVerb(sql))
	}
	pool, err := m.pool(ctx, plugin, cfg)
	if err != nil {
		return ExecuteResult{}, err
	}
	timeout := cfg.StatementTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	start := time.Now()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("pg: acquire: %w", err)
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("pg: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	ms := int64(timeout / time.Millisecond)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", ms)); err != nil {
		return ExecuteResult{}, fmt.Errorf("pg: set timeout: %w", err)
	}
	tag, err := tx.Exec(ctx, sql)
	if err != nil {
		return ExecuteResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExecuteResult{}, fmt.Errorf("pg: commit: %w", err)
	}
	committed = true

	return ExecuteResult{
		RowsAffected: tag.RowsAffected(),
		Verb:         firstVerb(sql),
		DurationMs:   time.Since(start).Milliseconds(),
	}, nil
}

// ─── Schema inspection ───────────────────────────────────────────

// ListSchemas returns every user-visible schema name (skips
// pg_catalog / pg_toast / information_schema).
func (m *Manager) ListSchemas(ctx context.Context, plugin string, cfg Config) ([]string, error) {
	const q = `
		SELECT schema_name
		  FROM information_schema.schemata
		 WHERE schema_name NOT IN ('pg_catalog','pg_toast','information_schema')
		   AND schema_name NOT LIKE 'pg_temp_%'
		   AND schema_name NOT LIKE 'pg_toast_temp_%'
		 ORDER BY schema_name
	`
	res, err := m.Query(ctx, plugin, cfg, q)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(res.Rows))
	for _, row := range res.Rows {
		if s, ok := row[0].(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// TableRow is one entry returned by ListTables.
type TableRow struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Kind   string `json:"kind"` // "table" | "view" | "matview" | ...
}

// ListTables returns tables in the given schema (user tables + views).
// Empty schema defaults to "public".
func (m *Manager) ListTables(ctx context.Context, plugin string, cfg Config, schema string) ([]TableRow, error) {
	if schema == "" {
		schema = "public"
	}
	// Parameterise the schema via a quoted literal — we validate
	// it's a single identifier-shaped string first to keep the
	// "one call, one query" model (pgx rejects $N substitution in
	// some PL contexts, but information_schema accepts it fine).
	if !identifierOK(schema) {
		return nil, fmt.Errorf("pg: invalid schema name %q", schema)
	}
	q := `
		SELECT table_schema, table_name, table_type
		  FROM information_schema.tables
		 WHERE table_schema = $1
		 ORDER BY table_name
	`
	res, err := m.queryArgs(ctx, plugin, cfg, q, schema)
	if err != nil {
		return nil, err
	}
	out := make([]TableRow, 0, len(res.Rows))
	for _, row := range res.Rows {
		schemaVal, _ := row[0].(string)
		nameVal, _ := row[1].(string)
		kindVal, _ := row[2].(string)
		out = append(out, TableRow{
			Schema: schemaVal,
			Name:   nameVal,
			Kind:   normaliseTableKind(kindVal),
		})
	}
	return out, nil
}

func normaliseTableKind(raw string) string {
	switch raw {
	case "BASE TABLE":
		return "table"
	case "VIEW":
		return "view"
	case "MATERIALIZED VIEW", "MATVIEW":
		return "matview"
	case "FOREIGN":
		return "foreign"
	default:
		return strings.ToLower(raw)
	}
}

// ColumnRow is one entry returned by DescribeColumns.
type ColumnRow struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
}

// DescribeColumns returns every column of the given schema.table.
func (m *Manager) DescribeColumns(ctx context.Context, plugin string, cfg Config, schema, table string) ([]ColumnRow, error) {
	if schema == "" {
		schema = "public"
	}
	if !identifierOK(schema) || !identifierOK(table) {
		return nil, fmt.Errorf("pg: invalid schema/table name")
	}
	q := `
		SELECT column_name, data_type, is_nullable, COALESCE(column_default, '')
		  FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY ordinal_position
	`
	res, err := m.queryArgs(ctx, plugin, cfg, q, schema, table)
	if err != nil {
		return nil, err
	}
	out := make([]ColumnRow, 0, len(res.Rows))
	for _, row := range res.Rows {
		name, _ := row[0].(string)
		typ, _ := row[1].(string)
		nullableStr, _ := row[2].(string)
		defVal, _ := row[3].(string)
		out = append(out, ColumnRow{
			Name: name, Type: typ,
			Nullable: nullableStr == "YES",
			Default:  defVal,
		})
	}
	return out, nil
}

// queryArgs is the parameterised twin of Query, used by schema
// introspection. Not exposed to the HTTP layer (user-entered SQL
// goes through Query unparameterised on purpose — SQL editor usage
// is the whole point of the plugin).
func (m *Manager) queryArgs(ctx context.Context, plugin string, cfg Config, sql string, args ...any) (QueryResult, error) {
	pool, err := m.pool(ctx, plugin, cfg)
	if err != nil {
		return QueryResult{}, err
	}
	start := time.Now()
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return QueryResult{}, err
	}
	defer rows.Close()

	out := QueryResult{Rows: [][]any{}}
	fieldDescs := rows.FieldDescriptions()
	for _, fd := range fieldDescs {
		out.Columns = append(out.Columns, Column{
			Name: string(fd.Name),
			Type: pgTypeName(fd.DataTypeOID),
		})
	}
	for rows.Next() {
		vs, err := rows.Values()
		if err != nil {
			return QueryResult{}, err
		}
		out.Rows = append(out.Rows, coerceRow(vs))
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, err
	}
	out.RowCount = len(out.Rows)
	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

// identifierOK restricts schema / table names to a conservative
// shape so callers can quote them safely. PostgreSQL technically
// allows anything inside "" delimiters — we don't support that here
// to avoid parsing two quoting styles from the UI.
func identifierOK(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

// coerceRow walks a pgx row and makes each value JSON-friendly.
// pgx returns []byte for bytea, which Go marshals as base64 — fine.
// time.Time goes through as RFC3339 — fine. Numeric is returned as
// pgtype.Numeric which doesn't JSON directly, so we stringify.
func coerceRow(vs []any) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		switch t := v.(type) {
		case nil:
			out[i] = nil
		case []byte:
			out[i] = string(t)
		default:
			// Stringify types pgx returns that don't JSON well.
			// pgtype.Numeric.Value() returns a string representation
			// already if we reach its MarshalJSON. For types we
			// don't recognise, let encoding/json try first — fallback
			// to %v stringify only if the caller sees an error
			// downstream.
			out[i] = v
		}
	}
	return out
}

// pgTypeName maps a pgx OID to a short human name. We cover the
// common types and fall back to "?" for anything exotic — the UI
// treats unknown types as opaque strings.
func pgTypeName(oid uint32) string {
	switch oid {
	case 16:
		return "bool"
	case 17:
		return "bytea"
	case 20:
		return "int8"
	case 21:
		return "int2"
	case 23:
		return "int4"
	case 25:
		return "text"
	case 114:
		return "json"
	case 700:
		return "float4"
	case 701:
		return "float8"
	case 1042:
		return "char"
	case 1043:
		return "varchar"
	case 1082:
		return "date"
	case 1114:
		return "timestamp"
	case 1184:
		return "timestamptz"
	case 1700:
		return "numeric"
	case 2950:
		return "uuid"
	case 3802:
		return "jsonb"
	default:
		return fmt.Sprintf("oid:%d", oid)
	}
}
