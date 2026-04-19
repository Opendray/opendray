// Package compat provides compatibility helpers for legacy OpenDray plugin
// manifests that predate the v1 plugin contract.
//
// The key export is [Synthesize], which projects a legacy Provider into the
// v1 shape entirely in memory. The on-disk manifest.json is never rewritten.
package compat

import (
	"log/slog"

	"github.com/opendray/opendray/plugin"
)

// Synthesize returns an in-memory v1 overlay for a legacy Provider.
// The result is a NEW Provider value — caller must not assume pointer
// equality with the input. The overlay is never persisted to disk; it
// exists so the rest of the platform can treat every loaded plugin as
// v1-shaped.
//
// Rules (07-lifecycle.md §Compat mode):
//
//	form      = "host" if p.Type in {cli,local,shell}, else "declarative"
//	publisher = "opendray-builtin"
//	engines   = {opendray: ">=0"}   — matches any host version
//	activation = ["onStartup"]      — builtins load eagerly
//	contributes                     — left empty (ContributesV1 has no
//	                                  AgentProviders or Views in M1)
//	permissions = nil               — builtins are trusted; no capability gate
//
// Passing a v1 manifest (IsV1()==true) is a programming error. Synthesize
// returns a copy unchanged with a log warning (no panic) — the caller's
// "is this legacy" check is wrong, not the data itself.
func Synthesize(p plugin.Provider) plugin.Provider {
	if p.IsV1() {
		slog.Warn("compat.Synthesize called on a v1 manifest; returning copy unchanged",
			"plugin", p.Name)
		// Return a copy (new value) but leave all fields as-is.
		return copyProvider(p)
	}

	// Start with a full copy so all legacy identity fields are preserved.
	out := copyProvider(p)

	// ── v1 identity-level fields ──────────────────────────────────────────
	out.Publisher = "opendray-builtin"
	out.Engines = &plugin.EnginesV1{Opendray: ">=0"}

	// form: derived from legacy type (EffectiveForm() already encodes this
	// logic, but we set Form explicitly so IsV1()'s Engines check passes AND
	// EffectiveForm returns the right value without re-deriving).
	out.Form = out.EffectiveForm()

	// activation: builtins activate eagerly like the pre-v1 runtime did.
	out.Activation = []string{"onStartup"}

	// ── contributes ───────────────────────────────────────────────────────
	// M1's ContributesV1 only has Commands/StatusBar/Keybindings/Menus.
	// M2 adds Views for legacy panel plugins so the Flutter workbench can
	// include them in the activity bar discovery list. The view entry is
	// metadata-only — legacy panels keep their bespoke widget rendering.
	if out.Contributes == nil {
		out.Contributes = &plugin.ContributesV1{}
	}

	// For legacy panel plugins: synthesize one view entry so the Flutter
	// workbench can discover the panel in the activity bar rail.
	// The view entry is metadata-only; legacy panels keep their existing
	// bespoke widget rendering (no entry path needed).
	//
	// Guard: only add when no view with the same id already exists (defensive
	// against callers who re-synthesize a provider that was already processed).
	if p.Type == plugin.ProviderTypePanel {
		alreadyPresent := false
		for _, v := range out.Contributes.Views {
			if v.ID == p.Name {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			title := p.DisplayName
			if title == "" {
				title = p.Name
			}
			out.Contributes.Views = append(out.Contributes.Views, plugin.ViewV1{
				ID:        p.Name,
				Title:     title,
				Container: "activityBar",
				Render:    "declarative",
				// Entry intentionally omitted: legacy panels render via
				// bespoke Flutter widgets, not the webview view host.
			})
		}
	}

	// ── permissions ───────────────────────────────────────────────────────
	// Builtins are trusted and skip the capability gate entirely.
	// Leave Permissions nil (zero) which the gate interprets as "unrestricted"
	// for the trusted-publisher path.
	out.Permissions = nil

	return out
}

// copyProvider returns a shallow copy of p. Pointer fields are re-pointed to
// new copies so the caller cannot observe aliasing bugs when modifying the
// result (e.g. replacing p.Engines must not affect the original).
func copyProvider(p plugin.Provider) plugin.Provider {
	out := p // shallow copy of all value fields

	// Deep-copy Engines pointer.
	if p.Engines != nil {
		e := *p.Engines
		out.Engines = &e
	}

	// Deep-copy Contributes pointer.
	if p.Contributes != nil {
		c := *p.Contributes
		// Copy slices so callers can't alias through the pointer.
		if len(c.Commands) > 0 {
			cmds := make([]plugin.CommandV1, len(c.Commands))
			copy(cmds, c.Commands)
			c.Commands = cmds
		}
		if len(c.StatusBar) > 0 {
			sb := make([]plugin.StatusBarItemV1, len(c.StatusBar))
			copy(sb, c.StatusBar)
			c.StatusBar = sb
		}
		if len(c.Keybindings) > 0 {
			kb := make([]plugin.KeybindingV1, len(c.Keybindings))
			copy(kb, c.Keybindings)
			c.Keybindings = kb
		}
		if len(c.Menus) > 0 {
			menus := make(map[string][]plugin.MenuEntryV1, len(c.Menus))
			for k, v := range c.Menus {
				entries := make([]plugin.MenuEntryV1, len(v))
				copy(entries, v)
				menus[k] = entries
			}
			c.Menus = menus
		}
		if len(c.ActivityBar) > 0 {
			ab := make([]plugin.ActivityBarItemV1, len(c.ActivityBar))
			copy(ab, c.ActivityBar)
			c.ActivityBar = ab
		}
		if len(c.Views) > 0 {
			views := make([]plugin.ViewV1, len(c.Views))
			copy(views, c.Views)
			c.Views = views
		}
		if len(c.Panels) > 0 {
			panels := make([]plugin.PanelV1, len(c.Panels))
			copy(panels, c.Panels)
			c.Panels = panels
		}
		out.Contributes = &c
	}

	// Deep-copy Permissions pointer.
	if p.Permissions != nil {
		perm := *p.Permissions
		out.Permissions = &perm
	}

	// Activation slice — copy to avoid shared backing array.
	if len(p.Activation) > 0 {
		act := make([]string, len(p.Activation))
		copy(act, p.Activation)
		out.Activation = act
	}

	// ConfigSchema slice — copy.
	if len(p.ConfigSchema) > 0 {
		cs := make([]plugin.ConfigField, len(p.ConfigSchema))
		copy(cs, p.ConfigSchema)
		out.ConfigSchema = cs
	}

	return out
}
