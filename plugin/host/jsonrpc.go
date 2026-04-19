// Package host — OpenDray plugin sidecar runtime (M3).
//
// The supervisor (supervisor.go, M3 T14) spawns one sidecar process
// per form:"host" plugin and speaks JSON-RPC 2.0 to it using the
// Language Server Protocol (LSP) Content-Length framing:
//
//	Content-Length: 42\r\n
//	\r\n
//	{"jsonrpc":"2.0","method":"ping","id":1}
//
// This file holds only the framing codec + minimal RPC types so T15
// can land independently of the supervisor and the mux.
package host

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// MaxBodyBytes is the hard cap on a single JSON-RPC message body.
// 16 MiB matches the bridge WS body-size limit.
const MaxBodyBytes = 16 << 20

// MaxHeaderBytes caps the header section between messages. LSP in
// practice never exceeds a few hundred bytes — 8 KiB is comfortably
// above every sighted real-world value.
const MaxHeaderBytes = 8 << 10

// ErrBodyTooLarge is returned by FramedReader.Read when a
// Content-Length header exceeds MaxBodyBytes. The stream is left in
// a recoverable state: the caller may skip the oversize payload
// (Read auto-consumes the body to resync).
var ErrBodyTooLarge = errors.New("host: jsonrpc body exceeds MaxBodyBytes")

// ErrHeaderTooLarge is returned when no Content-Length header is seen
// within MaxHeaderBytes of input. Typically a corrupt stream.
var ErrHeaderTooLarge = errors.New("host: jsonrpc header exceeds MaxHeaderBytes")

// ErrMalformedHeader flags a Content-Length value that isn't a
// non-negative integer, or a header block missing the \r\n\r\n
// terminator.
var ErrMalformedHeader = errors.New("host: malformed jsonrpc header")

// RPC is the minimal JSON-RPC 2.0 shape carried over stdio.
//
// A request has Method (+ID for non-notifications); a response has
// Result OR Error and echoes the ID. Param and Result payloads stay
// as json.RawMessage so the codec never has to know the concrete
// schemas — those live in the caller (LSP proxy, host_rpc_handler).
type RPC struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is the error envelope used inside RPC.Error. Implements
// the error interface so handlers can return it through the standard
// Go error chain — the Mux unwraps it back into the on-the-wire shape.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error makes *RPCError implement error. The format matches the wire
// shape so log lines are self-explanatory.
func (e *RPCError) Error() string {
	return fmt.Sprintf("jsonrpc %d: %s", e.Code, e.Message)
}

// RPC standard error codes (see JSON-RPC 2.0 §5.1). Ranges -32000 to
// -32099 are application-defined; we use -32001 to signal EPERM from
// the capability gate (see plugin/host.HostRPCHandler, M3 T17).
const (
	RPCErrParseError     = -32700
	RPCErrInvalidRequest = -32600
	RPCErrMethodNotFound = -32601
	RPCErrInvalidParams  = -32602
	RPCErrInternal       = -32603
	RPCErrEPERM          = -32001
	RPCErrEUNAVAIL       = -32002
)

// FramedReader reads one LSP-framed JSON-RPC message per Read call.
//
// It owns a bufio.Reader so repeated Read calls don't lose bytes
// between frames. The caller is expected to loop on Read until it
// returns io.EOF.
type FramedReader struct {
	br *bufio.Reader
}

// NewFramedReader wraps r for LSP framing. r is not closed by the
// reader — the owning supervisor handles process lifetime.
func NewFramedReader(r io.Reader) *FramedReader {
	return &FramedReader{br: bufio.NewReader(r)}
}

// Read blocks until the next complete JSON-RPC payload arrives, then
// returns its raw JSON body. Framing errors surface as ErrMalformed-
// Header / ErrBodyTooLarge / ErrHeaderTooLarge; io.EOF propagates
// cleanly when the stream closes between frames.
func (fr *FramedReader) Read() (json.RawMessage, error) {
	contentLen := -1
	headerBytes := 0
	for {
		line, err := fr.br.ReadString('\n')
		headerBytes += len(line)
		if err != nil {
			if err == io.EOF && line == "" {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("%w: %v", ErrMalformedHeader, err)
		}
		if headerBytes > MaxHeaderBytes {
			return nil, ErrHeaderTooLarge
		}
		// Trim the trailing \r\n (or bare \n from relaxed servers).
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			// End of headers.
			break
		}
		// Case-insensitive match — the spec requires "Content-Length"
		// but some servers emit "content-length".
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "content-length:") {
			val := strings.TrimSpace(line[len("content-length:"):])
			n, perr := strconv.Atoi(val)
			if perr != nil || n < 0 {
				return nil, fmt.Errorf("%w: Content-Length=%q", ErrMalformedHeader, val)
			}
			contentLen = n
		}
		// Unknown headers (Content-Type, etc.) are tolerated — LSP permits
		// but doesn't require them.
	}
	if contentLen < 0 {
		return nil, fmt.Errorf("%w: missing Content-Length", ErrMalformedHeader)
	}
	if contentLen > MaxBodyBytes {
		// Drain the oversize body so the stream stays aligned for the
		// next frame; the caller can then retry Read.
		if _, derr := io.CopyN(io.Discard, fr.br, int64(contentLen)); derr != nil {
			return nil, fmt.Errorf("%w (drain failed: %v)", ErrBodyTooLarge, derr)
		}
		return nil, ErrBodyTooLarge
	}
	body := make([]byte, contentLen)
	if _, err := io.ReadFull(fr.br, body); err != nil {
		return nil, fmt.Errorf("host: read body: %w", err)
	}
	return json.RawMessage(body), nil
}

// FramedWriter writes LSP-framed JSON-RPC messages. Safe for
// concurrent callers — every Write is atomic with respect to
// framing (header + body land together under the internal mutex).
type FramedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// NewFramedWriter wraps w for LSP framing. The writer does not close
// w — the supervisor owns process stdin lifetime.
func NewFramedWriter(w io.Writer) *FramedWriter {
	return &FramedWriter{w: w}
}

// Write encodes msg as JSON and emits the framed payload. msg is
// typically an RPC struct but any json.Marshal-compatible value works.
// Returns ErrBodyTooLarge if the encoded body exceeds MaxBodyBytes
// (frame is not written in that case).
func (fw *FramedWriter) Write(msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("host: marshal jsonrpc body: %w", err)
	}
	if len(body) > MaxBodyBytes {
		return ErrBodyTooLarge
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if _, werr := fw.w.Write([]byte(header)); werr != nil {
		return fmt.Errorf("host: write header: %w", werr)
	}
	if _, werr := fw.w.Write(body); werr != nil {
		return fmt.Errorf("host: write body: %w", werr)
	}
	return nil
}
