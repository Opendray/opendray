package roundtable

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// ModelOption is one selectable model for a seat provider. Value is what
// gets passed to the CLI's --model flag ("" = the CLI's own default);
// Label is what the operator sees.
type ModelOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ProviderModelOptions returns the selectable models per seat provider so the
// UI can offer a dropdown instead of a hand-typed model string (a one-char
// typo otherwise fails the whole seat). antigravity and opencode are
// enumerated LIVE from their CLIs (`agy models` / `opencode models`); claude,
// codex and grok have no easily-parsed model list, so they use a curated set —
// claude's documented aliases, codex the model that works on a plain ChatGPT
// plan (its own default, gpt-5.4, is rejected there), and grok its two
// documented build models.
func ProviderModelOptions(ctx context.Context) map[string][]ModelOption {
	return map[string][]ModelOption{
		"claude": {
			{Value: "", Label: "Default"},
			{Value: "opus", Label: "Opus"},
			{Value: "sonnet", Label: "Sonnet"},
			{Value: "haiku", Label: "Haiku"},
		},
		// No bare "Default" for codex: its config default (gpt-5.4) is not
		// allowed on a plain ChatGPT plan, so offering it would just fail.
		"codex": {
			{Value: "gpt-5.4-mini", Label: "gpt-5.4-mini"},
		},
		"antigravity": antigravityModels(ctx),
		// grok's documented build models (manifest knownModels). Default = the
		// CLI's own choice.
		"grok": {
			{Value: "", Label: "Default"},
			{Value: "grok-build", Label: "grok-build"},
			{Value: "grok-composer-2.5-fast", Label: "grok-composer-2.5-fast"},
		},
		"opencode": opencodeModels(ctx),
	}
}

// antigravityModels lists the models `agy` actually offers. Falls back to a
// bare Default option when the CLI is missing or errors.
func antigravityModels(ctx context.Context) []ModelOption {
	return cliModels(ctx, "agy", []string{"models"})
}

// opencodeModels lists the models `opencode models` offers (one provider/model
// per line, same shape as `agy models`). opencode is provider-agnostic, so the
// list reflects the operator's own opencode config/auth. Falls back to a bare
// Default option when the CLI is missing or errors.
func opencodeModels(ctx context.Context) []ModelOption {
	return cliModels(ctx, "opencode", []string{"models"})
}

// cliModels runs a CLI's model-list subcommand and returns one option per
// non-empty output line, prefixed with a bare Default. Any failure (missing
// CLI, non-zero exit) collapses to just Default so the dropdown still renders.
func cliModels(ctx context.Context, bin string, args []string) []ModelOption {
	opts := []ModelOption{{Value: "", Label: "Default"}}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if p, err := exec.LookPath(bin); err == nil {
		bin = p
	}
	cmd := exec.CommandContext(cctx, bin, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return opts
	}
	sc := bufio.NewScanner(&out)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		opts = append(opts, ModelOption{Value: line, Label: line})
	}
	return opts
}
