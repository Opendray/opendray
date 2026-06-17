package integration

import (
	"encoding/json"
	"testing"
)

func TestValidAndNormalizePermissionMode(t *testing.T) {
	if !ValidPermissionMode(PermissionModeDefault) || !ValidPermissionMode(PermissionModeBypass) {
		t.Error("default/bypass must be valid")
	}
	if ValidPermissionMode("yolo") || ValidPermissionMode("") {
		t.Error("unknown/empty must be invalid")
	}
	if NormalizePermissionMode("") != PermissionModeDefault {
		t.Error("empty must normalize to default")
	}
	if NormalizePermissionMode(PermissionModeBypass) != PermissionModeBypass {
		t.Error("bypass must pass through")
	}
}

func TestValidateMCPServers(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"empty ok", "", false},
		{"empty array ok", "[]", false},
		{"null ok", "null", false},
		{"valid array", `[{"name":"inv","command":"/bin/x"}]`, false},
		{"two valid", `[{"name":"a"},{"name":"b","url":"http://x"}]`, false},
		{"not an array", `{"name":"x"}`, true},
		{"object missing name", `[{"command":"/bin/x"}]`, true},
		{"object blank name", `[{"name":"  "}]`, true},
		{"malformed json", `[{"name":`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPServers(json.RawMessage(tt.raw))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPServers(%q) err=%v, wantErr=%v", tt.raw, err, tt.wantErr)
			}
		})
	}
}
