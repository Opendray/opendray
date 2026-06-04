package catalog

import (
	"context"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

// injectSessionIDFor is the building block the session manager relies on
// for account switching: when no resume UUID is in context it must mint a
// FRESH `--session-id` (a session the spawning account's CLI will create
// + recognise), and when a resume UUID is present it must `--resume` it.
//
// The account-switch fix (clearing ClaudeSessionID before respawn) hinges
// on the fresh branch: resuming a UUID minted under a *different* account
// fails with "No conversation found" and the CLI exits, which previously
// left a switched session stopped and unrestartable.
func TestInjectSessionIDFor_ClaudeResumeVsFresh(t *testing.T) {
	t.Run("no resume id mints a fresh --session-id", func(t *testing.T) {
		var out session.PrepareOutput
		ok := injectSessionIDFor(context.Background(), "claude", &out)
		if !ok {
			t.Fatal("expected injection for claude")
		}
		if !hasFlag(out.Args, "--session-id") {
			t.Errorf("fresh spawn should pass --session-id, got %v", out.Args)
		}
		if hasFlag(out.Args, "--resume") {
			t.Errorf("fresh spawn must NOT pass --resume, got %v", out.Args)
		}
		if out.ClaudeSessionID == "" {
			t.Error("fresh spawn should report a minted ClaudeSessionID")
		}
	})

	t.Run("a resume id resumes it", func(t *testing.T) {
		const rid = "c0626a3e-bbc4-4243-bcb1-c689cb269837"
		ctx := session.WithResumeClaudeSessionID(context.Background(), rid)
		var out session.PrepareOutput
		injectSessionIDFor(ctx, "claude", &out)
		if !hasFlag(out.Args, "--resume") || flagValue(out.Args, "--resume") != rid {
			t.Errorf("resume spawn should pass --resume %s, got %v", rid, out.Args)
		}
		if out.ClaudeSessionID != rid {
			t.Errorf("ClaudeSessionID = %q, want %q", out.ClaudeSessionID, rid)
		}
	})
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
