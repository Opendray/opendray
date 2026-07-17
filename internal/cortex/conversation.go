package cortex

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── curation conversations (Phase 4) ──────────────────────────
//
// The channel the operator uses to actively maintain the system by
// talking to the AI: update a doc section ("更新技术栈"), or discuss
// and re-draft a Foundational/Emergent knowledge page (重新制定方针).
// Lightweight by default (worker-fabric LLM, no real session); a
// conversation can be escalated to a full agent session when the
// revision needs codebase grounding.

// Conversation target kinds.
const (
	TargetDocSection = "doc_section"
	TargetKBPage     = "kb_page"
	TargetBlueprint  = "blueprint"
)

// Conversation statuses.
const (
	ConvOpen      = "open"
	ConvClosed    = "closed"
	ConvEscalated = "escalated"
)

// Conversation is one curation thread bound to a target.
type Conversation struct {
	ID                 string `json:"id"`
	TargetKind         string `json:"target_kind"`
	TargetCwd          string `json:"target_cwd"`
	TargetSlug         string `json:"target_slug"`
	Status             string `json:"status"`
	EscalatedSessionID string `json:"escalated_session_id,omitempty"`
	// The AI turns' model override (all empty = use the global `curation`
	// worker config). Mutually exclusive: SummarizerID pins a local/HTTP
	// model (a summarizer_providers row); ProviderID+Model pins a
	// cloud-agent CLI. See SetProvider.
	ProviderID string `json:"provider_id,omitempty"`
	Model      string `json:"model,omitempty"`
	// ClaudeAccountID pins which Claude (cliacct) account a claude turn
	// runs against — Claude is multi-account. Only meaningful when
	// ProviderID == "claude"; threaded into the worker as Config.AccountID.
	ClaudeAccountID string    `json:"claude_account_id,omitempty"`
	SummarizerID    string    `json:"summarizer_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// validConvProvider reports whether p is an allowed cloud-agent override
// provider (empty = no cloud-agent override). It lists the cloud-agent
// providers the Cortex conversation worker can actually drive headlessly and
// must stay in sync with the worker fabric's AgentWorker.Run/buildCommand
// switch (claude / codex / antigravity / grok / opencode).
func validConvProvider(p string) bool {
	switch p {
	case "", "claude", "codex", "antigravity", "grok", "opencode":
		return true
	}
	return false
}

// normalizeConvOverride validates + canonicalises a conversation override.
// Local model (summarizerID) and cloud-agent (providerID) are mutually
// exclusive; all empty means "use the global curation worker". A Claude
// account override is only meaningful for provider=claude and is cleared
// otherwise.
func normalizeConvOverride(providerID, model, summarizerID, claudeAccountID string) (p, m, s, acct string, err error) {
	providerID = strings.TrimSpace(providerID)
	summarizerID = strings.TrimSpace(summarizerID)
	claudeAccountID = strings.TrimSpace(claudeAccountID)
	if summarizerID != "" {
		if providerID != "" {
			return "", "", "", "", errors.New("cortex: set either a cloud-agent provider or a local model, not both")
		}
		// model is an OPTIONAL per-call override of the model the local
		// endpoint serves (empty → the provider row's configured model).
		// A local model has no Claude account.
		return "", model, summarizerID, "", nil
	}
	if !validConvProvider(providerID) {
		return "", "", "", "", fmt.Errorf("cortex: provider_id %q is not supported for AI discussion (want claude|codex|antigravity|grok|opencode)", providerID)
	}
	if providerID == "" {
		model = "" // a model with no provider is meaningless
	}
	if providerID != "claude" {
		claudeAccountID = "" // account selection only applies to claude
	}
	return providerID, model, "", claudeAccountID, nil
}

// Message is one turn in a conversation. RevisionAction records what
// the AI's structured revision did ("applied" | "proposed" | "").
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"` // operator | ai | system
	Content        string    `json:"content"`
	RevisionAction string    `json:"revision_action,omitempty"`
	RevisionRef    string    `json:"revision_ref,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ErrConversationNotFound is returned for unknown conversation ids.
var ErrConversationNotFound = errors.New("cortex: conversation not found")

// ConversationStore persists conversations + messages.
type ConversationStore struct {
	pool *pgxpool.Pool
}

// NewConversationStore wires the store on the shared pool.
func NewConversationStore(pool *pgxpool.Pool) *ConversationStore {
	return &ConversationStore{pool: pool}
}

func newConvID(prefix string) string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return prefix + base64.RawURLEncoding.EncodeToString(b[:])
}

func validTargetKind(k string) bool {
	return k == TargetDocSection || k == TargetKBPage || k == TargetBlueprint
}

// Create opens a conversation bound to a target. The override (providerID/
// model for a cloud agent, OR summarizerID for a local model) is optional;
// all empty = use the global curation worker.
func (s *ConversationStore) Create(ctx context.Context, targetKind, targetCwd, targetSlug, providerID, model, summarizerID, claudeAccountID string) (Conversation, error) {
	if !validTargetKind(targetKind) {
		return Conversation{}, fmt.Errorf("cortex: bad target_kind %q", targetKind)
	}
	if strings.TrimSpace(targetCwd) == "" || strings.TrimSpace(targetSlug) == "" {
		return Conversation{}, errors.New("cortex: target_cwd and target_slug are required")
	}
	providerID, model, summarizerID, claudeAccountID, err := normalizeConvOverride(providerID, model, summarizerID, claudeAccountID)
	if err != nil {
		return Conversation{}, err
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO cortex_conversations (id, target_kind, target_cwd, target_slug, provider_id, model, claude_account_id, summarizer_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, target_kind, target_cwd, target_slug, status,
		          COALESCE(escalated_session_id, ''), provider_id, model, claude_account_id, summarizer_id, created_at, updated_at`,
		newConvID("cc_"), targetKind, targetCwd, targetSlug, providerID, model, claudeAccountID, summarizerID)
	return scanConversation(row)
}

