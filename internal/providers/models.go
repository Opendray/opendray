// Package providers holds cross-feature knowledge about the agent CLIs
// opendray drives headlessly (claude / codex / antigravity / grok / opencode).
// It is deliberately dependency-free so both the Round Table (internal/
// roundtable) and the Cortex agent worker (internal/memory/worker) can share
// one source of truth for the selectable model list instead of each keeping a
// static catalog that drifts as the CLIs update.
package providers

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// ModelOption is one selectable model for an agent provider. Value is what
// gets passed to the CLI's --model flag ("" = the CLI's own default); Label
// is what the operator sees. Recommended marks the safe defaults (stable
// aliases that track the latest version, immune to version-number drift).
type ModelOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Recommended bool   `json:"recommended,omitempty"`
}

// AgentModelOptions returns the selectable models per agent provider for the
// full worker-backed set (claude / codex / antigravity / grok / opencode).
// antigravity and opencode are enumerated LIVE from their CLIs (`agy models` /
// `opencode models`) so the list can't drift; claude / codex / grok have no
// clean machine-readable model list, so they use a curated set — claude's
// documented aliases (which the CLI resolves to the current version), codex the
// model that works on a plain ChatGPT plan (its own default, gpt-5.4, is
// rejected there), and grok its two documented models (grok-4.5 is the CLI's
// own default).
func AgentModelOptions(ctx context.Context) map[string][]ModelOption {
	return map[string][]ModelOption{
		"claude": {
			{Value: "", Label: "Default", Recommended: true},
			{Value: "opus", Label: "Opus"},
			{Value: "sonnet", Label: "Sonnet"},
			{Value: "haiku", Label: "Haiku"},
		},
		// codex has no machine-readable model list, so it's curated. Offer
		// the whole gpt-5.4 family — a higher ChatGPT plan unlocks the fuller
		// models, so we must NOT suppress them to the one that happens to work
		// on a plain plan. mini is marked recommended because it's the one
		// that runs on ANY plan; the config default (bare gpt-5.4) is omitted
		// because a plain plan rejects it outright (an error, not a fallback).
		"codex": {
			{Value: "gpt-5.4-mini", Label: "gpt-5.4-mini — works on any plan", Recommended: true},
			{Value: "gpt-5.4-codex-mini", Label: "gpt-5.4-codex-mini — cheap coding"},
			{Value: "gpt-5.4-codex", Label: "gpt-5.4-codex — coding (higher plan)"},
			{Value: "gpt-5.4", Label: "gpt-5.4 — general (higher plan)"},
		},
		"antigravity": CLIModels(ctx, "agy", "models"),
		"grok": {
			{Value: "", Label: "Default", Recommended: true},
			{Value: "grok-4.5", Label: "grok-4.5"},
			{Value: "grok-composer-2.5-fast", Label: "grok-composer-2.5-fast"},
		},
		"opencode": CLIModels(ctx, "opencode", "models"),
	}
}

// CLIModels runs a CLI's model-list subcommand (e.g. `agy models`,
// `opencode models`) and returns one option per non-empty output line,
// prefixed with a bare Default. Any failure (missing CLI, non-zero exit)
// collapses to just Default so the dropdown still renders. The output is
// expected to be one model id per line — the format `agy` and `opencode` both
// use.
func CLIModels(ctx context.Context, bin string, args ...string) []ModelOption {
	opts := []ModelOption{{Value: "", Label: "Default", Recommended: true}}

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
