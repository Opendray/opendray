// Package contributions provides the in-memory contribution registry.
// It is the single source of truth for every installed plugin's declared
// workbench contribution points (commands, status bar, keybindings, menus).
//
// The registry is queried by HTTP handlers (T9) and mutated only by
// plugin.Runtime via Set/Remove — no direct user writes are allowed.
package contributions

import (
	"sort"
	"sync"

	"github.com/opendray/opendray/plugin"
)

// Registry is the in-memory join of every installed plugin's declared
// contribution points. Thread-safe for concurrent Set/Remove/Flatten calls.
type Registry struct {
	mu       sync.RWMutex
	byPlugin map[string]plugin.ContributesV1
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		byPlugin: make(map[string]plugin.ContributesV1),
	}
}

// isZero reports whether a ContributesV1 is entirely empty (no contributions).
// M1 slots (commands/statusBar/keybindings/menus) AND M2 slots
// (activityBar/views/panels) all count — a plugin that contributes
// ONLY a view (no commands at all) is still a legitimate registration.
func isZero(c plugin.ContributesV1) bool {
	return len(c.Commands) == 0 &&
		len(c.StatusBar) == 0 &&
		len(c.Keybindings) == 0 &&
		len(c.Menus) == 0 &&
		len(c.ActivityBar) == 0 &&
		len(c.Views) == 0 &&
		len(c.Panels) == 0
}

// Set replaces the contribution set for the given plugin name.
// If c is a zero ContributesV1, the entry is deleted (same behaviour as Remove).
func (r *Registry) Set(pluginName string, c plugin.ContributesV1) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if isZero(c) {
		delete(r.byPlugin, pluginName)
		return
	}
	r.byPlugin[pluginName] = c
}

// Remove drops a plugin's contributions. Safe to call for unknown names.
func (r *Registry) Remove(pluginName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byPlugin, pluginName)
}

// Has reports whether a plugin name has any contributions registered.
func (r *Registry) Has(pluginName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byPlugin[pluginName]
	return ok
}

// ── Owned wrappers ─────────────────────────────────────────────────────────
//
// Each item gains a PluginName field so the workbench knows the origin.

// OwnedCommand is a CommandV1 annotated with its contributing plugin name.
type OwnedCommand struct {
	PluginName string `json:"pluginName"`
	plugin.CommandV1
}

// OwnedStatusBarItem is a StatusBarItemV1 annotated with its contributing plugin name.
type OwnedStatusBarItem struct {
	PluginName string `json:"pluginName"`
	plugin.StatusBarItemV1
}

// OwnedKeybinding is a KeybindingV1 annotated with its contributing plugin name.
type OwnedKeybinding struct {
	PluginName string `json:"pluginName"`
	plugin.KeybindingV1
}

// OwnedMenuEntry is a MenuEntryV1 annotated with its contributing plugin name.
type OwnedMenuEntry struct {
	PluginName string `json:"pluginName"`
	plugin.MenuEntryV1
}

// OwnedActivityBarItem is an ActivityBarItemV1 annotated with its
// contributing plugin name (M2 webview slot).
type OwnedActivityBarItem struct {
	PluginName string `json:"pluginName"`
	plugin.ActivityBarItemV1
}

// OwnedView is a ViewV1 annotated with its contributing plugin name
// (M2 webview slot).
type OwnedView struct {
	PluginName string `json:"pluginName"`
	plugin.ViewV1
}

// OwnedPanel is a PanelV1 annotated with its contributing plugin name
// (M2 webview slot).
type OwnedPanel struct {
	PluginName string `json:"pluginName"`
	plugin.PanelV1
}

// FlatContributions is the materialised registry view returned to the workbench.
// Ordering within each slice is deterministic — sorted by (PluginName, ID/Key/etc)
// so the Flutter UI doesn't reshuffle between polls. StatusBar honours the
// priority field (higher priority first).
type FlatContributions struct {
	Commands    []OwnedCommand              `json:"commands"`
	StatusBar   []OwnedStatusBarItem        `json:"statusBar"`
	Keybindings []OwnedKeybinding           `json:"keybindings"`
	Menus       map[string][]OwnedMenuEntry `json:"menus"`

	// ── M2 webview slots ───────────────────────────────────────────
	ActivityBar []OwnedActivityBarItem `json:"activityBar"`
	Views       []OwnedView            `json:"views"`
	Panels      []OwnedPanel           `json:"panels"`
}

