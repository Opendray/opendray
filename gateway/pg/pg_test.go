package pg

import "testing"

// TestFirstVerb_NormalisesAndStripsComments covers the parser that
// every write-verb / destructive check builds on. A regression here
// lets DELETE-without-WHERE slip past the "is destructive" gate.
func TestFirstVerb_NormalisesAndStripsComments(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"SELECT 1", "SELECT"},
		{"  select 1", "SELECT"},
		{"-- comment\nSELECT 1", "SELECT"},
		{"-- line 1\n-- line 2\nSELECT 1", "SELECT"},
		{"/* block */ UPDATE t SET x=1", "UPDATE"},
		{"DELETE FROM users;", "DELETE"},
		{"DROP TABLE(x)", "DROP"},
		{"", ""},
		{"-- only comment", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := firstVerb(tc.in); got != tc.want {
				t.Errorf("firstVerb(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsWriteVerb_MatchesEveryModifyingStatement(t *testing.T) {
	writes := []string{
		"INSERT INTO t VALUES (1)",
		"UPDATE t SET x=1",
		"DELETE FROM t",
		"DROP TABLE t",
		"CREATE TABLE t (x int)",
		"ALTER TABLE t ADD COLUMN y int",
		"TRUNCATE t",
		"GRANT SELECT ON t TO kev",
		"REVOKE ALL ON t FROM kev",
		"VACUUM t",
	}
	for _, sql := range writes {
		if !IsWriteVerb(sql) {
			t.Errorf("IsWriteVerb(%q) = false, want true", sql)
		}
	}
	reads := []string{
		"SELECT 1",
		"SHOW search_path",
		"EXPLAIN SELECT 1",
		"WITH t AS (SELECT 1) SELECT * FROM t",
		// BEGIN/COMMIT are metadata — not "writes" per se, and the
		// tx scaffolding here wraps every query in its own BEGIN
		// anyway. Keep out of the write list to avoid false alarms.
		"BEGIN",
		"COMMIT",
	}
	for _, sql := range reads {
		if IsWriteVerb(sql) {
			t.Errorf("IsWriteVerb(%q) = true, want false", sql)
		}
	}
}

func TestIsDestructiveVerb_RequiresExplicitWHEREForDelete(t *testing.T) {
	// DELETE without WHERE is the classic footgun. Flag it so the
	// client can show an extra confirmation dialog.
	destructive := []string{
		"DROP TABLE x",
		"TRUNCATE x",
		"DELETE FROM x",
		"delete from x",
		"DELETE FROM x;",
	}
	for _, sql := range destructive {
		if !IsDestructiveVerb(sql) {
			t.Errorf("IsDestructiveVerb(%q) = false, want true", sql)
		}
	}
	safe := []string{
		"DELETE FROM x WHERE id = 1",
		"DELETE FROM x WHERE created_at < now()",
		"UPDATE x SET y=1 WHERE id=1",
		"SELECT * FROM x",
	}
	for _, sql := range safe {
		if IsDestructiveVerb(sql) {
			t.Errorf("IsDestructiveVerb(%q) = true, want false", sql)
		}
	}
}

func TestIdentifierOK_LetsThroughValidSchemaNames(t *testing.T) {
	ok := []string{"public", "my_schema", "Schema-42", "a"}
	for _, n := range ok {
		if !identifierOK(n) {
			t.Errorf("identifierOK(%q) = false, want true", n)
		}
	}
	bad := []string{"", "pg_temp_1; DROP TABLE x", "x y", "x;y", "x\"y"}
	for _, n := range bad {
		if identifierOK(n) {
			t.Errorf("identifierOK(%q) = true, want false (injection risk)", n)
		}
	}
	long := make([]byte, 64)
	for i := range long {
		long[i] = 'a'
	}
	if identifierOK(string(long)) {
		t.Error("identifierOK must reject names longer than 63 bytes (PG's NAMEDATALEN)")
	}
}

func TestConfigDSN_URLEncodesPassword(t *testing.T) {
	// Passwords with special chars (@, #, /, :) are common in
	// password managers. url.UserPassword escapes them inside the
	// UserInfo section so the resulting DSN still parses.
	cfg := Config{
		Host: "db.example", Port: 5432, User: "me",
		Password: "p@ss:word/!#", Database: "app", SSLMode: "require",
	}
	dsn := cfg.dsn()
	// The raw "@ss" must not appear — it would split the host.
	// We don't assert the exact escape form (url.UserPassword's
	// encoding is implementation-defined) — just that the raw
	// character isn't there.
	if containsAny(dsn, []string{":p@", "ss:word/"}) {
		t.Errorf("unescaped password leaked into DSN: %s", dsn)
	}
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		for i := 0; i+len(n) <= len(s); i++ {
			if s[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}
