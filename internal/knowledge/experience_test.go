package knowledge

import (
	"strings"
	"testing"
	"time"
)

func ep(session, cwd string, success bool, dur time.Duration, vec []float32) Episode {
	return Episode{
		SessionID: session,
		Cwd:       cwd,
		Title:     "Session " + session,
		Content:   "work in " + cwd,
		CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Duration:  dur,
		Success:   success,
		Embedding: vec,
		Embedder:  "test",
	}
}

func TestClusterEpisodes(t *testing.T) {
	a1 := []float32{1, 0, 0}
	a2 := []float32{0.98, 0.05, 0}
	b1 := []float32{0, 1, 0}
	eps := []Episode{
		ep("s1", "/p1", true, 10*time.Minute, a1),
		ep("s2", "/p2", true, 20*time.Minute, a2),
		ep("s3", "/p1", true, 5*time.Minute, b1),
	}
	clusters := clusterEpisodes(eps, 0.80)
	if len(clusters) != 2 {
		t.Fatalf("got %d clusters, want 2", len(clusters))
	}
	if len(clusters[0].episodes) != 2 || len(clusters[1].episodes) != 1 {
		t.Fatalf("unexpected cluster sizes: %d / %d", len(clusters[0].episodes), len(clusters[1].episodes))
	}
	// An episode with no vector is dropped, never clustered.
	if got := clusterEpisodes([]Episode{ep("s4", "/p", true, time.Minute, nil)}, 0.8); len(got) != 0 {
		t.Fatalf("vectorless episode should not cluster, got %d", len(got))
	}
}

func TestQualifyRequiresRepeatedSuccess(t *testing.T) {
	c := &ExperienceCompiler{minRecurrence: 2}
	vec := []float32{1, 0}

	// One success + one failure → does NOT qualify.
	cl := cluster{episodes: []Episode{
		ep("s1", "/p1", true, 10*time.Minute, vec),
		ep("s2", "/p1", false, 10*time.Minute, vec),
	}}
	if _, ok := c.qualify(cl); ok {
		t.Fatal("cluster with a single success must not qualify")
	}

	// The same session twice is one occurrence, not recurrence.
	cl = cluster{episodes: []Episode{
		ep("s1", "/p1", true, 10*time.Minute, vec),
		ep("s1", "/p1", true, 10*time.Minute, vec),
	}}
	if _, ok := c.qualify(cl); ok {
		t.Fatal("duplicate session ids must not count as recurrence")
	}

	// Two distinct successful sessions across two projects → qualifies,
	// and the stats drive the ranking.
	cl = cluster{episodes: []Episode{
		ep("s1", "/p1", true, 10*time.Minute, vec),
		ep("s2", "/p2", true, 30*time.Minute, vec),
		ep("s3", "/p1", false, 99*time.Minute, vec), // failure adds context, not recurrence
	}}
	st, ok := c.qualify(cl)
	if !ok {
		t.Fatal("two successful sessions should qualify")
	}
	if st.recurrence != 2 || len(st.projects) != 2 {
		t.Fatalf("stats: %+v", st)
	}
	if st.estMinutes != 30 { // median of {10, 30}
		t.Fatalf("estMinutes = %d, want 30", st.estMinutes)
	}
	if st.score != 60 { // recurrence × median minutes
		t.Fatalf("score = %v, want 60", st.score)
	}
}

func TestDurationMinutesBounds(t *testing.T) {
	if got := durationMinutes(5 * time.Second); got != 1 {
		t.Fatalf("floor: got %d", got)
	}
	if got := durationMinutes(20 * time.Hour); got != 240 {
		t.Fatalf("cap: got %d", got)
	}
	if got := durationMinutes(25 * time.Minute); got != 25 {
		t.Fatalf("plain: got %d", got)
	}
}

func TestConsumedBy(t *testing.T) {
	existing := []Node{{
		Provenance: map[string]any{"sessions": []any{"s1", "s2", "s3"}},
	}}
	if !consumedBy(existing, []string{"s2", "s3", "s9"}, 2) {
		t.Fatal("overlap of 2 should mark the cluster consumed")
	}
	if consumedBy(existing, []string{"s3", "s9"}, 2) {
		t.Fatal("overlap of 1 must not mark the cluster consumed")
	}
	if consumedBy(nil, []string{"s1", "s2"}, 2) {
		t.Fatal("no existing candidates → never consumed")
	}
}

