package cortex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/memory/worker"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// ContextSource supplies relevant prior context (memories / journal /
// docs) for the curation prompt. Backed by memquery in the app; nil
// disables context enrichment.
type ContextSource interface {
	RelevantContext(ctx context.Context, cwd, query string, topK int) (string, error)
}

// LaunchSpec describes the session to spawn when escalating a
// conversation. ProviderID/Model/ClaudeAccountID carry the
// conversation's cloud-agent override so the escalated session continues
// on the SAME CLI + account the discussion ran on. An empty ProviderID
// (no override, or a local-summarizer override that has no CLI to
// continue on) lets the launcher pick its default.
type LaunchSpec struct {
	Cwd             string
	Name            string
	SeedPrompt      string
	ProviderID      string
	Model           string
	ClaudeAccountID string
}

// SessionLauncher escalates a conversation into a full agent session.
// Implemented in the app over session.Manager — cortex never imports
// internal/session.
type SessionLauncher interface {
	Launch(ctx context.Context, spec LaunchSpec) (sessionID string, err error)
}

// CurationService runs the conversational maintenance channel.
type CurationService struct {
	store    *ConversationStore
	docs     *projectdoc.Service
	registry *worker.Registry
	bus      *eventbus.Hub
	context  ContextSource   // optional
	sessions SessionLauncher // optional — escalation 503s without it
	log      *slog.Logger

	// workspaceCwd is the cwd used when escalating a GLOBAL knowledge
	// conversation to a real session (there is no project cwd to use;
	// the opendray workspace is where policy evidence lives).
	workspaceCwd string
}

// NewCurationService wires the service. store/docs/registry required.
func NewCurationService(store *ConversationStore, docs *projectdoc.Service, reg *worker.Registry, bus *eventbus.Hub, log *slog.Logger) *CurationService {
	if log == nil {
		log = slog.Default()
	}
	return &CurationService{
		store:    store,
		docs:     docs,
		registry: reg,
		bus:      bus,
		log:      log.With("component", "cortex.curation"),
	}
}

// WithContextSource enables prompt enrichment with relevant memories.
func (s *CurationService) WithContextSource(c ContextSource) *CurationService {
	s.context = c
	return s
}

// WithSessionLauncher enables escalation to full agent sessions.
func (s *CurationService) WithSessionLauncher(l SessionLauncher, workspaceCwd string) *CurationService {
	s.sessions = l
	s.workspaceCwd = workspaceCwd
	return s
}

// curationReply is the strict response shape the worker returns.
type curationReply struct {
	Reply    string `json:"reply"`
	Revision struct {
		Action  string `json:"action"` // "none" | "update"
		Content string `json:"content"`
		Reason  string `json:"reason"`
	} `json:"revision"`
}

const curationJSONSchema = `{
  "name": "curation_reply",
  "schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "reply": {"type": "string"},
      "revision": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "action":  {"type": "string", "enum": ["none", "update"]},
          "content": {"type": "string"},
          "reason":  {"type": "string"}
        },
        "required": ["action", "content", "reason"]
      }
    },
    "required": ["reply", "revision"]
  },
  "strict": true
}`

// Send appends the operator's message and kicks off the AI turn in the
// background. The handler returns immediately with the stored operator
// message; the AI reply lands as a second message and is announced on
// the eventbus topic "cortex.conversation.reply" for live UI refresh.
func (s *CurationService) Send(ctx context.Context, conversationID, text string) (Message, error) {
	if strings.TrimSpace(text) == "" {
		return Message{}, errors.New("cortex: empty message")
	}
	conv, err := s.store.Get(ctx, conversationID)
	if err != nil {
		return Message{}, err
	}
	if conv.Status == ConvClosed {
		return Message{}, errors.New("cortex: conversation is closed")
	}
	msg, err := s.store.AppendMessage(ctx, Message{
		ConversationID: conversationID,
		Role:           "operator",
		Content:        text,
	})
	if err != nil {
		return Message{}, err
	}
	// The LLM turn runs detached: it can take minutes (agent-mode
	// workers) and must not hold the HTTP request open. Pass the id (not
	// the snapshot) so the turn re-reads the latest provider/model pin.
	go s.runAITurn(conv.ID)
	return msg, nil
}

