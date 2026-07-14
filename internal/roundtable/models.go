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
// typo otherwise fails the whole seat). antigravity is enumerated LIVE from
// `agy models`; claude and codex have no CLI model-list command, so they use
// a curated set — claude's documented aliases, and for codex the model that
// works on a plain ChatGPT plan (its own default, gpt-5.4, is rejected there).
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
	}
}

// antigravityModels lists the models `agy` actually offers. Falls back to a
// bare Default option when the CLI is missing or errors.
func antigravityModels(ctx context.Context) []ModelOption {
	opts := []ModelOption{{Value: "", Label: "Default"}}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	bin := "agy"
	if p, err := exec.LookPath(bin); err == nil {
		bin = p
	}
	cmd := exec.CommandContext(cctx, bin, "models")
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
