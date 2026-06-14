package catalog

import (
	"context"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

// injectCarryoverFor is the seam Manager.SwitchClaudeAccount relies on
// for "carry context": when the switch put a conversation recap on the
// context (session.WithCarryoverContext), the spawn must surface it to
// the new account's CLI — for claude, as a --append-system-prompt arg.
// When no recap is present (every spawn except an opted-in switch) it
// must add nothing.
func TestInjectCarryoverFor_Claude(t *testing.T) {
	const recap = "# Carried-over context from a previous account\n\nYou: do the thing"

	t.Run("recap on context injects --append-system-prompt", func(t *testing.T) {
		ctx := session.WithCarryoverContext(context.Background(), recap)
		var out session.PrepareOutput
		if err := injectCarryoverFor(ctx, "claude", t.TempDir(), &out); err != nil {
			t.Fatalf("injectCarryoverFor: %v", err)
		}
		if !hasFlag(out.Args, "--append-system-prompt") {
			t.Errorf("expected --append-system-prompt, got %v", out.Args)
		}
		if got := flagValue(out.Args, "--append-system-prompt"); got != recap {
			t.Errorf("injected text = %q, want %q", got, recap)
		}
	})

	t.Run("no recap on context injects nothing", func(t *testing.T) {
		var out session.PrepareOutput
		if err := injectCarryoverFor(context.Background(), "claude", t.TempDir(), &out); err != nil {
			t.Fatalf("injectCarryoverFor: %v", err)
		}
		if len(out.Args) != 0 {
			t.Errorf("expected no args without a recap, got %v", out.Args)
		}
	})
}
