package cortex

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// ─── rule suggestions ──────────────────────────────────────────
//
// A page's steering rules (Guidance / Excluded subjects) are operator-owned
// blueprint settings the AI drafter must never edit — but the curation
// discussion is exactly where a needed rule change surfaces. The AI files a
// structured suggestion on its reply; the operator applies it with one
// click and the backend merges it into the section. Approval stays human,
// the re-typing doesn't.

// ruleSuggestionFrom normalizes a reply's rules block into a stored
// suggestion, or nil when there is nothing actionable. Exclusions exist on
// global knowledge pages only, so for doc-section targets they are dropped
// at ingest — better no suggestion than one that can never apply. Blueprint
// targets carry no per-page rules at all.
func ruleSuggestionFrom(reply curationReply, targetKind string) *RuleSuggestion {
	if reply.Rules.Action != "suggest" || targetKind == TargetBlueprint {
		return nil
	}
	sug := RuleSuggestion{
		Guidance: strings.TrimSpace(reply.Rules.Guidance),
		Reason:   strings.TrimSpace(reply.Rules.Reason),
	}
	if targetKind == TargetKBPage {
		sug.ExclusionsAdd = cleanSubjects(reply.Rules.ExclusionsAdd)
		sug.ExclusionsRemove = cleanSubjects(reply.Rules.ExclusionsRemove)
	}
	if sug.Guidance == "" && len(sug.ExclusionsAdd) == 0 && len(sug.ExclusionsRemove) == 0 {
		return nil
	}
	return &sug
}

func cleanSubjects(in []string) []string {
	var out []string
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// mergeRuleSuggestion returns a copy of sec with the suggestion applied:
// a non-empty guidance replaces the prompt hint; removals match
// case-insensitively; additions dedupe against what survives. PutSection
// re-normalizes and caps the final list.
func mergeRuleSuggestion(sec projectdoc.Section, sug RuleSuggestion) projectdoc.Section {
	out := sec
	if g := strings.TrimSpace(sug.Guidance); g != "" {
		out.PromptHint = g
	}
	removed := make(map[string]struct{}, len(sug.ExclusionsRemove))
	for _, r := range sug.ExclusionsRemove {
		if r = strings.TrimSpace(r); r != "" {
			removed[strings.ToLower(r)] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(sec.Exclusions))
	next := make([]string, 0, len(sec.Exclusions)+len(sug.ExclusionsAdd))
	for _, e := range sec.Exclusions {
		k := strings.ToLower(strings.TrimSpace(e))
		if _, drop := removed[k]; drop {
			continue
		}
		next = append(next, e)
		seen[k] = struct{}{}
	}
	for _, a := range sug.ExclusionsAdd {
		if a = strings.TrimSpace(a); a == "" {
			continue
		}
		k := strings.ToLower(a)
		if _, dup := seen[k]; dup {
			continue
		}
		if _, drop := removed[k]; drop {
			continue // simultaneously added and removed → remove wins
		}
		next = append(next, a)
		seen[k] = struct{}{}
	}
	out.Exclusions = next
	return out
}

// describeRuleApply renders the system-message record of an applied
// suggestion, so later AI turns see the rules changed mid-thread.
func describeRuleApply(sug RuleSuggestion) string {
	parts := make([]string, 0, 3)
	if sug.Guidance != "" {
		parts = append(parts, "guidance replaced")
	}
	if len(sug.ExclusionsAdd) > 0 {
		parts = append(parts, "exclusions added: "+strings.Join(sug.ExclusionsAdd, ", "))
	}
	if len(sug.ExclusionsRemove) > 0 {
		parts = append(parts, "exclusions removed: "+strings.Join(sug.ExclusionsRemove, ", "))
	}
	return "Rule suggestion applied — " + strings.Join(parts, "; ") + "."
}

// ApplyRuleSuggestion merges a message's pending rule suggestion into the
// target's blueprint section. Only the operator reaches this (the HTTP
// handler), so the click IS the approval — no proposal detour. The mark
// happens after the merge succeeds; the UPDATE's pending-only guard makes
// a double-click a no-op error instead of a second merge.
func (s *CurationService) ApplyRuleSuggestion(ctx context.Context, conversationID, messageID string) (Message, error) {
	conv, err := s.store.Get(ctx, conversationID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.store.GetMessage(ctx, conversationID, messageID)
	if err != nil {
		return Message{}, err
	}
	if msg.RuleSuggestion == nil {
		return Message{}, errors.New("cortex: message carries no rule suggestion")
	}
	if msg.RuleAppliedAt != nil {
		return Message{}, errors.New("cortex: rule suggestion already applied")
	}

	secCwd := conv.TargetCwd
	switch conv.TargetKind {
	case TargetKBPage:
		secCwd = projectdoc.GlobalCwd
	case TargetDocSection:
		// exclusions were dropped at ingest; only guidance can be pending
	default:
		// Unreachable by construction — ruleSuggestionFrom never attaches a
		// suggestion to a blueprint conversation — kept so a future target
		// kind fails loudly instead of merging into a wrong section.
		return Message{}, fmt.Errorf("cortex: %s targets carry no page rules", conv.TargetKind)
	}
	// Read-modify-write with no version gate: a settings-dialog save landing
	// between GetSection and PutSection is lost (both surfaces are the one
	// operator, so the window is accepted — same shape as the dialog itself).
	sec, ok, err := s.docs.GetSection(ctx, secCwd, conv.TargetSlug)
	if err != nil {
		return Message{}, fmt.Errorf("cortex: load section: %w", err)
	}
	if !ok {
		return Message{}, fmt.Errorf("cortex: no blueprint section for %q", conv.TargetSlug)
	}
	if _, err := s.docs.PutSection(ctx, mergeRuleSuggestion(sec, *msg.RuleSuggestion)); err != nil {
		return Message{}, fmt.Errorf("cortex: apply rules: %w", err)
	}
	if err := s.store.MarkRuleApplied(ctx, conversationID, messageID); err != nil {
		return Message{}, err
	}
	// Journal the apply into the thread so later turns know the rules
	// changed; best-effort, the apply itself already succeeded.
	_, _ = s.store.AppendMessage(ctx, Message{
		ConversationID: conv.ID,
		Role:           "system",
		Content:        describeRuleApply(*msg.RuleSuggestion),
	})
	s.announce(conv.ID)
	return s.store.GetMessage(ctx, conversationID, messageID)
}