// runAITurn assembles the prompt, dispatches the worker, applies or
// proposes the revision per the target's lock state, and appends the
// AI message. Every failure path appends a system message so the
// operator is never staring at a silently dead conversation.
func (s *CurationService) runAITurn(conversationID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	fail := func(msg string, err error) {
		s.log.Warn("curation turn failed", "conversation_id", conversationID, "stage", msg, "err", err)
		_, _ = s.store.AppendMessage(ctx, Message{
			ConversationID: conversationID,
			Role:           "system",
			Content:        fmt.Sprintf("AI turn failed (%s): %v", msg, err),
		})
		s.announce(conversationID)
	}

	// Re-fetch fresh so a provider/model override the operator pinned just
	// before sending is always honoured (Send's snapshot may pre-date it).
	conv, err := s.store.Get(ctx, conversationID)
	if err != nil {
		fail("load", err)
		return
	}

	system, user, err := s.assemblePrompt(ctx, conv)
	if err != nil {
		fail("assemble", err)
		return
	}
	req := worker.Request{
		Task:                     worker.TaskCuration,
		SystemPrompt:             system,
		UserInput:                user,
		MaxTokens:                8192,
		Timeout:                  5 * time.Minute,
		ResponseFormatJSONSchema: curationJSONSchema,
	}
	// Per-conversation override: run THIS turn on the model the operator
	// pinned in the chat instead of the global `curation` worker config.
	// summarizer_id → a local/HTTP model; provider_id → a cloud-agent CLI;
	// neither → global default.
	var resp worker.Response
	switch {
	case conv.SummarizerID != "":
		resp, err = s.registry.RunWith(ctx, worker.Config{
			Task:         worker.TaskCuration,
			Kind:         worker.WorkerSummarizer,
			SummarizerID: conv.SummarizerID,
			Model:        conv.Model, // optional per-call model override
			Enabled:      true,
		}, req)
	case conv.ProviderID != "":
		resp, err = s.registry.RunWith(ctx, worker.Config{
			Task:       worker.TaskCuration,
			Kind:       worker.WorkerAgent,
			ProviderID: conv.ProviderID,
			Model:      conv.Model,
			AccountID:  conv.ClaudeAccountID, // multi-account claude: pin the chosen account
			Enabled:    true,
		}, req)
	default:
		resp, err = s.registry.Run(ctx, req)
	}
	if err != nil {
		fail("llm", err)
		return
	}
	reply, err := parseCurationReply(resp.Content)
	if err != nil {
		fail("parse", err)
		return
	}

	action, ref := "", ""
	if reply.Revision.Action == "update" && strings.TrimSpace(reply.Revision.Content) != "" {
		action, ref, err = s.applyRevision(ctx, conv, reply.Revision.Content, reply.Revision.Reason)
		if err != nil {
			fail("revision", err)
			return
		}
	}

	if _, err := s.store.AppendMessage(ctx, Message{
		ConversationID: conv.ID,
		Role:           "ai",
		Content:        reply.Reply,
		RevisionAction: action,
		RevisionRef:    ref,
	}); err != nil {
		s.log.Warn("curation: append ai message failed", "conversation_id", conv.ID, "err", err)
		return
	}
	s.announce(conv.ID)
}

func (s *CurationService) announce(conversationID string) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(eventbus.Event{
		Topic: "cortex.conversation.reply",
		Data:  map[string]any{"conversation_id": conversationID},
	})
}

