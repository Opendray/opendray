package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

type fakeOCAuth struct {
	auth string
	err  error
	got  string
}

func (f *fakeOCAuth) ResolveSpawnAuth(_ context.Context, id string) (string, error) {
	f.got = id
	return f.auth, f.err
}

// A bound opencode account must reach the CLI as OPENCODE_AUTH_CONTENT
// (which replaces opencode's on-disk auth.json); an unbound session must
// get no such env, so it keeps using opencode's own credentials.
func TestInjectOpenCodeAccount(t *testing.T) {
	const bundle = `{"moonshotai":{"type":"api","key":"sk-test"}}`

	t.Run("bound account injects OPENCODE_AUTH_CONTENT", func(t *testing.T) {
		r := &fakeOCAuth{auth: bundle}
		out := session.PrepareOutput{Env: map[string]string{}}
		if err := injectOpenCodeAccount(context.Background(), r, "oc_1", &out); err != nil {
			t.Fatal(err)
		}
		if out.Env["OPENCODE_AUTH_CONTENT"] != bundle {
			t.Errorf("OPENCODE_AUTH_CONTENT = %q, want bundle", out.Env["OPENCODE_AUTH_CONTENT"])
		}
		if r.got != "oc_1" {
			t.Errorf("resolved account %q, want oc_1", r.got)
		}
	})

	t.Run("unbound session injects nothing", func(t *testing.T) {
		r := &fakeOCAuth{auth: bundle}
		out := session.PrepareOutput{Env: map[string]string{}}
		if err := injectOpenCodeAccount(context.Background(), r, "", &out); err != nil {
			t.Fatal(err)
		}
		if _, ok := out.Env["OPENCODE_AUTH_CONTENT"]; ok {
			t.Error("unbound session must not get OPENCODE_AUTH_CONTENT")
		}
		if r.got != "" {
			t.Error("resolver must not be called for an unbound session")
		}
	})

	t.Run("resolver error fails the spawn", func(t *testing.T) {
		r := &fakeOCAuth{err: errors.New("disabled")}
		out := session.PrepareOutput{Env: map[string]string{}}
		if err := injectOpenCodeAccount(context.Background(), r, "oc_1", &out); err == nil {
			t.Fatal("expected error")
		}
		if _, ok := out.Env["OPENCODE_AUTH_CONTENT"]; ok {
			t.Error("no env on error")
		}
	})
}

// On restart / account switch the manager puts the conversation id on
// the context; the opencode spawn must resume it with --session. A fresh
// spawn carries no id and must start a new conversation.
func TestInjectSessionIDFor_OpenCodeResume(t *testing.T) {
	t.Run("resume id emits --session", func(t *testing.T) {
		const sid = "ses_ee7ec9773ffeqP6qrGkqiHYBYP"
		ctx := session.WithOpenCodeResumeSession(context.Background(), sid)
		var out session.PrepareOutput
		if !injectSessionIDFor(ctx, "opencode", &out) {
			t.Fatal("expected injection")
		}
		if flagValue(out.Args, "--session") != sid {
			t.Errorf("args = %v, want --session %s", out.Args, sid)
		}
		if out.ClaudeSessionID != "" {
			t.Error("opencode must not set ClaudeSessionID")
		}
	})
	t.Run("fresh spawn adds nothing", func(t *testing.T) {
		var out session.PrepareOutput
		if injectSessionIDFor(context.Background(), "opencode", &out) {
			t.Error("fresh opencode spawn must not inject")
		}
		if len(out.Args) != 0 {
			t.Errorf("args = %v, want none", out.Args)
		}
	})
}
