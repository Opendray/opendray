package database

import (
	"context"
	"strings"
	"testing"
)

func TestRunQuery_RejectsNonSelect(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{"empty", ""},
		{"whitespace only", "   \n\t  "},
		{"insert", "INSERT INTO foo VALUES (1)"},
		{"update", "UPDATE foo SET x = 1"},
		{"delete", "DELETE FROM foo"},
		{"drop", "DROP TABLE foo"},
		{"truncate", "TRUNCATE foo"},
		{"alter", "ALTER TABLE foo ADD COLUMN y int"},
		{"create", "CREATE TABLE foo (x int)"},
		{"grant", "GRANT SELECT ON foo TO bar"},
		{"multi-statement", "SELECT 1; DROP TABLE users"},
		{"case-insensitive drop", "select * from foo; drop table bar"},
		{"delete hidden by leading whitespace", "   delete from foo"},
		{"do block", "DO $$ BEGIN DELETE FROM foo; END $$"},
		{"call procedure", "CALL delete_everything()"},
		{"set command", "SET ROLE superuser"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunQuery(context.Background(), PGConfig{}, tc.sql)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.sql)
			}
		})
	}
}

func TestRunQuery_RejectsMultipleStatements(t *testing.T) {
	_, err := RunQuery(context.Background(), PGConfig{}, "SELECT 1; SELECT 2")
	if err == nil || !strings.Contains(err.Error(), "multiple statements") {
		t.Fatalf("expected multiple-statements error, got %v", err)
	}
}

func TestStripSQLComments(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"SELECT 1 -- drop table users\n", "SELECT 1 \n"},
		{"SELECT /* drop table */ 1", "SELECT  1"},
		{"SELECT '-- not a comment' FROM t", "SELECT '-- not a comment' FROM t"},
		{"SELECT \"-- quoted\" FROM t", "SELECT \"-- quoted\" FROM t"},
		{"/* nested /* still */ end */ SELECT 1", " end */ SELECT 1"},
	}
	for _, tc := range cases {
		got := stripSQLComments(tc.in)
		if got != tc.want {
			t.Errorf("stripSQLComments(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCommentedKeywordStillBlockedIfPresent(t *testing.T) {
	// -- comment hides DELETE but the next line has a real one
	sql := "SELECT 1\n-- DELETE FROM foo\nDELETE FROM foo"
	_, err := RunQuery(context.Background(), PGConfig{}, sql)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCommentedKeywordPermittedWhenOnlyInComment(t *testing.T) {
	// Keyword only inside a comment must not fail the check — but the query is
	// also not a SELECT start, so we assert that isolated from the start check.
	sql := "SELECT 1 -- DELETE FROM foo"
	stripped := stripSQLComments(sql)
	if findBlockedKeyword(stripped) != "" {
		t.Fatalf("expected no blocked keyword after stripping comment, got %q in %q", findBlockedKeyword(stripped), stripped)
	}
}

func TestSafeIdentifier(t *testing.T) {
	good := []string{"public", "my_table", "_underscore", "t1", "Camel_Case"}
	bad := []string{"", "1bad", "with space", "semi;colon", "quote'd", "--comment", "drop table"}
	for _, s := range good {
		if !safeIdentifier(s) {
			t.Errorf("expected %q to be safe", s)
		}
	}
	for _, s := range bad {
		if safeIdentifier(s) {
			t.Errorf("expected %q to be rejected", s)
		}
	}
}

func TestReadOnlyStartRegex(t *testing.T) {
	good := []string{
		"SELECT 1",
		"  select * from foo",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"SHOW server_version",
		"EXPLAIN SELECT 1",
		"VALUES (1), (2)",
		"TABLE foo",
	}
	bad := []string{
		"INSERT INTO foo VALUES (1)",
		"UPDATE foo SET x=1",
		"DELETE FROM foo",
		"BEGIN",
		"COMMIT",
		"",
	}
	for _, s := range good {
		if !readOnlyStartRe.MatchString(s) {
			t.Errorf("expected %q to match read-only start", s)
		}
	}
	for _, s := range bad {
		if readOnlyStartRe.MatchString(s) {
			t.Errorf("expected %q to NOT match read-only start", s)
		}
	}
}

func TestPGConfigDSN(t *testing.T) {
	cfg := PGConfig{Host: "h", Port: 5432, Database: "d", Username: "u", Password: "p"}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "host=h") || !strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("dsn missing expected fields: %s", dsn)
	}
}

func TestSanitizeErr(t *testing.T) {
	cfg := PGConfig{Password: "secret-pw"}
	err := sanitizeErr(errWithPassword("auth failed for secret-pw"), cfg)
	if strings.Contains(err.Error(), "secret-pw") {
		t.Errorf("password leaked in error: %s", err)
	}
}

type stringErr string

func (e stringErr) Error() string { return string(e) }

func errWithPassword(s string) error { return stringErr(s) }