// assemblePrompt builds the system + user blocks: the target's role
// and current content, section metadata where applicable, relevant
// prior context, and the conversation so far.
func (s *CurationService) assemblePrompt(ctx context.Context, conv Conversation) (string, string, error) {
	var system strings.Builder
	system.WriteString(`You are the curator of a developer's living documentation system. The operator is talking to you to keep a document accurate, restructure it, or re-draft standing policy. Be direct and concrete; this is a working session, not a chat.

When the operator asks for a change (explicitly or implicitly), produce the FULL replacement markdown for the target document in revision.content with action "update", and answer conversationally in reply (do not paste the whole document into reply). When no change is warranted, use action "none" and answer in reply.

NEVER include secrets (passwords, API keys, tokens) in any output. Preserve parts of the document the conversation did not touch verbatim.`)

	switch conv.TargetKind {
	case TargetKBPage:
		system.WriteString("\n\nThe target is a CROSS-PROJECT knowledge page")
		foundational := false
		if sections, err := s.docs.ListSections(ctx, projectdoc.GlobalCwd); err == nil {
			for _, sec := range sections {
				if sec.Slug == conv.TargetSlug {
					foundational = sec.Nature == "foundational"
					break
				}
			}
		}
		if foundational {
			system.WriteString(" of the FOUNDATIONAL nature: standing ground truth + binding rules injected into every project. This conversation is how the operator re-drafts policy with you (重新制定方针). Keep the \"## Rules (MUST follow)\" section explicit and imperative.")
		} else {
			system.WriteString(" of the EMERGENT nature: distilled lessons / reusable assets. Guidance, not law.")
		}
	case TargetDocSection:
		system.WriteString("\n\nThe target is one section of a project's official document.")
	case TargetBlueprint:
		system.WriteString("\n\nThe target is the project's doc BLUEPRINT (its section set). You cannot apply blueprint changes directly — discuss and recommend; the operator applies via the blueprint editor. Always use action \"none\".")
	}

	var user strings.Builder
	fmt.Fprintf(&user, "## Target\n\nkind: %s · cwd: `%s` · slug: `%s`\n\n", conv.TargetKind, conv.TargetCwd, conv.TargetSlug)

	// Section metadata steers the curator the same way it steers drift.
	if conv.TargetKind == TargetDocSection {
		if sections, err := s.docs.ListSections(ctx, conv.TargetCwd); err == nil {
			for _, sec := range sections {
				if sec.Slug != conv.TargetSlug {
					continue
				}
				fmt.Fprintf(&user, "Section: %q — %s\n", sec.Title, sec.Description)
				if sec.PromptHint != "" {
					fmt.Fprintf(&user, "Maintainer hint: %s\n", sec.PromptHint)
				}
				fmt.Fprintf(&user, "Maintainer mode: %s\n\n", sec.MaintainerMode)
				break
			}
		}
	}

	// Current content of the target document.
	if conv.TargetKind != TargetBlueprint {
		doc, err := s.docs.GetDoc(ctx, conv.TargetCwd, projectdoc.Kind(conv.TargetSlug))
		switch {
		case err == nil && strings.TrimSpace(doc.Content) != "":
			user.WriteString("## Current content\n\n")
			user.WriteString(truncate(doc.Content, 24000))
			user.WriteString("\n\n")
		case errors.Is(err, projectdoc.ErrNotFound) || (err == nil && strings.TrimSpace(doc.Content) == ""):
			user.WriteString("## Current content\n\n_(empty — you may draft it from scratch when asked)_\n\n")
		case err != nil:
			return "", "", fmt.Errorf("get target doc: %w", err)
		}
	} else if sections, err := s.docs.ListSections(ctx, conv.TargetCwd); err == nil {
		user.WriteString("## Current blueprint\n\n")
		for _, sec := range sections {
			fmt.Fprintf(&user, "- %s (%q, mode=%s, pos=%d, inject=%v)\n",
				sec.Slug, sec.Title, sec.MaintainerMode, sec.Position, sec.Inject)
		}
		user.WriteString("\n")
	}

	// Conversation history (the new operator message is already stored).
	msgs, err := s.store.Messages(ctx, conv.ID, 40)
	if err != nil {
		return "", "", fmt.Errorf("list messages: %w", err)
	}
	user.WriteString("## Conversation\n\n")
	for _, m := range msgs {
		role := strings.ToUpper(m.Role)
		fmt.Fprintf(&user, "%s: %s\n\n", role, truncate(m.Content, 4000))
	}

	// Relevant prior context — keyed off the latest operator message.
	if s.context != nil && len(msgs) > 0 {
		query := msgs[len(msgs)-1].Content
		cwd := conv.TargetCwd
		if cwd == projectdoc.GlobalCwd && s.workspaceCwd != "" {
			cwd = s.workspaceCwd
		}
		if extra, cerr := s.context.RelevantContext(ctx, cwd, query, 8); cerr == nil && strings.TrimSpace(extra) != "" {
			user.WriteString("## Possibly relevant prior context (memories / journal)\n\n")
			user.WriteString(truncate(extra, 6000))
			user.WriteString("\n")
		}
	}

	return system.String(), user.String(), nil
}

