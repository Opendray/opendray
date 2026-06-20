package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/session"
)

// OpenCode is provider-agnostic: it reads providers, the default model,
// MCP servers, and instruction files from a single JSON config. opendray
// generates a per-session config in baseDir and points OpenCode at it via
// OPENCODE_CONFIG, which OpenCode loads BETWEEN the user's global
// (~/.config/opencode) and project configs — so the user's own providers
// and auth keep working underneath this session overlay.
//
// Several spawn-prep steps (local-provider injection, MCP rendering,
// skill / memory instruction wiring) each contribute a slice of that
// config. They run sequentially within one spawn's Prepare, so each does
// a read-modify-write merge rather than fighting over the file.

const (
	openCodeConfigFile = "opencode.json"
	openCodeAgentsFile = "AGENTS.md"
	// openCodeLocalProvider is the provider key opendray registers when
	// the operator points OpenCode at a local OpenAI-compatible endpoint.
	openCodeLocalProvider = "opendray-local"
)

func openCodeConfigPath(baseDir string) string { return filepath.Join(baseDir, openCodeConfigFile) }
func openCodeAgentsPath(baseDir string) string { return filepath.Join(baseDir, openCodeAgentsFile) }

// mergeOpenCodeConfig read-modify-writes the per-session OpenCode config
// JSON in baseDir, applying mutate to the decoded map. Creates the file
// (with the OpenCode $schema) when absent. Callers run sequentially
// within one spawn's Prepare, so no locking is needed.
func mergeOpenCodeConfig(baseDir string, mutate func(map[string]any)) error {
	path := openCodeConfigPath(baseDir)
	cfg := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if _, ok := cfg["$schema"]; !ok {
		cfg["$schema"] = "https://opencode.ai/config.json"
	}
	mutate(cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal opencode config: %w", err)
	}
	// 0600: the config may embed a local-endpoint API key and the
	// resolved opendray-memory integration key inside the mcp block.
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// setOpenCodeConfigEnv points OpenCode at the generated per-session config.
func setOpenCodeConfigEnv(baseDir string, out *session.PrepareOutput) {
	if out.Env == nil {
		out.Env = map[string]string{}
	}
	out.Env["OPENCODE_CONFIG"] = openCodeConfigPath(baseDir)
}

// ensureOpenCodeInstructions makes the generated config reference the
// per-session AGENTS.md (where skills / memory guidance / ambient context
// are written) and points OPENCODE_CONFIG at the config. Safe to call
// repeatedly — the instructions entry is de-duplicated.
func ensureOpenCodeInstructions(baseDir string, out *session.PrepareOutput) error {
	agents := openCodeAgentsPath(baseDir)
	if err := mergeOpenCodeConfig(baseDir, func(cfg map[string]any) {
		var list []any
		if existing, ok := cfg["instructions"].([]any); ok {
			list = existing
		}
		for _, v := range list {
			if s, _ := v.(string); s == agents {
				return // already referenced
			}
		}
		cfg["instructions"] = append(list, agents)
	}); err != nil {
		return err
	}
	setOpenCodeConfigEnv(baseDir, out)
	return nil
}

// injectOpenCodeLocalProvider registers a `opendray-local` OpenAI-compatible
// provider in the generated config when the operator set a local endpoint
// base URL, and default-selects a model. A no-op when no local endpoint is
// configured (the operator uses their own ~/.config/opencode providers).
//
// OpenCode's custom OpenAI-compatible provider does NOT auto-discover models,
// so opendray enumerates them: the endpoint's /models is probed and every
// (chat) model id is added to the provider's `models` map. This way the
// operator only enters a base URL and OpenCode's /model picker lists the
// whole local catalog — `localModel`, if set, just pins the default.
func injectOpenCodeLocalProvider(ctx context.Context, baseDir string, cfg map[string]any, out *session.PrepareOutput) error {
	baseURL := strings.TrimSpace(stringFromConfig(cfg, "localBaseUrl"))
	if baseURL == "" {
		return nil
	}
	localModel := strings.TrimSpace(stringFromConfig(cfg, "localModel"))
	apiKey := strings.TrimSpace(stringFromConfig(cfg, "localApiKey"))
	explicitModel := strings.TrimSpace(stringFromConfig(cfg, "model")) != ""

	// Best-effort enumeration — a probe failure (endpoint down) just falls
	// back to the explicitly named localModel, so a spawn is never blocked.
	// probeErr distinguishes "unreachable" from "reached but no chat model"
	// so the spawn-time notice below can be specific.
	modelIDs, probeErr := probeOpenCodeModels(ctx, baseURL, apiKey)

	models := map[string]any{}
	for _, id := range modelIDs {
		models[id] = map[string]any{"name": id}
	}
	if localModel != "" {
		models[localModel] = map[string]any{"name": localModel}
	}
	// Default model: the operator's localModel if set, else the first probed
	// model so the session opens directly on a working local model.
	defaultModel := localModel
	if defaultModel == "" && len(modelIDs) > 0 {
		defaultModel = modelIDs[0]
	}

	if err := mergeOpenCodeConfig(baseDir, func(c map[string]any) {
		providers, _ := c["provider"].(map[string]any)
		if providers == nil {
			providers = map[string]any{}
		}
		options := map[string]any{"baseURL": baseURL}
		if apiKey != "" {
			options["apiKey"] = apiKey
		}
		providers[openCodeLocalProvider] = map[string]any{
			"npm":     "@ai-sdk/openai-compatible",
			"name":    "Local (opendray)",
			"options": options,
			"models":  models,
		}
		c["provider"] = providers
		// Default-select a local model unless the operator pinned an explicit
		// `model` (which goes through --model and wins anyway).
		if !explicitModel && defaultModel != "" {
			c["model"] = openCodeLocalProvider + "/" + defaultModel
		}
	}); err != nil {
		return err
	}
	setOpenCodeConfigEnv(baseDir, out)

	// Spawn-time diagnostics. The probe is best-effort and never blocks the
	// spawn, so without this the operator only sees OpenCode's opaque
	// "[buffer unavailable]" when the session can't produce a response.
	// Surface a one-time hint in the session terminal (and transcript)
	// describing the actual failure. Only fires when no chat model is
	// available — a healthy enumeration stays silent.
	if len(modelIDs) == 0 {
		havePinned := localModel != "" || explicitModel
		switch {
		case probeErr != nil && !havePinned:
			out.Notices = append(out.Notices, fmt.Sprintf(
				"opendray: the OpenCode local endpoint %s is unreachable (%v), so this session has no local model and OpenCode will fail with \"[buffer unavailable]\". "+
					"From the gateway host run:  curl %s/models  — check the URL ends in /v1, the endpoint serves on the LAN (not just localhost), and a chat model is loaded; then set a Default Local Model and start a new session.",
				baseURL, probeErr, baseURL))
		case probeErr != nil && havePinned:
			out.Notices = append(out.Notices, fmt.Sprintf(
				"opendray: couldn't reach the OpenCode local endpoint %s (%v) to verify models; falling back to your pinned model. If the session can't respond, confirm the endpoint is up and reachable from the gateway host.",
				baseURL, probeErr))
		case probeErr == nil && !havePinned:
			out.Notices = append(out.Notices, fmt.Sprintf(
				"opendray: the OpenCode local endpoint %s is reachable but served no chat-capable models (only embeddings/rerank, or none loaded), so this session has no local model and OpenCode will fail with \"[buffer unavailable]\". "+
					"Load a chat model on the endpoint, or set a Default Local Model, then start a new session.",
				baseURL))
		}
	}
	return nil
}

// probeOpenCodeModels lists the chat/coding model ids served at an
// OpenAI-compatible endpoint (GET <baseURL>/models). Embedding /
// transcription / rerank ids are filtered out so they don't clutter
// OpenCode's /model picker. Best-effort: a non-nil error means the
// endpoint was unreachable / refused / unparseable (the caller never
// blocks a spawn on it, but surfaces a one-time notice). A nil error with
// an empty slice means "reached it, but it served no chat-capable model".
// A package var so tests can stub it without a live endpoint.
var probeOpenCodeModels = func(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint returned HTTP %d", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode /models response: %w", err)
	}
	ids := make([]string, 0, len(body.Data))
	for _, m := range body.Data {
		if id := strings.TrimSpace(m.ID); id != "" && isOpenCodeChatModel(id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// isOpenCodeChatModel drops obvious non-chat model ids (embeddings,
// transcription, rerank, tts) by a conservative substring denylist so the
// /model picker only offers usable coding models.
func isOpenCodeChatModel(id string) bool {
	lc := strings.ToLower(id)
	for _, bad := range []string{"embed", "whisper", "rerank", "tts", "stt", "transcrib"} {
		if strings.Contains(lc, bad) {
			return false
		}
	}
	return true
}

// renderOpenCodeMCP merges the session's MCP servers into the generated
// config's `mcp` block (OpenCode's native shape: {type, command[], enabled,
// environment}) and returns the OPENCODE_CONFIG env so the spawn picks the
// file up. Empty server list → no-op.
func renderOpenCodeMCP(baseDir string, servers []MCPServer) ([]string, map[string]string, error) {
	block := map[string]any{}
	for _, s := range servers {
		entry := openCodeMCPEntry(s)
		if entry == nil {
			continue
		}
		block[s.Name] = entry
	}
	if len(block) == 0 {
		return nil, nil, nil
	}
	if err := mergeOpenCodeConfig(baseDir, func(cfg map[string]any) {
		existing, _ := cfg["mcp"].(map[string]any)
		if existing == nil {
			existing = map[string]any{}
		}
		for k, v := range block {
			existing[k] = v
		}
		cfg["mcp"] = existing
	}); err != nil {
		return nil, nil, err
	}
	return nil, map[string]string{"OPENCODE_CONFIG": openCodeConfigPath(baseDir)}, nil
}

// openCodeMCPEntry maps one MCPServer into OpenCode's mcp config shape.
// stdio/local → {type:"local", command:[cmd, args...], environment}.
// sse/http → {type:"remote", url, headers}. Returns nil for entries with
// neither a command nor a URL.
func openCodeMCPEntry(s MCPServer) map[string]any {
	if s.URL != "" {
		entry := map[string]any{"type": "remote", "url": s.URL, "enabled": true}
		if len(s.Headers) > 0 {
			entry["headers"] = s.Headers
		}
		return entry
	}
	if s.Command == "" {
		return nil
	}
	// command+args as a single list (OpenCode's local-server shape).
	// Appending s.Args to a cap-1 literal always reallocates, so the result
	// never aliases s.Args' backing array.
	command := append([]string{s.Command}, s.Args...)
	entry := map[string]any{"type": "local", "command": command, "enabled": true}
	if len(s.Env) > 0 {
		entry["environment"] = s.Env
	}
	return entry
}

// stringFromConfig reads a string value from a provider config map,
// returning "" for missing / non-string keys.
func stringFromConfig(cfg map[string]any, key string) string {
	v, _ := cfg[key].(string)
	return v
}

// wantsOpenCodeSessionConfig reports whether an OpenCode session needs the
// Prepare step solely to emit its generated OPENCODE_CONFIG (a local
// endpoint provider), independent of skills / MCP / schema env. Resolve
// uses this so the no-Prepare fast path can't silently drop a configured
// local endpoint when skills and memory MCP both happen to be off.
func wantsOpenCodeSessionConfig(id string, cfg map[string]any) bool {
	return id == "opencode" && strings.TrimSpace(stringFromConfig(cfg, "localBaseUrl")) != ""
}
