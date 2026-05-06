package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/wsutil"
)

const (
	eventChanBuf  = 64
	eventPingFreq = 20 * time.Second
	eventWriteTO  = 5 * time.Second
)

// EventsHandler serves the integration-only WS at /integrations/_events.
type EventsHandler struct {
	bus      *eventbus.Hub
	log      *slog.Logger
	upgrader websocket.Upgrader
}

func NewEventsHandler(bus *eventbus.Hub, log *slog.Logger) *EventsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &EventsHandler{
		bus: bus,
		log: log.With("component", "integration.events"),
		// Integration consumers may be server-to-server (no Origin
		// header) or run from a browser-based admin tab. Auth is the
		// bearer token in ?token=; routing this through wsutil
		// documents the deliberate any-origin policy and gives a
		// single grep target if we ever want to lock it down.
		upgrader: websocket.Upgrader{CheckOrigin: wsutil.AllowAnyOrigin()},
	}
}

// Serve handles GET /integrations/_events?topics=foo,bar.* — caller
// must already be auth'd as either an admin or an integration via the
// combined middleware. Per ADR 0009 admin principals subscribe without
// per-topic scope checks; integrations are still gated by
// event:subscribe:<topic>.
func (h *EventsHandler) Serve(w http.ResponseWriter, r *http.Request) {
	p, ok := CurrentPrincipal(r.Context())
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}
	isAdmin := p.Kind == KindAdmin
	isIntegration := p.Kind == KindIntegration
	if !isAdmin && !isIntegration {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}

	rawTopics := r.URL.Query().Get("topics")
	if rawTopics == "" {
		http.Error(w, "topics query param required (CSV)", http.StatusBadRequest)
		return
	}
	var allowed []string
	for _, t := range strings.Split(rawTopics, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !isAdmin && !HasScope(p.Scopes, "event:subscribe:"+t) {
			http.Error(w, "missing scope: event:subscribe:"+t, http.StatusForbidden)
			return
		}
		allowed = append(allowed, t)
	}
	if len(allowed) == 0 {
		http.Error(w, "no valid topics", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Debug("ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	subs := make([]<-chan eventbus.Event, 0, len(allowed))
	unsubs := make([]func(), 0, len(allowed))
	for _, t := range allowed {
		ch, unsub := h.bus.Subscribe(t, eventChanBuf)
		subs = append(subs, ch)
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Reader: detect client disconnect.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	// Fan-in.
	fan := make(chan eventbus.Event, eventChanBuf)
	for _, sub := range subs {
		go func(s <-chan eventbus.Event) {
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-s:
					if !ok {
						return
					}
					select {
					case fan <- ev:
					case <-ctx.Done():
						return
					}
				}
			}
		}(sub)
	}

	pinger := time.NewTicker(eventPingFreq)
	defer pinger.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-fan:
			if !ok {
				return
			}
			payload := map[string]any{
				"topic": ev.Topic,
				"ts":    ev.Time.UTC().Format(time.RFC3339Nano),
				"data":  ev.Data,
			}
			data, _ := json.Marshal(payload)
			_ = conn.SetWriteDeadline(time.Now().Add(eventWriteTO))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-pinger.C:
			if err := conn.WriteControl(websocket.PingMessage, nil,
				time.Now().Add(eventWriteTO)); err != nil {
				return
			}
		}
	}
}
