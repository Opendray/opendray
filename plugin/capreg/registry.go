// Package capreg holds the in-memory join of every plugin's runtime
// capability registrations: providers, channels, forges, MCP servers.
//
// It is the runtime sibling of plugin/contributions: contributions
// tracks UI slots (commands, views, menus) declared in manifests and
// rendered by the workbench; capreg tracks code-backed capabilities
// registered by plugins at activation time and consumed by the gateway
// (LLM router, source-control surface, MCP dispatcher, channel hub).
//
// Lookups are by capability kind + id. Each id is globally unique
// across all loaded plugins; a duplicate registration returns
// api.ErrDuplicateCapability and the second registration is rejected.
//
// Thread-safe for concurrent Register*/Get*/RemovePlugin/List calls.
package capreg

import (
	"fmt"
	"sort"
	"sync"

	"github.com/opendray/opendray/plugin/api"
)

// Registry is the capability-by-id index. Construct with New.
type Registry struct {
	mu sync.RWMutex

	providers  map[string]ownedProvider
	channels   map[string]ownedChannel
	forges     map[string]ownedForge
	mcpServers map[string]ownedMcpServer
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{
		providers:  make(map[string]ownedProvider),
		channels:   make(map[string]ownedChannel),
		forges:     make(map[string]ownedForge),
		mcpServers: make(map[string]ownedMcpServer),
	}
}

// ── Per-kind owned wrappers (private) ──────────────────────────────────

type ownedProvider struct {
	plugin string
	impl   api.Provider
}

type ownedChannel struct {
	plugin string
	impl   api.Channel
}

type ownedForge struct {
	plugin string
	impl   api.Forge
}

type ownedMcpServer struct {
	plugin string
	impl   api.McpServer
}

// ── Register* ──────────────────────────────────────────────────────────

// RegisterProvider claims provider id p.ID() for plugin. Returns
// api.ErrDuplicateCapability if id is already taken (whether by this
// plugin or another).
func (r *Registry) RegisterProvider(plugin string, p api.Provider) error {
	if p == nil {
		return fmt.Errorf("capreg: provider impl is nil")
	}
	id := p.ID()
	if id == "" {
		return fmt.Errorf("capreg: provider id is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.providers[id]; ok {
		return fmt.Errorf("%w: provider %q owned by plugin %q", api.ErrDuplicateCapability, id, existing.plugin)
	}
	r.providers[id] = ownedProvider{plugin: plugin, impl: p}
	return nil
}

// RegisterChannel claims channel id c.ID() for plugin.
func (r *Registry) RegisterChannel(plugin string, c api.Channel) error {
	if c == nil {
		return fmt.Errorf("capreg: channel impl is nil")
	}
	id := c.ID()
	if id == "" {
		return fmt.Errorf("capreg: channel id is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.channels[id]; ok {
		return fmt.Errorf("%w: channel %q owned by plugin %q", api.ErrDuplicateCapability, id, existing.plugin)
	}
	r.channels[id] = ownedChannel{plugin: plugin, impl: c}
	return nil
}

// RegisterForge claims forge id f.ID() for plugin.
func (r *Registry) RegisterForge(plugin string, f api.Forge) error {
	if f == nil {
		return fmt.Errorf("capreg: forge impl is nil")
	}
	id := f.ID()
	if id == "" {
		return fmt.Errorf("capreg: forge id is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.forges[id]; ok {
		return fmt.Errorf("%w: forge %q owned by plugin %q", api.ErrDuplicateCapability, id, existing.plugin)
	}
	r.forges[id] = ownedForge{plugin: plugin, impl: f}
	return nil
}

// RegisterMcpServer claims MCP-server id m.ID() for plugin.
func (r *Registry) RegisterMcpServer(plugin string, m api.McpServer) error {
	if m == nil {
		return fmt.Errorf("capreg: mcp server impl is nil")
	}
	id := m.ID()
	if id == "" {
		return fmt.Errorf("capreg: mcp server id is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.mcpServers[id]; ok {
		return fmt.Errorf("%w: mcp server %q owned by plugin %q", api.ErrDuplicateCapability, id, existing.plugin)
	}
	r.mcpServers[id] = ownedMcpServer{plugin: plugin, impl: m}
	return nil
}

// ── Get* ───────────────────────────────────────────────────────────────

// Provider returns the registered provider for id or (nil,false).
func (r *Registry) Provider(id string) (api.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.providers[id]
	if !ok {
		return nil, false
	}
	return o.impl, true
}

// Channel returns the registered channel for id or (nil,false).
func (r *Registry) Channel(id string) (api.Channel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.channels[id]
	if !ok {
		return nil, false
	}
	return o.impl, true
}

// Forge returns the registered forge for id or (nil,false).
func (r *Registry) Forge(id string) (api.Forge, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.forges[id]
	if !ok {
		return nil, false
	}
	return o.impl, true
}

// McpServer returns the registered MCP server for id or (nil,false).
func (r *Registry) McpServer(id string) (api.McpServer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.mcpServers[id]
	if !ok {
		return nil, false
	}
	return o.impl, true
}

// ── Listings ────────────────────────────────────────────────────────────

// Entry is the public listing element used by Flat() and the Owners
// API. Sorted by (Kind, ID) when returned from a List* method.
type Entry struct {
	Kind   string `json:"kind"`   // "provider" | "channel" | "forge" | "mcpServer"
	ID     string `json:"id"`
	Plugin string `json:"plugin"` // plugin name that registered this entry
}

// ListProviders returns all registered providers, sorted by id.
func (r *Registry) ListProviders() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.providers))
	for id, o := range r.providers {
		out = append(out, Entry{Kind: "provider", ID: id, Plugin: o.plugin})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ListChannels returns all registered channels, sorted by id.
func (r *Registry) ListChannels() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.channels))
	for id, o := range r.channels {
		out = append(out, Entry{Kind: "channel", ID: id, Plugin: o.plugin})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ListForges returns all registered forges, sorted by id.
func (r *Registry) ListForges() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.forges))
	for id, o := range r.forges {
		out = append(out, Entry{Kind: "forge", ID: id, Plugin: o.plugin})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ListMcpServers returns all registered MCP servers, sorted by id.
func (r *Registry) ListMcpServers() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.mcpServers))
	for id, o := range r.mcpServers {
		out = append(out, Entry{Kind: "mcpServer", ID: id, Plugin: o.plugin})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// All returns every registered capability of every kind, sorted by
// (Kind, ID). Stable for diagnostic snapshots.
func (r *Registry) All() []Entry {
	out := r.ListProviders()
	out = append(out, r.ListChannels()...)
	out = append(out, r.ListForges()...)
	out = append(out, r.ListMcpServers()...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ── Removal ─────────────────────────────────────────────────────────────

// RemovePlugin drops every capability registered under plugin. Safe to
// call for a plugin that has no registrations. Used at unload time.
func (r *Registry) RemovePlugin(plugin string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, o := range r.providers {
		if o.plugin == plugin {
			delete(r.providers, id)
		}
	}
	for id, o := range r.channels {
		if o.plugin == plugin {
			delete(r.channels, id)
		}
	}
	for id, o := range r.forges {
		if o.plugin == plugin {
			delete(r.forges, id)
		}
	}
	for id, o := range r.mcpServers {
		if o.plugin == plugin {
			delete(r.mcpServers, id)
		}
	}
}
