package session

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/wsutil"
)

// Service is the session-manager surface used by HTTP handlers,
// decoupled from *Manager so handlers can be tested without spawning
// real PTYs.
type Service interface {
	Create(ctx context.Context, req CreateRequest) (Session, error)
	Start(ctx context.Context, id string) (Session, error)
	Get(ctx context.Context, id string) (Session, error)
	List(ctx context.Context) ([]Session, error)
	Stop(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	Input(ctx context.Context, id string, data []byte) error
	Resize(ctx context.Context, id string, cols, rows uint16) error
	Subscribe(ctx context.Context, id string) (<-chan []byte, func(), error)
	Buffer(ctx context.Context, id string, since int64) (Replay, error)
	SwitchClaudeAccount(ctx context.Context, id, accountID string) (Session, error)
	History(ctx context.Context, id string, limit int) (HistoryResponse, error)
}

type Handlers struct {
	svc      Service
	log      *slog.Logger
	upgrader websocket.Upgrader
}

func NewHandlers(svc Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{
		svc: svc,
		log: log.With("component", "session.http"),
		upgrader: websocket.Upgrader{
			// Admin SPA in the browser. Token is in the ?token= query
			// (browsers can't set custom headers on WS handshake), so
			// CSWSH is mitigated by also checking Origin: same-host or
			// LAN private ranges only. Non-browser clients (mobile,
			// curl) send no Origin and are admitted as before.
			CheckOrigin: wsutil.SameOriginCheck(),
		},
	}
}

// Mount adds the session routes to the given chi.Router. Caller mounts
// this under /api/v1.
//
// Lifecycle: POST / creates+spawns; POST /{id}/start re-spawns a
// terminal row; POST /{id}/stop terminates the process but keeps the
// row; DELETE /{id} terminates and removes.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/sessions", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Delete("/", h.remove)
			r.Post("/start", h.start)
			r.Post("/stop", h.stop)
			r.Post("/input", h.input)
			r.Post("/resize", h.resize)
			r.Get("/buffer", h.buffer)
			r.Get("/stream", h.stream)
			r.Get("/history", h.history)
			r.Patch("/claude-account", h.switchClaudeAccount)
		})
	})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sess, err := h.svc.Create(r.Context(), req)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []Session{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

// remove handles DELETE /sessions/{id}. Running sessions are SIGTERMed
// first; then the DB row is dropped.
func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Remove(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// stop handles POST /sessions/{id}/stop. Process is terminated; the
// DB row remains so the user can Start it again.
func (h *Handlers) stop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Stop(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	sess, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

// start handles POST /sessions/{id}/start. Re-spawns a previously
// stopped or ended session under the original provider/cwd/args/account.
func (h *Handlers) start(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := h.svc.Start(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *Handlers) input(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req InputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.Input(r.Context(), id, []byte(req.Data)); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) resize(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req ResizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Cols == 0 || req.Rows == 0 {
		writeError(w, http.StatusBadRequest, errors.New("cols and rows must be > 0"))
		return
	}
	if err := h.svc.Resize(r.Context(), id, req.Cols, req.Rows); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) buffer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	since := int64(0)
	if v := r.URL.Query().Get("since"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, errors.New("invalid since: must be non-negative integer"))
			return
		}
		since = n
	}
	rep, err := h.svc.Buffer(r.Context(), id, since)
	if err != nil {
		h.respondError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-OpenDray-Buffer-Start", strconv.FormatInt(rep.Start, 10))
	w.Header().Set("X-OpenDray-Buffer-Cursor", strconv.FormatInt(rep.Written, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(rep.Bytes)
}

func (h *Handlers) stream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, unsub, err := h.svc.Subscribe(r.Context(), id)
	if err != nil {
		// For ended/stopped sessions, complete the WebSocket handshake
		// and send a clean close (1001 going-away) so the client's
		// reconnect loop latches onto a real "ended" signal instead of
		// firing repeated 404s through the proxy. Browsers treat
		// pre-upgrade HTTP errors as abnormal closes (code 1006), which
		// our reconnect logic interprets as a transient failure and
		// keeps retrying.
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrAlreadyEnded) {
			h.streamCloseEnded(w, r)
			return
		}
		h.respondError(w, err)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		unsub()
		h.log.Debug("ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()
	defer unsub()

	if rep, err := h.svc.Buffer(r.Context(), id, 0); err == nil && len(rep.Bytes) > 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, rep.Bytes); err != nil {
			return
		}
	}

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for data := range ch {
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		}
	}()

	conn.SetReadLimit(64 * 1024)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if len(data) == 0 {
			continue
		}
		if err := h.svc.Input(r.Context(), id, data); err != nil {
			h.log.Debug("ws input error", "session", id, "err", err)
			break
		}
	}

	select {
	case <-writerDone:
	case <-time.After(time.Second):
	}
}

// streamCloseEnded completes the WebSocket handshake and immediately
// sends a normal-close (1001 going-away) frame so the client's
// reconnect loop sees a clean termination instead of HTTP 404 →
// abnormal close → retry. Optional JSON `{type:"ended"}` payload
// gives clients that want it a richer signal before the close.
func (h *Handlers) streamCloseEnded(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade failed; nothing to do — client will see HTTP error.
		return
	}
	defer conn.Close()
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ended"}`))
	closeMsg := websocket.FormatCloseMessage(
		websocket.CloseGoingAway,
		"session ended",
	)
	_ = conn.WriteControl(
		websocket.CloseMessage,
		closeMsg,
		time.Now().Add(time.Second),
	)
}

// history handles GET /sessions/{id}/history. Returns up to `limit`
// (default 200, max 1000) of the user's past prompts in this project,
// pulled from Claude's JSONL transcripts under ~/.claude/projects.
//
// Sessions whose provider isn't Claude (codex/gemini/shell) get
// {entries: [], unsupported_provider: true} so the UI can render the
// right empty state without a separate /providers lookup.
func (h *Handlers) history(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, errors.New("invalid limit: must be non-negative integer"))
			return
		}
		if n > 1000 {
			n = 1000
		}
		limit = n
	}
	res, err := h.svc.History(r.Context(), id, limit)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// switchClaudeAccount handles PATCH /sessions/{id}/claude-account.
// Body: {"account_id": "<id>"} ("" clears the binding). The session
// process is terminated and respawned in-place under the new credential;
// the row id (and therefore the UI tab) survives.
func (h *Handlers) switchClaudeAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req SwitchAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sess, err := h.svc.SwitchClaudeAccount(r.Context(), id, req.AccountID)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *Handlers) respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, ErrUnknownProvider):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, ErrAccountSwitchUnsupported):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, ErrProviderUnavailable):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, ErrAlreadyEnded):
		writeError(w, http.StatusConflict, err)
	default:
		h.log.Error("session handler", "err", err)
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
