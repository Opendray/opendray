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

// telegramRecentChats lists chats that have recently messaged the bot
// — drives the "Detect chat" button in the setup wizard so a non-
// technical user doesn't need @userinfobot.
//
// Workflow: user pastes bot token → bot starts polling → user sends
// /start (or anything) to their bot in Telegram → that update flows
// through the bot's pollLoop, which records the chat → UI calls this
// endpoint, shows the chat picker.
func (s *Server) telegramRecentChats(w http.ResponseWriter, r *http.Request) {
	if s.telegram == nil {
		respondError(w, http.StatusServiceUnavailable, "telegram manager not initialised")
		return
	}
	chats := s.telegram.RecentChats()
	out := make([]map[string]any, 0, len(chats))
	for _, c := range chats {
		out = append(out, map[string]any{
			"chatId":   c.ID,
			"type":     c.Type,
			"title":    c.Title,
			"username": c.Username,
			"name":     c.Name,
			"lastSeen": c.LastSeen.UnixMilli(),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{"chats": out})
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
