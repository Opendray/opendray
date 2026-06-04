package voice

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opendray/opendray-v2/internal/mcp"
)

// mcpProtocolVersion is the MCP protocol version we negotiate with the
// server. Servers that don't recognise this respond with the version
// they support; we accept their value and proceed.
const mcpProtocolVersion = "2025-06-18"

// Client talks JSON-RPC 2.0 over stdio to a voice MCP server. One
// Client wraps one configured server entry from the vault; each
// Transcribe / Synthesize call spawns the server, runs initialize +
// tools/call, then tears it down.
//
// Spawn-per-call keeps the lifecycle dead simple — no long-running
// state to manage, no zombie processes, no warm-pool tuning. Cost is
// the per-call cold start (≈100–500 ms for a Node stdio server),
// which is acceptable for voice-note traffic. If usage justifies it,
// a future iteration can keep one process warm per server.
type Client struct {
	srv mcp.Server // resolved server config from the vault
	log *slog.Logger
}

// NewClient binds a Client to one MCP server entry. The Client itself
// is stateless and safe to reuse across calls.
func NewClient(srv mcp.Server, log *slog.Logger) *Client {
	if log == nil {
		log = slog.Default()
	}
	return &Client{srv: srv, log: log.With("voice_mcp", srv.ID)}
}

// Transcribe asks the bound server to convert audio bytes into a
// Transcript via the voice/transcribe tool.
func (c *Client) Transcribe(ctx context.Context, audio []byte, mime string) (Transcript, error) {
	if len(audio) > maxAudioIn {
		return Transcript{}, &Error{Code: "audio_too_large",
			Message: fmt.Sprintf("audio is %d bytes; max %d", len(audio), maxAudioIn)}
	}
	if mime == "" {
		mime = "audio/ogg"
	}

	args := map[string]any{
		"audio_b64": base64.StdEncoding.EncodeToString(audio),
		"mime_type": mime,
	}
	raw, err := c.call(ctx, CapTranscribe, args)
	if err != nil {
		return Transcript{}, err
	}

	var out struct {
		Text       string  `json:"text"`
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
		DurationMS int     `json:"duration_ms"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return Transcript{}, fmt.Errorf("voice: decode transcript: %w", err)
	}
	return Transcript{
		Text:       out.Text,
		Language:   out.Language,
		Confidence: out.Confidence,
		DurationMS: out.DurationMS,
	}, nil
}

// Synthesize asks the bound server to convert text into audio bytes
// via the voice/synthesize tool.
func (c *Client) Synthesize(ctx context.Context, text string, opts SynthesizeOpts) (Audio, error) {
	if opts.Format == "" {
		opts.Format = "ogg-opus"
	}
	args := map[string]any{
		"text":   text,
		"format": opts.Format,
	}
	if opts.Voice != "" {
		args["voice"] = opts.Voice
	}
	if opts.Speed != 0 {
		args["speed"] = opts.Speed
	}

	raw, err := c.call(ctx, CapSynthesize, args)
	if err != nil {
		return Audio{}, err
	}

	var out struct {
		AudioB64   string `json:"audio_b64"`
		MimeType   string `json:"mime_type"`
		DurationMS int    `json:"duration_ms"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return Audio{}, fmt.Errorf("voice: decode audio: %w", err)
	}
	body, err := base64.StdEncoding.DecodeString(out.AudioB64)
	if err != nil {
		return Audio{}, fmt.Errorf("voice: decode audio_b64: %w", err)
	}
	if len(body) > maxAudioOut {
		return Audio{}, &Error{Code: "audio_too_large",
			Message: fmt.Sprintf("synth output is %d bytes; max %d", len(body), maxAudioOut)}
	}
	return Audio{Body: body, MimeType: out.MimeType, DurationMS: out.DurationMS}, nil
}

// call runs the full stdio round trip: spawn → initialize → tools/call
// → close. Returns the tool result's first content block (interpreted
// as JSON text).
func (c *Client) call(parent context.Context, tool string, args map[string]any) ([]byte, error) {
	ctx, cancel := ctxWithCallTimeout(parent)
	defer cancel()

	if c.srv.Transport != "" && c.srv.Transport != "stdio" {
		return nil, fmt.Errorf("voice: transport %q not supported (stdio only)", c.srv.Transport)
	}
	if c.srv.Command == "" {
		return nil, errors.New("voice: server has no command")
	}

	cmd := exec.CommandContext(ctx, c.srv.Command, c.srv.Args...)
	cmd.Env = envSlice(c.srv.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("voice: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("voice: stdout pipe: %w", err)
	}
	// Capture stderr for diagnostics on failure; never bubble noise
	// back to the user as the error message.
	var stderrBuf logCollector
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("voice: start %s: %w", c.srv.Command, err)
	}

	// Make sure the subprocess gets cleaned up even if a step below
	// returns early. CommandContext kills on ctx done; this is a
	// belt-and-braces.
	defer func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}()

	dec := json.NewDecoder(bufio.NewReader(stdout))
	enc := json.NewEncoder(stdin)
	// MCP framing is newline-delimited JSON; json.Encoder writes a
	// trailing newline by default, which the server's line reader
	// interprets as message boundary.

	id := newIDGen()

	// 1. initialize
	if _, err := rpcCall(ctx, enc, dec, id.next(), "initialize", map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "opendray-voice",
			"version": "0.1",
		},
	}); err != nil {
		return nil, fmt.Errorf("voice: initialize: %w (stderr: %s)", err, stderrBuf.tail())
	}

	// 2. initialized notification — no id, no response expected
	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}); err != nil {
		return nil, fmt.Errorf("voice: send initialized: %w", err)
	}

	// 3. tools/call
	result, err := rpcCall(ctx, enc, dec, id.next(), "tools/call", map[string]any{
		"name":      tool,
		"arguments": args,
	})
	if err != nil {
		return nil, err // already wraps the contract error
	}

	return extractToolText(result)
}

