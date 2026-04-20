package gateway

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray/plugin"
)

// controlMsg is a JSON message sent over the WebSocket text channel.
type controlMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// wsConn wraps a websocket.Conn with a write mutex so multiple goroutines
// (output loop, ping keepalive, idle notifier) can safely write.
type wsConn struct {
	*websocket.Conn
	mu sync.Mutex
}

func (c *wsConn) writeMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

func (c *wsConn) writeControl(messageType int, data []byte, deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteControl(messageType, data, deadline)
}

func (c *wsConn) sendControl(msg controlMsg) error {
	data, _ := json.Marshal(msg)
	return c.writeMessage(websocket.TextMessage, data)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ts, ok := s.hub.GetTerminalSession(id)
	if !ok {
		respondError(w, http.StatusNotFound, "session not running")
		return
	}

	raw, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("ws: upgrade failed", "error", err)
		return
	}
	defer raw.Close()
	conn := &wsConn{Conn: raw}

	// Subscribe to output BEFORE snapshot to avoid data loss
	subID, outputCh := ts.Buffer().Subscribe()
	defer ts.Buffer().Unsubscribe(subID)

	conn.sendControl(controlMsg{Type: "replay_start"})

	snapshot := ts.Buffer().Snapshot()
	if len(snapshot) > 0 {
		conn.writeMessage(websocket.BinaryMessage, snapshot)
	}

	conn.sendControl(controlMsg{Type: "replay_end"})

	s.plugins.HookBus().DispatchSessionEvent(plugin.HookOnSessionStart, id)

	done := make(chan struct{})

	// Read loop: user input → PTY
	go func() {
		defer close(done)
		for {
			msgType, data, err := raw.ReadMessage()
			if err != nil {
				return
			}
			switch msgType {
			case websocket.BinaryMessage:
				ts.WriteInput(data)
			case websocket.TextMessage:
				var msg controlMsg
				if json.Unmarshal(data, &msg) == nil {
					switch msg.Type {
					case "resize":
						ts.Resize(msg.Rows, msg.Cols)
					}
				} else {
					ts.WriteInput(data)
				}
			}
		}
	}()

	// Ping keepalive
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ts.Done():
				return
			case <-ticker.C:
				if err := conn.writeControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			}
		}
	}()

	// Idle detection: notify client when session is waiting for input.
	// NOTE: HookOnIdle is NOT dispatched here — hub.watchIdle is the sole
	// authority for idle hook events. This goroutine only sends the
	// "waiting_for_input" control message to the WebSocket client.
	go func() {
		for {
			ch := ts.IdleCh()
			select {
			case <-done:
				return
			case <-ts.Done():
				return
			case <-ch:
				if err := conn.sendControl(controlMsg{Type: "waiting_for_input"}); err != nil {
					return
				}
			}
			// Phase 2: wait for the session to produce new output (which
			// re-arms the idle channel) before looping back. Without this
			// we'd spin on the already-closed channel.
			ticker := time.NewTicker(2 * time.Second)
			waiting := true
			for waiting {
				select {
				case <-done:
					ticker.Stop()
					return
				case <-ts.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					if ts.IdleCh() != ch {
						waiting = false
					}
				}
			}
			ticker.Stop()
		}
	}()

	// Write loop: PTY output → WebSocket
	for {
		select {
		case <-done:
			return
		case <-ts.Done():
			conn.sendControl(controlMsg{Type: "process_exit"})
			return
		case chunk, ok := <-outputCh:
			if !ok {
				return
			}
			if err := conn.writeMessage(websocket.BinaryMessage, chunk); err != nil {
				return
			}
		}
	}
}
