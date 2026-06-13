// Package voice is Opendray's client side of the voice-MCP contract
// (see docs/mcp-voice.md).
//
// Voice providers — Deepgram, ElevenLabs, AssemblyAI, local Whisper,
// anything else — are not compiled into Opendray. Each is an MCP
// server installed by the operator under <vault>/mcp/, identified by
// id, and reachable through the same Loader the AI providers already
// use.
//
// This package supplies:
//
//   - Client: a thin JSON-RPC 2.0 stdio client that spawns the MCP
//     server per call, performs initialize → tools/call → shutdown,
//     and returns a Transcript or Audio.
//
//   - Service: ResolveProvider(ctx, id) → *Client. Looks the server
//     up in the vault, applies ${SECRET} placeholder substitution,
//     returns a ready-to-call client.
//
// Channels call into Service.ResolveProvider with the value of their
// own voice_mcp_server config field. They never see a Deepgram API
// key (or anything else third-party-specific).
package voice

import (
	"context"
	"errors"
	"time"
)

// Capability lists the two tool names the voice contract defines.
const (
	CapTranscribe = "voice/transcribe"
	CapSynthesize = "voice/synthesize"
)

// Transcript is the result of a voice/transcribe call.
//
// Text="" means the server received audio but found no speech; the
// caller should treat that as a non-error "nothing heard".
type Transcript struct {
	Text       string
	Language   string  // BCP-47, may be empty
	Confidence float64 // 0..1, 0 = not reported
	DurationMS int     // 0 = not reported
}

// Audio is the result of a voice/synthesize call. Body holds the
// decoded audio bytes; MimeType is what the server declared.
type Audio struct {
	Body       []byte
	MimeType   string
	DurationMS int
}

// SynthesizeOpts are the optional knobs on a TTS call. Format is
// required.
type SynthesizeOpts struct {
	Format string  // "ogg-opus" | "mp3" | "wav" (server may add more)
	Voice  string  // server-specific voice id
	Speed  float64 // 0 = server default
}

// Error wraps a tool-call failure with the contract's well-known code
// (see docs/mcp-voice.md §Error model). Callers compare via errors.As
// or check Code directly to render user-facing messages.
type Error struct {
	Code    string // e.g. "auth_failed", "audio_too_large"
	Message string // server-supplied human-readable detail
}

func (e *Error) Error() string {
	if e.Code == "" {
		return e.Message
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

// Well-known error codes from the contract. Kept as exported vars so
// callers do `errors.Is(err, voice.ErrAuthFailed)` for branches.
var (
	ErrAuthFailed          = &Error{Code: "auth_failed"}
	ErrAudioTooLarge       = &Error{Code: "audio_too_large"}
	ErrUnsupportedFormat   = &Error{Code: "unsupported_format"}
	ErrLanguageUnsupported = &Error{Code: "language_unsupported"}
	ErrProviderUnavailable = &Error{Code: "provider_unavailable"}
)

// Is matches by code so the well-known sentinels work with errors.Is
// even though we instantiate them fresh per call.
func (e *Error) Is(target error) bool {
	var te *Error
	if !errors.As(target, &te) {
		return false
	}
	return te.Code == e.Code
}

// callTimeout caps every tool call. Voice notes are short; if a
// provider hasn't responded in 30s something is wrong.
const callTimeout = 30 * time.Second

// Audio payload safety caps. Bigger than this and we don't even try
// the round-trip — keeps tool-call latency bounded and protects the
// Telegram sendVoice path (1 MB waveform / 50 MB document).
const (
	maxAudioIn  = 5 * 1024 * 1024
	maxAudioOut = 5 * 1024 * 1024
)

// ctxWithCallTimeout wraps the caller's context with the per-call
// budget, returning the derived context and its cancel func.
func ctxWithCallTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, callTimeout)
}
