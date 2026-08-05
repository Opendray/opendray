package canvas

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// TopicUpdated is broadcast whenever an artifact is rendered; the Canvas panel
// (filtered by cwd) refetches the artifact by id.
const TopicUpdated = "canvas.updated"

// TopicFocus is broadcast when a project's focused canvas changes, so other
// open panels for the same project follow along.
const TopicFocus = "canvas.focus_changed"

// PromptInjector seeds a prompt into a live agent session.
type PromptInjector interface {
	Seed(ctx context.Context, sessionID, prompt string) error
}

// Service is the canvas business layer: render mocks (persist + broadcast) and
// route the operator's on-mock annotations back into the session.
type Service struct {
	store  *Store
	bus    *eventbus.Hub
	inject PromptInjector
	log    *slog.Logger

	// focus holds the canvas the operator is currently working on, per project
	// (cwd) — the MCP tools only know the project scope, not a session id. It is
	// the answer to "which canvas are we talking about" for plain conversation.
	// In-memory: the panel re-asserts it on mount/selection, so a gateway restart
	// costs nothing.
	focusMu sync.RWMutex
	focus   map[string]string // cwd -> slug
}

// NewService wires the service. store required; bus/inject optional.
func NewService(store *Store, bus *eventbus.Hub, inject PromptInjector, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		store:  store,
		bus:    bus,
		inject: inject,
		log:    log.With("component", "canvas"),
		focus:  map[string]string{},
	}
}

// GetFocus returns the slug of the project's focused canvas ("" if none).
func (s *Service) GetFocus(cwd string) string {
	s.focusMu.RLock()
	defer s.focusMu.RUnlock()
	return s.focus[strings.TrimSpace(cwd)]
}

// SetFocus records which canvas the operator is working on. When notify is set
// (an explicit switch, not a panel mount) and sessionID is given, a short focus
// line is seeded into that session so plain conversation — "make the title
// bigger" — resolves to this canvas without the operator repeating its name.
// Re-selecting the same canvas never re-notifies.
func (s *Service) SetFocus(ctx context.Context, cwd, slug, sessionID string, notify bool) (Artifact, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return Artifact{}, errors.New("canvas: cwd is required")
	}
	slug = strings.TrimSpace(slug)

	s.focusMu.Lock()
	prev := s.focus[cwd]
	if slug == "" {
		delete(s.focus, cwd)
	} else {
		s.focus[cwd] = slug
	}
	s.focusMu.Unlock()

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: TopicFocus,
			Data:  map[string]any{"cwd": cwd, "slug": slug},
		})
	}
	if slug == "" {
		return Artifact{}, nil
	}

	a, err := s.store.GetBySlug(ctx, cwd, slug)
	if err != nil {
		return Artifact{}, err
	}
	if notify && slug != prev && s.inject != nil && strings.TrimSpace(sessionID) != "" {
		if err := s.inject.Seed(ctx, sessionID, formatFocusNote(a)); err != nil {
			// The focus itself is recorded; a failed nudge is not fatal.
			s.log.Warn("canvas: focus note not seeded", "error", err, "session_id", sessionID)
		}
	}
	return a, nil
}

// formatFocusNote is the one-line context nudge seeded on an explicit switch.
func formatFocusNote(a Artifact) string {
	title := a.Title
	if title == "" {
		title = a.Slug
	}
	return fmt.Sprintf(
		"[Canvas focus] The operator switched the Canvas panel to the %s canvas %q (slug=%q). From now on, when they say \"this canvas\" / \"this design\" / \"this diagram\" — or ask for a change without naming one — they mean THIS canvas: update it in place with `canvas_render` slug=%q. No reply needed; just keep it in mind.",
		kindLabel(a.Kind), title, a.Slug, a.Slug)
}

