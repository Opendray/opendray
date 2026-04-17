package gateway

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opendray/opendray/gateway/telegram"
)

// telegramStatus returns the current bot connection status — used by the
// frontend Messaging panel to show "@bot connected" / "offline" / errors.
func (s *Server) telegramStatus(w http.ResponseWriter, r *http.Request) {
	if s.telegram == nil {
		respondError(w, http.StatusServiceUnavailable, "telegram manager not initialised")
		return
	}
	respondJSON(w, http.StatusOK, s.telegram.Snapshot())
}

// telegramTest sends a test message to the configured notification chat.
func (s *Server) telegramTest(w http.ResponseWriter, r *http.Request) {
	if s.telegram == nil {
		respondError(w, http.StatusServiceUnavailable, "telegram manager not initialised")
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body) // ignore — empty body is fine
	}
	chatID, err := s.telegram.SendTest(r.Context(), body.Text)
	if err != nil {
		// Sentinel errors → 400 (user fixable). Network errors → 502.
		var serr interface{ Error() string }
		if errors.As(err, &serr) {
			respondError(w, http.StatusBadRequest, err.Error())
		} else {
			respondError(w, http.StatusBadGateway, err.Error())
		}
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"sent": true,
		"chat": chatID,
	})
}

// telegramLinks lists active chat ↔ session bindings for the panel page.
func (s *Server) telegramLinks(w http.ResponseWriter, r *http.Request) {
	if s.telegram == nil {
		respondError(w, http.StatusServiceUnavailable, "telegram manager not initialised")
		return
	}
	links := s.telegram.Links().All()
	out := make([]map[string]any, 0, len(links))
	for _, l := range links {
		out = append(out, map[string]any{
			"chatId":    l.ChatID,
			"sessionId": l.SessionID,
			"linkedAt":  l.LinkedAt.UnixMilli(),
		})
	}
	respondJSON(w, http.StatusOK, out)
}

// telegramUnlink removes a link from the panel UI (analogue of /unlink in
// Telegram itself).
func (s *Server) telegramUnlink(w http.ResponseWriter, r *http.Request) {
	if s.telegram == nil {
		respondError(w, http.StatusServiceUnavailable, "telegram manager not initialised")
		return
	}
	var body struct {
		ChatID int64 `json:"chatId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ChatID == 0 {
		respondError(w, http.StatusBadRequest, "chatId required")
		return
	}
	prior := s.telegram.Links().Remove(body.ChatID)
	respondJSON(w, http.StatusOK, map[string]any{
		"removed":   prior != "",
		"sessionId": prior,
	})
}

// Compile-time guard that we're using the imported package.
var _ = telegram.Status{}
