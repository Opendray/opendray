// Package channel is the unified hub over messaging services
// (telegram, slack, imessage, ...).
//
// Per design §8.3 the package owns:
//   - The Channel interface plus per-kind implementations under
//     internal/channel/<kind>/.
//   - A Hub that loads enabled rows from the channels table at
//     startup, drives each impl's lifecycle, and dispatches outbound
//     event-bus topics (session.idle, session.ended) to channels
//     whose config opts in via notify_on.
//   - Inbound persistence + event publishing — but NOT inbound→session
//     routing, which is delegated to event-bus consumers per ADR 0005.
//
// Adding a new channel kind = implement Channel, register a factory
// from package init(), drop the package import in app/. The Hub
// requires no changes.
package channel

import (
	"context"
	"errors"
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
	// Send pushes one outbound message.
	Send(ctx context.Context, msg ChannelMessage) error
}

// InboundFunc is invoked by Channel impls when a message arrives. The
// Hub provides this callback during Start.
type InboundFunc func(ctx context.Context, msg ChannelMessage) error

var (
	ErrNotFound      = errors.New("channel not found")
	ErrUnknownKind   = errors.New("unknown channel kind")
	ErrAlreadyExists = errors.New("channel already exists")
)
