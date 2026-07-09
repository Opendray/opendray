package dbtool

import "testing"

func TestDialects(t *testing.T) {
	tests := []struct {
		name      string
		d         Dialect
		wantQuote string // QuoteIdent("a", "b")
		wantPlace string // Placeholder(3)
		wantNull  string
		wantRet   bool
	}{
		{"postgres", pgDialect{}, `"a"."b"`, "$3", "IS NOT DISTINCT FROM", true},
		{"mysql", mysqlDialect{returning: false}, "`a`.`b`", "?", "<=>", false},
		{"mariadb", mysqlDialect{returning: true}, "`a`.`b`", "?", "<=>", true},
		{"sqlite", sqliteDialect{}, `"a"."b"`, "?", "IS", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.QuoteIdent("a", "b"); got != tt.wantQuote {
				t.Errorf("QuoteIdent = %q, want %q", got, tt.wantQuote)
			}
			if got := tt.d.Placeholder(3); got != tt.wantPlace {
				t.Errorf("Placeholder(3) = %q, want %q", got, tt.wantPlace)
			}
			if got := tt.d.NullSafeEq(); got != tt.wantNull {
				t.Errorf("NullSafeEq = %q, want %q", got, tt.wantNull)
			}
			if got := tt.d.SupportsReturning(); got != tt.wantRet {
				t.Errorf("SupportsReturning = %v, want %v", got, tt.wantRet)
			}
		})
	}
}

// Identifier quoting must escape the quote char by doubling it — the
// injection barrier for every generated statement.
func TestDialectQuoteEscaping(t *testing.T) {
	if got := (mysqlDialect{}).QuoteIdent("a`b"); got != "`a``b`" {
		t.Errorf("mysql escape = %q", got)
	}
	if got := (sqliteDialect{}).QuoteIdent(`a"b`); got != `"a""b"` {
		t.Errorf("sqlite escape = %q", got)
	}
	if got := (pgDialect{}).QuoteIdent(`a"b`); got != `"a""b"` {
		t.Errorf("pg escape = %q", got)
	}
}

// Filter operators differ: ILIKE is PostgreSQL-only.
func TestDialectFilterOps(t *testing.T) {
	if !(pgDialect{}).FilterOps()["ILIKE"] {
		t.Error("pg should allow ILIKE")
	}
	if _, ok := (mysqlDialect{}).FilterOps()["ILIKE"]; ok {
		t.Error("mysql must not allow ILIKE")
	}
	if _, ok := (sqliteDialect{}).FilterOps()["ILIKE"]; ok {
		t.Error("sqlite must not allow ILIKE")
	}
}

// The shared builders must honour a non-postgres dialect: "?" params,
// engine-specific quoting and null-safe operator, and RETURNING only when
// supported.
func TestBuildWithMySQLDialect(t *testing.T) {
	d := mysqlDialect{returning: false}
	sql, args, err := buildTableDataSQL(TableDataReq{
		Schema: "app", Table: "users", Limit: 10,
		Filters: []Filter{{Column: "status", Op: "=", Value: "open"}},
		Sort:    []Sort{{Column: "id", Desc: true}},
	}, d)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT * FROM `app`.`users` WHERE `status` = ? ORDER BY `id` DESC LIMIT ?"
	if sql != want {
		t.Fatalf("sql = %q\nwant  %q", sql, want)
	}
	if len(args) != 2 {
		t.Fatalf("args = %#v", args)
	}

	ins, _, err := buildInsertSQL(RowInsertReq{Schema: "app", Table: "t", Values: map[string]any{"a": 1}}, d)
	if err != nil {
		t.Fatal(err)
	}
	if want := "INSERT INTO `app`.`t` (`a`) VALUES (?)"; ins != want {
		t.Fatalf("insert = %q, want %q (no RETURNING for mysql)", ins, want)
	}

	upd, _, err := buildUpdateSQL(RowUpdateReq{Schema: "app", Table: "t", PK: map[string]any{"id": 1}, Values: map[string]any{"a": 2}}, d)
	if err != nil {
		t.Fatal(err)
	}
	if want := "UPDATE `app`.`t` SET `a` = ? WHERE `id` <=> ?"; upd != want {
		t.Fatalf("update = %q, want %q", upd, want)
	}
}
