package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/linivek/ntc/kernel/store"
)

// OpenCode doesn't consume plain OPENAI_* env vars the way the rest of
// the OpenAI-compatible CLI ecosystem does — its providers are named
// and have to be declared in its own `opencode.json` config. To make a
// Mac Ollama or LM Studio endpoint show up we generate a per-session
// config that routes one provider name to the llm_providers row the
// user picked. Provider name == the row's `name` (lowercase slug), so
// the --model argument becomes "<providerName>/<modelId>".
//
// Placement: OpenCode exposes an OPENCODE_CONFIG env var that takes a
// direct path to the config file — preferred over XDG_CONFIG_HOME
// because it's unambiguous and doesn't require a nested directory
// layout to match OpenCode's internal search rules. We also set
// top-level `model` in the JSON so versions that ignore the --model
// flag still land on the right selection.

type opencodeConfig struct {
	Schema   string                  `json:"$schema,omitempty"`
	Model    string                  `json:"model,omitempty"`
	Provider map[string]opencodeProv `json:"provider"`
}

type opencodeProv struct {
	NPM     string                    `json:"npm"`
	Name    string                    `json:"name,omitempty"`
	Options map[string]any            `json:"options"`
	Models  map[string]map[string]any `json:"models"`
}

// OpenCodeInjection is the env-var payload the hub injects for an
// OpenCode session: both a file path (human-debuggable) and the raw
// JSON (matches Ollama's official integration, robust against config-
// discovery quirks).
type OpenCodeInjection struct {
	ConfigPath    string // absolute path to the generated opencode.json
	ConfigContent string // exact JSON bytes for OPENCODE_CONFIG_CONTENT
}

// buildOpenCodeConfig creates a per-session opencode.json and also
// returns its inline JSON. The model declared in the config is
// exactly sess.Model — OpenCode requires models to be listed under
// the provider block, so we always include the one the user picked.
//
// Directory is deterministic per session ID so cleanupOpenCodeConfig
// can find and remove it on exit without bookkeeping.
func buildOpenCodeConfig(sessionID string, p store.LLMProvider, apiKey, model string) (OpenCodeInjection, error) {
	if model == "" {
		return OpenCodeInjection{}, fmt.Errorf("opencode: session has no model selected; pick one from the provider's model list")
	}

	dir := openCodeConfigDir(sessionID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return OpenCodeInjection{}, fmt.Errorf("opencode: mkdir config: %w", err)
	}

	opts := map[string]any{
		"baseURL": p.BaseURL,
	}
	if apiKey != "" {
		opts["apiKey"] = apiKey
	}

	cfg := opencodeConfig{
		Schema: "https://opencode.ai/config.json",
		Model:  p.Name + "/" + model,
		Provider: map[string]opencodeProv{
			p.Name: {
				NPM:     "@ai-sdk/openai-compatible",
				Name:    p.DisplayName,
				Options: opts,
				Models: map[string]map[string]any{
					// A non-empty block helps OpenCode recognise
					// this as a valid model declaration rather than
					// skipping it silently.
					model: {"name": model},
				},
			},
		},
	}

	buf, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return OpenCodeInjection{}, fmt.Errorf("opencode: marshal config: %w", err)
	}
	path := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return OpenCodeInjection{}, fmt.Errorf("opencode: write config: %w", err)
	}
	return OpenCodeInjection{ConfigPath: path, ConfigContent: string(buf)}, nil
}

// cleanupOpenCodeConfig removes the per-session config tree. Safe to
// call even if we never wrote one (the RemoveAll is a no-op on a
// missing path).
func cleanupOpenCodeConfig(sessionID string) {
	_ = os.RemoveAll(openCodeConfigDir(sessionID))
}

// openCodeConfigDir picks a predictable, user-findable location for
// the generated config. macOS's os.TempDir() returns a hashed path
// like /var/folders/xx/yy/T/ which is awful to debug; putting our
// files under ~/.ntc makes them trivial to `ls` and diff.
func openCodeConfigDir(sessionID string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// Fallback to OS temp dir if somehow HOME is unavailable.
		return filepath.Join(os.TempDir(), "ntc-opencode-"+sessionID)
	}
	return filepath.Join(home, ".ntc", "opencode-sessions", sessionID)
}

// rewriteModelArg replaces the value of any existing "--model <val>"
// pair in args with newVal, or appends the pair if it's absent. We
// have to do this because the agent plugin's ResolveCLI already
// appended `--model <sess.Model>`, but OpenCode requires the value to
// be provider-qualified ("<providerName>/<model>").
func rewriteModelArg(args []string, newVal string) []string {
	out := make([]string, 0, len(args)+2)
	replaced := false
	for i := 0; i < len(args); i++ {
		if !replaced && args[i] == "--model" && i+1 < len(args) {
			out = append(out, "--model", newVal)
			i++ // skip the old value
			replaced = true
			continue
		}
		out = append(out, args[i])
	}
	if !replaced {
		out = append(out, "--model", newVal)
	}
	return out
}
