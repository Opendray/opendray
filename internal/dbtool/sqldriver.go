package dbtool

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Shared database/sql plumbing for the MySQL/MariaDB and SQLite drivers.
// PostgreSQL keeps its own pgx-based path (postgres.go); these two ride
// database/sql, so they share the handle type, row rendering and the
// read-only / affected-rows execution helpers here.

// sqlHandle wraps a database/sql pool as a dbtool Handle.
type sqlHandle struct{ db *sql.DB }

func (h *sqlHandle) Close() { _ = h.db.Close() }

func sqlDB(h Handle) (*sql.DB, error) {
	sh, ok := h.(*sqlHandle)
	if !ok || sh.db == nil {
		return nil, errors.New("dbtool: handle is not a database/sql pool")
	}
	return sh.db, nil
}

// binaryTypeNames are the column DatabaseTypeName()s whose []byte payload
// is genuinely binary and must be base64-encoded. Everything else that
// scans as []byte (VARCHAR/TEXT/JSON/DECIMAL/…) is UTF-8 text and is
// returned as a string — database/sql hands back []byte for text columns,
// so a blanket base64 (like pgx's bytea path) would corrupt normal text.
var binaryTypeNames = map[string]bool{
	"BLOB": true, "TINYBLOB": true, "MEDIUMBLOB": true, "LONGBLOB": true,
	"BINARY": true, "VARBINARY": true, "BYTEA": true,
}

// sqlCell renders one database/sql cell JSON-safe. []byte is text unless
// the column type is binary; other types defer to the shared jsonSafe.
func sqlCell(v any, binary bool) any {
	if b, ok := v.([]byte); ok {
		if binary {
			return base64.StdEncoding.EncodeToString(b)
		}
		return string(b)
	}
	return jsonSafe(v)
}

// sqlExecute runs one statement over database/sql.
//   - readOnly wraps it in a read-only transaction (the second fence behind
//     the classifier): MySQL issues START TRANSACTION READ ONLY, SQLite sets
//     PRAGMA query_only — both via database/sql's TxOptions.ReadOnly.
//   - wantRows collects a result set (SELECT, or INSERT … RETURNING); when
//     false the statement is a bare write and only RowsAffected/Command are
//     returned.
func sqlExecute(ctx context.Context, db *sql.DB, sqlText string, args []any, readOnly, wantRows bool, maxRows int, timeout time.Duration) (*ResultSet, error) {
	if maxRows <= 0 {
		maxRows = 500
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		return nil, fmt.Errorf("dbtool: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	start := time.Now()
	rs := &ResultSet{Columns: []ColumnMeta{}, Rows: [][]any{}}

	if !wantRows {
		res, err := tx.ExecContext(ctx, sqlText, args...)
		if err != nil {
			return nil, err
		}
		if n, err := res.RowsAffected(); err == nil {
			rs.RowsAffected = n
		}
		rs.Command = firstWord(sqlText)
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		rs.DurationMs = time.Since(start).Milliseconds()
		return rs, nil
	}

	rows, err := tx.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, err
	}
	colTypes, _ := rows.ColumnTypes()
	binary := make([]bool, len(cols))
	for i, name := range cols {
		typeName := ""
		if i < len(colTypes) && colTypes[i] != nil {
			typeName = strings.ToUpper(colTypes[i].DatabaseTypeName())
			binary[i] = binaryTypeNames[typeName]
		}
		rs.Columns = append(rs.Columns, ColumnMeta{Name: name, Type: typeName})
	}
	for rows.Next() {
		if len(rs.Rows) >= maxRows {
			rs.Truncated = true
			break
		}
		holders := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			return nil, fmt.Errorf("dbtool: read row: %w", err)
		}
		row := make([]any, len(cols))
		for i, v := range holders {
			row[i] = sqlCell(v, binary[i])
		}
		rs.Rows = append(rs.Rows, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rs.Command = firstWord(sqlText)
	rs.RowsAffected = int64(len(rs.Rows))
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	rs.DurationMs = time.Since(start).Milliseconds()
	return rs, nil
}

// sqlExecuteAffected runs a write statement and returns rows affected
// (row update/delete). Always a read-write transaction.
func sqlExecuteAffected(ctx context.Context, db *sql.DB, sqlText string, args []any, timeout time.Duration) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("dbtool: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, sqlText, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

// firstWord returns the upper-cased leading keyword of a statement (its
// command tag: INSERT / UPDATE / DELETE / SELECT …).
func firstWord(sqlText string) string {
	fields := strings.Fields(strings.TrimSpace(sqlText))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(fields[0])
}
