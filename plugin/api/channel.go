package api

import "context"

// Channel is a messaging integration — Telegram, Slack, Discord, Matrix,
// LINE, etc. A Channel plugin owns the connection to its messaging
// service and translates between the service's wire format and
// OpenDray's neutral ChannelMessage shape.
//
// Channels are bidirectional: Send() pushes outbound messages from the
// host to the service, and the channel publishes inbound messages by
// dispatching the HookMessageReceived event on the host's hook bus.
type Channel interface {
	// ID is the stable registry key (e.g. "telegram", "slack").
	// MUST equal the id declared in manifest.contributes.channels[].id.
	ID() string

	// Start activates the channel: connect to the messaging service,
	// subscribe to inbound events, etc. Should return promptly; long
	// connect logic must run on a goroutine and respect ctx.
	Start(ctx context.Context) error

	// Stop closes the channel and releases all resources. After Stop
	// returns, Send() may be called again only after a new Start.
	Stop(ctx context.Context) error

	// Send delivers a message to the messaging service. Blocks until
	// the service has accepted the message or returns an error.
	Send(ctx context.Context, msg ChannelMessage) error
}

// ChannelMessage is the neutral shape exchanged between the host and
// channel plugins. Service-specific fields go in Metadata.
type ChannelMessage struct {
	// ConversationID identifies the thread/chat/channel within the
	// service. Format is service-specific but stable per conversation.
	ConversationID string `json:"conversationId"`

	// Author is the sender's identifier within the service (user id,
	// handle, etc). Empty for outbound messages from the host.
	Author string `json:"author,omitempty"`

	// Text is the plain-text body. Markdown formatting is allowed; the
	// channel is responsible for converting to the service's preferred
	// format.
	Text string `json:"text"`

	// Attachments carry binary or linked content. Channels MAY refuse
	// attachment kinds their service does not support.
	Attachments []ChannelAttachment `json:"attachments,omitempty"`

	// Metadata is service-specific extension data (e.g. Telegram
	// reply-to message ids). Hosts treat it as opaque.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ChannelAttachment is one piece of non-text content. Exactly one of
// URL or Bytes should be set; setting both is implementation-defined.
type ChannelAttachment struct {
	MimeType string `json:"mimeType"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
	Bytes    []byte `json:"bytes,omitempty"`
}