// applyRevision routes the AI's revision through the SAME lock
// semantics every other writer uses: an ai-maintained, unlocked target
// is updated in place (the conversation IS the operator's intent); a
// human-locked target or human-mode section gets a proposal — the
// conversation can never silently overwrite something a human claimed.
func (s *CurationService) applyRevision(ctx context.Context, conv Conversation, content, reason string) (action, ref string, err error) {
	if conv.TargetKind == TargetBlueprint {
		return "", "", nil // discussed, never applied from here
	}
	kind := projectdoc.Kind(conv.TargetSlug)

	locked := false
	if doc, derr := s.docs.GetDoc(ctx, conv.TargetCwd, kind); derr == nil {
		locked = doc.UpdatedBy == projectdoc.AuthorOperator
	}
	humanMode := false
	if conv.TargetKind == TargetDocSection {
		if sections, serr := s.docs.ListSections(ctx, conv.TargetCwd); serr == nil {
			for _, sec := range sections {
				if sec.Slug == conv.TargetSlug {
					humanMode = sec.MaintainerMode == "human"
					break
				}
			}
		}
	}

	if reason == "" {
		reason = "Curation conversation " + conv.ID
	}
	if locked || humanMode {
		p, perr := s.docs.ProposeDoc(ctx, conv.TargetCwd, kind, content, reason, "")
		if perr != nil {
			return "", "", fmt.Errorf("propose: %w", perr)
		}
		return "proposed", p.ID, nil
	}
	d, uerr := s.docs.PutDoc(ctx, conv.TargetCwd, kind, content, projectdoc.AuthorAgent)
	if uerr != nil {
		return "", "", fmt.Errorf("apply: %w", uerr)
	}
	return "applied", d.ID, nil
}

// Escalate promotes the conversation to a full agent session seeded
// with the transcript + instructions to ground the revision in the
// codebase and file changes through the standard proposal tools.
func (s *CurationService) Escalate(ctx context.Context, conversationID string) (Conversation, error) {
	if s.sessions == nil {
		return Conversation{}, errors.New("cortex: session launcher not configured")
	}
	conv, err := s.store.Get(ctx, conversationID)
	if err != nil {
		return Conversation{}, err
	}
	if conv.Status == ConvEscalated && conv.EscalatedSessionID != "" {
		return conv, nil // idempotent
	}

	cwd := conv.TargetCwd
	if cwd == projectdoc.GlobalCwd {
		cwd = s.workspaceCwd
	}
	if strings.TrimSpace(cwd) == "" {
		return Conversation{}, errors.New("cortex: no cwd to escalate into (workspace cwd not configured)")
	}

	msgs, err := s.store.Messages(ctx, conv.ID, 40)
	if err != nil {
		return Conversation{}, err
	}
	var seed strings.Builder
	fmt.Fprintf(&seed, "You are taking over a documentation curation conversation about %s `%s` (project `%s`).\n\n",
		conv.TargetKind, conv.TargetSlug, conv.TargetCwd)
	seed.WriteString("Conversation so far:\n\n")
	for _, m := range msgs {
		fmt.Fprintf(&seed, "%s: %s\n\n", strings.ToUpper(m.Role), truncate(m.Content, 2000))
	}
	seed.WriteString("Ground the requested change in the actual codebase (read the relevant files), then update the document. ")
	seed.WriteString("Use the project_goal_set / project_plan_set MCP tools (or the project-docs HTTP API) so changes flow through the standard proposal/approval path — do not edit doc mirrors on disk.\n")

	name := "curation: " + conv.TargetSlug
	sessionID, err := s.sessions.Launch(ctx, LaunchSpec{
		Cwd:             cwd,
		Name:            name,
		SeedPrompt:      seed.String(),
		ProviderID:      conv.ProviderID,
		Model:           conv.Model,
		ClaudeAccountID: conv.ClaudeAccountID,
	})
	if err != nil {
		return Conversation{}, fmt.Errorf("cortex: launch session: %w", err)
	}
	if err := s.store.SetStatus(ctx, conv.ID, ConvEscalated, sessionID); err != nil {
		return Conversation{}, err
	}
	_, _ = s.store.AppendMessage(ctx, Message{
		ConversationID: conv.ID,
		Role:           "system",
		Content:        fmt.Sprintf("Escalated to agent session `%s` in `%s`.", sessionID, cwd),
	})
	s.announce(conv.ID)
	return s.store.Get(ctx, conv.ID)
}

// parseCurationReply tolerates fenced / preambled JSON.
func parseCurationReply(raw string) (curationReply, error) {
	body := strings.TrimSpace(raw)
	if i := strings.IndexByte(body, '{'); i >= 0 {
		if j := strings.LastIndexByte(body, '}'); j > i {
			body = body[i : j+1]
		}
	}
	var out curationReply
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return curationReply{}, fmt.Errorf("cortex: unparseable curation reply: %w", err)
	}
	if strings.TrimSpace(out.Reply) == "" {
		return curationReply{}, errors.New("cortex: curation reply has no reply text")
	}
	return out, nil
}
