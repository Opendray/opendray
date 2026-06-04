package main

import (
	"path/filepath"
	"testing"
)

func TestIsTCCProtectedPath(t *testing.T) {
	home := "/Users/alice"
	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join(home, "Documents", "proj", "config.toml"), true},
		{filepath.Join(home, "Desktop", "config.toml"), true},
		{filepath.Join(home, "Downloads", "x"), true},
		{"/Volumes/UNAS/x", true},
		{filepath.Join(home, "Documents"), true},
		{filepath.Join(home, ".opendray", "config.toml"), false},
		{filepath.Join(home, "Library", "Application Support", "opendray", "c.toml"), false},
		{"/opt/opendray/config.toml", false},
		{"", false},
		// A sibling that merely shares a prefix must NOT match.
		{filepath.Join(home, "Documents-archive", "c.toml"), false},
	}
	for _, c := range cases {
		if got := isTCCProtectedPath(home, c.path); got != c.want {
			t.Errorf("isTCCProtectedPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestSignatureIsAdhoc(t *testing.T) {
	adhoc := "Identifier=a.out\nCodeDirectory v=20400 flags=0x20002(adhoc,linker-signed)\nSignature=adhoc\n"
	if !signatureIsAdhoc(adhoc) {
		t.Error("expected ad-hoc signature to be detected")
	}
	stable := "Identifier=online.opendray.gateway\nAuthority=opendray-codesign\nSignature size=...\n"
	if signatureIsAdhoc(stable) {
		t.Error("stable signature must not be reported as ad-hoc")
	}
}

func TestSignatureHasIdentifier(t *testing.T) {
	out := "Identifier=online.opendray.gateway\nFormat=Mach-O\n"
	if !signatureHasIdentifier(out, "online.opendray.gateway") {
		t.Error("expected identifier match")
	}
	if signatureHasIdentifier(out, "something.else") {
		t.Error("unexpected identifier match")
	}
}

func TestConfigArgFromPlistXML(t *testing.T) {
	plist := `<array>
    <string>/Users/alice/.opendray/bin/opendray</string>
    <string>serve</string>
    <string>-config</string>
    <string>/Users/alice/Documents/proj/config.toml</string>
  </array>`
	if got := configArgFromPlistXML(plist); got != "/Users/alice/Documents/proj/config.toml" {
		t.Errorf("configArgFromPlistXML = %q", got)
	}
	if got := configArgFromPlistXML("<array><string>serve</string></array>"); got != "" {
		t.Errorf("no -config should yield empty, got %q", got)
	}
}
