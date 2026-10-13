package codexacct

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CheckUsable returns nil only when id is an existing, enabled account with
// a login on disk. Used to guard an explicit account switch so a switch to
// a logged-out account is rejected up-front (400) instead of stopping the
// session and then failing to respawn.
func (s *Service) CheckUsable(ctx context.Context, id string) error {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if !a.Enabled {
		return ErrDisabled
	}
	if !accountHasCredentials(a.ConfigDir) {
		return ErrNotLoggedIn
	}
	return nil
}

// AccountHome returns the CODEX_HOME an account's credential and durable
// conversation rollouts live under; "" id → the gateway user's default
// codex home. Implements session.CodexAccountResolver so an account switch
// can carry the conversation rollout from the old home to the new one.
func (s *Service) AccountHome(ctx context.Context, id string) (string, error) {
	if id == "" {
		return DefaultHome(), nil
	}
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return a.ConfigDir, nil
}

// DefaultHome is the gateway user's own codex home (CODEX_HOME or
// ~/.codex) — what an unbound codex session authenticates with.
func DefaultHome() string { return defaultCodexHome() }

// SetAPIKey logs account id into its CODEX_HOME with an OpenAI API key by
// running `codex login --with-api-key` (key on stdin, never argv, so it
// can't leak via the process list). codex writes <CODEX_HOME>/auth.json
// itself; opendray never persists the key. ChatGPT (OAuth) login stays
// out-of-band (`CODEX_HOME=<dir> codex login --device-auth`) since it
// needs an interactive browser/device confirmation.
func (s *Service) SetAPIKey(ctx context.Context, id, key string) (Account, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Account{}, errors.New("api_key is required")
	}
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return Account{}, err
	}
	if a.ConfigDir == "" {
		return Account{}, fmt.Errorf("codex account %q has no CODEX_HOME directory configured", a.Name)
	}
	if err := os.MkdirAll(a.ConfigDir, 0o700); err != nil {
		return Account{}, fmt.Errorf("mkdir codex home: %w", err)
	}
	exe := s.codexBin
	if exe == "" {
		exe, err = exec.LookPath("codex")
		if err != nil {
			return Account{}, ErrNoCodexBinary
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, exe, "login", "--with-api-key")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+a.ConfigDir)
	cmd.Stdin = strings.NewReader(key + "\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return Account{}, fmt.Errorf("codex login --with-api-key: %w: %s", err, strings.TrimSpace(out.String()))
	}
	return s.Get(ctx, id)
}