// kindLabel renders a canvas kind for prompts.
func kindLabel(kind string) string {
	switch kind {
	case KindFlow:
		return "flowchart"
	case KindMindmap:
		return "mind-map"
	case KindGraph:
		return "relationship-diagram"
	case KindDoc:
		return "document"
	default:
		return "UI-mock"
	}
}

// ResolveSlug picks the slug a render should target: the explicit one, else the
// project's focused canvas, else the default. This is why an agent that just
// calls canvas_render (no slug) updates what the operator is looking at.
func (s *Service) ResolveSlug(cwd, slug string) string {
	if slug = strings.TrimSpace(slug); slug != "" {
		return slug
	}
	if f := s.GetFocus(cwd); f != "" {
		return f
	}
	return DefaultSlug
}

// Render upserts the artifact and broadcasts canvas.updated. An empty slug
// targets the project's focused canvas (see ResolveSlug).
func (s *Service) Render(ctx context.Context, cwd, slug, title, kind, html string) (Artifact, error) {
	a, err := s.store.Upsert(ctx, cwd, s.ResolveSlug(cwd, slug), title, kind, html)
	if err != nil {
		return Artifact{}, err
	}
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: TopicUpdated,
			Data: map[string]any{
				"id":      a.ID,
				"cwd":     a.Cwd,
				"slug":    a.Slug,
				"title":   a.Title,
				"kind":    a.Kind,
				"version": a.Version,
			},
		})
	}
	return a, nil
}

