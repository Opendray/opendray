package knowledge

import (
	"context"
	"encoding/json"
	"strings"
)

// LLM is the minimal completion surface the entity extractor needs. The app
// wires a summarizer-backed adapter; knowledge owns this interface so it never
// imports internal/memory (the one-way dependency rule).
type LLM interface {
	// Complete runs one system+user prompt and returns the raw model text.
	Complete(ctx context.Context, system, user string) (string, error)
}

// ExtractedEntity is one entity the LLM pulled from a fact.
type ExtractedEntity struct {
	Name string
	Type EntityType
}

const entityExtractSystem = `You extract the few CANONICAL, REUSABLE entities named in one project fact.
Return ONLY compact JSON: {"entities":[{"name":"...","type":"..."}]}
"type" MUST be exactly one of: service, host, project, tool, decision, tech, person.
Rules:
- AT MOST 3 entities; fewer is better. If the fact names none, return {"entities":[]}.
- Only NAMED, specific things (e.g. "PostgreSQL", "kv01", "pnpm"). Do NOT extract
  generic concepts, features, adjectives, or activities (skip "app", "feature",
  "AR", "measurement", "the database", "industrial-grade").
- name = the canonical proper name only, never a phrase.
- Output JSON only: no prose, no markdown fences.`

// ExtractEntities asks the LLM to pull entities from a fact and returns the
// valid, de-duplicated ones (unknown types are dropped). An empty result is a
// normal outcome, not an error.
func ExtractEntities(ctx context.Context, llm LLM, factText string) ([]ExtractedEntity, error) {
	raw, err := llm.Complete(ctx, entityExtractSystem, factText)
	if err != nil {
		return nil, err
	}
	return parseExtracted(raw), nil
}

// parseExtracted is a defensive JSON parser: it tolerates code fences / stray
// prose around the object, drops unknown entity types, and de-dupes by
// (type, lowercased name). Returns nil on unparseable input.
func parseExtracted(raw string) []ExtractedEntity {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '{'); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndexByte(raw, '}'); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	var parsed struct {
		Entities []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"entities"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	out := make([]ExtractedEntity, 0, len(parsed.Entities))
	seen := map[string]struct{}{}
	for _, e := range parsed.Entities {
		name := strings.TrimSpace(e.Name)
		et := EntityType(strings.ToLower(strings.TrimSpace(e.Type)))
		if name == "" || !et.Valid() {
			continue
		}
		key := string(et) + "\x00" + strings.ToLower(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ExtractedEntity{Name: name, Type: et})
	}
	// Hard cap as a guard against an over-eager model, even if the prompt
	// is ignored — keeps the graph from bloating with low-value entities.
	if len(out) > 4 {
		out = out[:4]
	}
	return out
}
