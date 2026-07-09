package dbtool

import (
	"context"
	"strings"
	"testing"
	"time"
)

// These tests are the executable proof behind the CodeQL go/sql-injection
// findings on the identifier-concatenation sites (PRAGMA args can't be
// bound; generated SELECT/INSERT/UPDATE quote identifiers). They pin the
// escaping so a future refactor of QuoteIdent — or a new dialect — can't
// silently reintroduce a real injection at those sinks.

// Injection-safety property for a quoted identifier: after stripping the
// outer quotes, every inner quote char must belong to a doubled ("") pair —
// no lone quote survives, so an attacker's quote can never terminate the
// identifier early and start a second statement. (Byte-for-byte round-trip
// is stronger, but pgx.Identifier strips NUL bytes — still safe, since a
// removed byte can't inject; this checks the property that actually matters.)
func TestQuoteIdentNoBreakout(t *testing.T) {
	payloads := []string{
		`t") ; DROP TABLE secret; --`,
		`a"b"c`,
		"back`tick",
		`); DROP TABLE x;--`,
		"line\nbreak",
		"tab\there",
		"nul\x00byte",
		`" OR 1=1 --`,
		`"" UNION SELECT * FROM secret --`,
		"plain",
		"",
	}
	check := func(name, quoted string, q byte) {
		if len(quoted) < 2 || quoted[0] != q || quoted[len(quoted)-1] != q {
			t.Errorf("%s: %q not wrapped in %c", name, quoted, q)
			return
		}
		inner := quoted[1 : len(quoted)-1]
		stripped := strings.ReplaceAll(inner, string([]byte{q, q}), "")
		if strings.IndexByte(stripped, q) >= 0 {
			t.Errorf("%s: %q has an unpaired %c — identifier breakout possible", name, quoted, q)
		}
	}
	for _, p := range payloads {
		check("sqlite", sqliteDialect{}.QuoteIdent(p), '"')
		check("pg", pgDialect{}.QuoteIdent(p), '"')
		check("mysql", mysqlDialect{}.QuoteIdent(p), '`')
	}
}

// A hostile filter operator must be rejected by the whitelist, never
// concatenated into SQL (operators are the one non-identifier,
// non-parameter token in buildTableDataSQL).
func TestFilterOperatorWhitelistRejectsInjection(t *testing.T) {
	for _, d := range []Dialect{pgDialect{}, mysqlDialect{}, sqliteDialect{}} {
		_, _, err := buildTableDataSQL(TableDataReq{
			Schema: "s", Table: "t",
			Filters: []Filter{{Column: "a", Op: "= 1 OR 1=1 --", Value: 1}},
		}, d)
		if err == nil {
			t.Errorf("%T: injection via operator was not rejected", d)
		}
	}
}

// Live proof: firing crafted identifiers at the real PRAGMA path
// (TableMeta) and the generated-INSERT path must not execute a smuggled
// second statement — a target table survives untouched.
func TestSQLiteInjectionResistanceLive(t *testing.T) {
	dir := t.TempDir()
	c := Connection{Driver: "sqlite", Cwd: dir, DBName: "t.db"}
	d := sqliteDriver{}
	ctx := context.Background()
	to := 10 * time.Second

	h, err := d.Open(ctx, c, DriverOpts{})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	mustExec := func(sql string, class StatementClass) {
		if _, err := d.Query(ctx, h, QueryReq{SQL: sql}, class, 100, to); err != nil {
			t.Fatalf("setup %q: %v", sql, err)
		}
	}
	mustExec(`CREATE TABLE secret (id INTEGER PRIMARY KEY)`, ClassDDL)
	mustExec(`INSERT INTO secret VALUES (1)`, ClassWrite)
	mustExec(`CREATE TABLE t (a INTEGER)`, ClassDDL)

	evil := `secret"); DROP TABLE secret; --`
	// Hostile table name down the PRAGMA path — errors or empties, never drops.
	_, _ = d.TableMeta(ctx, h, "main", evil, to)
	// Hostile column name down the generated-INSERT path.
	_, _ = d.InsertRow(ctx, h, RowInsertReq{
		Schema: "main", Table: "t",
		Values: map[string]any{`a"); DROP TABLE secret; --`: 1},
	}, to)

	// secret must still exist with its row.
	rs, err := d.Query(ctx, h, QueryReq{SQL: `SELECT count(*) FROM secret`}, ClassRead, 100, to)
	if err != nil {
		t.Fatalf("secret table was dropped by an injection: %v", err)
	}
	if len(rs.Rows) != 1 {
		t.Fatalf("unexpected result shape: %#v", rs.Rows)
	}
}
