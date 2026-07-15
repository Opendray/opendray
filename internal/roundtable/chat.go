package roundtable

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/memory/worker"
)

// ContextSource supplies relevant prior context (memories / journal / docs)
// for a reply prompt, keyed off the table's cwd. Backed by memquery in the
// app; nil disables enrichment. Mirrors cortex.ContextSource.
type ContextSource interface {
	RelevantContext(ctx context.Context, cwd, query string, topK int) (string, error)
}

// tuning — chat replies are short and conversational; summaries are longer.
const (
	replyMaxTokens   = 2048
	summaryMaxTokens = 4096
	callTimeout      = 5 * time.Minute
	roundTimeout     = 20 * time.Minute
	// maxAutoRounds caps how many EXTRA rounds the members may trigger among
	// themselves (round 0 is the operator's @mentions). Prevents an infinite
	// @-ping-pong and bounds cost: with N seats the worst case is
	// (maxAutoRounds+1) × N headless calls per operator message.
	maxAutoRounds = 2
)

// Service drives the group chat: it appends operator messages, invokes the
// @mentioned members, and appends their replies. It holds no chair / verdict
// logic — the discussion is open-ended.
type Service struct {
	store    *Store
	registry *worker.Registry
	bus      *eventbus.Hub
	context  ContextSource   // optional
	sessions SessionLauncher // optional — Handoff 503s without it
	log      *slog.Logger
}

// NewService wires the service. store + registry required.
func NewService(store *Store, reg *worker.Registry, bus *eventbus.Hub, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		store:    store,
		registry: reg,
		bus:      bus,
		log:      log.With("component", "roundtable.chat"),
	}
}

// WithContextSource enables reply-prompt enrichment with relevant memories.
func (s *Service) WithContextSource(c ContextSource) *Service {
	s.context = c
	return s
}

// PostMessage stores the operator's message and, for every member it
// @mentions, kicks off a reply in the background. Returns the stored
// operator message immediately; replies land asynchronously and are
// announced on the eventbus topic "roundtable.updated" for the UI to poll.
func (s *Service) PostMessage(ctx context.Context, id, content string) (Message, error) {
	if strings.TrimSpace(content) == "" {
		return Message{}, fmt.Errorf("roundtable: empty message")
	}
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return Message{}, err
	}
	if rt.Status == StatusClosed {
		return Message{}, fmt.Errorf("roundtable: chat is closed")
	}
	mentions := parseMentions(content, rt.Seats)
	msg, err := s.store.AppendMessage(ctx, Message{
		RoundTableID: id, Role: RoleOperator, Content: content, Mentions: mentions,
	})
	if err != nil {
		return Message{}, err
	}
	// Auto-name the chat from its first message so the operator never has to
	// title it upfront.
	if strings.TrimSpace(rt.Topic) == "" {
		if title := deriveTitle(content); title != "" {
			_ = s.store.SetTopic(ctx, id, title)
		}
	}
	s.announce(id)
	if len(mentions) > 0 {
		// Detached: agent-mode replies take minutes and must not hold the
		// HTTP request. The mentioned members reply in seat order; each
		// re-reads the thread so later members see earlier replies.
		go s.runReplies(id, mentions)
	}
	return msg, nil
}

