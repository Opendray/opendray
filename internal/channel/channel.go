// Package channel is the unified hub over messaging services
// (telegram, slack, bridge, ...).
//
// Per design §8.3 the package owns:
//   - The Channel interface plus per-kind implementations under
//     internal/channel/<kind>/.
//   - A Hub that loads enabled rows from the channels table at
//     startup, drives each impl's lifecycle, and dispatches outbound
//     event-bus topics (session.idle, session.ended) to every enabled,
//     unmuted channel (deduped by each channel's repeat policy).
//   - Inbound persistence + event publishing — but NOT inbound→session
//     routing, which is delegated to event-bus consumers per ADR 0005.
//
// # Capability interfaces
//
// Channel itself is intentionally minimal: Kind/ID/Start/Stop/Send.
// Richer abilities (cards, buttons, typing indicators, message
// updates, files) are exposed as small *optional* interfaces an impl
// may also implement. The Hub uses type assertions and falls back to
// Send(text) when an optional capability is missing — so adding a
// new platform never gets blocked on supporting every feature.
//
// Adding a new channel kind = implement Channel (+ any capability
// interface that fits), register a factory from package init(), drop
// the package import in app/. The Hub requires no changes.
package channel

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

// Direction matches channel_messages.direction in the schema.
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// ChannelMessage is the canonical shape exchanged between Hub and
// Channel impls. It maps onto channel_messages rows.
//
// ReplyCtx is opaque to the Hub — it carries platform-specific routing
// data (Telegram chat_id+message_id, Slack thread_ts, Feishu
// open_chat_id+open_message_id, ...) that the Channel impl needs to
// route an outbound reply back to the right thread/message. The Hub
// just preserves it across inbound→outbound flows; impls cast it back
// to their concrete type.
type ChannelMessage struct {
	ID             int64          `json:"id,omitempty"`
	ChannelID      string         `json:"channel_id"`
	Direction      Direction      `json:"direction"`
	ConversationID string         `json:"conversation_id"`
	SessionID      string         `json:"session_id,omitempty"`
	Author         string         `json:"author,omitempty"`
	Text           string         `json:"text"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Timestamp      time.Time      `json:"ts"`

	// ReplyCtx is the platform-routing handle the Channel impl needs
	// to deliver a reply. Hub treats it as opaque.
	ReplyCtx any `json:"-"`
}

// SessionKey returns the canonical "{kind}:{conversation_id}:{author}"
// identifier used to scope session routing across channels.
//
// Empty Author falls back to ConversationID as the user scope so
// notification-only flows still produce a stable key.
func (m ChannelMessage) SessionKey(kind string) string {
	conv := m.ConversationID
	if conv == "" {
		conv = "default"
	}
	user := m.Author
	if user == "" {
		user = conv
	}
	return strings.Join([]string{kind, conv, user}, ":")
}

// Channel is one configured messaging integration.
type Channel interface {
	// Kind returns the registered factory key (e.g. "telegram").
	Kind() string
	// ID returns the channels.id db row.
	ID() string
	// Start begins listening for inbound messages and stays running
	// until Stop is called. The supplied InboundFunc is the Hub's
	// callback for persistence + event publishing.
	Start(ctx context.Context, inbound InboundFunc) error
	// Stop tears down resources gracefully.
	Stop(ctx context.Context) error
	// Send pushes one outbound text message. The mandatory baseline —
	// every richer capability eventually degrades to this.
	Send(ctx context.Context, msg ChannelMessage) error
}

// InboundFunc is invoked by Channel impls when a message arrives. The
// Hub provides this callback during Start.
type InboundFunc func(ctx context.Context, msg ChannelMessage) error

// CardSender is implemented by channels that natively render
// structured Card payloads. Channels that do not implement this get
// a plain-text fallback (Card.RenderText sent via Channel.Send) from
// the Hub.
type CardSender interface {
	SendCard(ctx context.Context, msg ChannelMessage, card *Card) error
}

// ButtonSender is implemented by channels that can attach an inline
// button keyboard to a text message (subset of CardSender for impls
// that support buttons but not the wider Card model).
type ButtonSender interface {
	SendWithButtons(ctx context.Context, msg ChannelMessage, buttons [][]ButtonOption) error
}

// MessageUpdater is implemented by channels that can edit a
// previously-sent message in place. Used for streaming agent output.
//
// PreviewHandle is whatever opaque token the channel returned in
// ChannelMessage.Metadata["preview_handle"] when the original message
// was sent.
type MessageUpdater interface {
	UpdateMessage(ctx context.Context, msg ChannelMessage, previewHandle string, newText string) error
}

// TypingIndicator is implemented by channels that can show a transient
// "agent is working" hint. StartTyping returns a stop func the caller
// must invoke when work completes.
type TypingIndicator interface {
	StartTyping(ctx context.Context, msg ChannelMessage) (stop func())
}

// ImageSender is implemented by channels that can deliver an image
// attachment.
type ImageSender interface {
	SendImage(ctx context.Context, msg ChannelMessage, img ImageAttachment) error
}

// FileSender is implemented by channels that can deliver a generic
// file attachment.
type FileSender interface {
	SendFile(ctx context.Context, msg ChannelMessage, file FileAttachment) error
}

// ImageAttachment is the canonical shape for image deliveries.
// Either Path (local file) or URL must be set.
type ImageAttachment struct {
	Path    string `json:"path,omitempty"`
	URL     string `json:"url,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// FileAttachment is the canonical shape for file deliveries.
type FileAttachment struct {
	Path     string `json:"path,omitempty"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

// Capabilities enumerates which optional capability interfaces a
// concrete channel implements. Used by the Hub to advertise to admin
// UIs and by the bridge protocol to negotiate features with external
// adapters.
//
// Treat the slice as a set; order is not significant.
type Capability string

const (
	CapText           Capability = "text"
	CapCard           Capability = "card"
	CapButtons        Capability = "buttons"
	CapImage          Capability = "image"
	CapFile           Capability = "file"
	CapTyping         Capability = "typing"
	CapUpdateMessage  Capability = "update_message"
	CapReplyToMessage Capability = "reply_to_message"
)

// Capabilities returns the set of optional capabilities a Channel
// implementation supports. Text is always implied.
//
// CapabilityProvider takes precedence — see its docstring for why.
func Capabilities(c Channel) []Capability {
	if cp, ok := c.(CapabilityProvider); ok {
		caps := cp.DeclaredCapabilities()
		// Always include text, dedupe.
		seen := map[Capability]bool{CapText: true}
		out := []Capability{CapText}
		for _, k := range caps {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
		return out
	}
	out := []Capability{CapText}
	if _, ok := c.(CardSender); ok {
		out = append(out, CapCard)
	}
	if _, ok := c.(ButtonSender); ok {
		out = append(out, CapButtons)
	}
	if _, ok := c.(ImageSender); ok {
		out = append(out, CapImage)
	}
	if _, ok := c.(FileSender); ok {
		out = append(out, CapFile)
	}
	if _, ok := c.(TypingIndicator); ok {
		out = append(out, CapTyping)
	}
	if _, ok := c.(MessageUpdater); ok {
		out = append(out, CapUpdateMessage)
	}
	if _, ok := c.(ReplyCapable); ok {
		out = append(out, CapReplyToMessage)
	}
	return out
}

// ReplyCapable is a marker interface for channels that wire ReplyCtx
// into their outbound (i.e. they can post a message *as a reply* to
// the inbound message that triggered it). Most modern messaging apps
// support this; webhook-only ones may not.
type ReplyCapable interface {
	SupportsReply() bool
}

var (
	ErrNotFound      = errors.New("channel not found")
	ErrUnknownKind   = errors.New("unknown channel kind")
	ErrAlreadyExists = errors.New("channel already exists")

	// ErrNotSupported is returned by capability-implementing methods
	// (SendCard, SendImage, ...) when the underlying connector cannot
	// fulfil the request — e.g. a bridge adapter that did not claim
	// the corresponding capability on register. The Hub treats this
	// error as a signal to fall back to Channel.Send(text).
	ErrNotSupported = errors.New("channel: capability not supported")
)

// CapabilityProvider lets a channel publish the exact capability set
// it actually supports (rather than relying on type-assertion against
// the optional interfaces). The Bridge channel uses this to mirror
// whatever its connected adapter declared on register; without it the
// detector would assume Bridge supports everything regardless of the
// adapter on the other end of the wire.
type CapabilityProvider interface {
	DeclaredCapabilities() []Capability
}

// WebhookHandler is implemented by channels that receive inbound
// events via an external HTTP POST (Feishu, DingTalk, WeCom, generic
// webhook bridges). The Hub exposes a single public route
//
//	/api/v1/channels/{id}/webhook
//
// that dispatches to the matching channel's HandleWebhook. Auth is
// the channel's responsibility — typically a signed payload or a
// shared verification token in the request body / headers.
type WebhookHandler interface {
	HandleWebhook(w http.ResponseWriter, r *http.Request)
}
