package roundtable

import (
	"reflect"
	"testing"
)

func TestNextRoundMentions(t *testing.T) {
	tests := []struct {
		name     string
		queue    []string
		mentions []string
		self     string
		want     []string
	}{
		{
			name:     "excludes self",
			queue:    nil,
			mentions: []string{"claude", "codex"},
			self:     "claude",
			want:     []string{"codex"},
		},
		{
			name:     "dedups against queue",
			queue:    []string{"codex"},
			mentions: []string{"codex", "antigravity"},
			self:     "claude",
			want:     []string{"codex", "antigravity"},
		},
		{
			name:     "preserves order, drops dups within mentions",
			queue:    nil,
			mentions: []string{"antigravity", "codex", "antigravity"},
			self:     "claude",
			want:     []string{"antigravity", "codex"},
		},
		{
			name:     "no mentions leaves queue untouched",
			queue:    []string{"codex"},
			mentions: nil,
			self:     "claude",
			want:     []string{"codex"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nextRoundMentions(tc.queue, tc.mentions, tc.self)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("nextRoundMentions(%v,%v,%q) = %v, want %v",
					tc.queue, tc.mentions, tc.self, got, tc.want)
			}
		})
	}
}

func TestPendingSpeakers(t *testing.T) {
	seats := []Seat{{Provider: "claude"}, {Provider: "codex"}}

	t.Run("reads pending from the paused note, filtered to seated", func(t *testing.T) {
		msgs := []Message{
			{Role: RoleSeat, SeatProvider: "claude"},
			{Role: RoleSystem, Mentions: []string{"codex", "grok"}}, // grok not seated
		}
		got := pendingSpeakers(msgs, seats)
		if !reflect.DeepEqual(got, []string{"codex"}) {
			t.Errorf("want [codex] (seated only), got %v", got)
		}
	})

	t.Run("nothing pending when last message is a real turn", func(t *testing.T) {
		msgs := []Message{
			{Role: RoleSystem, Mentions: []string{"codex"}},
			{Role: RoleSeat, SeatProvider: "codex"}, // spoke after the pause
		}
		if got := pendingSpeakers(msgs, seats); got != nil {
			t.Errorf("want nil (a turn happened after the pause), got %v", got)
		}
	})

	t.Run("nil when no messages", func(t *testing.T) {
		if got := pendingSpeakers(nil, seats); got != nil {
			t.Errorf("want nil, got %v", got)
		}
	})
}