// rpcCall sends one JSON-RPC request and waits for the matching
// response. Notifications and out-of-order responses for other ids are
// discarded (a voice client only has one in-flight call at a time).
func rpcCall(ctx context.Context, enc *json.Encoder, dec *json.Decoder, id int, method string, params any) (json.RawMessage, error) {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	if err := enc.Encode(req); err != nil {
		return nil, fmt.Errorf("send %s: %w", method, err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var resp struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      *int            `json:"id"`
			Result  json.RawMessage `json:"result"`
			Error   *struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			} `json:"error"`
			Method string `json:"method"`
		}
		if err := dec.Decode(&resp); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("%s: server exited", method)
			}
			return nil, fmt.Errorf("%s: read: %w", method, err)
		}
		// Server-initiated notification — ignore and keep reading.
		if resp.ID == nil {
			continue
		}
		if *resp.ID != id {
			// Stale response from a previous call (shouldn't happen in
			// our one-call lifecycle, but be tolerant).
			continue
		}
		if resp.Error != nil {
			return nil, decodeContractError(resp.Error.Code, resp.Error.Message, resp.Error.Data)
		}
		return resp.Result, nil
	}
}

// extractToolText pulls the first text content block out of a
// tools/call result. MCP tool results are wrapped as:
//
//	{ "content": [ { "type": "text", "text": "<json>" } ] }
//
// We expect the server to put the JSON-encoded result into the text
// field, since voice tools return structured data not chat content.
func extractToolText(result json.RawMessage) ([]byte, error) {
	var wrap struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &wrap); err != nil {
		return nil, fmt.Errorf("voice: parse tools/call result: %w", err)
	}
	if wrap.IsError {
		// Tool reported a logical failure inside a success envelope.
		// Best effort: decode the text as a contract error.
		for _, b := range wrap.Content {
			if b.Type == "text" {
				return nil, decodeContractError(0, b.Text, nil)
			}
		}
		return nil, &Error{Code: "provider_unavailable", Message: "tool returned isError without text"}
	}
	for _, b := range wrap.Content {
		if b.Type == "text" {
			return []byte(b.Text), nil
		}
	}
	return nil, errors.New("voice: tools/call result has no text content")
}

// decodeContractError reads a JSON-RPC error and maps it into the
// voice contract's well-known codes when possible.
func decodeContractError(rpcCode int, message string, data json.RawMessage) error {
	out := &Error{Message: message}
	if len(data) > 0 {
		var d struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(data, &d); err == nil {
			out.Code = d.Code
		}
	}
	if out.Code == "" && rpcCode != 0 {
		// Map common JSON-RPC codes to contract codes.
		switch rpcCode {
		case -32601: // method not found
			out.Code = "unsupported_format"
		case -32602: // invalid params
			out.Code = "unsupported_format"
		default:
			out.Code = "provider_unavailable"
		}
	}
	return out
}

// envSlice flattens a name→value map into the os/exec format. Keeps
// only the explicitly configured vars — the MCP server does not inherit
// Opendray's process env, matching the catalog adapter's behavior at
// session spawn (no accidental credential leakage into a sandboxed
// subprocess).
func envSlice(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

// idGen hands out monotonically increasing JSON-RPC request ids.
type idGen struct{ n atomic.Int64 }

func newIDGen() *idGen { return &idGen{} }

func (g *idGen) next() int { return int(g.n.Add(1)) }

// logCollector captures the last ~4KB of subprocess stderr so we can
// surface a useful tail when initialize fails (network error, missing
// binary, "key invalid", etc.).
type logCollector struct {
	mu   sync.Mutex
	buf  []byte
	full bool
}

const logCollectorCap = 4096

func (l *logCollector) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = append(l.buf, p...)
	if len(l.buf) > logCollectorCap {
		l.buf = l.buf[len(l.buf)-logCollectorCap:]
		l.full = true
	}
	return len(p), nil
}

func (l *logCollector) tail() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	s := string(l.buf)
	if l.full {
		return "…" + s
	}
	return s
}

// silence unused-import warning if time isn't otherwise touched.
var _ = time.Second