// RequestDesign seeds an operator design request into the session as a prompt
// that explicitly names canvas_render — opendray's deterministic entry point to
// the Canvas (doesn't rely on the agent guessing, doesn't override its native
// abilities). When targetSlug is set (the operator has a specific mock selected
// in the panel), the prompt tells the agent to UPDATE that canvas in place — so
// it doesn't spawn a new mock when the intent is to iterate on an existing one.
func (s *Service) RequestDesign(ctx context.Context, sessionID, prompt, targetSlug, targetTitle, kind string) error {
	if s.inject == nil {
		return fmt.Errorf("canvas: request injection not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("canvas: session_id is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("canvas: empty request")
	}
	return s.inject.Seed(ctx, sessionID, formatDesignRequest(prompt, targetSlug, targetTitle, kind))
}

func formatDesignRequest(prompt, targetSlug, targetTitle, kind string) string {
	var b strings.Builder
	b.WriteString("[Canvas request from the operator] ")
	if slug := strings.TrimSpace(targetSlug); slug != "" {
		fmt.Fprintf(&b, "Work on the EXISTING canvas %s— update it in place by calling `canvas_render` with slug=%q (the SAME slug); do NOT create a new canvas unless the request below explicitly asks for a NEW/separate one. ", titleHint(targetTitle), slug)
	} else if k := NormKind(kind); k != "" && k != KindUI {
		fmt.Fprintf(&b, "Draw this as a NEW %s on the opendray Canvas: call the `canvas_render` MCP tool with kind=%q, a fresh slug, and a COMPLETE, self-contained HTML document. Author the diagram as inline SVG (or plain HTML/CSS) — no external scripts or fonts — so it renders reliably, stays crisp at any size, and the operator can pin/annotate its parts. ", kindLabel(k), k)
	} else {
		b.WriteString("Render this to the opendray Canvas by calling the `canvas_render` MCP tool with a COMPLETE, self-contained HTML document. ")
	}
	b.WriteString("Use the current project context. The operator is viewing the Canvas panel and will pin/annotate the result (their notes come back to you). If it previews existing UI, base it on the real code.\n\nRequest: ")
	b.WriteString(strings.TrimSpace(prompt))
	return b.String()
}

// titleHint renders a "(titled "X") " fragment for prompts, or "" when unknown.
func titleHint(title string) string {
	if t := strings.TrimSpace(title); t != "" {
		return fmt.Sprintf("(titled %q) ", t)
	}
	return ""
}

// Annotation is one operator mark on a mock. Coordinates are percentages of the
// preview (0–100) so they survive a resize.
type Annotation struct {
	N        int      `json:"n"`
	Kind     string   `json:"kind"` // "pin" | "region"
	Note     string   `json:"note"`
	Selector string   `json:"selector,omitempty"`
	HTML     string   `json:"html,omitempty"`
	Elements []string `json:"elements,omitempty"` // components inside a mock region
	X        float64  `json:"x,omitempty"`
	Y        float64  `json:"y,omitempty"`
	W        float64  `json:"w,omitempty"`
	H        float64  `json:"h,omitempty"`
}

// Feedback is the operator's on-canvas response, routed to a specific session.
type Feedback struct {
	SessionID   string       `json:"session_id"`
	Message     string       `json:"message,omitempty"`
	Annotations []Annotation `json:"annotations"`
}

func (fb Feedback) isEmpty() bool {
	return strings.TrimSpace(fb.Message) == "" && len(fb.Annotations) == 0
}

// SubmitFeedback formats the operator's mock annotations into a prompt and seeds
// it into the target session — element-level, unambiguous change requests.
func (s *Service) SubmitFeedback(ctx context.Context, artifactID string, fb Feedback) error {
	if s.inject == nil {
		return fmt.Errorf("canvas: feedback injection not configured")
	}
	if strings.TrimSpace(fb.SessionID) == "" {
		return fmt.Errorf("canvas: session_id is required")
	}
	if fb.isEmpty() {
		return fmt.Errorf("canvas: empty feedback")
	}
	a, err := s.store.Get(ctx, artifactID)
	if err != nil {
		return err
	}
	return s.inject.Seed(ctx, fb.SessionID, formatFeedback(a, fb))
}

func formatFeedback(a Artifact, fb Feedback) string {
	title := a.Title
	if title == "" {
		title = a.Slug
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Canvas feedback on \"%s\"] The operator reviewed the preview you rendered and wants these changes. Apply each point, then UPDATE THE SAME canvas in place by calling canvas_render with slug=%q (do NOT create a new canvas) so they can review again.\n\n", title, a.Slug)
	b.WriteString(annotationsBlock(fb))
	return b.String()
}

func annotationsBlock(fb Feedback) string {
	var b strings.Builder
	if len(fb.Annotations) > 0 {
		b.WriteString("Annotations:\n")
		for _, an := range fb.Annotations {
			kind := an.Kind
			if kind == "" {
				kind = "note"
			}
			fmt.Fprintf(&b, "%d. (%s) ", an.N, kind)
			switch {
			case an.Kind == "region" && an.Selector != "":
				// Mock region: the framed DOM block, with coords as a hint.
				fmt.Fprintf(&b, "region framing element `%s` (~x:%.0f%% y:%.0f%% w:%.0f%% h:%.0f%%) ", an.Selector, an.X, an.Y, an.W, an.H)
			case an.Selector != "":
				fmt.Fprintf(&b, "element `%s` ", an.Selector)
			case an.Kind == "region":
				fmt.Fprintf(&b, "region at ~x:%.0f%% y:%.0f%% w:%.0f%% h:%.0f%% ", an.X, an.Y, an.W, an.H)
			default:
				fmt.Fprintf(&b, "point at ~x:%.0f%% y:%.0f%% ", an.X, an.Y)
			}
			if note := strings.TrimSpace(an.Note); note != "" {
				fmt.Fprintf(&b, "— %s", note)
			}
			b.WriteString("\n")
			if len(an.Elements) > 0 {
				fmt.Fprintf(&b, "   contains: %s\n", oneLine(strings.Join(an.Elements, " · ")))
			}
			if snip := strings.TrimSpace(an.HTML); snip != "" {
				fmt.Fprintf(&b, "   markup: %s\n", oneLine(snip))
			}
		}
	}
	if msg := strings.TrimSpace(fb.Message); msg != "" {
		fmt.Fprintf(&b, "\nOverall: %s\n", msg)
	}
	return b.String()
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 400
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