// Get returns one conversation by id.
func (s *ConversationStore) Get(ctx context.Context, id string) (Conversation, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, target_kind, target_cwd, target_slug, status,
		       COALESCE(escalated_session_id, ''), provider_id, model, claude_account_id, summarizer_id, created_at, updated_at
		  FROM cortex_conversations WHERE id = $1`, id)
	c, err := scanConversation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrConversationNotFound
	}
	return c, err
}

// SetProvider pins (or clears) the conversation's model override. All empty
// resets it to the global curation worker config; otherwise either a
// cloud-agent provider (+model) OR a local summarizer provider.
func (s *ConversationStore) SetProvider(ctx context.Context, id, providerID, model, summarizerID, claudeAccountID string) error {
	providerID, model, summarizerID, claudeAccountID, err := normalizeConvOverride(providerID, model, summarizerID, claudeAccountID)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE cortex_conversations
		   SET provider_id = $1, model = $2, summarizer_id = $3, claude_account_id = $4, updated_at = NOW()
		 WHERE id = $5`, providerID, model, summarizerID, claudeAccountID, id)
	if err != nil {
		return fmt.Errorf("cortex: set conversation provider: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConversationNotFound
	}
	return nil
}

// ListByTarget returns conversations for a target, newest first.
// Empty cwd+slug lists every conversation (admin overview).
func (s *ConversationStore) ListByTarget(ctx context.Context, targetCwd, targetSlug string, limit int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if targetCwd == "" && targetSlug == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, target_kind, target_cwd, target_slug, status,
			       COALESCE(escalated_session_id, ''), provider_id, model, claude_account_id, summarizer_id, created_at, updated_at
			  FROM cortex_conversations
			 ORDER BY updated_at DESC LIMIT $1`, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, target_kind, target_cwd, target_slug, status,
			       COALESCE(escalated_session_id, ''), provider_id, model, claude_account_id, summarizer_id, created_at, updated_at
			  FROM cortex_conversations
			 WHERE target_cwd = $1 AND target_slug = $2
			 ORDER BY updated_at DESC LIMIT $3`, targetCwd, targetSlug, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("cortex: list conversations: %w", err)
	}
	defer rows.Close()
	var out []Conversation
	for rows.Next() {
		c, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetStatus updates status (+ optional escalated session link).
func (s *ConversationStore) SetStatus(ctx context.Context, id, status, escalatedSessionID string) error {
	var esc any
	if escalatedSessionID != "" {
		esc = escalatedSessionID
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE cortex_conversations
		   SET status = $1,
		       escalated_session_id = COALESCE($2, escalated_session_id),
		       updated_at = NOW()
		 WHERE id = $3`, status, esc, id)
	if err != nil {
		return fmt.Errorf("cortex: set conversation status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConversationNotFound
	}
	return nil
}

// AppendMessage adds one turn and bumps the conversation's updated_at.
func (s *ConversationStore) AppendMessage(ctx context.Context, m Message) (Message, error) {
	if m.Role != "operator" && m.Role != "ai" && m.Role != "system" {
		return Message{}, fmt.Errorf("cortex: bad message role %q", m.Role)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO cortex_conversation_messages
			(id, conversation_id, role, content, revision_action, revision_ref)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, conversation_id, role, content, revision_action, revision_ref, created_at`,
		newConvID("ccm_"), m.ConversationID, m.Role, m.Content, m.RevisionAction, m.RevisionRef)
	var out Message
	if err := row.Scan(&out.ID, &out.ConversationID, &out.Role, &out.Content,
		&out.RevisionAction, &out.RevisionRef, &out.CreatedAt); err != nil {
		return Message{}, fmt.Errorf("cortex: append message: %w", err)
	}
	_, _ = s.pool.Exec(ctx,
		`UPDATE cortex_conversations SET updated_at = NOW() WHERE id = $1`, m.ConversationID)
	return out, nil
}

// Messages returns a conversation's turns, oldest first.
func (s *ConversationStore) Messages(ctx context.Context, conversationID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, conversation_id, role, content, revision_action, revision_ref, created_at
		  FROM cortex_conversation_messages
		 WHERE conversation_id = $1
		 ORDER BY created_at ASC LIMIT $2`, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("cortex: list messages: %w", err)
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content,
			&m.RevisionAction, &m.RevisionRef, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type convRowScanner interface{ Scan(dest ...any) error }

func scanConversation(row convRowScanner) (Conversation, error) {
	var c Conversation
	err := row.Scan(&c.ID, &c.TargetKind, &c.TargetCwd, &c.TargetSlug,
		&c.Status, &c.EscalatedSessionID, &c.ProviderID, &c.Model, &c.ClaudeAccountID, &c.SummarizerID,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return Conversation{}, err
	}
	return c, nil
}
