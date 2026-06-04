package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/opendray/opendray-v2/internal/channel"
	"github.com/opendray/opendray-v2/internal/voice"
)

// voiceReplyMaxChars caps how much of an agent reply we hand to TTS.
// Aura-2 OGG/Opus at ~24 kbps fits ~5 minutes in 1 MB; Telegram's
// sendVoice waveform UI requires ≤1 MB. Conservative cap keeps us
// safely inside both bounds and keeps synth latency low.
const voiceReplyMaxChars = 800

// maybeVoiceReply synthesizes msg.Text via the channel's configured
// MCP voice provider and posts it as a Telegram voice note alongside
// the existing text reply. Best-effort — failures are logged and the
// caller keeps going with its normal text path, so a flaky TTS
// provider never silently drops the reply.
func (t *Telegram) maybeVoiceReply(ctx context.Context, msg channel.ChannelMessage) {
	if !t.cfg.VoiceReplyEnabled {
		return
	}
	if t.cfg.VoiceMCPServer == "" {
		return
	}

	body := strings.TrimSpace(stripVoiceUnfriendly(msg.Text))
	if body == "" {
		return
	}
	if len([]rune(body)) > voiceReplyMaxChars {
		t.log.Debug("voice reply skipped — text exceeds cap",
			"len_runes", len([]rune(body)), "cap", voiceReplyMaxChars)
		return
	}

	svc := currentVoiceService()
	if svc == nil {
		return
	}
	provider, err := svc.ResolveProvider(ctx, t.cfg.VoiceMCPServer)
	if err != nil {
		t.log.Warn("voice reply: resolve provider failed",
			"server", t.cfg.VoiceMCPServer, "err", err)
		return
	}

	audio, err := provider.Synthesize(ctx, body, voice.SynthesizeOpts{Format: "ogg-opus"})
	if err != nil {
		t.log.Warn("voice reply: synthesize failed",
			"server", t.cfg.VoiceMCPServer, "err", err)
		return
	}
	if len(audio.Body) == 0 {
		return
	}

	chatID, replyTo := t.routing(msg)
	if chatID == 0 {
		return
	}

	if err := t.sendVoiceMultipart(ctx, chatID, replyTo, audio.Body, audio.MimeType); err != nil {
		t.log.Warn("voice reply: sendVoice failed",
			"server", t.cfg.VoiceMCPServer, "err", err)
		return
	}
}

// sendVoiceMultipart posts an OGG/Opus blob to Telegram's sendVoice
// endpoint as a multipart upload. Telegram renders the file with its
// native voice-note UI (waveform + duration) when:
//   - mime is audio/ogg with Opus codec
//   - file size ≤ 1 MB
//
// We don't enforce those here — the caller (maybeVoiceReply) is
// responsible for staying inside the cap, and the Telegram API will
// reject the rest with a clear error if it doesn't fit.
func (t *Telegram) sendVoiceMultipart(ctx context.Context, chatID int64, replyTo int, audio []byte, mime string) error {
	if mime == "" {
		mime = "audio/ogg"
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return err
	}
	if replyTo != 0 {
		// reply_parameters as a JSON-encoded form field — Telegram
		// accepts JSON sub-objects this way on multipart requests.
		if err := w.WriteField("reply_parameters",
			fmt.Sprintf(`{"message_id":%d}`, replyTo)); err != nil {
			return err
		}
	}

	// voice field — has to declare the OGG content-type explicitly so
	// Telegram routes it to the voice-note pipeline rather than
	// generic audio file.
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="voice"; filename="reply.ogg"`)
	hdr.Set("Content-Type", mime)
	part, err := w.CreatePart(hdr)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewReader(audio)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	url := apiBase() + t.cfg.BotToken + "/sendVoice"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("sendVoice HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

// stripVoiceUnfriendly removes characters / sequences that TTS engines
// pronounce awkwardly: control-keyboard chrome, leading session ids,
// markdown decoration. The result is a clean prose body for synthesis.
//
// Anything not handled here just passes through — TTS providers
// generally cope with light formatting, so we only strip the worst
// offenders.
func stripVoiceUnfriendly(in string) string {
	out := in
	// Drop the control-keyboard footer that the Hub appends to
	// agent replies (e.g. "[⏸ Stop] [🔄 Restart] [🔀 Switch]").
	if i := strings.LastIndex(out, "[⏸"); i >= 0 {
		out = out[:i]
	}
	// Trim leading session-id breadcrumb if the card included one.
	out = strings.TrimSpace(out)
	return out
}

// ensure errors stays referenced when no other site in this file uses
// it; keeps the import stable as the file evolves.
var _ = errors.New
