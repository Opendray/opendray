package dbtool

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Engine-agnostic SQL builders shared by every driver. Identifier quoting,
// bind-parameter spelling and the null-safe operator come from the passed
// Dialect; every value is a positional parameter and every identifier is
// quoted+escaped, so these builders never widen the injection surface.

// buildTableDataSQL renders the paged SELECT for req.
func buildTableDataSQL(req TableDataReq, d Dialect) (string, []any, error) {
	if req.Schema == "" || req.Table == "" {
		return "", nil, errors.New("dbtool: schema and table are required")
	}
	var sb strings.Builder
	var args []any
	sb.WriteString("SELECT * FROM ")
	sb.WriteString(d.QuoteIdent(req.Schema, req.Table))
	ops := d.FilterOps()
	for i, f := range req.Filters {
		op := strings.ToUpper(strings.TrimSpace(f.Op))
		takesValue, ok := ops[op]
		if !ok {
			return "", nil, fmt.Errorf("dbtool: unsupported filter operator %q", f.Op)
		}
		if f.Column == "" {
			return "", nil, errors.New("dbtool: filter column is required")
		}
		if i == 0 {
			sb.WriteString(" WHERE ")
		} else {
			sb.WriteString(" AND ")
		}
		sb.WriteString(d.QuoteIdent(f.Column))
		sb.WriteString(" ")
		sb.WriteString(op)
		if takesValue {
			args = append(args, f.Value)
			sb.WriteString(" ")
			sb.WriteString(d.Placeholder(len(args)))
		}
	}
	for i, s := range req.Sort {
		if s.Column == "" {
			return "", nil, errors.New("dbtool: sort column is required")
		}
		if i == 0 {
			sb.WriteString(" ORDER BY ")
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(d.QuoteIdent(s.Column))
		if s.Desc {
			sb.WriteString(" DESC")
		}
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit+1) // +1 row to detect truncation
	sb.WriteString(" LIMIT ")
	sb.WriteString(d.Placeholder(len(args)))
	if req.Offset > 0 {
		args = append(args, req.Offset)
		sb.WriteString(" OFFSET ")
		sb.WriteString(d.Placeholder(len(args)))
	}
	return sb.String(), args, nil
}

// buildInsertSQL renders INSERT … [RETURNING *]. Column order is made
// deterministic for testability. RETURNING is emitted only when the
// dialect supports it (MySQL 8 does not — its driver re-selects instead).
func buildInsertSQL(req RowInsertReq, d Dialect) (string, []any, error) {
	if req.Schema == "" || req.Table == "" {
		return "", nil, errors.New("dbtool: schema and table are required")
	}
	if len(req.Values) == 0 {
		return "", nil, errors.New("dbtool: insert requires at least one column value")
	}
	cols := sortedKeys(req.Values)
	var sb strings.Builder
	var args []any
	sb.WriteString("INSERT INTO ")
	sb.WriteString(d.QuoteIdent(req.Schema, req.Table))
	sb.WriteString(" (")
	for i, c := range cols {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(d.QuoteIdent(c))
	}
	sb.WriteString(") VALUES (")
	for i, c := range cols {
		if i > 0 {
			sb.WriteString(", ")
		}
		args = append(args, req.Values[c])
		sb.WriteString(d.Placeholder(len(args)))
	}
	sb.WriteString(")")
	if d.SupportsReturning() {
		sb.WriteString(" RETURNING *")
	}
	return sb.String(), args, nil
}

// buildUpdateSQL renders UPDATE … WHERE <full pk>. The service validates
// that req.PK covers the table's actual primary key before calling this.
func buildUpdateSQL(req RowUpdateReq, d Dialect) (string, []any, error) {
	if req.Schema == "" || req.Table == "" {
		return "", nil, errors.New("dbtool: schema and table are required")
	}
	if len(req.Values) == 0 {
		return "", nil, errors.New("dbtool: update requires at least one column value")
	}
	if len(req.PK) == 0 {
		return "", nil, errors.New("dbtool: update requires the row's primary key")
	}
	var sb strings.Builder
	var args []any
	sb.WriteString("UPDATE ")
	sb.WriteString(d.QuoteIdent(req.Schema, req.Table))
	sb.WriteString(" SET ")
	for i, c := range sortedKeys(req.Values) {
		if i > 0 {
			sb.WriteString(", ")
		}
		args = append(args, req.Values[c])
		sb.WriteString(d.QuoteIdent(c))
		sb.WriteString(" = ")
		sb.WriteString(d.Placeholder(len(args)))
	}
	sb.WriteString(" WHERE ")
	writePKWhere(&sb, &args, req.PK, d)
	return sb.String(), args, nil
}

// writePKWhere appends "pk1 <eq> $n AND pk2 <eq> $m" (NULL-safe equality
// via the dialect) for the pk map, in deterministic column order.
func writePKWhere(sb *strings.Builder, args *[]any, pk map[string]any, d Dialect) {
	for i, c := range sortedKeys(pk) {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		*args = append(*args, pk[c])
		sb.WriteString(d.QuoteIdent(c))
		sb.WriteString(" ")
		sb.WriteString(d.NullSafeEq())
		sb.WriteString(" ")
		sb.WriteString(d.Placeholder(len(*args)))
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
