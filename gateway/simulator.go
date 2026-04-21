package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// simulatorScreenshot captures a screenshot from the running iOS Simulator or
// Android Emulator and returns it as a PNG image.
func (s *Server) simulatorScreenshot(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "ios"
	}
	device := r.URL.Query().Get("device")

	imgData, err := captureScreenshot(r.Context(), platform, device)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	w.Write(imgData) //nolint:errcheck
}

// simulatorInput sends a touch/key/text event to the running emulator.
func (s *Server) simulatorInput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform string `json:"platform"`
		Device   string `json:"device"`
		Action   string `json:"action"` // tap | swipe | keyevent | text
		X        int    `json:"x"`
		Y        int    `json:"y"`
		X2       int    `json:"x2"`
		Y2       int    `json:"y2"`
		Duration int    `json:"duration"` // swipe duration ms
		Key      string `json:"key"`      // KEYCODE_* for keyevent
		Text     string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Platform == "" {
		req.Platform = "android"
	}
	if err := sendInput(r.Context(), req.Platform, req.Device, req.Action,
		req.X, req.Y, req.X2, req.Y2, req.Duration, req.Key, req.Text); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ── Internal helpers ─────────────────────────────────────────────

func captureScreenshot(ctx context.Context, platform, device string) ([]byte, error) {
	switch platform {
	case "android":
		args := []string{"exec-out", "screencap", "-p"}
		if device != "" {
			args = append([]string{"-s", device}, args...)
		}
		data, err := exec.CommandContext(ctx, "adb", args...).Output()
		if err != nil {
			return nil, fmt.Errorf("adb screencap: %w", err)
		}
		return data, nil

	default: // ios
		if device == "" {
			device = "booted"
		}
		tmpPath := fmt.Sprintf("/tmp/opendray_sim_%d.png", time.Now().UnixNano())
		defer os.Remove(tmpPath)
		if err := exec.CommandContext(ctx, "xcrun", "simctl", "io", device, "screenshot", tmpPath).Run(); err != nil {
			return nil, fmt.Errorf("xcrun simctl screenshot: %w", err)
		}
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return nil, fmt.Errorf("read screenshot: %w", err)
		}
		return data, nil
	}
}

func sendInput(ctx context.Context, platform, device, action string, x, y, x2, y2, duration int, key, text string) error {
	switch platform {
	case "android":
		return sendADBInput(ctx, device, action, x, y, x2, y2, duration, key, text)
	default:
		// iOS simulator does not expose a general touch injection API.
		// Keyevent only (e.g. hardware home/back via simctl).
		if action == "keyevent" {
			return sendSimctlKey(ctx, device, key)
		}
		return fmt.Errorf("touch input not supported for iOS simulator — use Android emulator")
	}
}

func sendADBInput(ctx context.Context, device, action string, x, y, x2, y2, duration int, key, text string) error {
	adb := func(args ...string) *exec.Cmd {
		if device != "" {
			return exec.CommandContext(ctx, "adb", append([]string{"-s", device}, args...)...)
		}
		return exec.CommandContext(ctx, "adb", args...)
	}
	switch action {
	case "tap":
		return adb("shell", "input", "tap",
			strconv.Itoa(x), strconv.Itoa(y)).Run()
	case "swipe":
		if duration <= 0 {
			duration = 300
		}
		return adb("shell", "input", "swipe",
			strconv.Itoa(x), strconv.Itoa(y),
			strconv.Itoa(x2), strconv.Itoa(y2),
			strconv.Itoa(duration)).Run()
	case "keyevent":
		return adb("shell", "input", "keyevent", key).Run()
	case "text":
		// ADB text input treats spaces specially
		escaped := strings.ReplaceAll(text, " ", "%s")
		return adb("shell", "input", "text", escaped).Run()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func sendSimctlKey(ctx context.Context, device, key string) error {
	if device == "" {
		device = "booted"
	}
	// Map common key names to simctl button names
	button := map[string]string{
		"KEYCODE_HOME":   "home",
		"KEYCODE_BACK":   "back",
		"home":           "home",
		"back":           "back",
	}[key]
	if button == "" {
		return fmt.Errorf("unsupported iOS key: %s", key)
	}
	return exec.CommandContext(ctx, "xcrun", "simctl", "ui", device, "button", button).Run()
}
