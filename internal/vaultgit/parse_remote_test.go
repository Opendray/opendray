package vaultgit

import "testing"

// The owner decides WHICH credential authenticates the vault's push.
// Get it wrong and the sync silently authenticates as the wrong
// identity — which surfaces as a 403 from the forge, not as anything
// pointing back here. Hence a case per remote shape people actually
// paste in.
func TestParseRemote(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantScheme string
		wantHost   string
		wantOwner  string
	}{
		{
			name:       "https with .git",
			raw:        "https://github.com/octo-org/handbook.git",
			wantScheme: "https", wantHost: "github.com", wantOwner: "octo-org",
		},
		{
			name:       "https without .git",
			raw:        "https://github.com/Opendray/opendray",
			wantScheme: "https", wantHost: "github.com", wantOwner: "Opendray",
		},
		{
			name:       "https with trailing slash",
			raw:        "https://github.com/octo-org/handbook.git/",
			wantScheme: "https", wantHost: "github.com", wantOwner: "octo-org",
		},
		{
			name:       "scp-like ssh",
			raw:        "git@github.com:Opendray/opendray.git",
			wantScheme: "ssh", wantHost: "github.com", wantOwner: "Opendray",
		},
		{
			name:       "ssh:// url",
			raw:        "ssh://git@gitea.example.com:2222/octo-org/notes.git",
			wantScheme: "ssh", wantHost: "gitea.example.com:2222", wantOwner: "octo-org",
		},
		{
			name:       "self-hosted https with port",
			raw:        "https://gitea.example.com:3000/octo-org/notes.git",
			wantScheme: "https", wantHost: "gitea.example.com:3000", wantOwner: "octo-org",
		},
		{
			name:       "nested group path takes the FIRST segment",
			raw:        "https://gitlab.com/group/subgroup/repo.git",
			wantScheme: "https", wantHost: "gitlab.com", wantOwner: "group",
		},
		{
			name:       "host root, no owner",
			raw:        "https://example.com/",
			wantScheme: "https", wantHost: "example.com", wantOwner: "",
		},
		{
			name: "empty", raw: "",
			wantScheme: "", wantHost: "", wantOwner: "",
		},
		{
			name: "garbage is not guessed at", raw: "not a url",
			wantScheme: "", wantHost: "", wantOwner: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, host, owner := parseRemote(tt.raw)
			if scheme != tt.wantScheme || host != tt.wantHost || owner != tt.wantOwner {
				t.Fatalf("parseRemote(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.raw, scheme, host, owner,
					tt.wantScheme, tt.wantHost, tt.wantOwner)
			}
		})
	}
}
