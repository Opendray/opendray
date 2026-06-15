package channel

import (
	"encoding/json"
	"strings"
)

// At-rest encryption for the secret fields inside channels.config.
//
// Channel credentials (bot tokens, app secrets, webhook keys) live as
// string fields inside the per-kind config JSON. When the backup feature
// is armed they're wrapped with the backup cipher's AES-GCM field
// envelope ("v1:..."), the same key + format used for git-host tokens
// and backup_targets passwords; otherwise they stay plaintext (the
// historical behaviour). Encryption is transparent: the store encrypts
// on write and decrypts on read, so the hub, the channel adapters and
// the admin API all see plaintext exactly as before — only the DB column
// changes.
//
// The design is deliberately round-trip safe. The mute toggle and other
// config patches read the config, change one field, and write it back;
// if the backup key has rotated and a secret can't be decrypted, decrypt
// leaves the "v1:" ciphertext in place (never blanks it) and encrypt
// skips anything already wrapped — so a read-modify-write can never lose
// a secret it couldn't read. A rotated key just means the channel fails
// to connect with its (still-stored) credential until the operator
// re-enters it.

// FieldCipher wraps a short secret at rest. The backup cipher satisfies
// it; nil (or a cipher whose backup feature isn't armed) means secrets
// stay plaintext.
type FieldCipher interface {
	EncryptField(plain string) (string, error)
	DecryptField(envelope string) (string, error)
}

// encryptedFieldPrefix marks a value wrapped by FieldCipher.EncryptField.
// A stored value without it is legacy plaintext.
const encryptedFieldPrefix = "v1:"

// channelSecretFields lists, per channel kind, the top-level config JSON
// fields that hold credentials and must be encrypted at rest.
var channelSecretFields = map[string][]string{
	"telegram": {"bot_token"},
	"slack":    {"bot_token", "app_token"},
	"discord":  {"bot_token"},
	"feishu":   {"app_secret", "verification_token"},
	"dingtalk": {"secret"},
	"wecom":    {"webhook_key"},
	"wechat":   {"app_token"},
}

// encryptConfigSecrets returns config with this kind's secret fields
// wrapped. Fields that are absent, empty, non-string, or already wrapped
// are left untouched; a cipher that isn't armed (EncryptField errors or
// returns "") leaves the plaintext as-is. The input is returned verbatim
// when there's nothing to do, so callers can store it directly.
func encryptConfigSecrets(c FieldCipher, kind string, raw json.RawMessage) json.RawMessage {
	fields := channelSecretFields[kind]
	if c == nil || len(fields) == 0 || len(raw) == 0 {
		return raw
	}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw // not a JSON object — nothing to wrap
	}
	changed := false
	for _, f := range fields {
		val, ok := obj[f]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(val, &s); err != nil || s == "" {
			continue // not a string / empty
		}
		if strings.HasPrefix(s, encryptedFieldPrefix) {
			continue // already wrapped — never double-encrypt
		}
		enc, err := c.EncryptField(s)
		if err != nil || enc == "" {
			continue // not armed — keep plaintext
		}
		encoded, err := json.Marshal(enc)
		if err != nil {
			continue
		}
		obj[f] = encoded
		changed = true
	}
	if !changed {
		return raw
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return out
}

// decryptConfigSecrets returns config with this kind's wrapped secret
// fields unwrapped. A field that can't be decrypted (no cipher, or the
// backup key rotated) is left as its "v1:" ciphertext — deliberately NOT
// blanked — so a later read-modify-write re-stores it intact rather than
// erasing a secret it couldn't read. The channel then simply fails to
// connect with the unreadable credential until it's re-entered.
func decryptConfigSecrets(c FieldCipher, kind string, raw json.RawMessage) json.RawMessage {
	fields := channelSecretFields[kind]
	if len(fields) == 0 || len(raw) == 0 {
		return raw
	}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw
	}
	changed := false
	for _, f := range fields {
		val, ok := obj[f]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(val, &s); err != nil {
			continue
		}
		if !strings.HasPrefix(s, encryptedFieldPrefix) {
			continue // plaintext already
		}
		if c == nil {
			continue // can't decrypt — leave ciphertext (don't lose it)
		}
		plain, err := c.DecryptField(s)
		if err != nil {
			continue // key rotated — leave ciphertext (don't lose it)
		}
		encoded, err := json.Marshal(plain)
		if err != nil {
			continue
		}
		obj[f] = encoded
		changed = true
	}
	if !changed {
		return raw
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return out
}
