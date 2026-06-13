# Voice tool contract for MCP servers

Opendray treats voice (STT + TTS) as a pluggable capability delivered
through MCP servers. **No voice-provider code ships in the Opendray
binary.** Operators install an MCP server that implements the contract
below; channels reference that server by name and call its tools.

This document is the contract. Anyone can implement it.

## Why MCP

Opendray already has the install/delete plugin model (`internal/mcp/`):
servers are vault-stored, support `${SECRET}` placeholders, can run as
stdio subprocesses or remote HTTP. Reusing it for voice gives operators
one mental model instead of two.

When a third-party voice service's API URL changes, or the operator
swaps Deepgram for ElevenLabs, **the Opendray binary stays unchanged**.
Only the MCP server package is updated or replaced.

## Tools

A voice MCP server **MAY** implement either or both of the following
tools. Channels detect support by listing the server's tools (standard
MCP `tools/list` call) and matching names exactly.

### `voice/transcribe` — speech-to-text

**Input:**

```jsonc
{
  "audio_b64": "string",        // required — base64-encoded audio bytes
  "mime_type": "string",        // required — e.g. "audio/ogg", "audio/wav", "audio/mpeg"
  "language_hint": "string?",   // BCP-47 tag ("en", "fr-CA"); server may auto-detect
  "model_hint":    "string?"    // server-specific model preference; server may ignore
}
```

`audio_b64` is the entire payload base64-encoded. Opendray caps inbound
voice notes at 5 MB raw (≈6.7 MB base64) to keep tool-call latency
bounded. Servers SHOULD reject anything larger with `audio_too_large`.

**Output:**

```jsonc
{
  "text":       "string",       // required — the transcript (may be "")
  "language":   "string?",      // BCP-47 detected language, if known
  "confidence": "number?",      // 0.0..1.0, if available
  "duration_ms": "number?"      // total audio duration, if known
}
```

Empty `text` means "no speech detected" — Opendray treats this as a
silent drop, not an error.

### `voice/synthesize` — text-to-speech

**Input:**

```jsonc
{
  "text":   "string",           // required
  "format": "string",           // required — "ogg-opus" | "mp3" | "wav" (server may add more)
  "voice":  "string?",          // server-specific voice id (e.g. "aura-2-thalia-en")
  "speed":  "number?"           // 0.5..2.0; 1.0 = default
}
```

**Output:**

```jsonc
{
  "audio_b64":   "string",      // required — base64-encoded audio bytes
  "mime_type":   "string",      // required — e.g. "audio/ogg;codecs=opus"
  "duration_ms": "number?"      // synthesized clip length, if known
}
```

Opendray will reject audio_b64 larger than 5 MB to keep Telegram
`sendVoice` happy (the platform's voice-note waveform render caps at
~1 MB; documents go up to 50 MB but lose the voice-note UX).

## Error model

Tool calls follow standard MCP error semantics. A server SHOULD surface
these `code` strings in `error.data.code` so channels can render
appropriate user-facing messages:

| `code`                | Meaning                                      | Opendray's response       |
|-----------------------|----------------------------------------------|---------------------------|
| `auth_failed`         | API key missing / invalid / quota exhausted  | "Voice unavailable — check provider auth." |
| `audio_too_large`     | Payload exceeds the server's accepted size   | "Voice note too long — try shorter." |
| `unsupported_format`  | mime_type / output format not handled        | "Unsupported audio format." |
| `language_unsupported`| Detected/hinted language not supported       | "Language not supported." |
| `provider_unavailable`| Upstream third party is down or rate-limited | "Voice service temporarily down." |

Any other error is logged and surfaced as the generic
"Couldn't process that voice note" reply.

## Discovery

Opendray's existing MCP loader (`internal/mcp/mcp.go`) lists every
server installed under `<vault>/mcp/`. The channel admin UI filters
the list to servers whose `tools/list` includes one or both voice
tool names, and presents them as the **Voice provider** dropdown on
each channel's config form.

A server registers itself by being installed into the vault — same
mechanism as any other MCP server. No additional manifest is required.

## Channel configuration

A channel that wants voice references a server by name and toggles
which capabilities it uses:

```jsonc
{
  "voice_mcp_server":            "deepgram-default",  // matches the server's id in the vault
  "voice_transcription_enabled": true,                 // requires the server to expose voice/transcribe
  "voice_reply_enabled":         false                 // requires the server to expose voice/synthesize
}
```

`voice_mcp_server` empty / absent = voice off (default). Toggling a
capability for which the bound server has no tool is a no-op with a
logged warning.

## Reference implementation

`@opendray/voice-deepgram` (separate repo, published to npm) is a
stdio MCP server that implements both tools against the Deepgram REST
API:

- `voice/transcribe` → POST `/v1/listen` (Nova-3, language=multi,
  smart_format)
- `voice/synthesize` → POST `/v1/speak` (Aura-2)

Operators install it via the Plugins page (or `pnpm install -g`); the
Deepgram API key is supplied via Opendray's secrets file using a
`${DEEPGRAM_API_KEY}` placeholder in the server's `mcp.json`.

The reference implementation is one supported option, not the only
one. A community-maintained Whisper-local server, an ElevenLabs
server, an AssemblyAI server, etc., all conform to the same contract
and interoperate without Opendray changes.

## Versioning

This contract is `voice-mcp/1`. Backwards-incompatible changes go in
`voice-mcp/2` with a separate document and a deprecation window. Servers
MAY advertise the contract version they implement in their MCP
`serverInfo.version` field; Opendray treats absence as "1".
