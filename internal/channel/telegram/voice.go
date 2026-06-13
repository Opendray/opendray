package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/opendray/opendray-v2/internal/voice"
)

// voiceSvc is the package-level handle. The app sets it at startup
// (after the voice service is constructed); a nil value means voice
// is unavailable and any voice-note inbound will surface a friendly
// "voice not configured" reply.
var (
	voiceMu  sync.RWMutex
	voiceSvc *voice.Service
)

// SetVoiceService is called once by the gateway wiring to hand the
// Telegram channel its voice service. Safe to call before any
// channels start; if called later, only voice notes that arrive
// afterwards pick it up.
func SetVoiceService(s *voice.Service) {
	voiceMu.Lock()
	voiceSvc = s
	voiceMu.Unlock()
}

func currentVoiceService() *voice.Service {
	voiceMu.RLock()
	defer voiceMu.RUnlock()
	return voiceSvc
}

const maxVoiceDownload = 5 * 1024 * 1024

// downloadVoiceFile fetches the OGG blob behind a Telegram file_id.
// Returns the raw bytes + the platform-reported content-type. The
// bounded read guards against a runaway upload; Telegram's own voice
// notes are well under this cap.
func (t *Telegram) downloadVoiceFile(ctx context.Context, fileID string) ([]byte, string, error) {
	var resp struct {
		Ok     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
			FileSize int64  `json:"file_size"`
		} `json:"result"`
	}
	if err := t.callAPI(ctx, "getFile", map[string]any{"file_id": fileID}, &resp); err != nil {
		return nil, "", fmt.Errorf("getFile: %w", err)
	}
	if !resp.Ok || resp.Result.FilePath == "" {
		return nil, "", errors.New("getFile: empty file_path")
	}

	// Telegram file CDN URLs are bot-scoped and contain the token.
	// apiBase ends in "bot"; strip it to compose the file CDN prefix.
	dlURL := strings.TrimSuffix(apiBase(), "bot") + "file/bot" +
		url.PathEscape(t.cfg.BotToken) + "/" + resp.Result.FilePath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return nil, "", err
	}
	httpResp, err := t.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("download: HTTP %d", httpResp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(httpResp.Body, maxVoiceDownload+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxVoiceDownload {
		return nil, "", fmt.Errorf("download: payload exceeds %d bytes", maxVoiceDownload)
	}
	return data, httpResp.Header.Get("Content-Type"), nil
}

// transcribeVoice runs the inbound voice-note pipeline: download from
// Telegram, hand the audio bytes off to the bound MCP voice provider,
// return the transcript text.
//
// Errors are wrapped with enough context for log diagnosis; the caller
// is responsible for picking the user-facing chat message.
func (t *Telegram) transcribeVoice(ctx context.Context, v *tgVoice) (string, error) {
	if v == nil || v.FileID == "" {
		return "", errors.New("voice: empty file_id")
	}
	svc := currentVoiceService()
	if svc == nil {
		return "", errors.New("voice: service not wired")
	}
	if t.cfg.VoiceMCPServer == "" {
		return "", errors.New("voice: no provider configured")
	}

	provider, err := svc.ResolveProvider(ctx, t.cfg.VoiceMCPServer)
	if err != nil {
		return "", fmt.Errorf("voice: resolve provider %q: %w", t.cfg.VoiceMCPServer, err)
	}

	body, mime, err := t.downloadVoiceFile(ctx, v.FileID)
	if err != nil {
		return "", fmt.Errorf("voice: download: %w", err)
	}
	if mime == "" {
		mime = v.MimeType
	}
	if mime == "" {
		mime = "audio/ogg"
	}

	transcript, err := provider.Transcribe(ctx, body, mime)
	if err != nil {
		return "", fmt.Errorf("voice: transcribe: %w", err)
	}
	return strings.TrimSpace(transcript.Text), nil
}
