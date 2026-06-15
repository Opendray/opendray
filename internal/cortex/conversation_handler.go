package cortex

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// mountConversations registers the curation-conversation routes under
// the already-entered /cortex group:
//
//	POST /conversations                     body: {target_kind, target_cwd, target_slug}
//	GET  /conversations?cwd=&slug=          → {conversations}
//	GET  /conversations/{id}                → {conversation, messages}
//	POST /conversations/{id}/messages       body: {content} → operator Message
//	     (AI reply lands async; listen for eventbus topic
//	      "cortex.conversation.reply" or poll GET /{id})
//	POST /conversations/{id}/escalate       → Conversation (full agent session)
//	POST /conversations/{id}/close          → 204
func (h *Handlers) mountConversations(r chi.Router) {
	r.Route("/conversations", func(r chi.Router) {
		r.Post("/", h.createConversation)
		r.Get("/", h.listConversations)
		r.Get("/{id}", h.getConversation)
		r.Post("/{id}/messages", h.sendConversationMessage)
		r.Post("/{id}/provider", h.setConversationProvider)
		r.Post("/{id}/escalate", h.escalateConversation)
		r.Post("/{id}/close", h.closeConversation)
	})
}

func (h *Handlers) curationReady(w http.ResponseWriter) bool {
	if h.curation == nil || h.convStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "curation not configured"})
		return false
	}
	return true
}

func (h *Handlers) createConversation(w http.ResponseWriter, r *http.Request) {
	if !h.curationReady(w) {
		return
	}
	var body struct {
		TargetKind   string `json:"target_kind"`
		TargetCwd    string `json:"target_cwd"`
		TargetSlug   string `json:"target_slug"`
		ProviderID   string `json:"provider_id"`
		Model        string `json:"model"`
		SummarizerID string `json:"summarizer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	conv, err := h.convStore.Create(r.Context(), body.TargetKind, body.TargetCwd, body.TargetSlug, body.ProviderID, body.Model, body.SummarizerID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

// setConversationProvider pins (or clears with empty provider_id) the
// conversation's cloud-agent model override. Returns the updated row.
func (h *Handlers) setConversationProvider(w http.ResponseWriter, r *http.Request) {
	if !h.curationReady(w) {
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		ProviderID   string `json:"provider_id"`
		Model        string `json:"model"`
		SummarizerID string `json:"summarizer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.convStore.SetProvider(r.Context(), id, body.ProviderID, body.Model, body.SummarizerID); err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	conv, err := h.convStore.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (h *Handlers) listConversations(w http.ResponseWriter, r *http.Request) {
	if !h.curationReady(w) {
		return
	}
	q := r.URL.Query()
	convs, err := h.convStore.ListByTarget(r.Context(), q.Get("cwd"), q.Get("slug"), 0)
	if err != nil {
		h.log.Error("list conversations failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	if convs == nil {
		convs = []Conversation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": convs})
}

func (h *Handlers) getConversation(w http.ResponseWriter, r *http.Request) {
	if !h.curationReady(w) {
		return
	}
	id := chi.URLParam(r, "id")
	conv, err := h.convStore.Get(r.Context(), id)
	if errors.Is(err, ErrConversationNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	msgs, err := h.convStore.Messages(r.Context(), id, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if msgs == nil {
		msgs = []Message{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs})
}

func (h *Handlers) sendConversationMessage(w http.ResponseWriter, r *http.Request) {
	if !h.curationReady(w) {
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	msg, err := h.curation.Send(r.Context(), chi.URLParam(r, "id"), body.Content)
	if errors.Is(err, ErrConversationNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, msg)
}

func (h *Handlers) escalateConversation(w http.ResponseWriter, r *http.Request) {
	if !h.curationReady(w) {
		return
	}
	conv, err := h.curation.Escalate(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrConversationNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		h.log.Error("escalate failed", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (h *Handlers) closeConversation(w http.ResponseWriter, r *http.Request) {
	if !h.curationReady(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.convStore.SetStatus(r.Context(), id, ConvClosed, ""); err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
