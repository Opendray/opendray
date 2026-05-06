// Package bridge — HTTP/WebSocket entry point for external adapters.
//
// The single endpoint /api/v1/channels/bridge/ws upgrades a request
// to WebSocket, reads the first frame (which MUST be type=register
// with a valid token), looks up the matching Bridge channel via the
// broker, and hands the connection over.
package bridge

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/channel"
	"github.com/opendray/opendray-v2/internal/wsutil"
)

// Handlers wires the bridge WS endpoint into an admin-less router.
// The endpoint authenticates by token (carried in the register frame
// or via Authorization/X-Bridge-Token headers / ?token= query).
type Handlers struct {
	broker *Broker
	log    *slog.Logger

	upgrader websocket.Upgrader
}

// NewHandlers builds the public WS handler bound to the given broker.
// Pass DefaultBroker() unless writing tests.
func NewHandlers(b *Broker, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{
		broker: b,
		log:    log.With("component", "channel.bridge.http"),
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
			ReadBufferSize:   16 * 1024,
			WriteBufferSize:  16 * 1024,
			// Bridge adapters connect from outside the browser — origin
			// checks would block legitimate use. The token in the
			// register frame is the real auth.
			CheckOrigin: wsutil.AllowAnyOrigin(),
		},
	}
}

// Mount adds /channels/bridge/ws to the public (non-admin) router.
// Token-based auth makes this safe to expose without the admin
// bearer-token middleware.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/channels/bridge/ws", h.serve)
}

// registerFrame is the first message every adapter must send.
type registerFrame struct {
	Type         string               `json:"type"`
	Token        string               `json:"token,omitempty"`
	Platform     string               `json:"platform"`
	Capabilities []channel.Capability `json:"capabilities"`
	Metadata     map[string]any       `json:"metadata,omitempty"`
}

func (h *Handlers) serve(w http.ResponseWriter, r *http.Request) {
	headerToken := extractToken(r)
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade itself wrote the error response.
		return
	}

	if err := conn.SetReadDeadline(time.Now().Add(registerWait)); err != nil {
		_ = conn.Close()
		return
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		h.log.Warn("bridge: register read failed", "err", err)
		_ = conn.Close()
		return
	}
	var reg registerFrame
	if err := json.Unmarshal(raw, &reg); err != nil || reg.Type != "register" {
		writeAck(conn, false, "first frame must be type=register")
		_ = conn.Close()
		return
	}
	token := headerToken
	if token == "" {
		token = reg.Token
	}
	if token == "" {
		writeAck(conn, false, "missing token")
		_ = conn.Close()
		return
	}
	br := h.broker.LookupByToken(token)
	if br == nil {
		writeAck(conn, false, "invalid token")
		_ = conn.Close()
		return
	}
	platform := strings.TrimSpace(reg.Platform)
	if platform == "" {
		platform = br.cfg.Name
	}
	// Send the ack before handing the conn to the bridge — once
	// attach() returns the read/ping pumps own the socket and a
	// concurrent writeAck would race with subsequent outbound frames.
	writeAck(conn, true, "")
	br.attach(conn, reg.Capabilities, platform)
}

func writeAck(conn *websocket.Conn, ok bool, msg string) {
	payload := map[string]any{"type": "register_ack", "ok": ok}
	if msg != "" {
		payload["error"] = msg
	}
	raw, _ := json.Marshal(payload)
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = conn.WriteMessage(websocket.TextMessage, raw)
}

// extractToken pulls the bridge token from one of the three accepted
// transports: Authorization: Bearer X, X-Bridge-Token: X, ?token=X.
// Returns "" when none present (the register frame may still carry it).
func extractToken(r *http.Request) string {
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	if t := r.Header.Get("X-Bridge-Token"); t != "" {
		return t
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[len("Bearer "):])
	}
	return ""
}
