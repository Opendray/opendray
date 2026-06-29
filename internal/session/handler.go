package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/integration"
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
	SwitchClaudeAccount(ctx context.Context, id, accountID string, carryContext bool) (Session, error)
	SwitchAntigravityAccount(ctx context.Context, id, accountID string) (Session, error)
	History(ctx context.Context, id string, limit int) (HistoryResponse, error)
}

// AntigravityAccountChecker is the minimal agyacct surface the session
// handler needs to validate `antigravity_account_id` early (before the
// switch mutates the row). Mirrors ClaudeAccountChecker; a nil checker
// disables validation (deferred error at spawn time).
type AntigravityAccountChecker interface {
	// CheckEnabled returns nil when id refers to an existing, enabled
	// antigravity account; an error otherwise (distinguishing
	// not-found from disabled so the handler maps both to 400).
	CheckEnabled(ctx context.Context, id string) error
}

// ClaudeAccountChecker is the minimal cliacct surface the session
// handler needs to (a) validate `claude_account_id` early — before
// the row is persisted (create) or mutated (switch) — and (b) auto-
// assign the least-loaded enabled account when the caller omits it.
// Wiring this as an interface, rather than importing cliacct directly,
// keeps the session package free of the cliacct ↔ session cyclic-dep
// risk and makes the handler trivially testable with a fake.
//
// A nil checker disables both validation and auto-assign; existing
// callers that don't wire one keep their previous behavior (deferred
// error at spawn time, always-CLI-default for unpinned sessions).
type ClaudeAccountChecker interface {
	// CheckClaudeAccountEnabled returns nil when id refers to an
	// existing, enabled account; an error otherwise. Implementations
	// should distinguish "not found" from "disabled" via the returned
	// error so the handler can map both to 400 with a useful message.
	CheckClaudeAccountEnabled(ctx context.Context, id string) error

	// PickAutoAssignClaudeAccount returns the id of the enabled
	// account that should receive the next session, or "" when
	// auto-assign should be skipped (fewer than 2 enabled accounts,
	// no enabled accounts at all, etc.). Implementations are free to
	// pick by least-loaded, round-robin, or any other policy — the
	// handler treats the returned id as opaque.
	PickAutoAssignClaudeAccount(ctx context.Context) (string, error)
}

// IntegrationDefaults resolves the spawn defaults an integration has
// configured (provider / model / claude account). create() applies them
// to integration-origin sessions for any field the request omits. Kept
// as a small interface so the session package needn't import the
// integration registry directly. Optional; a nil resolver disables
// default application.
type IntegrationDefaults interface {
	// DefaultsFor returns the configured spawn defaults for the
	// integration with the given id. Any field may be empty ("no
	// default"). The caller treats a non-nil error as "no defaults".
	DefaultsFor(ctx context.Context, integrationID string) (provider, model, claudeAccount string, err error)
}

type Handlers struct {
	svc      Service
	acct     ClaudeAccountChecker      // optional; nil disables early validation
	agyAcct  AntigravityAccountChecker // optional; nil disables early validation
	defaults IntegrationDefaults       // optional; nil disables integration spawn defaults
	log      *slog.Logger
	upgrader websocket.Upgrader
}

// HandlerOption mutates Handlers at construction time. Used so adding
// new optional deps (the account checker, future per-provider checks)
// doesn't break NewHandlers' positional signature.
type HandlerOption func(*Handlers)

// WithClaudeAccountChecker wires the cliacct surface used to validate
// claude_account_id in create() and switchClaudeAccount(). Passing nil
// is equivalent to omitting the option (validation is skipped).
func WithClaudeAccountChecker(c ClaudeAccountChecker) HandlerOption {
	return func(h *Handlers) { h.acct = c }
}

// WithAntigravityAccountChecker wires the agyacct surface used to
// validate antigravity_account_id in switchAntigravityAccount(). Passing
// nil is equivalent to omitting the option (validation is skipped).
func WithAntigravityAccountChecker(c AntigravityAccountChecker) HandlerOption {
	return func(h *Handlers) { h.agyAcct = c }
}

// WithIntegrationDefaults wires the resolver used to fill provider /
// model / claude account from an integration's configured spawn
// defaults in create(). Passing nil is equivalent to omitting it
// (defaults are not applied).
func WithIntegrationDefaults(d IntegrationDefaults) HandlerOption {
	return func(h *Handlers) { h.defaults = d }
}

