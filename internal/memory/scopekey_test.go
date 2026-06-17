package memory

import "testing"

func TestIntegrationScopeKey(t *testing.T) {
	if got := IntegrationScopeKey("intg-42"); got != "integration:intg-42" {
		t.Errorf("IntegrationScopeKey = %q, want integration:intg-42", got)
	}
}

func TestIsIntegrationScopeKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"integration:intg-42", true},
		{"integration:", true},
		{"/home/dev/proj", false},
		{"operator", false},
		{"", false},
		{"/srv/integration-stuff", false}, // substring, not prefix
	}
	for _, tt := range tests {
		if got := IsIntegrationScopeKey(tt.key); got != tt.want {
			t.Errorf("IsIntegrationScopeKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}
