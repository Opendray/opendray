package dbtool

import (
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Dialect captures the per-engine SQL surface the shared statement
// builders (sqlbuild.go) need: how identifiers are quoted, how bind
// parameters are spelled, the NULL-safe equality operator, whether
// INSERT … RETURNING is available, and which filter operators are legal.
//
// The builders are engine-agnostic; only these five knobs differ between
// PostgreSQL, MySQL/MariaDB and SQLite. Values are always positional
// parameters and identifiers always quoted+escaped, so a Dialect never
// widens the injection surface — it only changes the spelling.
type Dialect interface {
	// QuoteIdent quotes and escapes a (possibly dotted) identifier, e.g.
	// schema+table → "public"."users" (pg/sqlite) or `db`.`users` (mysql).
	QuoteIdent(parts ...string) string
	// Placeholder renders the n-th (1-based) bind parameter: "$n" for
	// postgres, "?" for mysql/sqlite (n ignored).
	Placeholder(n int) string
	// NullSafeEq is the operator that treats NULL = NULL as true, used for
	// primary-key WHERE clauses so a NULL PK column still matches its row.
	NullSafeEq() string
	// SupportsReturning reports whether INSERT … RETURNING * works.
	SupportsReturning() bool
	// FilterOps whitelists table-data filter operators → whether each takes
	// a value parameter.
	FilterOps() map[string]bool
}

// pgFilterOps is the PostgreSQL filter whitelist — includes ILIKE.
var pgFilterOps = map[string]bool{
	"=": true, "!=": true, "<>": true, "<": true, ">": true, "<=": true, ">=": true,
	"LIKE": true, "ILIKE": true, "NOT LIKE": true, "NOT ILIKE": true,
	"IS NULL": false, "IS NOT NULL": false,
}

// sqlFilterOps is the MySQL/SQLite whitelist — no ILIKE (neither engine
// has it; MySQL LIKE is case-insensitive by collation, SQLite LIKE is
// case-insensitive for ASCII).
var sqlFilterOps = map[string]bool{
	"=": true, "!=": true, "<>": true, "<": true, ">": true, "<=": true, ">=": true,
	"LIKE": true, "NOT LIKE": true,
	"IS NULL": false, "IS NOT NULL": false,
}

// pgDialect — PostgreSQL. QuoteIdent delegates to pgx.Identifier.Sanitize
// so quoting stays byte-identical to the original hand-rolled builders.
type pgDialect struct{}

func (pgDialect) QuoteIdent(parts ...string) string { return pgx.Identifier(parts).Sanitize() }
func (pgDialect) Placeholder(n int) string          { return "$" + strconv.Itoa(n) }
func (pgDialect) NullSafeEq() string                { return "IS NOT DISTINCT FROM" }
func (pgDialect) SupportsReturning() bool           { return true }
func (pgDialect) FilterOps() map[string]bool        { return pgFilterOps }

// mysqlDialect — MySQL and MariaDB. Backtick identifiers, "?" params,
// "<=>" null-safe equality. MySQL 8 has no RETURNING; MariaDB 10.5+ does,
// so `returning` is set per connection.
type mysqlDialect struct{ returning bool }

func (mysqlDialect) QuoteIdent(parts ...string) string { return quoteJoin(parts, '`') }
func (mysqlDialect) Placeholder(int) string            { return "?" }
func (mysqlDialect) NullSafeEq() string                { return "<=>" }
func (d mysqlDialect) SupportsReturning() bool         { return d.returning }
func (mysqlDialect) FilterOps() map[string]bool        { return sqlFilterOps }

// sqliteDialect — SQLite. Standard double-quoted identifiers, "?" params,
// "IS" null-safe equality, RETURNING (3.35+, satisfied by modernc v1.47).
type sqliteDialect struct{}

func (sqliteDialect) QuoteIdent(parts ...string) string { return quoteJoin(parts, '"') }
func (sqliteDialect) Placeholder(int) string            { return "?" }
func (sqliteDialect) NullSafeEq() string                { return "IS" }
func (sqliteDialect) SupportsReturning() bool           { return true }
func (sqliteDialect) FilterOps() map[string]bool        { return sqlFilterOps }

// quoteJoin quotes each part with q, escaping an embedded quote by
// doubling it, and joins with ".". This is the standard SQL identifier
// escape for both backtick (MySQL) and double-quote (SQLite) styles.
func quoteJoin(parts []string, q byte) string {
	esc := string(q) + string(q)
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = string(q) + strings.ReplaceAll(p, string(q), esc) + string(q)
	}
	return strings.Join(out, ".")
}
