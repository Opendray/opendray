// Package actions implements the revocation action dispatcher —
// the bridge between market/revocation.Poller (which decides
// "this installed plugin is on the kill list") and the concrete
// platform effects (uninstall via install.Installer, disable via
// plugin.Runtime, user-visible banner via the workbench bus).
//
// This package lives separately so the test surface for the
// dispatcher stays tight (fakes for three callbacks, no Installer
// or Runtime boot) and so future action kinds (e.g. "block-update")
// land in one file.
package actions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/opendray/opendray/plugin/market/revocation"
)

// UninstallFunc removes a plugin completely — files on disk,
// consents, DB rows — as if the user hit Uninstall. Matches
// install.Installer.Uninstall's signature so the caller can pass
// the method value directly.
type UninstallFunc func(ctx context.Context, name string) error

// SetEnabledFunc flips the `enabled` column in the plugins table
// (and in-memory runtime) without touching files or consents.
// Matches plugin.Runtime.SetEnabled.
type SetEnabledFunc func(ctx context.Context, name string, enabled bool) error

// NotifyFunc surfaces a revocation banner to the user. kind is
// the action that fired ("uninstall" / "disable" / "warn"),
// pluginName is the affected plugin ("publisher/name"), reason is
// the operator-supplied short explanation.
//
// The concrete implementation in main.go publishes a
// WorkbenchBus event so every connected Flutter client sees the
// banner; tests use a recorder fake.
type NotifyFunc func(kind, pluginName, reason string)

// Config wires the dispatcher to the platform. Every field is
// required except Logger.
type Config struct {
	Uninstall  UninstallFunc
	SetEnabled SetEnabledFunc
	Notify     NotifyFunc
	Logger     *slog.Logger
}

// Handler is the dispatcher. Construct once at boot, pass
// Handler.Dispatch to revocation.Config.OnAction.
type Handler struct {
	cfg Config
}

// New constructs a Handler. Missing required callbacks return an
// error so misconfiguration fails loudly at wire-time.
func New(cfg Config) (*Handler, error) {
	if cfg.Uninstall == nil {
		return nil, fmt.Errorf("actions: Uninstall is required")
	}
	if cfg.SetEnabled == nil {
		return nil, fmt.Errorf("actions: SetEnabled is required")
	}
	if cfg.Notify == nil {
		return nil, fmt.Errorf("actions: Notify is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Handler{cfg: cfg}, nil
}

// Dispatch implements [revocation.ActionHandler]. Called once per
// (Entry, InstalledPlugin) match from the poller.
//
// Every action — including "warn" — fires Notify so the user
// always sees a banner; disable additionally flips the runtime
// state, and uninstall additionally removes the plugin.
//
// Error returned from Dispatch bubbles back to the poller, which
// logs it and continues to the next match.
func (h *Handler) Dispatch(ctx context.Context, entry revocation.Entry, target revocation.InstalledPlugin) error {
	targetName := target.Publisher + "/" + target.Name
	action := entry.NormalisedAction()
	h.cfg.Logger.Info("revocation: action fired",
		"plugin", targetName,
		"version", target.Version,
		"action", action,
		"reason", entry.Reason,
	)

	// Notify first so the user sees the banner even when the
	// follow-up effect fails (e.g. uninstall can't remove a locked
	// file on Windows). Keeps the spec's "advisory" posture —
	// users aren't left wondering what happened.
	h.cfg.Notify(action, targetName, entry.Reason)

	switch action {
	case revocation.ActionUninstall:
		// Plugin name in the DB is bare (no publisher prefix), so
		// pass target.Name not targetName.
		if err := h.cfg.Uninstall(ctx, target.Name); err != nil {
			return fmt.Errorf("uninstall %s: %w", targetName, err)
		}
	case revocation.ActionDisable:
		if err := h.cfg.SetEnabled(ctx, target.Name, false); err != nil {
			return fmt.Errorf("disable %s: %w", targetName, err)
		}
	case revocation.ActionWarn:
		// Banner-only — Notify already fired.
	default:
		// NormalisedAction should prevent this branch; defensive.
		return fmt.Errorf("unhandled action %q", action)
	}
	return nil
}
