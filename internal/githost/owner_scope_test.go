package githost

import (
	"context"
	"errors"
	"testing"
)

// Credential resolution decides which identity a push authenticates as.
// A wrong answer here doesn't fail loudly — it authenticates as
// somebody else and the forge returns a 403 that points nowhere near
// this code, which is exactly how the host-only scheme wasted an
// afternoon. So: one case per branch of GetForRepo.
func TestGetForRepo_Resolution(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	ctx := context.Background()
	svc := NewService(pool, nil)
	host := uniqHost("resolve")

	mk := func(owner, token string) Host {
		h, err := svc.Create(ctx, CreateRequest{
			Kind: KindGitHub, Host: host, Owner: owner,
			Name: "n", Token: token,
		})
		if err != nil {
			t.Fatalf("create owner=%q: %v", owner, err)
		}
		t.Cleanup(func() { _ = svc.Delete(context.Background(), h.ID) })
		return h
	}

	// The whole point: two identities on ONE host, which the old
	// UNIQUE(host) made impossible.
	wide := mk("", "tok-host-wide")
	org := mk("Opendray", "tok-org")

	tests := []struct {
		name      string
		owner     string
		wantToken string
		wantID    string
	}{
		{"owner with its own row", "Opendray", "tok-org", org.ID},
		{"owner is matched case-insensitively", "opendray", "tok-org", org.ID},
		{"different case again", "OPENDRAY", "tok-org", org.ID},
		{"unknown owner falls back to host-wide", "someone-else", "tok-host-wide", wide.ID},
		{"no owner asks for the host-wide row", "", "tok-host-wide", wide.ID},
		{"whitespace is not an owner", "   ", "tok-host-wide", wide.ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetForRepo(ctx, host, tt.owner)
			if err != nil {
				t.Fatalf("GetForRepo(%q): %v", tt.owner, err)
			}
			if got.Token != tt.wantToken || got.ID != tt.wantID {
				t.Fatalf("owner %q resolved to token %q (id %s), want %q (id %s)",
					tt.owner, got.Token, got.ID, tt.wantToken, tt.wantID)
			}
		})
	}

	// GetByHost is the host-wide lookup, unchanged for existing callers.
	t.Run("GetByHost still means host-wide", func(t *testing.T) {
		got, err := svc.GetByHost(ctx, host)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != wide.ID {
			t.Fatalf("GetByHost returned %s, want the host-wide row %s", got.ID, wide.ID)
		}
	})
}

// Without a host-wide row there is nothing to fall back TO, and an
// unmatched owner must say so rather than borrow another owner's token.
func TestGetForRepo_NoFallbackRow(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	ctx := context.Background()
	svc := NewService(pool, nil)
	host := uniqHost("nofallback")

	h, err := svc.Create(ctx, CreateRequest{
		Kind: KindGitHub, Host: host, Owner: "only-this-org",
		Name: "n", Token: "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), h.ID) })

	for _, owner := range []string{"", "another-org"} {
		if _, err := svc.GetForRepo(ctx, host, owner); !errors.Is(err, ErrNotFound) {
			t.Fatalf("GetForRepo(%q) err = %v, want ErrNotFound — a token "+
				"scoped to one owner must not answer for another", owner, err)
		}
	}
}

// Owners are case-insensitive on every forge here, so the uniqueness
// must be too — otherwise Opendray and opendray become two rows and
// which one answers is down to insertion order.
func TestCreate_RejectsDuplicateOwnerRegardlessOfCase(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	ctx := context.Background()
	svc := NewService(pool, nil)
	host := uniqHost("dupowner")

	first, err := svc.Create(ctx, CreateRequest{
		Kind: KindGitHub, Host: host, Owner: "Acme", Name: "n", Token: "t1",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), first.ID) })

	dup, err := svc.Create(ctx, CreateRequest{
		Kind: KindGitHub, Host: host, Owner: "acme", Name: "n", Token: "t2",
	})
	if err == nil {
		_ = svc.Delete(ctx, dup.ID)
		t.Fatal("a second credential for the same owner in different case was accepted")
	}
}