// Flatten materialises the registry into a stable, sorted view.
// Safe to call concurrently with Set/Remove.
func (r *Registry) Flatten() FlatContributions {
	r.mu.RLock()
	// Snapshot the plugin names so we hold the read lock for the minimum time.
	// We copy the map entries rather than just names so we don't re-acquire the lock.
	type entry struct {
		name string
		c    plugin.ContributesV1
	}
	entries := make([]entry, 0, len(r.byPlugin))
	for name, c := range r.byPlugin {
		entries = append(entries, entry{name: name, c: c})
	}
	r.mu.RUnlock()

	// Sort entries by plugin name for deterministic iteration.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	flat := FlatContributions{
		Commands:    make([]OwnedCommand, 0),
		StatusBar:   make([]OwnedStatusBarItem, 0),
		Keybindings: make([]OwnedKeybinding, 0),
		Menus:       make(map[string][]OwnedMenuEntry),
		ActivityBar: make([]OwnedActivityBarItem, 0),
		Views:       make([]OwnedView, 0),
		Panels:      make([]OwnedPanel, 0),
	}

	for _, e := range entries {
		for _, cmd := range e.c.Commands {
			flat.Commands = append(flat.Commands, OwnedCommand{
				PluginName: e.name,
				CommandV1:  cmd,
			})
		}
		for _, sb := range e.c.StatusBar {
			flat.StatusBar = append(flat.StatusBar, OwnedStatusBarItem{
				PluginName:      e.name,
				StatusBarItemV1: sb,
			})
		}
		for _, kb := range e.c.Keybindings {
			flat.Keybindings = append(flat.Keybindings, OwnedKeybinding{
				PluginName:   e.name,
				KeybindingV1: kb,
			})
		}
		for menuPath, menuEntries := range e.c.Menus {
			for _, me := range menuEntries {
				flat.Menus[menuPath] = append(flat.Menus[menuPath], OwnedMenuEntry{
					PluginName:  e.name,
					MenuEntryV1: me,
				})
			}
		}
		for _, ab := range e.c.ActivityBar {
			flat.ActivityBar = append(flat.ActivityBar, OwnedActivityBarItem{
				PluginName:        e.name,
				ActivityBarItemV1: ab,
			})
		}
		for _, v := range e.c.Views {
			flat.Views = append(flat.Views, OwnedView{
				PluginName: e.name,
				ViewV1:     v,
			})
		}
		for _, pn := range e.c.Panels {
			flat.Panels = append(flat.Panels, OwnedPanel{
				PluginName: e.name,
				PanelV1:    pn,
			})
		}
	}

	// ── Sort Commands: primary PluginName asc, secondary ID asc ──────────
	sort.Slice(flat.Commands, func(i, j int) bool {
		if flat.Commands[i].PluginName != flat.Commands[j].PluginName {
			return flat.Commands[i].PluginName < flat.Commands[j].PluginName
		}
		return flat.Commands[i].ID < flat.Commands[j].ID
	})

	// ── Sort StatusBar: alignment ("left" before "right"), priority desc,
	//    then PluginName asc as tie-break ──────────────────────────────────
	alignOrder := func(a string) int {
		if a == "left" {
			return 0
		}
		return 1 // "right" or anything else
	}
	sort.Slice(flat.StatusBar, func(i, j int) bool {
		ai := alignOrder(flat.StatusBar[i].Alignment)
		aj := alignOrder(flat.StatusBar[j].Alignment)
		if ai != aj {
			return ai < aj
		}
		if flat.StatusBar[i].Priority != flat.StatusBar[j].Priority {
			return flat.StatusBar[i].Priority > flat.StatusBar[j].Priority // higher first
		}
		return flat.StatusBar[i].PluginName < flat.StatusBar[j].PluginName
	})

	// ── Sort Keybindings: primary PluginName asc, secondary Key asc ──────
	sort.Slice(flat.Keybindings, func(i, j int) bool {
		if flat.Keybindings[i].PluginName != flat.Keybindings[j].PluginName {
			return flat.Keybindings[i].PluginName < flat.Keybindings[j].PluginName
		}
		return flat.Keybindings[i].Key < flat.Keybindings[j].Key
	})

	// ── Sort ActivityBar: primary PluginName asc, secondary ID asc ──────
	sort.Slice(flat.ActivityBar, func(i, j int) bool {
		if flat.ActivityBar[i].PluginName != flat.ActivityBar[j].PluginName {
			return flat.ActivityBar[i].PluginName < flat.ActivityBar[j].PluginName
		}
		return flat.ActivityBar[i].ID < flat.ActivityBar[j].ID
	})

	// ── Sort Views: primary PluginName asc, secondary ID asc ────────────
	sort.Slice(flat.Views, func(i, j int) bool {
		if flat.Views[i].PluginName != flat.Views[j].PluginName {
			return flat.Views[i].PluginName < flat.Views[j].PluginName
		}
		return flat.Views[i].ID < flat.Views[j].ID
	})

	// ── Sort Panels: primary PluginName asc, secondary ID asc ───────────
	sort.Slice(flat.Panels, func(i, j int) bool {
		if flat.Panels[i].PluginName != flat.Panels[j].PluginName {
			return flat.Panels[i].PluginName < flat.Panels[j].PluginName
		}
		return flat.Panels[i].ID < flat.Panels[j].ID
	})

	// ── Sort each menu path: primary Group asc (empty group last),
	//    secondary PluginName asc ───────────────────────────────────────────
	for path := range flat.Menus {
		entries := flat.Menus[path]
		sort.Slice(entries, func(i, j int) bool {
			gi, gj := entries[i].Group, entries[j].Group
			// Empty group sorts last.
			if gi == "" && gj != "" {
				return false
			}
			if gi != "" && gj == "" {
				return true
			}
			if gi != gj {
				return gi < gj
			}
			return entries[i].PluginName < entries[j].PluginName
		})
		flat.Menus[path] = entries
	}

	return flat
}
