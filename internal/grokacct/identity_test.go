package grokacct

import (
	"os"
	"path/filepath"
	"testing"
)

func writeAuth(t *testing.T, home, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, grokAuthRelPath), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadOAuthIdentity(t *testing.T) {
	t.Run("extracts email and full name from auth.json", func(t *testing.T) {
		home := t.TempDir()
		writeAuth(t, home, `{
			"https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828": {
				"email": "ada@example.com",
				"first_name": "Ada",
				"last_name": "Lovelace",
				"user_id": "u-123"
			}
		}`)
		got := readOAuthIdentity(home)
		if got.Email != "ada@example.com" {
			t.Errorf("email = %q, want ada@example.com", got.Email)
		}
		if got.Name != "Ada Lovelace" {
			t.Errorf("name = %q, want \"Ada Lovelace\"", got.Name)
		}
	})

	t.Run("email only when names are empty", func(t *testing.T) {
		home := t.TempDir()
		writeAuth(t, home, `{"https://auth.x.ai::x": {"email": "solo@example.com"}}`)
		got := readOAuthIdentity(home)
		if got.Email != "solo@example.com" || got.Name != "" {
			t.Errorf("got %+v, want email=solo@example.com name=\"\"", got)
		}
	})

	t.Run("empty for missing file", func(t *testing.T) {
		got := readOAuthIdentity(t.TempDir())
		if got.Email != "" || got.Name != "" {
			t.Errorf("got %+v, want zero identity", got)
		}
	})

	t.Run("empty for blank home", func(t *testing.T) {
		if got := readOAuthIdentity(""); got.Email != "" {
			t.Errorf("got %+v, want zero identity", got)
		}
	})

	t.Run("skips entries without an email, takes the one that has it", func(t *testing.T) {
		home := t.TempDir()
		writeAuth(t, home, `{
			"https://auth.x.ai::empty": {"first_name": "No", "last_name": "Mail"},
			"https://auth.x.ai::real": {"email": "real@example.com", "first_name": "Real"}
		}`)
		got := readOAuthIdentity(home)
		if got.Email != "real@example.com" {
			t.Errorf("email = %q, want real@example.com", got.Email)
		}
	})
}
