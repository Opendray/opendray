package dbtool

import (
	"reflect"
	"testing"
)

func TestBuildTableDataSQL(t *testing.T) {
	tests := []struct {
		name     string
		req      TableDataReq
		wantSQL  string
		wantArgs []any
		wantErr  bool
	}{
		{
			name:     "plain page",
			req:      TableDataReq{Schema: "public", Table: "users", Limit: 50},
			wantSQL:  `SELECT * FROM "public"."users" LIMIT $1`,
			wantArgs: []any{51},
		},
		{
			name:     "default limit with offset",
			req:      TableDataReq{Schema: "public", Table: "users", Offset: 200},
			wantSQL:  `SELECT * FROM "public"."users" LIMIT $1 OFFSET $2`,
			wantArgs: []any{101, 200},
		},
		{
			name: "filters and sort",
			req: TableDataReq{
				Schema: "public", Table: "orders", Limit: 10,
				Filters: []Filter{
					{Column: "status", Op: "=", Value: "open"},
					{Column: "note", Op: "IS NULL"},
					{Column: "name", Op: "ilike", Value: "%a%"},
				},
				Sort: []Sort{{Column: "created_at", Desc: true}, {Column: "id"}},
			},
			wantSQL: `SELECT * FROM "public"."orders" WHERE "status" = $1 AND "note" IS NULL AND "name" ILIKE $2` +
				` ORDER BY "created_at" DESC, "id" LIMIT $3`,
			wantArgs: []any{"open", "%a%", 11},
		},
		{
			name: "hostile identifiers are quoted",
			req: TableDataReq{
				Schema: `pub"lic`, Table: `us"; DROP TABLE x; --ers`, Limit: 1,
				Sort: []Sort{{Column: `id"; --`}},
			},
			wantSQL:  `SELECT * FROM "pub""lic"."us""; DROP TABLE x; --ers" ORDER BY "id""; --" LIMIT $1`,
			wantArgs: []any{2},
		},
		{
			name:    "unsupported operator",
			req:     TableDataReq{Schema: "s", Table: "t", Filters: []Filter{{Column: "a", Op: "IN"}}},
			wantErr: true,
		},
		{
			name:    "injection via operator rejected",
			req:     TableDataReq{Schema: "s", Table: "t", Filters: []Filter{{Column: "a", Op: "= 1 OR 1=1 --"}}},
			wantErr: true,
		},
		{
			name:    "missing table",
			req:     TableDataReq{Schema: "s"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := buildTableDataSQL(tt.req, pgDialect{})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got sql %q", sql)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.wantSQL {
				t.Fatalf("sql = %q\nwant  %q", sql, tt.wantSQL)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, tt.wantArgs)
			}
		})
	}
}

func TestBuildInsertSQL(t *testing.T) {
	sql, args, err := buildInsertSQL(RowInsertReq{
		Schema: "public", Table: "users",
		Values: map[string]any{"name": "bob", "age": 42},
	}, pgDialect{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Columns are emitted in sorted order for determinism.
	want := `INSERT INTO "public"."users" ("age", "name") VALUES ($1, $2) RETURNING *`
	if sql != want {
		t.Fatalf("sql = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{42, "bob"}) {
		t.Fatalf("args = %#v", args)
	}

	if _, _, err := buildInsertSQL(RowInsertReq{Schema: "s", Table: "t"}, pgDialect{}); err == nil {
		t.Fatal("expected error for empty values")
	}
}

func TestBuildUpdateSQL(t *testing.T) {
	sql, args, err := buildUpdateSQL(RowUpdateReq{
		Schema: "public", Table: "users",
		PK:     map[string]any{"id": 7},
		Values: map[string]any{"name": "alice"},
	}, pgDialect{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `UPDATE "public"."users" SET "name" = $1 WHERE "id" IS NOT DISTINCT FROM $2`
	if sql != want {
		t.Fatalf("sql = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{"alice", 7}) {
		t.Fatalf("args = %#v", args)
	}

	if _, _, err := buildUpdateSQL(RowUpdateReq{Schema: "s", Table: "t", Values: map[string]any{"a": 1}}, pgDialect{}); err == nil {
		t.Fatal("expected error for missing pk")
	}
	if _, _, err := buildUpdateSQL(RowUpdateReq{Schema: "s", Table: "t", PK: map[string]any{"id": 1}}, pgDialect{}); err == nil {
		t.Fatal("expected error for empty values")
	}
}

func TestBuildUpdateSQLCompositePK(t *testing.T) {
	sql, args, err := buildUpdateSQL(RowUpdateReq{
		Schema: "s", Table: "t",
		PK:     map[string]any{"b": 2, "a": 1},
		Values: map[string]any{"v": "x"},
	}, pgDialect{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `UPDATE "s"."t" SET "v" = $1 WHERE "a" IS NOT DISTINCT FROM $2 AND "b" IS NOT DISTINCT FROM $3`
	if sql != want {
		t.Fatalf("sql = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{"x", 1, 2}) {
		t.Fatalf("args = %#v", args)
	}
}

// fakeCipher is a reversible toy cipher for round-trip tests.
type fakeCipher struct{ broken bool }

func (f fakeCipher) EncryptField(plain string) (string, error) {
	return "v1:" + plain, nil
}

func (f fakeCipher) DecryptField(envelope string) (string, error) {
	if f.broken {
		return "", errFakeRotated
	}
	return envelope[len("v1:"):], nil
}

var errFakeRotated = &rotatedError{}

type rotatedError struct{}

func (*rotatedError) Error() string { return "key rotated" }

func TestStoreCipherRoundTrip(t *testing.T) {
	s := &Store{cipher: fakeCipher{}}
	enc := s.encrypt("hunter2")
	if enc != "v1:hunter2" {
		t.Fatalf("encrypt = %q", enc)
	}
	// Never double-encrypt an already-wrapped value.
	if again := s.encrypt(enc); again != enc {
		t.Fatalf("double encrypt = %q", again)
	}
	if got := s.decrypt(enc); got != "hunter2" {
		t.Fatalf("decrypt = %q", got)
	}
	// Empty passes through untouched.
	if got := s.encrypt(""); got != "" {
		t.Fatalf("encrypt empty = %q", got)
	}
}

func TestStoreCipherUnarmedAndRotated(t *testing.T) {
	// No cipher: plaintext at rest, envelopes unreadable but preserved.
	plain := &Store{}
	if got := plain.encrypt("secret"); got != "secret" {
		t.Fatalf("unarmed encrypt = %q", got)
	}
	if got := plain.decrypt("v1:abc"); got != "" {
		t.Fatalf("unarmed decrypt = %q, want empty", got)
	}
	// Legacy plaintext read back verbatim.
	if got := plain.decrypt("legacy"); got != "legacy" {
		t.Fatalf("plaintext decrypt = %q", got)
	}
	// Rotated key: decrypt yields empty (caller can't connect) but
	// nothing is blanked in storage — encrypt of the stored envelope on
	// a read-modify-write keeps the ciphertext intact.
	rot := &Store{cipher: fakeCipher{broken: true}}
	if got := rot.decrypt("v1:abc"); got != "" {
		t.Fatalf("rotated decrypt = %q, want empty", got)
	}
	if got := rot.encrypt("v1:abc"); got != "v1:abc" {
		t.Fatalf("rotated re-encrypt = %q, want ciphertext kept", got)
	}
}
