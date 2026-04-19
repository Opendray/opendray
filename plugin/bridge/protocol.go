package bridge

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion is the on-wire bridge envelope version. Every plugin
// handshake must carry this number; mismatches are a hard connection
// reject on the host side.
//
// This value is frozen as part of the v1 plugin contract — bumping it
// breaks every installed plugin until they re-sync the SDK. Any change
// requires coordinated updates to docs/plugin-platform/04-bridge-api.md,
// the @opendray/plugin-sdk npm package, and a migration announcement.
const ProtocolVersion = 1

// Envelope is the sole message shape on the bridge WebSocket. Every
// plugin-to-host call, host-to-plugin event, and stream chunk shares
// this structure — the protocol is deliberately flat so SDKs in any
// language can implement it with zero custom codecs.
//
// Field semantics:
//
//	V       — protocol version; must equal ProtocolVersion on inbound
//	ID      — correlation id; req-scoped for call/response, sub-scoped
//	          for streams. Absent on fire-and-forget events.
//	NS      — namespace ("workbench" | "storage" | "events" | ...).
//	          Dispatch key on host; echoed on response.
//	Method  — method within the namespace (e.g. "get", "subscribe").
//	Args    — method input payload, shape defined by the method.
//	Result  — method output payload on success.
//	Error   — structured error; mutually exclusive with Result.
//	Stream  — "chunk" for in-band stream data, "end" for stream close.
//	          Absent on plain req/response.
//	Data    — stream payload (per chunk); absent on "end".
//	Token   — auth handshake carrier (only used on the hello envelope).
//
// `json.RawMessage` is used for Args/Result/Data so the wire format
// preserves whatever the caller sent — handy for future-proofing and
// for passing opaque payloads through without double-encoding.
type Envelope struct {
	V      int             `json:"v"`
	ID     string          `json:"id,omitempty"`
	NS     string          `json:"ns,omitempty"`
	Method string          `json:"method,omitempty"`
	Args   json.RawMessage `json:"args,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *WireError      `json:"error,omitempty"`
	Stream string          `json:"stream,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	Token  string          `json:"token,omitempty"`
}

// WireError is the machine-readable failure shape. Code is drawn from
// a fixed set so SDK callers can branch without string-sniffing:
//
//	EPERM     — capability denied (permission not granted / revoked)
//	EINVAL    — malformed args / method
//	ENOENT    — target (key, view, session) not found
//	ETIMEOUT  — request deadline exceeded
//	EUNAVAIL  — functionality deferred to a later milestone (M3/M4/M6)
//	EINTERNAL — host-side bug; caller should retry then report
//
// Message is human-readable English; SDKs localise on their side.
// Data carries optional structured context (e.g. the needed capability
// on EPERM, the expected field on EINVAL).
type WireError struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// NewOK builds a success response envelope for correlation id `id`
// with a `result` payload that will be JSON-marshaled. Returns the
// marshal error verbatim rather than swallowing it — a broken
// response envelope would strand the SDK caller forever, so the caller
// must surface the failure.
func NewOK(id string, result any) (Envelope, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return Envelope{}, fmt.Errorf("bridge: marshal result for %s: %w", id, err)
	}
	return Envelope{V: ProtocolVersion, ID: id, Result: raw}, nil
}

// NewErr builds a structured error envelope. code must be one of the
// documented WireError codes; validation happens at the handler layer
// via a `NewErrWithData`-style extension, not here — keeping NewErr
// cheap and infallible is load-bearing for the error paths.
func NewErr(id, code, msg string) Envelope {
	return Envelope{
		V:     ProtocolVersion,
		ID:    id,
		Error: &WireError{Code: code, Message: msg},
	}
}

// NewStreamChunk builds an in-band stream-data envelope. Stream IDs
// are allocated by the host on subscribe and echoed on every chunk
// until the caller (or the stream source) closes with NewStreamEnd.
func NewStreamChunk(id string, data any) (Envelope, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, fmt.Errorf("bridge: marshal stream chunk for %s: %w", id, err)
	}
	return Envelope{V: ProtocolVersion, ID: id, Stream: "chunk", Data: raw}, nil
}

// NewStreamEnd closes an open stream. No payload — SDKs fire their
// `onComplete` callback on receipt and free the correlation id.
func NewStreamEnd(id string) Envelope {
	return Envelope{V: ProtocolVersion, ID: id, Stream: "end"}
}

// NewStreamChunkErr emits an error chunk inside a live stream. Used
// when a stream-producing capability (exec.spawn non-zero exit mid-
// stream, http.stream truncated, fs.watch errored) needs to surface
// a recoverable problem without tearing the subscription down. The
// SDK receives it as a chunk with an error payload — unlike
// NewStreamEnd, the correlation id stays live and more chunks may
// follow.
//
// For terminal errors, callers should pair this with a subsequent
// NewStreamEnd.
func NewStreamChunkErr(id string, we *WireError) Envelope {
	return Envelope{V: ProtocolVersion, ID: id, Stream: "chunk", Error: we}
}
