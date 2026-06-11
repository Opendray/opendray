package knowledge

import "testing"

func TestValidateDraftPlaybook(t *testing.T) {
	good := draftPlaybook{
		Title:       "Ship an unreleased build to the local Mac",
		AppliesWhen: "After committing changes that need live validation",
		Steps: []string{
			"Run `opendray-v2-update-local.sh --restart` from the repo root",
			"Verify the service came back with curl http://127.0.0.1:8770/admin/",
			"Check migrations applied via the startup log",
		},
		Evidence: []string{"operator ran the update script and validated /admin responded"},
	}
	if reason, ok := validateDraftPlaybook(good); !ok {
		t.Fatalf("good draft rejected: %s", reason)
	}

	tests := []struct {
		name  string
		mutil func(d draftPlaybook) draftPlaybook
	}{
		{"greeting-grade title", func(d draftPlaybook) draftPlaybook { d.Title = "Hello"; return d }},
		{"single-word title", func(d draftPlaybook) draftPlaybook { d.Title = "Deployment"; return d }},
		{"no trigger", func(d draftPlaybook) draftPlaybook { d.AppliesWhen = ""; return d }},
		{"too few steps", func(d draftPlaybook) draftPlaybook { d.Steps = d.Steps[:2]; return d }},
		{"prose-only steps", func(d draftPlaybook) draftPlaybook {
			d.Steps = []string{
				"Think carefully about the problem",
				"Do the needful and be diligent",
				"Confirm everything is fine afterwards",
			}
			return d
		}},
		{"no evidence", func(d draftPlaybook) draftPlaybook { d.Evidence = nil; return d }},
		{"evidence too short", func(d draftPlaybook) draftPlaybook { d.Evidence = []string{"yes"}; return d }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if reason, ok := validateDraftPlaybook(tt.mutil(good)); ok {
				t.Errorf("draft should be rejected (%s) but passed", tt.name)
			} else if reason == "" {
				t.Errorf("reject must carry a reason")
			}
		})
	}
}

func TestConcreteArtifactRe(t *testing.T) {
	hits := []string{
		"Run `pct create 8650` on kv01",
		"Edit /etc/systemd/system/opendray.service",
		"pnpm build with --force",
		"curl http://192.168.3.88:5432",
		"update config.toml",
	}
	for _, h := range hits {
		if !concreteArtifactRe.MatchString(h) {
			t.Errorf("should match concrete artifact: %q", h)
		}
	}
	misses := []string{
		"Think about the architecture",
		"Talk to the operator first",
	}
	for _, m := range misses {
		if concreteArtifactRe.MatchString(m) {
			t.Errorf("should NOT match prose: %q", m)
		}
	}
}