func NewHandlers(svc Service, log *slog.Logger, opts ...HandlerOption) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	h := &Handlers{
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
	for _, opt := range opts {
		opt(h)
	}
	return h
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
			r.Patch("/antigravity-account", h.switchAntigravityAccount)
			r.Post("/uploads", h.upload)
		})
	})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Provenance — derived from the authenticated principal, never the
	// JSON body (Cortex Phase 2). Integration-created sessions are
	// tagged so the memory capture pipeline can route their facts by
	// the integration's memory_policy instead of polluting durable
	// memory. No principal (route not behind auth middleware, e.g. in
	// tests) falls through to the operator default.
	//
	// Stamped first so integration defaults below see the resolved
	// origin, and so the default-filled provider/account flow through
	// the same validation + auto-assign as a client-supplied request.
	if p, ok := integration.CurrentPrincipal(r.Context()); ok && p.Kind == integration.KindIntegration {
		req.SetOrigin(OriginIntegration, p.ID)
		h.applyIntegrationDefaults(r.Context(), p.ID, &req)
	} else if ok {
		req.SetOrigin(OriginOperator, "")
	}
	// Validate claude_account_id up front so the operator gets a clean
	// 400 instead of a 500 on the deferred spawn-time error. We only
	// check when an id was actually supplied; empty means "use the
	// CLI's keychain default" and stays the supported fallback.
	if h.acct != nil && req.ClaudeAccountID != "" {
		if err := h.acct.CheckClaudeAccountEnabled(r.Context(), req.ClaudeAccountID); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("claude_account_id: %w", err))
			return
		}
	}
	// Auto-assign: when caller didn't pin an account and the provider
	// is "claude", let the checker pick the least-loaded enabled
	// account so multi-account setups balance traffic without the
	// operator having to touch the spawn dialog. Empty return means
	// "no auto-assign" (e.g. <2 enabled accounts) — fall through to
	// the existing CLI-default behavior.
	if h.acct != nil && req.ClaudeAccountID == "" && req.ProviderID == "claude" {
		if id, err := h.acct.PickAutoAssignClaudeAccount(r.Context()); err == nil && id != "" {
			req.ClaudeAccountID = id
		}
		// Auto-assign errors are non-fatal: we still try to spawn
		// with the CLI default rather than failing the create call.
	}
	sess, err := h.svc.Create(r.Context(), req)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