func TestRejectExperienceVerifiesQuotes(t *testing.T) {
	labeled := map[string]string{
		"E1": "Session one\nran opendray update --restart then curl http://127.0.0.1:8770/healthz returned ok",
		"E2": "Session two\nagain ran opendray update --restart and the healthz check passed cleanly",
	}
	good := draftExperience{
		Title:       "Update the local opendray service",
		AppliesWhen: "After landing changes that need the live gateway",
		Steps: []string{
			"Run `opendray update --restart`",
			"Wait for the service to come back",
			"curl http://127.0.0.1:8770/healthz and expect ok",
		},
		Evidence: []experienceQuote{
			{Episode: "E1", Quote: "ran opendray update --restart then curl http://127.0.0.1:8770/healthz"},
			{Episode: "e2", Quote: "again ran opendray update --restart"}, // label case-insensitive
		},
	}
	if reason, ok := rejectExperience(good, labeled, 2); !ok {
		t.Fatalf("good draft rejected: %s", reason)
	}

	// Fabricated quote → not verbatim → rejected.
	fabricated := good
	fabricated.Evidence = []experienceQuote{
		{Episode: "E1", Quote: "ran opendray update --restart then curl"},
		{Episode: "E2", Quote: "completely invented narrative about the deploy"},
	}
	if reason, ok := rejectExperience(fabricated, labeled, 2); ok {
		t.Fatal("fabricated evidence must be rejected")
	} else if !strings.Contains(reason, "verified evidence") {
		t.Fatalf("unexpected reason: %s", reason)
	}

	// Both quotes from the SAME episode → no recurrence proof → rejected.
	oneEpisode := good
	oneEpisode.Evidence = []experienceQuote{
		{Episode: "E1", Quote: "ran opendray update --restart then curl"},
		{Episode: "E1", Quote: "curl http://127.0.0.1:8770/healthz returned ok"},
	}
	if _, ok := rejectExperience(oneEpisode, labeled, 2); ok {
		t.Fatal("single-episode evidence must be rejected")
	}

	// The structural floor still applies after quote verification.
	thin := good
	thin.Steps = thin.Steps[:2]
	if reason, ok := rejectExperience(thin, labeled, 2); ok {
		t.Fatal("draft below the structural floor must be rejected")
	} else if !strings.Contains(reason, "steps") {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestParseExperience(t *testing.T) {
	raw := "```json\n{\"experience\":{\"title\":\"Deploy Go to Proxmox\",\"applies_when\":\"shipping a Go API\",\"steps\":[\"a\",\"b\"],\"evidence\":[{\"episode\":\"E1\",\"quote\":\"q\"}],\"script\":\"echo hi\",\"validation\":\"curl localhost\"}}\n```"
	d, ok := parseExperience(raw)
	if !ok {
		t.Fatal("should parse fenced JSON")
	}
	if d.Title != "Deploy Go to Proxmox" || d.Script != "echo hi" || len(d.Evidence) != 1 {
		t.Fatalf("unexpected draft: %+v", d)
	}
	if _, ok := parseExperience(`{"experience":null}`); ok {
		t.Fatal("declined cluster must parse to ok=false")
	}
	if _, ok := parseExperience("not json"); ok {
		t.Fatal("garbage must parse to ok=false")
	}
}

func TestComposeRunScript(t *testing.T) {
	out := composeRunScript("echo hi")
	if !strings.HasPrefix(out, "#!/usr/bin/env bash\nset -euo pipefail") || !strings.Contains(out, "echo hi") {
		t.Fatalf("missing safety header:\n%s", out)
	}
	withBang := composeRunScript("#!/bin/sh\necho hi")
	if !strings.HasPrefix(withBang, "#!/bin/sh") {
		t.Fatalf("existing shebang must be preserved:\n%s", withBang)
	}
}

func TestRenderExperienceBody(t *testing.T) {
	body := renderExperienceBody(draftExperience{
		AppliesWhen: "shipping a Go API",
		Steps:       []string{"read infra registry", "create LXC"},
		Pitfalls:    []string{"TCC denial"},
		Evidence:    []experienceQuote{{Episode: "E1", Quote: "created the LXC"}},
		Validation:  "pct status 8650 shows running",
	}, clusterStats{recurrence: 3, projects: []string{"/a", "/b"}, estMinutes: 25}, true)
	for _, want := range []string{
		"succeeded in 3 sessions across 2 project(s), ~25 min each",
		"## Steps", "## Pitfalls", "E1 — created the LXC",
		"## Verification", "pct status 8650",
		"run.sh",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestParseVector(t *testing.T) {
	v := ParseVector("[1,0.5,-2]")
	if len(v) != 3 || v[0] != 1 || v[1] != 0.5 || v[2] != -2 {
		t.Fatalf("got %v", v)
	}
	if ParseVector("") != nil || ParseVector("1,2") != nil || ParseVector("[a,b]") != nil {
		t.Fatal("malformed input must return nil")
	}
}

func TestEpisodesSignatureChangesWithFeedstock(t *testing.T) {
	a := []Episode{ep("s1", "/p", true, time.Minute, nil)}
	b := []Episode{ep("s1", "/p", true, time.Minute, nil), ep("s2", "/p", true, time.Minute, nil)}
	if episodesSignature(a) == episodesSignature(b) {
		t.Fatal("signature must change when episodes change")
	}
	if episodesSignature(a) != episodesSignature(a) {
		t.Fatal("signature must be deterministic")
	}
}
