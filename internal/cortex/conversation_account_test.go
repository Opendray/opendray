package cortex

import "testing"

func TestNormalizeConvOverride_ClaudeAccount(t *testing.T) {
	tests := []struct {
		name                                 string
		provider, model, summarizer, account string
		wantProvider, wantAccount            string
		wantErr                              bool
	}{
		{"claude keeps account", "claude", "opus", "", "acc-1", "claude", "acc-1", false},
		{"non-claude clears account", "antigravity", "", "", "acc-1", "antigravity", "", false},
		{"codex clears account", "codex", "", "", "acc-1", "codex", "", false},
		{"summarizer clears account+provider", "", "", "sum-1", "acc-1", "", "", false},
		{"empty provider clears account", "", "", "", "acc-1", "", "", false},
		{"account trimmed", "claude", "", "", "  acc-2  ", "claude", "acc-2", false},
		{"both provider and summarizer is error", "claude", "", "sum-1", "acc-1", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _, _, a, err := normalizeConvOverride(tt.provider, tt.model, tt.summarizer, tt.account)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if p != tt.wantProvider || a != tt.wantAccount {
				t.Errorf("got provider=%q account=%q, want provider=%q account=%q", p, a, tt.wantProvider, tt.wantAccount)
			}
		})
	}
}
