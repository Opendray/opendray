// Package dbtool is the Database tool: per-project (cwd-keyed) external
// database connections with schema introspection, table data browsing,
// row-level CRUD and a SQL console. It is exposed over REST (dual-auth,
// db:read / db:write scopes) and re-exported to agent sessions as the
// opendray-dbtool MCP server.
//
// The package deliberately opens its own connections to the *target*
// databases — it never touches opendray's own store pool except through
// its store for connection metadata.
package dbtool

import (
	"context"
	"time"
)

// Connection is one registered external database. Password holds the
// decrypted secret in memory only — it is never serialized (the API
// exposes HasPassword instead).
type Connection struct {
	ID        string         `json:"id"`
	Cwd       string         `json:"cwd"`
	Name      string         `json:"name"`
	Driver    string         `json:"driver"`
	Host      string         `json:"host"`
	Port      int            `json:"port"`
	DBName    string         `json:"db_name"`
	Username  string         `json:"username"`
	Password  string         `json:"-"`
	SSLMode   string         `json:"ssl_mode"`
	ReadOnly  bool           `json:"read_only"`
	Options   map[string]any `json:"options"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// HasPassword tells the UI a secret is stored without revealing it.
	HasPassword bool `json:"has_password"`
}

// DriverOpts carries the per-connection runtime knobs a driver needs
// when opening handles or pinging.
type DriverOpts struct {
	MaxConns       int
	ConnectTimeout time.Duration
}

// PingResult is what a connectivity test reports back to the UI.
// IsSuperuser powers the "you are connected as a superuser" warning
// (operator rule: never run project work as the PG superuser).
type PingResult struct {
	OK            bool   `json:"ok"`
	ServerVersion string `json:"server_version,omitempty"`
	IsSuperuser   bool   `json:"is_superuser"`
	LatencyMs     int64  `json:"latency_ms"`
	Error         string `json:"error,omitempty"`
}

// Schema is one namespace in the target database.
type Schema struct {
	Name string `json:"name"`
}

// Table is one relation inside a schema. Kind is "table", "view" or
// "foreign"; RowEstimate comes from planner statistics (cheap, inexact).
type Table struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	RowEstimate int64  `json:"row_estimate"`
}

// Column describes one column of a table.
type Column struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
	Position int    `json:"position"`
}

// Index describes one index on a table.
type Index struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
	Unique     bool   `json:"unique"`
	Primary    bool   `json:"primary"`
}

// ForeignKey describes one outgoing FK constraint.
type ForeignKey struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	RefSchema  string   `json:"ref_schema"`
	RefTable   string   `json:"ref_table"`
	RefColumns []string `json:"ref_columns"`
}

// TableMeta is the full structural description of one table — what the
// data grid needs to render, edit and key rows.
type TableMeta struct {
	Schema      string       `json:"schema"`
	Table       string       `json:"table"`
	Columns     []Column     `json:"columns"`
	PrimaryKey  []string     `json:"primary_key"`
	Indexes     []Index      `json:"indexes"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
}

// Filter is one ANDed predicate on a table-data request. Op must be one
// of the whitelisted operators (see filterOps in postgres.go).
type Filter struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  any    `json:"value,omitempty"`
}

// Sort is one ORDER BY term.
type Sort struct {
	Column string `json:"column"`
	Desc   bool   `json:"desc"`
}

// TableDataReq pages through one table.
type TableDataReq struct {
	Schema  string   `json:"schema"`
	Table   string   `json:"table"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
	Sort    []Sort   `json:"sort"`
	Filters []Filter `json:"filters"`
}

// RowInsertReq inserts one row.
type RowInsertReq struct {
	Schema string         `json:"schema"`
	Table  string         `json:"table"`
	Values map[string]any `json:"values"`
}

// RowUpdateReq updates one row addressed by its full primary key.
type RowUpdateReq struct {
	Schema string         `json:"schema"`
	Table  string         `json:"table"`
	PK     map[string]any `json:"pk"`
	Values map[string]any `json:"values"`
}

// RowDeleteReq deletes rows, each addressed by its full primary key.
type RowDeleteReq struct {
	Schema string           `json:"schema"`
	Table  string           `json:"table"`
	PKs    []map[string]any `json:"pks"`
}

// QueryReq runs one SQL statement from the console.
type QueryReq struct {
	SQL     string `json:"sql"`
	MaxRows int    `json:"max_rows,omitempty"`
}

// ColumnMeta names one result column and its type.
type ColumnMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ResultSet is the JSON-safe result of any execution: cell values are
// rendered to JSON-encodable Go types (bytea → base64, timestamps →
// RFC 3339, exotic types → their text form).
type ResultSet struct {
	Columns      []ColumnMeta `json:"columns"`
	Rows         [][]any      `json:"rows"`
	RowsAffected int64        `json:"rows_affected"`
	Command      string       `json:"command,omitempty"`
	Truncated    bool         `json:"truncated"`
	DurationMs   int64        `json:"duration_ms"`
}

// StatementClass is the coarse classification the read-only gate keys on.
type StatementClass string

const (
	ClassRead  StatementClass = "read"
	ClassWrite StatementClass = "write"
	ClassDDL   StatementClass = "ddl"
)

// Handle is a driver-owned open connection (a *pgxpool.Pool for
// postgres). The service caches handles per connection id and closes
// them on idle-eviction, update, delete and shutdown.
type Handle interface {
	Close()
}

// Driver abstracts one database engine. v1 ships postgres only; MySQL /
// SQLite implement this same interface later without touching the
// service, handler or MCP layers.
//
// Read-only enforcement contract: TableData and read-class Query MUST run
// inside a read-only transaction with a statement timeout, so statements
// the classifier can't see through (writing CTEs, DO blocks) fail
// server-side rather than mutating through a "read" path.
type Driver interface {
	Ping(ctx context.Context, c Connection, opts DriverOpts) PingResult
	Open(ctx context.Context, c Connection, opts DriverOpts) (Handle, error)
	Schemas(ctx context.Context, h Handle, timeout time.Duration) ([]Schema, error)
	Tables(ctx context.Context, h Handle, schema string, timeout time.Duration) ([]Table, error)
	TableMeta(ctx context.Context, h Handle, schema, table string, timeout time.Duration) (TableMeta, error)
	TableData(ctx context.Context, h Handle, req TableDataReq, timeout time.Duration) (*ResultSet, error)
	InsertRow(ctx context.Context, h Handle, req RowInsertReq, timeout time.Duration) (*ResultSet, error)
	UpdateRow(ctx context.Context, h Handle, req RowUpdateReq, timeout time.Duration) (int64, error)
	DeleteRows(ctx context.Context, h Handle, req RowDeleteReq, timeout time.Duration) (int64, error)
	Query(ctx context.Context, h Handle, req QueryReq, class StatementClass, maxRows int, timeout time.Duration) (*ResultSet, error)
}
