package memory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/summarizer"
)

// SummarizerGatekeeper implements Gatekeeper by re-using the
// existing summarizer subsystem: wrap the candidate memory as a
// single USER message and run Summarize. If the model extracts
// zero facts, the candidate is noise; if it extracts ≥1 fact, the
// candidate is durable and we lift the top fact's category back to
// the memory-side enum.
//
// Why this design vs. a separate "is_durable?" prompt:
//
//   - Zero new dependencies. Summarizer providers (anthropic, ollama,
//     openai-compat) already speak strict JSON with categories +
//     confidence. We get all of that for free.
//   - One source of truth for "what counts as durable" — the system
//     prompt in summarizer/prompt.go. Adding a parallel gatekeeper
//     prompt would drift.
//   - Cheap. Single user message means input tokens ≈ candidate
//     length (typically <200 tokens). With LM Studio + a 3B model
//     a judgement takes ~200ms on an M-series Mac.
//
// The mapping summarizer category → memory category:
//
//	preference → user_preference
//	identifier → project_fact
//	decision   → project_fact   (project-level decisions = facts)
//	task       → reference      (a pointer to ongoing work)
//	other      → project_fact   (fallback)
type SummarizerGatekeeper struct {
	reg        *summarizer.Registry
	providerID string        // empty → registry default
	maxLatency time.Duration // hard cap per judge call
	log        *slog.Logger
}

// NewSummarizerGatekeeper wires a gatekeeper backed by the
// summarizer registry. providerID picks a specific configured
// provider (memory_summarizer_providers.id); empty falls back to
// the registry's is_default row. maxLatency caps the per-call
// timeout — exceeding it returns an error which the Service
// degrades to "allow".
func NewSummarizerGatekeeper(
	reg *summarizer.Registry,
	providerID string,
	maxLatency time.Duration,
	log *slog.Logger,
) *SummarizerGatekeeper {
	if log == nil {
		log = slog.Default()
	}
	if maxLatency <= 0 {
		maxLatency = 2 * time.Second
	}
	return &SummarizerGatekeeper{
		reg:        reg,
		providerID: providerID,
		maxLatency: maxLatency,
		log:        log.With("component", "memory.gatekeeper"),
	}
}

// Judge implements Gatekeeper. Returns (durable, memoryCategory,
// reason, err) — when err is non-nil the caller should degrade to
// "allow" rather than blocking the write.
func (g *SummarizerGatekeeper) Judge(ctx context.Context, text string) (bool, string, string, error) {
	if g.reg == nil {
		return true, "", "", errors.New("gatekeeper: no registry")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return false, "", "empty text", nil
	}

	var (
		provider summarizer.Provider
		err      error
	)
	if g.providerID != "" {
		provider, err = g.reg.Build(ctx, g.providerID)
	} else {
		provider, err = g.reg.Default(ctx)
	}
	if err != nil {
		return true, "", "", fmt.Errorf("gatekeeper: build provider: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, g.maxLatency)
	defer cancel()

	msgs := []summarizer.Message{
		{
			Role:      summarizer.RoleUser,
			Text:      text,
			Timestamp: time.Now().UTC(),
		},
	}
	res, err := provider.Summarize(callCtx, msgs)
	if err != nil {
		if errors.Is(err, summarizer.ErrEmptyConversation) {
			return false, "", "summarizer found no extractable content", nil
		}
		return true, "", "", fmt.Errorf("gatekeeper: summarize: %w", err)
	}
	if len(res.Facts) == 0 {
		return false, "", "model did not extract any durable fact", nil
	}

	// Pick the highest-confidence fact and lift its category.
	top := res.Facts[0]
	for _, f := range res.Facts[1:] {
		if f.Confidence > top.Confidence {
			top = f
		}
	}
	mapped := mapSummarizerCategory(top.Category)
	g.log.Debug("gatekeeper.judge",
		"text_len", len(text),
		"facts", len(res.Facts),
		"top_confidence", top.Confidence,
		"summarizer_category", top.Category,
		"memory_category", mapped,
	)
	return true, mapped, "", nil
}

// mapSummarizerCategory translates the summarizer's category enum
// (preference / identifier / decision / task / other) into the
// memory subsystem's category enum surfaced in the MCP
// memoryGuidanceText (user_preference / project_fact / feedback /
// reference). Kept here so adding a new summarizer category later
// is a one-line change.
func mapSummarizerCategory(cat summarizer.Category) string {
	switch cat {
	case summarizer.CategoryPreference:
		return "user_preference"
	case summarizer.CategoryIdentifier, summarizer.CategoryDecision, summarizer.CategoryOther:
		return "project_fact"
	case summarizer.CategoryTask:
		return "reference"
	default:
		return "project_fact"
	}
}
