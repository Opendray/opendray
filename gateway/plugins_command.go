package gateway

// T11 — Command invoke HTTP endpoint.
//
// POST /api/plugins/{name}/commands/{id}/invoke
//
// Thin HTTP wrapper over commandInvoker.Invoke. The handler decodes the
// optional args body, dispatches via the local commandInvoker interface
// (satisfied at runtime by plugin/commands.(*Dispatcher) when main.go wires
// the Server), and marshals the raw result directly as JSON.
//
// Decoupling rule: this file MUST NOT import plugin/commands or plugin/bridge
// concrete types. Error translation uses message-prefix sniffing (v1 approach)
// plus the invokeError interface escape hatch (v2 optimisation path).

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// commandInvoker is the minimum surface T11 needs from whatever dispatches
// commands. plugin/commands.(*Dispatcher) will satisfy this when main.go is
// wired in a later task. This indirection keeps T11 parallelizable with T10
// and avoids a concrete package dependency.
type commandInvoker interface {
	Invoke(ctx context.Context, pluginName, commandID string, args map[string]any) (any, error)
}

// invokeError lets the handler translate opaque errors returned from
// commandInvoker.Invoke into HTTP status codes without importing the concrete
// types. Implementations (plugin/commands, plugin/bridge) can opt in by
// implementing InvokeStatus() int.
//
// v1 compromise: T10 emits clear error messages so message-prefix sniffing
// is the primary path. InvokeStatus() is the v2 escape hatch — documented here
// so future packages know the protocol without a breaking API change.
type invokeError interface {
	InvokeStatus() int
}

// commandInvokeRequest is the request body for the command invoke endpoint.
// Args is optional; omitting it (or sending an empty body) is equivalent to
// passing no arguments.
type commandInvokeRequest struct {
	Args map[string]any `json:"args,omitempty"`
}

// commandInvoke handles POST /api/plugins/{name}/commands/{id}/invoke.
//
// Status codes:
//
//	200 — ok, body is the Result marshaled directly (no envelope)
//	400 — malformed JSON body (EINVAL)
//	403 — capability denied, EPERM in error message (EPERM)
//	404 — command not found, "command not found" in error message (ENOTFOUND)
//	501 — run kind not implemented in M1, "requires M2" or "requires M3" (ENOTIMPL)
//	503 — no CommandInvoker wired (EINVOKER)
//	500 — other errors (EINTERNAL)
//
// Error-mapping strategy (v1 — message-prefix sniffing):
//   - If the error implements invokeError, use InvokeStatus() directly.
//   - Otherwise classify by Error() string content:
//     "command not found" → 404 ENOTFOUND
//     "requires M2" | "requires M3" → 501 ENOTIMPL
//     "EPERM" → 403 EPERM
//     else → 500 EINTERNAL
//
// This is an acceptable v1 compromise: T10 emits these exact strings and the
// handler stays decoupled from the concrete error types.
func (s *Server) commandInvoke(w http.ResponseWriter, r *http.Request) {
	// 503 if the invoker was never wired.
	if s.cmdInvoker == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "EINVOKER",
			"command invoker not wired")
		return
	}

	pluginName := chi.URLParam(r, "name")
	commandID := chi.URLParam(r, "id")

	// Decode request body. An empty body is treated as {} — not an error.
	// We use io.EOF detection to distinguish "empty body" from "bad JSON".
	var req commandInvokeRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// io.EOF means the body was empty (no bytes sent); treat as {}.
			if !isJSONEOF(err) {
				writeJSONError(w, http.StatusBadRequest, "EINVAL",
					"malformed JSON body: "+err.Error())
				return
			}
		}
	}

	result, err := s.cmdInvoker.Invoke(r.Context(), pluginName, commandID, req.Args)
	if err != nil {
		status, code := classifyInvokeError(err)
		writeJSONError(w, status, code, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result) //nolint:errcheck
}

// classifyInvokeError maps an error from commandInvoker.Invoke to an HTTP
// status code and machine-readable error code string.
//
// Priority:
//  1. invokeError interface (InvokeStatus() int) — allows dispatcher to opt
//     in to precise status codes without this package importing its types.
//  2. Message-prefix sniffing — v1 fallback matching T10's emitted strings.
func classifyInvokeError(err error) (status int, code string) {
	// v2 path: error implements InvokeStatus().
	var ie invokeError
	// Use a type assertion to avoid reflect; invokeError is an interface so
	// errors.As does a standard unwrap walk.
	if asInvokeError(err, &ie) {
		s := ie.InvokeStatus()
		switch s {
		case http.StatusNotFound:
			return s, "ENOTFOUND"
		case http.StatusForbidden:
			return s, "EPERM"
		case http.StatusNotImplemented:
			return s, "ENOTIMPL"
		default:
			return s, "EINTERNAL"
		}
	}

	// v1 path: message-prefix sniffing.
	msg := err.Error()
	switch {
	case strings.Contains(msg, "command not found"):
		return http.StatusNotFound, "ENOTFOUND"
	case strings.Contains(msg, "requires M2") || strings.Contains(msg, "requires M3"):
		return http.StatusNotImplemented, "ENOTIMPL"
	case strings.Contains(msg, "EPERM"):
		return http.StatusForbidden, "EPERM"
	default:
		return http.StatusInternalServerError, "EINTERNAL"
	}
}

// isJSONEOF returns true when err signals that the JSON stream had no tokens
// at all — i.e., the request body was empty. Both io.EOF and
// io.ErrUnexpectedEOF may be returned by json.Decoder on an empty read;
// io.EOF is the canonical case for a zero-length body.
func isJSONEOF(err error) bool {
	return err == io.EOF || err == io.ErrUnexpectedEOF
}

// asInvokeError attempts a direct type assertion to invokeError, then falls
// back to errors.As-style unwrapping for wrapped errors. Returns true and
// populates target if successful.
func asInvokeError(err error, target *invokeError) bool {
	if ie, ok := err.(invokeError); ok { //nolint:errorlint
		*target = ie
		return true
	}
	// Unwrap chain (errors.As semantics without generics constraint).
	type unwrapper interface{ Unwrap() error }
	for {
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
		if err == nil {
			return false
		}
		if ie, ok := err.(invokeError); ok { //nolint:errorlint
			*target = ie
			return true
		}
	}
}
