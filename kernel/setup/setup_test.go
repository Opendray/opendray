package setup

import (
	"testing"

	"github.com/opendray/opendray/kernel/config"
)

func TestTokenValidation(t *testing.T) {
	m, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !m.ValidateToken(m.BootstrapToken()) {
		t.Error("own token failed to validate")
	}
	if m.ValidateToken("") {
		t.Error("empty token accepted")
	}
	if m.ValidateToken("wrong") {
		t.Error("bogus token accepted")
	}
}

func TestDraftMutation(t *testing.T) {
	m, _ := New(nil)
	m.UpdateDraft(func(c *config.Config) {
		c.Auth.JWTSecret = "set-by-wizard"
	})
	if got := m.Draft().Auth.JWTSecret; got != "set-by-wizard" {
		t.Errorf("draft JWT = %q, want from mutator", got)
	}
}

func TestStatusProgression(t *testing.T) {
	m, _ := New(nil)
	s := m.Status()
	if !s.NeedsSetup {
		t.Error("fresh manager should need setup")
	}
	if s.Step != StepDB && s.Step != StepWelcome {
		t.Errorf("initial step = %q, want welcome/db", s.Step)
	}

	// Simulate embedded commit
	m.MarkDBTested("embedded")
	m.UpdateDraft(func(c *config.Config) { c.DB.Mode = "embedded" })
	s = m.Status()
	if !s.DBTested || s.DBMode != "embedded" {
		t.Errorf("after commit: dbTested=%v mode=%q", s.DBTested, s.DBMode)
	}

	// Simulate admin set
	m.UpdateDraft(func(c *config.Config) { c.Auth.AdminBootstrapPassword = "secret" })
	s = m.Status()
	if !s.AdminConfigured {
		t.Error("adminConfigured should flip true once password is staged")
	}
}

func TestFinalizeInvokesCallback(t *testing.T) {
	called := false
	m, _ := New(func() { called = true })
	m.UpdateDraft(func(c *config.Config) {
		c.Auth.JWTSecret = "x"
		c.DB.Mode = "embedded"
		c.DB.Embedded.DataDir = t.TempDir()
		c.DB.Embedded.Port = 5432
	})

	// Redirect the save target to avoid touching the user's home dir.
	t.Setenv("OPENDRAY_CONFIG", t.TempDir()+"/config.toml")

	if err := m.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if !called {
		t.Error("onFinish callback not invoked")
	}
	if m.Active() {
		t.Error("Active should be false after Finalize")
	}
}
