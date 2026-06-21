package agyacct

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// agy persists each conversation as a SQLite db keyed by UUID under
// <HOME>/.gemini/antigravity-cli/conversations/, and records the
// most-recent conversation per working directory in
// <HOME>/.gemini/antigravity-cli/cache/last_conversations.json
// ({"<cwd>": "<conversation-uuid>"}). Because the db is a plain portable
// file and auth lives separately (the OAuth token), a conversation can be
// resumed under a *different* account by copying its db into that
// account's HOME — which is how the account switch preserves the session.
const (
	agyConversationsRel = ".gemini/antigravity-cli/conversations"
	agyLastConvRel      = ".gemini/antigravity-cli/cache/last_conversations.json"
)

// AccountHome returns the HOME directory for account id. Empty id (no
// pinned account) → the gateway user's real HOME, i.e. the default agy
// login (~/.gemini). Implements session.AntigravityAccountResolver.
func (s *Service) AccountHome(ctx context.Context, id string) (string, error) {
	if id == "" {
		return os.UserHomeDir()
	}
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if a.ConfigDir == "" {
		return os.UserHomeDir()
	}
	return a.ConfigDir, nil
}

// ConversationIDForCwd returns the id of the most-recent agy conversation
// for cwd under home, or "" when home has no record for that cwd.
func (s *Service) ConversationIDForCwd(home, cwd string) string {
	return conversationIDForCwd(home, cwd)
}

func conversationIDForCwd(home, cwd string) string {
	if home == "" || cwd == "" {
		return ""
	}
	body, err := os.ReadFile(filepath.Join(home, agyLastConvRel))
	if err != nil {
		return ""
	}
	var m map[string]string
	if json.Unmarshal(body, &m) != nil {
		return ""
	}
	return m[cwd]
}

// CopyConversation copies conversation convID's SQLite files from
// srcHome's conversations dir into dstHome's, then records cwd->convID in
// dstHome's last_conversations.json so a subsequent resume finds it.
// No-op when src == dst. Used by the account switch to carry a running
// session's conversation onto the new account's HOME before respawn.
// Safe because the switch stops the source agy process first.
func (s *Service) CopyConversation(srcHome, dstHome, convID, cwd string) error {
	return copyConversation(srcHome, dstHome, convID, cwd)
}

func copyConversation(srcHome, dstHome, convID, cwd string) error {
	if srcHome == "" || dstHome == "" || convID == "" {
		return fmt.Errorf("copy conversation: missing src/dst/id")
	}
	if srcHome == dstHome {
		return nil
	}
	srcDir := filepath.Join(srcHome, agyConversationsRel)
	dstDir := filepath.Join(dstHome, agyConversationsRel)
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return fmt.Errorf("mkdir conversations: %w", err)
	}
	// A conversation is a .db plus optional -wal/-shm sidecars.
	copied := false
	for _, suffix := range []string{".db", ".db-wal", ".db-shm"} {
		data, err := os.ReadFile(filepath.Join(srcDir, convID+suffix))
		if err != nil {
			if suffix == ".db" {
				return fmt.Errorf("read conversation %s: %w", convID, err)
			}
			continue // wal/shm optional (checkpointed dbs have none)
		}
		if err := os.WriteFile(filepath.Join(dstDir, convID+suffix), data, 0o600); err != nil {
			return fmt.Errorf("write conversation %s%s: %w", convID, suffix, err)
		}
		copied = true
	}
	if !copied {
		return fmt.Errorf("conversation %s not found under %s", convID, srcDir)
	}
	if cwd != "" {
		if err := setLastConversation(dstHome, cwd, convID); err != nil {
			return fmt.Errorf("record last conversation: %w", err)
		}
	}
	return nil
}

func setLastConversation(home, cwd, convID string) error {
	path := filepath.Join(home, agyLastConvRel)
	m := map[string]string{}
	if body, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(body, &m)
	}
	m[cwd] = convID
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}