// applyIntegrationDefaults fills provider / model / claude account from
// the integration's configured spawn defaults, but only for fields the
// request left empty — a client-supplied value always wins. A nil
// resolver (option not wired) or any lookup error is a silent no-op:
// defaults are a convenience, never a hard requirement for spawning.
func (h *Handlers) applyIntegrationDefaults(ctx context.Context, integrationID string, req *CreateRequest) {
	if h.defaults == nil {
		return
	}
	provider, model, account, err := h.defaults.DefaultsFor(ctx, integrationID)
	if err != nil {
		h.log.Warn("integration defaults lookup failed",
			"integration", integrationID, "err", err)
		return
	}
	if req.ProviderID == "" {
		req.ProviderID = provider
	}
	// default_model is a provider-specific string (e.g. an antigravity
	// model name like "Gemini 3.5 Flash (Medium)"). Only inherit it when
	// the session actually runs on the integration's default provider —
	// otherwise a mismatched model would be passed to a different CLI
	// (e.g. `claude --model "Gemini 3.5 Flash (Medium)"`), which fails to
	// spawn. A session on another provider falls back to that provider's
	// own default (no --model flag), which is always safe.
	if req.Model == "" && req.ProviderID == provider {
		req.Model = model
	}
	if req.ClaudeAccountID == "" {
		req.ClaudeAccountID = account
	}
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	p, ok := integration.CurrentPrincipal(r.Context())
	list = visibleSessions(p, ok, list)
	if list == nil {
		list = []Session{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

// visibleSessions filters the session list for the calling principal.
// An integration token sees only its own integration's sessions; the
// operator console (admin token, or no integration principal) never sees
// integration-origin sessions at all. Third-party consumer sessions are
// private — they may serve many end users — so they must not surface in
// the operator's session list. Returns a fresh slice (never mutates).
func visibleSessions(p integration.Principal, ok bool, list []Session) []Session {
	asIntegration := ok && p.Kind == integration.KindIntegration
	out := make([]Session, 0, len(list))
	for _, s := range list {
		if asIntegration {
			if s.IntegrationID == p.ID {
				out = append(out, s)
			}
			continue
		}
		if s.Origin == OriginIntegration {
			continue
		}
		out = append(out, s)
	}
	return out
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

// uploadMaxBytes caps each /uploads request — Claude Code can read
// images up to a few MB; bigger payloads are usually a mistake.
const uploadMaxBytes = 16 * 1024 * 1024 // 16 MiB

// upload handles POST /sessions/{id}/uploads. The body is a multipart
// form with a "file" part. The bytes are saved to a per-session
// directory under the gateway host's tempdir; the saved absolute
// path is returned so the client can paste it into the terminal as
// e.g. an `@/path/to/file.png` reference for the running CLI.
//
// Path safety: the filename is regenerated server-side as a random
// hex token + the client-supplied extension (lowercased + filtered to
// known image suffixes). The original filename is never touched on
// disk — guards against path traversal and odd shell-active chars.
func (h *Handlers) upload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Validate the session exists so we don't accumulate orphan
	// directories from typo'd ids.
	if _, err := h.svc.Get(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, uploadMaxBytes)
	if err := r.ParseMultipartForm(uploadMaxBytes); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("parse multipart: %w", err))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing file part: %w", err))
		return
	}
	defer file.Close()

	ext := normalizeUploadExt(header.Filename)
	dir := filepath.Join(os.TempDir(), "opendray-uploads", id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tokenBytes := make([]byte, 8)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	name := hex.EncodeToString(tokenBytes) + ext
	outPath := filepath.Join(dir, name)
	out, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	written, copyErr := io.Copy(out, file)
	if cerr := out.Close(); cerr != nil && copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		_ = os.Remove(outPath)
		writeError(w, http.StatusInternalServerError, copyErr)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"path":          outPath,
		"size":          written,
		"original_name": header.Filename,
	})
}

// normalizeUploadExt extracts a safe file extension from the
// client-supplied filename. We accept a small allowlist of image
// types Claude / Codex are known to read; anything else
// becomes ".bin" so the file lives somewhere predictable but the
// CLI won't accidentally try to render it as JPEG.
func normalizeUploadExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".heic", ".bmp":
		return ext
	default:
		return ".bin"
	}
}

// history handles GET /sessions/{id}/history. Returns up to `limit`
// (default 200, max 1000) of the user's past prompts in this project,
// pulled from Claude's JSONL transcripts under ~/.claude/projects.
//
// Sessions whose provider isn't Claude (codex/shell) get
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
	// Mirror create()'s up-front check so a switch to a deleted or
	// disabled account fails BEFORE we stop the live session. Without
	// this, Manager.SwitchClaudeAccount would stop the PTY, then
	// discover the bad id at spawn time, leaving the session stopped
	// with the *original* account preserved on the row — surprising
	// state for the operator. Empty AccountID is the valid "clear
	// pinning, use CLI default" intent, so we skip validation there.
	if h.acct != nil && req.AccountID != "" {
		if err := h.acct.CheckClaudeAccountEnabled(r.Context(), req.AccountID); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("account_id: %w", err))
			return
		}
	}
	sess, err := h.svc.SwitchClaudeAccount(r.Context(), id, req.AccountID, req.CarryContext)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

// switchAntigravityAccount handles PATCH /sessions/{id}/antigravity-account.
// Mirrors switchClaudeAccount: validate the target up-front (so a bad id
// fails before the live session is stopped), then rebind. carry_context
// is accepted in the body for client symmetry but ignored — agy has no
// cross-account recap builder yet, so the switch is always clean-slate.
func (h *Handlers) switchAntigravityAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req SwitchAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if h.agyAcct != nil && req.AccountID != "" {
		if err := h.agyAcct.CheckEnabled(r.Context(), req.AccountID); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("account_id: %w", err))
			return
		}
	}
	sess, err := h.svc.SwitchAntigravityAccount(r.Context(), id, req.AccountID)
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
