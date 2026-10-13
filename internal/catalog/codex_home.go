package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/opendray/opendray-v2/internal/session"
)

// finalizeCodexHome is the last codex step of Prepare. It decides which
// CODEX_HOME the spawn runs under, makes it authenticated as the session's
// account, resumes the session's conversation when there is one to
// resume, and arranges for durable state to survive the temp dir.
//
//   - credHome is the session's credential home: the bound account's
//     CODEX_HOME, or the gateway default (~/.codex) when unbound. It holds
//     the login (auth.json) and the durable conversation rollouts.
//   - defaultHome is the gateway default codex home — the source of the
//     operator's shared codex config (rules, skills, plugins, config.toml
//     base), mirrored into every scratch home so all accounts behave alike.
//   - resumeID is the codex thread id recorded on the session row ("" on a
//     fresh spawn).
//
// Modes:
//
//   - Scratch home (out.Env["CODEX_HOME"] already set by MCP / skills /
//     memory injection, or forced here for an account-bound session so its
//     credential never runs in place where two sessions would race token
//     refreshes): mirror the shared config from defaultHome, overlay the
//     account's auth.json, seed the resumed rollout, and register an exit
//     hook that syncs rollouts + a refreshed auth.json back into credHome
//     before the temp dir is removed.
//   - Direct default home (unbound, nothing injected): codex runs on
//     ~/.codex itself, so rollouts are already durable; only resume.
//
// Resume is emitted only when the rollout actually exists, so a missing
// conversation starts fresh instead of `codex resume` exiting with "no
// rollout found for thread id".
func finalizeCodexHome(baseDir, credHome, defaultHome, resumeID string, accountBound bool, out *session.PrepareOutput) error {
	scratch := out.Env["CODEX_HOME"]
	if scratch == "" && accountBound {
		scratch = filepath.Join(baseDir, "codex-home")
		if err := os.MkdirAll(scratch, 0o700); err != nil {
			return fmt.Errorf("mkdir codex home: %w", err)
		}
		out.Env["CODEX_HOME"] = scratch
	}

	if scratch == "" {
		if resumeID != "" && session.FindCodexRollout(credHome, resumeID) != "" {
			out.LeadingArgs = []string{"resume", resumeID}
		}
		return nil
	}

	if defaultHome != "" && filepath.Clean(defaultHome) != filepath.Clean(scratch) {
		if err := mirrorCodexHome(defaultHome, scratch); err != nil {
			return fmt.Errorf("mirror codex home: %w", err)
		}
	}
	if credHome != "" && filepath.Clean(credHome) != filepath.Clean(defaultHome) {
		if err := overlayCodexAuth(credHome, scratch); err != nil {
			return err
		}
	}

	if resumeID != "" {
		ok, err := session.SeedCodexRollout(credHome, scratch, resumeID)
		if err != nil {
			return fmt.Errorf("seed codex conversation %s: %w", resumeID, err)
		}
		if ok {
			out.LeadingArgs = []string{"resume", resumeID}
		}
	}

	out.OnExit = func() string { return session.SyncCodexHome(scratch, credHome) }
	return nil
}

// overlayCodexAuth replaces the scratch home's auth.json with the bound
// account's, so the session authenticates as that account (the mirror step
// copied the default home's). The source must be a non-empty regular file —
// a symlink planted in an account dir is never followed.
func overlayCodexAuth(credHome, scratch string) error {
	src := filepath.Join(credHome, "auth.json")
	st, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("codex account login missing at %s: %w", src, err)
	}
	if !st.Mode().IsRegular() || st.Size() == 0 {
		return fmt.Errorf("codex account login at %s is not a regular file", src)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read codex account login: %w", err)
	}
	dst := filepath.Join(scratch, "auth.json")
	_ = os.Remove(dst) // drop the mirrored default login (or a stale link)
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write codex account login: %w", err)
	}
	return nil
}