// runReplies drives the discussion in rounds. Round 0 is the members the
// operator @mentioned; each member's reply is appended before the next speaks,
// so members react to each other within a round. When a reply @mentions OTHER
// seated members, they form the next round — an autonomous debate — capped at
// maxAutoRounds to bound cost and prevent an infinite @-ping-pong. A member
// that errors gets a system note; the round continues.
func (s *Service) runReplies(id string, providers []string) {
	ctx, cancel := context.WithTimeout(context.Background(), roundTimeout)
	defer cancel()

	rt, err := s.store.Get(ctx, id)
	if err != nil {
		s.log.Error("roundtable replies: load failed", "id", id, "err", err)
		return
	}
	seatByProvider := map[string]Seat{}
	for _, seat := range rt.Seats {
		seatByProvider[seat.Provider] = seat
	}

	current := providers
	for round := 0; len(current) > 0; round++ {
		var next []string
		for _, provider := range current {
			seat, ok := seatByProvider[provider]
			if !ok {
				continue
			}
			system := chatSystemPrompt(rt, seat)
			// Re-read the thread each turn so a member sees earlier replies
			// (within this round and prior rounds).
			rt, err = s.store.Get(ctx, id)
			if err != nil {
				s.log.Error("roundtable replies: reload failed", "id", id, "err", err)
				return
			}
			user, err := s.buildTranscript(ctx, rt, seat.Provider)
			if err != nil {
				s.memberFailed(ctx, id, seat, err)
				continue
			}
			resp, err := s.invokeSeat(ctx, seat, system, user, replyMaxTokens)
			if err != nil {
				s.memberFailed(ctx, id, seat, err)
				continue
			}
			reply := strings.TrimSpace(resp)
			if reply == "" {
				s.memberFailed(ctx, id, seat, fmt.Errorf("empty reply"))
				continue
			}
			// A member may @mention others to pull them into the debate; record
			// those on the message (drives the UI) and queue them for the next
			// round.
			mentions := parseMentions(reply, rt.Seats)
			if _, err := s.store.AppendMessage(ctx, Message{
				RoundTableID: id, Role: RoleSeat, SeatProvider: seat.Provider,
				SeatModel: seat.Model, Content: reply, Mentions: mentions,
			}); err != nil {
				s.log.Warn("roundtable: append reply failed", "id", id, "provider", provider, "err", err)
				continue
			}
			s.announce(id)
			next = nextRoundMentions(next, mentions, seat.Provider)
		}
		if len(next) > 0 && round >= maxAutoRounds {
			// Cap reached but members still want to talk. Post a paused note and
			// stash the pending speakers in its Mentions so Continue can resume
			// exactly where the debate left off (no @mention needed).
			_, _ = s.store.AppendMessage(ctx, Message{
				RoundTableID: id, Role: RoleSystem,
				Content:  fmt.Sprintf("Auto-discussion paused at its %d-round limit. Continue to let them keep going.", maxAutoRounds),
				Mentions: next,
			})
			s.announce(id)
			return
		}
		current = next
	}
}

// Continue resumes a paused auto-discussion for another maxAutoRounds burst.
// It seeds from the pending speakers stashed on the last paused system note;
// if none are pending (the debate ended cleanly), it re-engages every seated
// member for a fresh round. Runs in the background like PostMessage.
func (s *Service) Continue(ctx context.Context, id string) error {
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if rt.Status == StatusClosed {
		return fmt.Errorf("roundtable: chat is closed")
	}
	if len(rt.Seats) == 0 {
		return fmt.Errorf("roundtable: no members to continue")
	}
	msgs, err := s.store.Messages(ctx, id, 200)
	if err != nil {
		return fmt.Errorf("roundtable: load messages: %w", err)
	}
	seeds := pendingSpeakers(msgs, rt.Seats)
	if len(seeds) == 0 {
		// Nothing pending — give everyone another turn.
		seeds = make([]string, len(rt.Seats))
		for i, seat := range rt.Seats {
			seeds[i] = seat.Provider
		}
	}
	go s.runReplies(id, seeds)
	return nil
}

// pendingSpeakers returns the members a paused auto-discussion still owes a
// turn: the Mentions stashed on the most recent paused system note, filtered to
// members still seated. Empty when the last note isn't a paused one.
func pendingSpeakers(msgs []Message, seats []Seat) []string {
	seated := make(map[string]bool, len(seats))
	for _, s := range seats {
		seated[s.Provider] = true
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role != RoleSystem {
			// A real turn happened after any paused note → nothing pending.
			return nil
		}
		if len(m.Mentions) > 0 {
			var out []string
			for _, p := range m.Mentions {
				if seated[p] {
					out = append(out, p)
				}
			}
			return out
		}
	}
	return nil
}

// nextRoundMentions accumulates the seated members a reply addressed into the
// next round's queue, excluding the speaker itself and any already queued (so
// each member speaks at most once per round). Order is preserved for
// deterministic sequencing.
func nextRoundMentions(queue, mentions []string, self string) []string {
	seen := make(map[string]bool, len(queue))
	for _, q := range queue {
		seen[q] = true
	}
	for _, m := range mentions {
		if m == self || seen[m] {
			continue
		}
		seen[m] = true
		queue = append(queue, m)
	}
	return queue
}

