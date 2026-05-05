package backup

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestNewPgDump_PATHLookup(t *testing.T) {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		t.Skip("pg_dump not on PATH; skipping")
	}
	pd, err := NewPgDump("")
	if err != nil {
		t.Fatalf("NewPgDump(\"\"): %v", err)
	}
	if pd.BinPath() == "" {
		t.Error("BinPath empty after PATH resolution")
	}
}

func TestNewPgDump_MissingBinary(t *testing.T) {
	_, err := NewPgDump("/definitely/not/a/real/path/pg_dump")
	if !errors.Is(err, ErrPgDumpUnavailable) {
		t.Fatalf("got %v, want ErrPgDumpUnavailable", err)
	}
}

func TestNewPgDump_PATHFallback_Missing(t *testing.T) {
	// Force PATH to empty so PATH lookup fails.
	t.Setenv("PATH", "")
	_, err := NewPgDump("")
	if !errors.Is(err, ErrPgDumpUnavailable) {
		t.Fatalf("got %v, want ErrPgDumpUnavailable", err)
	}
}

func TestPgDump_Version(t *testing.T) {
	pd := newTestPgDump(t)
	out, err := pd.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if !strings.Contains(out, "pg_dump") {
		t.Errorf("Version output missing 'pg_dump': %q", out)
	}
	if got := ParsePGMajorMinor(out); got == "" {
		t.Errorf("could not parse major.minor from %q", out)
	}
}

func TestPgDump_Dump_BadDSN_FailsViaWait(t *testing.T) {
	pd := newTestPgDump(t)
	res, err := pd.Dump(context.Background(), "postgres://nobody@127.0.0.1:1/nodb?sslmode=disable&connect_timeout=1")
	if err != nil {
		// Some platforms may fail on Start; that's also acceptable.
		t.Skipf("Start failed (acceptable on some platforms): %v", err)
	}
	// drain stdout so the child can exit
	_, _ = readAll(res.Reader)
	if err := res.Wait(); err == nil {
		t.Error("expected non-nil error from pg_dump against unreachable DSN")
	}
}

func TestPgDump_Dump_EmptyDSN(t *testing.T) {
	pd := newTestPgDump(t)
	if _, err := pd.Dump(context.Background(), ""); err == nil {
		t.Error("empty DSN should error")
	}
}

func TestParsePGMajorMinor(t *testing.T) {
	cases := map[string]string{
		"pg_dump (PostgreSQL) 14.19 (Homebrew)":       "14.19",
		"pg_dump (PostgreSQL) 16.2":                   "16.2",
		"pg_dump (PostgreSQL) 17.0 (Debian 17.0-1)":   "17.0",
		"":                                            "",
		"pg_dump (PostgreSQL) bogus":                  "",
	}
	for in, want := range cases {
		if got := ParsePGMajorMinor(in); got != want {
			t.Errorf("ParsePGMajorMinor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCappedBuf_Cap(t *testing.T) {
	b := newCappedBuf(8)
	_, _ = b.Write([]byte("aaaaaa"))
	_, _ = b.Write([]byte("bbbbbb"))
	got := b.String()
	if len(got) != 8 {
		t.Errorf("len = %d, want 8 (cap)", len(got))
	}
	if !strings.HasSuffix("aaaaaabbbbbb", got) {
		t.Errorf("got tail %q, want suffix of input", got)
	}
}

// helpers

func newTestPgDump(t *testing.T) *PgDump {
	t.Helper()
	if _, err := exec.LookPath("pg_dump"); err != nil {
		t.Skip("pg_dump not on PATH; skipping")
	}
	pd, err := NewPgDump("")
	if err != nil {
		t.Fatalf("NewPgDump: %v", err)
	}
	return pd
}

func readAll(r interface{ Read(p []byte) (int, error) }) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}
