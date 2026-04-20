package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/image/draw"

	"net/http"
)

// simulatorStreamWS upgrades to a WebSocket and streams JPEG screenshots
// at adaptive FPS. Replaces the HTTP-per-frame polling model.
//
// Protocol:
//   Server → Client (text):   {"type":"size","width":W,"height":H}   (first frame)
//   Server → Client (binary): JPEG bytes                             (each frame)
//   Client → Server (text):   {"type":"tap|swipe|keyevent|text",...}  (input events)
//   Client → Server (text):   {"type":"quality","fps":N,"q":N}       (quality request)
//
// The server captures at up to 8 FPS during interaction, drops to 1 FPS
// after 5 seconds of no input. JPEG quality 50 brings frame size from
// 1-3 MB (PNG) down to 30-80 KB.
func (s *Server) simulatorStreamWS(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "android"
	}
	device := r.URL.Query().Get("device")

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Read plugin config for quality/FPS settings
	activeFPS, idleFPS, jpegQ, maxWidth := 8, 1, 50, 720
	for _, pi := range s.plugins.ListInfo() {
		if pi.Provider.Name == "simulator-preview" && pi.Enabled {
			if v, ok := pi.Config["activeFps"].(float64); ok && v > 0 && v <= 15 {
				activeFPS = int(v)
			}
			if v, ok := pi.Config["idleFps"].(float64); ok && v >= 0 {
				idleFPS = int(v)
			}
			if v, ok := pi.Config["quality"].(float64); ok && v >= 10 && v <= 95 {
				jpegQ = int(v)
			}
			if v, ok := pi.Config["maxWidth"].(float64); ok {
				maxWidth = int(v)
			}
			break
		}
	}

	stream := &simStream{
		platform:   platform,
		device:     device,
		conn:       conn,
		logger:     s.logger,
		activeFPS:  activeFPS,
		idleFPS:    idleFPS,
		jpegQ:      jpegQ,
		maxWidth:   maxWidth,
		lastInput:  time.Now(),
		idleAfter:  5 * time.Second,
	}

	// Read loop: input events + quality changes
	go stream.readLoop(ctx, cancel)
	// Write loop: screenshot frames
	stream.writeLoop(ctx)
}

type simStream struct {
	platform  string
	device    string
	conn      *websocket.Conn
	logger    *slog.Logger
	activeFPS int
	idleFPS   int
	jpegQ     int
	maxWidth  int
	idleAfter time.Duration

	mu        sync.Mutex
	lastInput time.Time
	sentSize  bool
}

func (ss *simStream) readLoop(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	for {
		_, raw, err := ss.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Type     string `json:"type"`
			Action   string `json:"action"`
			X        int    `json:"x"`
			Y        int    `json:"y"`
			X2       int    `json:"x2"`
			Y2       int    `json:"y2"`
			Duration int    `json:"duration"`
			Key      string `json:"key"`
			Text     string `json:"text"`
			FPS      int    `json:"fps"`
			Q        int    `json:"q"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		ss.mu.Lock()
		ss.lastInput = time.Now()
		ss.mu.Unlock()

		switch msg.Type {
		case "tap", "swipe", "keyevent", "text":
			_ = sendInput(ctx, ss.platform, ss.device, msg.Type,
				msg.X, msg.Y, msg.X2, msg.Y2, msg.Duration, msg.Key, msg.Text)
		case "quality":
			if msg.FPS > 0 && msg.FPS <= 15 {
				ss.activeFPS = msg.FPS
			}
			if msg.Q > 10 && msg.Q <= 95 {
				ss.jpegQ = msg.Q
			}
		}
	}
}

func (ss *simStream) writeLoop(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		// Adaptive FPS: fast during interaction, slow when idle
		ss.mu.Lock()
		idle := time.Since(ss.lastInput) > ss.idleAfter
		ss.mu.Unlock()
		fps := ss.activeFPS
		if idle {
			fps = ss.idleFPS
		}
		interval := time.Second / time.Duration(fps)

		start := time.Now()
		if err := ss.captureAndSend(ctx); err != nil {
			ss.logger.Debug("simulator stream: capture failed", "error", err)
			// Don't hammer on failure — wait a bit
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		// Sleep for the remaining interval
		elapsed := time.Since(start)
		if elapsed < interval {
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval - elapsed):
			}
		}
	}
}

func (ss *simStream) captureAndSend(ctx context.Context) error {
	// Capture PNG
	pngData, err := captureScreenshot(ctx, ss.platform, ss.device)
	if err != nil {
		return err
	}

	// Decode PNG
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Send size on first frame
	if !ss.sentSize {
		sizeMsg, _ := json.Marshal(map[string]any{
			"type": "size", "width": w, "height": h,
		})
		_ = ss.conn.WriteMessage(websocket.TextMessage, sizeMsg)
		ss.sentSize = true
	}

	// Scale down if wider than maxWidth (saves bandwidth on mobile)
	if ss.maxWidth > 0 && w > ss.maxWidth {
		ratio := float64(ss.maxWidth) / float64(w)
		newW := ss.maxWidth
		newH := int(float64(h) * ratio)
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
		img = dst
	}

	// Encode to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: ss.jpegQ}); err != nil {
		return err
	}

	// Send binary frame
	return ss.conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
}
