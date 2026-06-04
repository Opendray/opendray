package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Voice-note transcription. Telegram → Deepgram → text.
//
// Inbound flow (deliverMessage):
//  1. tgMessage.Voice arrives with an opaque FileID.
//  2. getFile (Telegram Bot API) resolves FileID → file_path.
//  3. Download the OGG/Opus blob from the file CDN.
//  4. POST it to Deepgram /v1/listen.
//  5. The transcript becomes msg.Text; everything downstream of
//     deliverMessage stays unchanged.
//
// Telegram caps voice notes at 1MB on the receive side (~5 min at the
// usual 24 kbps), so we don't stream — one bounded read fits.

const (
	deepgramListenURL    = "https://api.deepgram.com/v1/listen"
	deepgramTimeout      = 30 * time.Second
	maxVoiceDownloadSize = 5 * 1024 * 1024 // 5MB hard ceiling on the OGG payload
)

// errVoiceTranscriptionDisabled is returned when a voice message
// arrives but the channel doesn't have transcription enabled. Caller
// surfaces a friendly reply instead of silently dropping.
var errVoiceTranscriptionDisabled = errors.New("voice transcription not enabled for this channel")

// transcribeVoice pulls the voice payload from Telegram, hands it to
// Deepgram, and returns the transcript text.
//
// The caller is responsible for checking VoiceTranscriptionEnabled
// before invoking — this function will error out if the API key is
// missing.
func (t *Telegram) transcribeVoice(ctx context.Context, v *tgVoice) (string, error) {
	if v == nil || v.FileID == "" {
		return "", errors.New("voice: empty file_id")
	}
	if t.cfg.VoiceTranscriptionAPIKey == "" {
		return "", errors.New("voice: deepgram api key not configured")
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

	return t.deepgramTranscribe(ctx, body, mime)
}

// downloadVoiceFile fetches the OGG blob behind a Telegram file_id.
// Returns the raw bytes + the platform-reported content-type.
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
	// Build manually (apiBase strips "bot" off so callAPI can't be reused).
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

	// Bounded read — Telegram caps voice notes well below this, the
	// limit is just a runaway guard.
	data, err := io.ReadAll(io.LimitReader(httpResp.Body, maxVoiceDownloadSize+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxVoiceDownloadSize {
		return nil, "", fmt.Errorf("download: payload exceeds %d bytes", maxVoiceDownloadSize)
	}
	return data, httpResp.Header.Get("Content-Type"), nil
}

// deepgramTranscribe sends the audio bytes to Deepgram and returns the
// first alternative transcript across all channels.
//
// We use nova-3 multilingual with smart_format on by default —
// punctuation + capitalisation make the transcript usable as session
// stdin without further cleanup.
func (t *Telegram) deepgramTranscribe(ctx context.Context, audio []byte, mime string) (string, error) {
	q := url.Values{}
	q.Set("model", "nova-3")
	q.Set("language", "multi")
	q.Set("smart_format", "true")
	q.Set("punctuate", "true")

	reqURL := deepgramListenURL + "?" + q.Encode()
	ctx, cancel := context.WithTimeout(ctx, deepgramTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(audio))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Token "+t.cfg.VoiceTranscriptionAPIKey)
	req.Header.Set("Content-Type", mime)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepgram request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("deepgram read: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		// Deepgram returns JSON errors but a non-2xx with a short body
		// is enough for the operator to diagnose (bad key, quota, etc.).
		return "", fmt.Errorf("deepgram HTTP %d: %s", resp.StatusCode, truncForLog(body))
	}

	var dg struct {
		Results struct {
			Channels []struct {
				Alternatives []struct {
					Transcript string  `json:"transcript"`
					Confidence float64 `json:"confidence"`
				} `json:"alternatives"`
			} `json:"channels"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &dg); err != nil {
		return "", fmt.Errorf("deepgram decode: %w", err)
	}
	if len(dg.Results.Channels) == 0 || len(dg.Results.Channels[0].Alternatives) == 0 {
		return "", errors.New("deepgram: no transcript returned")
	}
	text := strings.TrimSpace(dg.Results.Channels[0].Alternatives[0].Transcript)
	if text == "" {
		return "", errors.New("deepgram: empty transcript")
	}
	return text, nil
}

func truncForLog(b []byte) string {
	const max = 200
	s := string(bytes.TrimSpace(b))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
