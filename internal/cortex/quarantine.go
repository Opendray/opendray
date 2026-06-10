package cortex

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/memory"
)

// QuarantineSource is the slice of *memory.Service the review queue
// needs. Nil when memory is disabled.
type QuarantineSource interface {
	ListQuarantined(ctx context.Context, limit int) ([]memory.Memory, error)
	CountQuarantined(ctx context.Context) (int, error)
	PromoteQuarantined(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

// mountQuarantine registers the quarantine review queue under the
// already-entered /cortex route group:
//
//	GET  /memory/quarantine              → {memories, count}
//	POST /memory/quarantine/{id}/promote → 204 (tier → durable)
//	POST /memory/quarantine/{id}/discard → 204 (delete)
//
// Registered BEFORE the re-mounted memory handlers so chi matches the
// static /quarantine segment ahead of memory's /{id} wildcard.
func (h *Handlers) mountQuarantine(r chi.Router) {
	r.Route("/memory/quarantine", func(r chi.Router) {
		r.Get("/", h.listQuarantine)
		r.Post("/{id}/promote", h.promoteQuarantine)
		r.Post("/{id}/discard", h.discardQuarantine)
	})
}

func (h *Handlers) quarantineReady(w http.ResponseWriter) bool {
	if h.quarantine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "memory disabled"})
		return false
	}
	return true
}

func (h *Handlers) listQuarantine(w http.ResponseWriter, r *http.Request) {
	if !h.quarantineReady(w) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("n"))
	rows, err := h.quarantine.ListQuarantined(r.Context(), limit)
	if err != nil {
		h.log.Error("quarantine list failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	if rows == nil {
		rows = []memory.Memory{}
	}
	count, err := h.quarantine.CountQuarantined(r.Context())
	if err != nil {
		h.log.Warn("quarantine count failed", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"memories": rows, "count": count})
}

func (h *Handlers) promoteQuarantine(w http.ResponseWriter, r *http.Request) {
	if !h.quarantineReady(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.quarantine.PromoteQuarantined(r.Context(), id); err != nil {
		if errors.Is(err, memory.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not a quarantined memory"})
			return
		}
		h.log.Error("quarantine promote failed", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "promote failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) discardQuarantine(w http.ResponseWriter, r *http.Request) {
	if !h.quarantineReady(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.quarantine.Delete(r.Context(), id); err != nil {
		if errors.Is(err, memory.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		h.log.Error("quarantine discard failed", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "discard failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