// Summarize asks one member to condense the discussion so far into a plan,
// posted as a summary message. Runs in the background; the summary lands
// asynchronously. If provider is empty, the first seat is used.
func (s *Service) Summarize(ctx context.Context, id, provider string) error {
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if len(rt.Seats) == 0 {
		return fmt.Errorf("roundtable: no members to summarize")
	}
	provider = strings.TrimSpace(provider)
	var chosen Seat
	found := false
	for _, seat := range rt.Seats {
		if provider == "" || seat.Provider == provider {
			chosen = seat
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("roundtable: %q is not a member of this chat", provider)
	}
	go s.runSummary(id, chosen)
	return nil
}

func (s *Service) runSummary(id string, seat Seat) {
	ctx, cancel := context.WithTimeout(context.Background(), roundTimeout)
	defer cancel()
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		s.log.Error("roundtable summary: load failed", "id", id, "err", err)
		return
	}
	user, err := s.buildTranscript(ctx, rt, seat.Provider)
	if err != nil {
		s.memberFailed(ctx, id, seat, err)
		return
	}
	resp, err := s.invokeSeat(ctx, seat, summarySystemPrompt(rt), user, summaryMaxTokens)
	if err != nil {
		s.memberFailed(ctx, id, seat, err)
		return
	}
	if _, err := s.store.AppendMessage(ctx, Message{
		RoundTableID: id, Role: RoleSeat, SeatProvider: seat.Provider,
		SeatModel: seat.Model, Kind: KindSummary, Content: strings.TrimSpace(resp),
	}); err != nil {
		s.log.Warn("roundtable: append summary failed", "id", id, "err", err)
		return
	}
	s.announce(id)
}

// invokeSeat dispatches one headless agent call for a member via the worker
// registry's per-call override path. TaskCuration is reused purely as the
// metrics label (round table needs no memory_workers row of its own — see
// ROLLBACK.md).
func (s *Service) invokeSeat(ctx context.Context, seat Seat, system, user string, maxTokens int) (string, error) {
	resp, err := s.registry.RunWith(ctx, worker.Config{
		Task:       worker.TaskCuration,
		Kind:       worker.WorkerAgent,
		ProviderID: seat.Provider,
		Model:      seat.Model,
		AccountID:  seat.AccountID,
		Enabled:    true,
	}, worker.Request{
		Task:         worker.TaskCuration,
		SystemPrompt: system,
		UserInput:    user,
		MaxTokens:    maxTokens,
		Timeout:      callTimeout,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (s *Service) memberFailed(ctx context.Context, id string, seat Seat, err error) {
	s.log.Warn("roundtable member failed", "id", id, "provider", seat.Provider, "err", err)
	_, _ = s.store.AppendMessage(ctx, Message{
		RoundTableID: id, Role: RoleSystem, SeatProvider: seat.Provider,
		Content: fmt.Sprintf("%s could not reply: %v", seat.Provider, err),
	})
	s.announce(id)
}

func (s *Service) announce(id string) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(eventbus.Event{
		Topic: "roundtable.updated",
		Data:  map[string]any{"round_table_id": id},
	})
}

// buildTranscript renders the chat history as the user block for a member's
// next turn, plus optional relevant memory context. selfProvider is the
// member about to speak (its own past lines are marked so it doesn't
// impersonate others).
func (s *Service) buildTranscript(ctx context.Context, rt RoundTable, selfProvider string) (string, error) {
	msgs, err := s.store.Messages(ctx, rt.ID, 200)
	if err != nil {
		return "", fmt.Errorf("load messages: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Group chat: %s\n\n", strings.TrimSpace(rt.Topic))
	b.WriteString("Conversation so far:\n\n")
	for _, m := range msgs {
		b.WriteString(speakerLabel(m, selfProvider))
		b.WriteString(": ")
		b.WriteString(strings.TrimSpace(m.Content))
		b.WriteString("\n\n")
	}

	if s.context != nil && strings.TrimSpace(rt.Cwd) != "" {
		if extra, cerr := s.context.RelevantContext(ctx, rt.Cwd, rt.Topic, 6); cerr == nil && strings.TrimSpace(extra) != "" {
			b.WriteString("## Possibly relevant prior context (memories / journal)\n\n")
			b.WriteString(truncate(extra, 4000))
			b.WriteString("\n\n")
		}
	}
	return b.String(), nil
}

// deriveTitle makes a short chat title from its first message: the first
// non-empty line, with @mentions stripped, truncated to ~80 runes.
func deriveTitle(content string) string {
	line := content
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	// Drop leading @mention tokens so the title reads as the actual ask.
	fields := strings.Fields(line)
	kept := fields[:0]
	for _, f := range fields {
		if strings.HasPrefix(f, "@") {
			continue
		}
		kept = append(kept, f)
	}
	line = strings.TrimSpace(strings.Join(kept, " "))
	if line == "" {
		line = strings.TrimSpace(content)
	}
	r := []rune(line)
	if len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return line
}

func speakerLabel(m Message, selfProvider string) string {
	switch m.Role {
	case RoleOperator:
		return "Operator"
	case RoleSystem:
		return "System"
	default:
		if m.SeatProvider == selfProvider {
			return m.SeatProvider + " (you)"
		}
		return m.SeatProvider
	}
}
