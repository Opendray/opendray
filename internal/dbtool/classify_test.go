package dbtool

import (
	"errors"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want StatementClass
		err  error
	}{
		{"select", "SELECT * FROM users", ClassRead, nil},
		{"select lowercase", "select 1", ClassRead, nil},
		{"select with trailing semicolon", "SELECT 1;", ClassRead, nil},
		{"select with leading spaces", "   \n\tSELECT 1", ClassRead, nil},
		{"values", "VALUES (1), (2)", ClassRead, nil},
		{"table", "TABLE users", ClassRead, nil},
		{"show", "SHOW search_path", ClassRead, nil},
		{"describe", "DESCRIBE users", ClassRead, nil},
		{"desc", "DESC users", ClassRead, nil},

		{"line comment then select", "-- hi\nSELECT 1", ClassRead, nil},
		{"block comment then select", "/* multi\nline */ SELECT 1", ClassRead, nil},
		{"nested block comment", "/* outer /* inner */ still */ SELECT 1", ClassRead, nil},
		{"comment only", "-- nothing here", "", ErrEmptyStatement},
		{"empty", "   ", "", ErrEmptyStatement},

		{"insert", "INSERT INTO t (a) VALUES (1)", ClassWrite, nil},
		{"update", "UPDATE t SET a = 1", ClassWrite, nil},
		{"delete", "DELETE FROM t", ClassWrite, nil},
		{"merge", "MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DO NOTHING", ClassWrite, nil},
		{"copy", "COPY t FROM STDIN", ClassWrite, nil},

		{"with select", "WITH x AS (SELECT 1) SELECT * FROM x", ClassRead, nil},
		{"with writing cte", "WITH x AS (INSERT INTO t VALUES (1) RETURNING *) SELECT * FROM x", ClassWrite, nil},
		{"with delete tail", "WITH x AS (SELECT 1) DELETE FROM t", ClassWrite, nil},
		{"with update in string literal", "WITH x AS (SELECT * FROM audit WHERE action = 'update') SELECT * FROM x", ClassRead, nil},
		{"with quoted identifier named update", `WITH x AS (SELECT "update" FROM t) SELECT * FROM x`, ClassRead, nil},
		{"with backtick identifier named update", "WITH x AS (SELECT `update` FROM t) SELECT * FROM x", ClassRead, nil},
		{"with dollar-quoted insert text", "WITH x AS (SELECT $tag$INSERT INTO$tag$) SELECT * FROM x", ClassRead, nil},

		{"explain select", "EXPLAIN SELECT 1", ClassRead, nil},
		{"explain update plans only", "EXPLAIN UPDATE t SET a = 1", ClassRead, nil},
		{"explain analyze update executes", "EXPLAIN ANALYZE UPDATE t SET a = 1", ClassWrite, nil},
		{"explain option-list analyze", "EXPLAIN (ANALYZE, BUFFERS) DELETE FROM t", ClassWrite, nil},
		{"explain option-list no analyze", "EXPLAIN (COSTS, VERBOSE) UPDATE t SET a = 1", ClassRead, nil},
		{"explain analyze select", "EXPLAIN ANALYZE SELECT 1", ClassRead, nil},

		{"create table", "CREATE TABLE t (id int)", ClassDDL, nil},
		{"alter", "ALTER TABLE t ADD COLUMN b int", ClassDDL, nil},
		{"drop", "DROP TABLE t", ClassDDL, nil},
		{"truncate", "TRUNCATE t", ClassDDL, nil},
		{"grant", "GRANT SELECT ON t TO u", ClassDDL, nil},
		{"vacuum", "VACUUM t", ClassDDL, nil},

		{"do block fails safe as write", "DO $$ BEGIN PERFORM 1; END $$", ClassWrite, nil},
		{"call fails safe as write", "CALL proc()", ClassWrite, nil},
		{"set fails safe as write", "SET search_path = public", ClassWrite, nil},

		{"begin rejected", "BEGIN", "", ErrTxnControl},
		{"commit rejected", "COMMIT", "", ErrTxnControl},
		{"rollback rejected", "ROLLBACK", "", ErrTxnControl},

		{"multi-statement rejected", "SELECT 1; SELECT 2", "", ErrMultiStatement},
		{"multi-statement write rejected", "SELECT 1; DROP TABLE t", "", ErrMultiStatement},
		{"semicolon inside string ok", "SELECT 'a;b'", ClassRead, nil},
		{"semicolon inside dollar quote ok", "SELECT $$a;b$$", ClassRead, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Classify(tt.sql)
			if tt.err != nil {
				if !errors.Is(err, tt.err) {
					t.Fatalf("Classify(%q) err = %v, want %v", tt.sql, err, tt.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Classify(%q) unexpected error: %v", tt.sql, err)
			}
			if got != tt.want {
				t.Fatalf("Classify(%q) = %q, want %q", tt.sql, got, tt.want)
			}
		})
	}
}

// Do-block inner content is dollar-quoted, so the classifier must not
// mistake its body for extra statements even though it contains
// semicolons; the DO leader itself fails safe as a write.
func TestClassifyDoBlockSingleStatement(t *testing.T) {
	got, err := Classify("DO $$ BEGIN INSERT INTO t VALUES (1); END $$;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ClassWrite {
		t.Fatalf("got %q, want write", got)
	}
}
