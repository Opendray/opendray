package channel

import (
	"context"
	"sort"
	"strings"
	"testing"
)

func TestChannelMessage_SessionKey(t *testing.T) {
	cases := []struct {
		name string
		msg  ChannelMessage
		kind string
		want string
	}{
		{
			name: "full triple",
			msg:  ChannelMessage{ConversationID: "42", Author: "@alice"},
			kind: "telegram",
			want: "telegram:42:@alice",
		},
		{
			name: "missing author falls back to conv",
			msg:  ChannelMessage{ConversationID: "42"},
			kind: "telegram",
			want: "telegram:42:42",
		},
		{
			name: "missing both yields default",
			msg:  ChannelMessage{},
			kind: "bridge",
			want: "bridge:default:default",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.msg.SessionKey(c.kind); got != c.want {
				t.Errorf("SessionKey = %q, want %q", got, c.want)
			}
		})
	}
}

// stubChannel is the minimal Channel impl — only Send. Used to assert
// the capability detector returns just "text" for bare impls.
type stubChannel struct{}

func (stubChannel) Kind() string                                   { return "stub" }
func (stubChannel) ID() string                                     { return "ch_stub" }
func (stubChannel) Start(_ context.Context, _ InboundFunc) error   { return nil }
func (stubChannel) Stop(_ context.Context) error                   { return nil }
func (stubChannel) Send(_ context.Context, _ ChannelMessage) error { return nil }

// richChannel implements every optional capability — used to assert
// the detector picks them all up.
type richChannel struct{ stubChannel }

func (richChannel) SendCard(_ context.Context, _ ChannelMessage, _ *Card) error { return nil }
func (richChannel) SendWithButtons(_ context.Context, _ ChannelMessage, _ [][]ButtonOption) error {
	return nil
}
func (richChannel) SendImage(_ context.Context, _ ChannelMessage, _ ImageAttachment) error {
	return nil
}
func (richChannel) SendFile(_ context.Context, _ ChannelMessage, _ FileAttachment) error { return nil }
func (richChannel) StartTyping(_ context.Context, _ ChannelMessage) (stop func())        { return func() {} }
func (richChannel) UpdateMessage(_ context.Context, _ ChannelMessage, _ string, _ string) error {
	return nil
}
func (richChannel) SupportsReply() bool { return true }

func TestCapabilities(t *testing.T) {
	t.Run("bare channel", func(t *testing.T) {
		caps := capStrings(Capabilities(stubChannel{}))
		if want := []string{string(CapText)}; !equalSorted(caps, want) {
			t.Errorf("bare caps = %v, want %v", caps, want)
		}
	})
	t.Run("rich channel", func(t *testing.T) {
		caps := capStrings(Capabilities(richChannel{}))
		want := []string{
			string(CapText), string(CapCard), string(CapButtons),
			string(CapImage), string(CapFile), string(CapTyping),
			string(CapUpdateMessage), string(CapReplyToMessage),
		}
		if !equalSorted(caps, want) {
			t.Errorf("rich caps = %v\nwant       = %v", caps, want)
		}
	})
}

func capStrings(caps []Capability) []string {
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = string(c)
	}
	return out
}

func equalSorted(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	return strings.Join(x, "|") == strings.Join(y, "|")
}
