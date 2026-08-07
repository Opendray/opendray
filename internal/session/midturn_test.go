package session

import "testing"

// AnyMidTurn is the keep-awake activity signal: it must fire only for a
// session that is actively producing output, not for idle or terminal
// ones sitting in the live map.
func TestAnyMidTurn(t *testing.T) {
	tests := []struct {
		name   string
		states []State
		want   bool
	}{
		{"no live sessions", nil, false},
		{"only idle sessions", []State{StateIdle, StateIdle}, false},
		{"terminal states do not count", []State{StateEnded, StateStopped}, false},
		{"one running among idle", []State{StateIdle, StateRunning, StateIdle}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{sessions: make(map[string]*runningSession)}
			for i, st := range tt.states {
				m.sessions[string(rune('a'+i))] = &runningSession{sess: Session{State: st}}
			}
			if got := m.AnyMidTurn(); got != tt.want {
				t.Fatalf("AnyMidTurn() = %v, want %v", got, tt.want)
			}
		})
	}
}
