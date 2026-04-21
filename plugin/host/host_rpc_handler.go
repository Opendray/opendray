package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/opendray/opendray/plugin/bridge"
)

// NSDispatcher is the namespace surface HostRPCHandler routes to. It
// intentionally drops the envID and *bridge.Conn parameters that the
// WebSocket-bound gateway.Namespace interface carries — sidecar
// JSON-RPC doesn't (yet) surface streams, so those parameters would
// be unused noise here.
//
// main.go adapts each concrete namespace (*bridge.FSAPI, *bridge.ExecAPI,
// *bridge.HTTPAPI, *bridge.SecretAPI, *bridge.WorkbenchAPI, *bridge.StorageAPI,
// *bridge.EventsAPI) to this interface via NamespaceAdapter below.
type NSDispatcher interface {
	Dispatch(ctx context.Context, plugin, method string, args json.RawMessage) (any, error)
}

// NamespaceAdapter wraps a gateway.Namespace-style dispatcher (the one
// with envID + *bridge.Conn parameters) for the sidecar's stdio path.
// The wrapped namespace is called with envID="" and conn=nil — neither
// is read by the M3 namespaces in the non-stream path.
type NamespaceAdapter struct {
	// Inner is a function type that matches every *bridge.*API's
	// Dispatch signature. Main.go constructs adapters like:
	//
	//   host.NamespaceAdapter{Inner: fsAPI.Dispatch}
	//
	// Keeping the adapter as a first-class struct (rather than a raw
	// func type) lets us extend it later with per-namespace policy
	// (timeouts, audit tags) without churning every call site.
	Inner func(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *bridge.Conn) (any, error)
}

// Dispatch implements NSDispatcher.
func (a NamespaceAdapter) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage) (any, error) {
	return a.Inner(ctx, plugin, method, args, "", nil)
}

// HostRPCConfig carries dependencies for HostRPCHandler.
type HostRPCConfig struct {
	// Plugin is the owning plugin name. Bound at construction — the
	// sidecar cannot override it through any RPC method, which is the
	// fundamental "sidecar impersonation" mitigation from M3-PLAN §6.
	Plugin string

	// Namespaces maps a namespace name ("fs", "exec", "http", "secret",
	// "workbench", "storage", "events") to its dispatcher. Unknown
	// namespaces return RPCErrMethodNotFound.
	Namespaces map[string]NSDispatcher
}

// HostRPCHandler routes sidecar → host JSON-RPC calls through the
// same bridge.Namespace surface webview plugins use. Capability
// enforcement runs identically for both transports.
//
// Method format on the wire: "<ns>/<method>" — the slash is the
// discriminator. "fs/readFile", "exec/run", etc. Methods with more
// than one slash, or any of "..", "\x00" are rejected as injection
// attempts per the §6 threat matrix.
type HostRPCHandler struct {
	cfg HostRPCConfig
}

// NewHostRPCHandler constructs a handler. Plugin and at least one
// Namespace are required. Returns an error (not a panic) so wiring
// code can surface config problems without the supervisor dying.
func NewHostRPCHandler(cfg HostRPCConfig) (*HostRPCHandler, error) {
	if cfg.Plugin == "" {
		return nil, errors.New("host: HostRPCConfig.Plugin is required")
	}
	if len(cfg.Namespaces) == 0 {
		return nil, errors.New("host: HostRPCConfig.Namespaces is empty")
	}
	return &HostRPCHandler{cfg: cfg}, nil
}

// Handle implements RPCHandler. It routes "ns/method" to the matching
// NSDispatcher, or returns MethodNotFound for unknown namespaces.
// EPERM errors from the capability gate (wrapped *bridge.PermError)
// surface as RPCError with Code=RPCErrEPERM so sidecars see a stable
// code rather than a free-form message.
func (h *HostRPCHandler) Handle(ctx context.Context, method string, params json.RawMessage) (any, error) {
	ns, inner, err := splitMethod(method)
	if err != nil {
		return nil, &RPCError{Code: RPCErrInvalidRequest, Message: err.Error()}
	}
	d, ok := h.cfg.Namespaces[ns]
	if !ok {
		return nil, &RPCError{
			Code:    RPCErrMethodNotFound,
			Message: fmt.Sprintf("unknown namespace %q", ns),
		}
	}

	result, err := d.Dispatch(ctx, h.cfg.Plugin, inner, params)
	if err == nil {
		return result, nil
	}

	// Map bridge-layer error kinds to the RPC error shape.
	var permErr *bridge.PermError
	if errors.As(err, &permErr) {
		return nil, &RPCError{Code: RPCErrEPERM, Message: permErr.Msg}
	}
	var wireErr *bridge.WireError
	if errors.As(err, &wireErr) {
		return nil, &RPCError{
			Code:    wireCodeToRPC(wireErr.Code),
			Message: wireErr.Message,
			Data:    wireErr.Data,
		}
	}
	return nil, &RPCError{Code: RPCErrInternal, Message: err.Error()}
}

// splitMethod splits "ns/method" into its parts, rejecting injection
// attempts (multi-slash, traversal, null byte).
func splitMethod(method string) (ns, inner string, err error) {
	if method == "" {
		return "", "", errors.New("method is empty")
	}
	if strings.Contains(method, "\x00") {
		return "", "", errors.New("method contains null byte")
	}
	if strings.Contains(method, "..") {
		return "", "", errors.New(`method contains ".."`)
	}
	idx := strings.Index(method, "/")
	if idx < 0 {
		return "", "", errors.New(`method missing "/" separator`)
	}
	// Reject more than one slash — keeps the method-space flat and
	// unambiguous.
	if strings.Count(method, "/") != 1 {
		return "", "", errors.New(`method must be "<namespace>/<method>" (one slash)`)
	}
	ns = method[:idx]
	inner = method[idx+1:]
	if ns == "" || inner == "" {
		return "", "", errors.New("namespace or method half is empty")
	}
	return ns, inner, nil
}

// wireCodeToRPC maps a bridge WireError.Code onto the JSON-RPC error
// code space. See plugin/host/jsonrpc.go constants.
func wireCodeToRPC(code string) int {
	switch code {
	case "EPERM":
		return RPCErrEPERM
	case "EUNAVAIL":
		return RPCErrEUNAVAIL
	case "EINVAL":
		return RPCErrInvalidParams
	case "EINTERNAL":
		return RPCErrInternal
	default:
		return RPCErrInternal
	}
}
