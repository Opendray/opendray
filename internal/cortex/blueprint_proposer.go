package cortex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/worker"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// BlueprintProposer asks an LLM (via the worker fabric, task
// "blueprint") to propose a doc section set tailored to what a
// project actually is — a mobile app wants different sections than a
// service or a CLI. Operator-triggered only and applied only on
// operator accept: no surprise LLM spend, AI proposes / human decides.
type BlueprintProposer struct {
	docs     *projectdoc.Service
	registry *worker.Registry
}

// NewBlueprintProposer wires the proposer. Either dep nil disables it
// (Propose returns an explanatory error).
func NewBlueprintProposer(docs *projectdoc.Service, reg *worker.Registry) *BlueprintProposer {
	return &BlueprintProposer{docs: docs, registry: reg}
}

// BlueprintProposal is the LLM's verdict: a full replacement section
// set plus the classification that justified it.
type BlueprintProposal struct {
	ProjectType string               `json:"project_type"`
	Reason      string               `json:"reason"`
	Sections    []projectdoc.Section `json:"sections"`
}

const blueprintSystemPrompt = `You design the documentation BLUEPRINT for a software project: the set of living doc sections an AI will keep current as work happens.

You are given the project's path, its current section set, and its scanned tech stack. Classify what kind of project this is (mobile app / web service / CLI tool / library / infrastructure / monorepo / …) and propose the section set that best documents THAT kind of project.

Rules:
- ALWAYS keep these slugs: "overview" (the official front page), "goal", "plan". Keep "tech_stack" and "recent_activity" (maintainer_mode "scanner") unless they make no sense.
- Add 1-4 sections specific to the project type. Examples: a mobile app might want "screens_and_flows" and "release_process"; a service might want "api_surface" and "operations"; a library might want "public_api" and "compatibility".
- Slugs: lowercase a-z0-9_ only, 2-48 chars, must not start with "kb_".
- maintainer_mode: "ai" for sections the AI should keep current from session work; "human" for sections only the operator should author; "scanner" only for tech_stack/recent_activity.
- prompt_hint: one sentence steering the AI maintainer of that section.
- position: 0-based display order. pinned=true only for "overview". inject=true for sections worth including in an agent's spawn context (keep the total lean — overview should be inject=false).
- Preserve the existing custom sections unless they clearly don't fit.

Respond ONLY with a JSON object (no prose, no code fences):
{
  "project_type": "<one short label>",
  "reason": "<one sentence shown to the operator>",
  "sections": [
    {"slug": "...", "title": "...", "description": "...", "position": 0,
     "maintainer_mode": "ai|human|scanner", "prompt_hint": "...",
     "pinned": false, "inject": true}
  ]
}`

const blueprintJSONSchema = `{
  "name": "blueprint_proposal",
  "schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "project_type": {"type": "string"},
      "reason":       {"type": "string"},
      "sections": {
        "type": "array",
        "items": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "slug":            {"type": "string"},
            "title":           {"type": "string"},
            "description":     {"type": "string"},
            "position":        {"type": "integer"},
            "maintainer_mode": {"type": "string"},
            "prompt_hint":     {"type": "string"},
            "pinned":          {"type": "boolean"},
            "inject":          {"type": "boolean"}
          },
          "required": ["slug", "title", "description", "position",
                       "maintainer_mode", "prompt_hint", "pinned", "inject"]
        }
      }
    },
    "required": ["project_type", "reason", "sections"]
  },
  "strict": true
}`

// Propose classifies cwd and returns a proposed section set. Nothing
// is persisted — the caller (UI) shows the proposal and applies it via
// the blueprint apply endpoint on operator accept.
func (p *BlueprintProposer) Propose(ctx context.Context, cwd string) (BlueprintProposal, error) {
	if p == nil || p.docs == nil || p.registry == nil {
		return BlueprintProposal{}, errors.New("cortex: blueprint proposer not configured")
	}
	if strings.TrimSpace(cwd) == "" {
		return BlueprintProposal{}, errors.New("cortex: cwd required")
	}

	sections, err := p.docs.ListSections(ctx, cwd)
	if err != nil {
		return BlueprintProposal{}, fmt.Errorf("cortex: list sections: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Project path\n\n`%s`\n\n", cwd)
	b.WriteString("## Current blueprint\n\n")
	for _, sec := range sections {
		fmt.Fprintf(&b, "- %s (%q, mode=%s, pos=%d)\n", sec.Slug, sec.Title, sec.MaintainerMode, sec.Position)
	}
	// The scanner-maintained tech_stack doc is the repo signal: stacks,
	// entry files, structure. Best-effort — a never-scanned project
	// still gets a (less informed) proposal.
	if d, derr := p.docs.GetDoc(ctx, cwd, projectdoc.KindTechStack); derr == nil && strings.TrimSpace(d.Content) != "" {
		b.WriteString("\n## Scanned tech stack\n\n")
		b.WriteString(truncate(d.Content, 6000))
		b.WriteString("\n")
	}
	if d, derr := p.docs.GetDoc(ctx, cwd, projectdoc.KindGoal); derr == nil && strings.TrimSpace(d.Content) != "" {
		b.WriteString("\n## Project goal\n\n")
		b.WriteString(truncate(d.Content, 2000))
		b.WriteString("\n")
	}

	resp, err := p.registry.Run(ctx, worker.Request{
		Task:                     worker.TaskBlueprint,
		SystemPrompt:             blueprintSystemPrompt,
		UserInput:                b.String(),
		MaxTokens:                4096,
		Timeout:                  3 * time.Minute,
		ResponseFormatJSONSchema: blueprintJSONSchema,
	})
	if err != nil {
		return BlueprintProposal{}, fmt.Errorf("cortex: blueprint llm: %w", err)
	}

	out, err := parseBlueprintProposal(resp.Content)
	if err != nil {
		return BlueprintProposal{}, err
	}
	// Validate + normalize before the UI ever sees it.
	cleaned := out.Sections[:0]
	for _, sec := range out.Sections {
		sec.Cwd = cwd
		if !projectdoc.ValidSectionSlug(sec.Slug) || !projectdoc.ValidMaintainerMode(sec.MaintainerMode) {
			continue
		}
		if sec.Slug == projectdoc.SlugOverview {
			sec.Pinned = true
		}
		cleaned = append(cleaned, sec)
	}
	out.Sections = cleaned
	if !hasSlug(out.Sections, projectdoc.SlugOverview) {
		return BlueprintProposal{}, errors.New("cortex: proposal dropped the reserved overview section")
	}
	return out, nil
}

func hasSlug(sections []projectdoc.Section, slug string) bool {
	for _, s := range sections {
		if s.Slug == slug {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…(truncated)"
}

// parseBlueprintProposal tolerates fenced / preambled JSON the same
// way the drift parser does.
func parseBlueprintProposal(raw string) (BlueprintProposal, error) {
	body := strings.TrimSpace(raw)
	if i := strings.IndexByte(body, '{'); i >= 0 {
		if j := strings.LastIndexByte(body, '}'); j > i {
			body = body[i : j+1]
		}
	}
	var out BlueprintProposal
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return BlueprintProposal{}, fmt.Errorf("cortex: unparseable blueprint proposal: %w", err)
	}
	if len(out.Sections) == 0 {
		return BlueprintProposal{}, errors.New("cortex: blueprint proposal has no sections")
	}
	return out, nil
}
