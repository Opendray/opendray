package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Hook event types that plugins can subscribe to.
const (
	HookOnOutput       = "onOutput"
	HookOnIdle         = "onIdle"
	HookOnSessionStart = "onSessionStart"
	HookOnSessionStop  = "onSessionStop"
)

// HookEvent is the payload sent to plugin hook callbacks.
type HookEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId,omitempty"`
	Data      string `json:"data,omitempty"` // base64 for binary, plain for text
	Timestamp int64  `json:"timestamp"`
}

// subscriber tracks a plugin's hook callback registration.
type subscriber struct {
	pluginName  string
	callbackURL string
	hooks       map[string]bool
}

// LocalListener is an in-process hook callback. Used by Go-side bridges
// (e.g. the Telegram plugin) so they can react to events without running
// a HTTP server inside the same process. The callback runs in a goroutine
// — it must not block.
type LocalListener struct {
	id    uint64
	hooks map[string]bool
	fn    func(HookEvent)
}

// HookBus dispatches terminal events to subscribed plugins.
type HookBus struct {
	mu          sync.RWMutex
	subscribers map[string]*subscriber // pluginName → subscriber (HTTP)
	locals      map[uint64]*LocalListener
	nextLocalID uint64
	client      *http.Client
	logger      *slog.Logger
}

// NewHookBus creates a hook event dispatcher.
func NewHookBus(logger *slog.Logger) *HookBus {
	if logger == nil {
		logger = slog.Default()
	}
	return &HookBus{
		subscribers: make(map[string]*subscriber),
		locals:      make(map[uint64]*LocalListener),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// SubscribeLocal registers an in-process listener for the given hook types.
// Returns an unsubscribe function — callers MUST call it on shutdown to
// avoid leaking goroutines / closures.
func (hb *HookBus) SubscribeLocal(hooks []string, fn func(HookEvent)) func() {
	hb.mu.Lock()
	hb.nextLocalID++
	id := hb.nextLocalID
	set := make(map[string]bool, len(hooks))
	for _, h := range hooks {
		set[h] = true
	}
	hb.locals[id] = &LocalListener{id: id, hooks: set, fn: fn}
	hb.mu.Unlock()
	return func() {
		hb.mu.Lock()
		delete(hb.locals, id)
		hb.mu.Unlock()
	}
}

// Register adds or updates a plugin's hook subscriptions.
func (hb *HookBus) Register(pluginName, callbackURL string, hooks []string) {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	hookSet := make(map[string]bool, len(hooks))
	for _, h := range hooks {
		hookSet[h] = true
	}

	hb.subscribers[pluginName] = &subscriber{
		pluginName:  pluginName,
		callbackURL: callbackURL,
		hooks:       hookSet,
	}
}

// Unregister removes a plugin's hook subscriptions.
func (hb *HookBus) Unregister(pluginName string) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	delete(hb.subscribers, pluginName)
}

// Dispatch sends an event to all plugins subscribed to the given hook type.
// This is non-blocking; callbacks are fired in goroutines.
func (hb *HookBus) Dispatch(event HookEvent) {
	hb.mu.RLock()
	defer hb.mu.RUnlock()

	for _, sub := range hb.subscribers {
		if !sub.hooks[event.Type] {
			continue
		}
		go hb.deliver(sub, event)
	}
	for _, lis := range hb.locals {
		if !lis.hooks[event.Type] {
			continue
		}
		fn := lis.fn
		go func() {
			defer func() {
				if r := recover(); r != nil {
					hb.logger.Warn("local hook listener panic", "error", r)
				}
			}()
			fn(event)
		}()
	}
}

func (hb *HookBus) deliver(sub *subscriber, event HookEvent) {
	body, err := json.Marshal(event)
	if err != nil {
		return
	}

	resp, err := hb.client.Post(sub.callbackURL, "application/json", bytes.NewReader(body))
	if err != nil {
		hb.logger.Warn("hook delivery failed",
			"plugin", sub.pluginName,
			"hook", event.Type,
			"error", err,
		)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		hb.logger.Warn("hook callback returned error",
			"plugin", sub.pluginName,
			"hook", event.Type,
			"status", resp.StatusCode,
		)
	}
}

// ListSubscriptions returns a summary of all registered hook subscriptions.
func (hb *HookBus) ListSubscriptions() map[string][]string {
	hb.mu.RLock()
	defer hb.mu.RUnlock()

	result := make(map[string][]string, len(hb.subscribers))
	for name, sub := range hb.subscribers {
		hooks := make([]string, 0, len(sub.hooks))
		for h := range sub.hooks {
			hooks = append(hooks, h)
		}
		result[name] = hooks
	}
	return result
}

// DispatchSessionEvent is a convenience wrapper for session lifecycle events.
func (hb *HookBus) DispatchSessionEvent(hookType, sessionID string) {
	hb.Dispatch(HookEvent{
		Type:      hookType,
		SessionID: sessionID,
		Timestamp: time.Now().UnixMilli(),
	})
}

// DispatchOutput sends an output event. For efficiency, only dispatches
// if any subscriber is listening for onOutput.
func (hb *HookBus) DispatchOutput(sessionID string, data []byte) {
	hb.mu.RLock()
	hasListener := false
	for _, sub := range hb.subscribers {
		if sub.hooks[HookOnOutput] {
			hasListener = true
			break
		}
	}
	if !hasListener {
		for _, lis := range hb.locals {
			if lis.hooks[HookOnOutput] {
				hasListener = true
				break
			}
		}
	}
	hb.mu.RUnlock()

	if !hasListener {
		return
	}

	hb.Dispatch(HookEvent{
		Type:      HookOnOutput,
		SessionID: sessionID,
		Data:      fmt.Sprintf("%s", data),
		Timestamp: time.Now().UnixMilli(),
	})
}
