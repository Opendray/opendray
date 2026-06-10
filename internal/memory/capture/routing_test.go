package capture

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/memory/summarizer"
)

// fakePolicy implements PolicyResolver with a canned answer.
type fakePolicy struct {
	policy string
	err    error
	calls  int
}

func (f *fakePolicy) MemoryPolicy(ctx context.Context, integrationID string) (string, error) {
	f.calls++
	return f.policy, f.err
}

func TestRouteForSession(t *testing.T) {
	tests := []struct {
		name       string
		sess       SessionInfo
		policy     PolicyResolver
		wantTier   string
		wantExpiry bool
		wantSkip   bool
	}{
		{
			name:     "operator session is durable",
			sess:     SessionInfo{ID: "s", Origin: "operator"},
			wantTier: memory.TierDurable,
		},
		{
			name:     "empty origin (pre-0044 rows, tests) is durable",
			sess:     SessionInfo{ID: "s"},
			wantTier: memory.TierDurable,
		},
		{
			name:     "cli session is durable",
			sess:     SessionInfo{ID: "s", Origin: "cli"},
			wantTier: memory.TierDurable,
		},
		{
			name:       "integration with nil resolver quarantines (safe default)",
			sess:       SessionInfo{ID: "s", Origin: "integration", IntegrationID: "i1"},
			wantTier:   memory.TierQuarantine,
			wantExpiry: true,
		},
		{
			name:     "integration policy none skips capture entirely",
			sess:     SessionInfo{ID: "s", Origin: "integration", IntegrationID: "i1"},
			policy:   &fakePolicy{policy: "none"},
			wantSkip: true,
		},
		{
			name:     "integration policy full is durable",
			sess:     SessionInfo{ID: "s", Origin: "integration", IntegrationID: "i1"},
			policy:   &fakePolicy{policy: "full"},
			wantTier: memory.TierDurable,
		},
		{
			name:       "integration policy quarantine quarantines with TTL",
			sess:       SessionInfo{ID: "s", Origin: "integration", IntegrationID: "i1"},
			policy:     &fakePolicy{policy: "quarantine"},
			wantTier:   memory.TierQuarantine,
			wantExpiry: true,
		},
		{
			name:       "resolver error degrades to quarantine",
			sess:       SessionInfo{ID: "s", Origin: "integration", IntegrationID: "i1"},
			policy:     &fakePolicy{err: errors.New("db down")},
			wantTier:   memory.TierQuarantine,
			wantExpiry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &runner{log: newSilentLogger(), policy: tt.policy}
			tier, expiry, skip := r.routeForSession(context.Background(), tt.sess)
			if skip != tt.wantSkip {
				t.Fatalf("skip = %v, want %v", skip, tt.wantSkip)
			}
			if skip {
				return
			}
			if tier != tt.wantTier {
				t.Errorf("tier = %q, want %q", tier, tt.wantTier)
			}
			if (expiry != nil) != tt.wantExpiry {
				t.Errorf("expiry = %v, wantExpiry %v", expiry, tt.wantExpiry)
			}
			if expiry != nil && time.Until(*expiry) <= 0 {
				t.Errorf("expiry %v is not in the future", *expiry)
			}
		})
	}
}

// TestIntegrationSessionQuarantinesFacts drives the FULL capture path
// (trigger → summarize → store) via runForceForSession with the
// worker-provider default, proving captured facts land in quarantine
// with TTL + integration attribution.
func TestIntegrationSessionQuarantinesFacts(t *testing.T) {
	mem := &fakeMemory{}
	prov := &fakeProvider{
		name: "fake", kind: "anthropic",
		res: summarizer.SummarizeResult{
			Facts: []summarizer.Fact{{Text: "third-party fact", Confidence: 0.9, Category: summarizer.CategoryOther}},
		},
	}
	r := &runner{
		memory:         mem,
		workerProvider: prov,
		callLog:        &fakeCallLog{},
		state:          newStateMap(),
		historyLimit:   100,
		log:            newSilentLogger(),
		policy:         &fakePolicy{policy: "quarantine"},
		quarantineTTL:  time.Hour,
	}
	r.history = mockHistory{entries: []TranscriptEntry{{Ts: time.Now(), Text: "hello"}}}
	rule := Rule{
		ID: "rule-q", Enabled: true,
		TriggerKind: "after_messages", TriggerConfig: map[string]any{"n": float64(1)},
		DedupThreshold: 0, TargetScope: "project",
	}
	sess := SessionInfo{ID: "s-int", ProviderID: "claude", Cwd: "/x",
		Origin: "integration", IntegrationID: "intg-42"}

	r.runForceForSession(context.Background(), rule, sess)

	if len(mem.storeCalls) != 1 {
		t.Fatalf("store calls = %d, want 1", len(mem.storeCalls))
	}
	got := mem.storeCalls[0]
	if got.Tier != memory.TierQuarantine {
		t.Errorf("tier = %q, want quarantine", got.Tier)
	}
	if got.QuarantineExpiresAt == nil {
		t.Errorf("quarantine expiry not set")
	}
	if got.Metadata["integration_id"] != "intg-42" {
		t.Errorf("metadata integration_id = %v, want intg-42", got.Metadata["integration_id"])
	}
}

// TestPolicyNoneSkipsSessionEntirely proves a policy=none session
// never reads history, never calls the summarizer, never stores.
func TestPolicyNoneSkipsSessionEntirely(t *testing.T) {
	mem := &fakeMemory{}
	prov := &fakeProvider{
		res: summarizer.SummarizeResult{
			Facts: []summarizer.Fact{{Text: "should never be stored", Confidence: 0.9}},
		},
	}
	r := &runner{
		memory:         mem,
		workerProvider: prov,
		callLog:        &fakeCallLog{},
		state:          newStateMap(),
		historyLimit:   100,
		log:            newSilentLogger(),
		policy:         &fakePolicy{policy: "none"},
	}
	r.history = mockHistory{entries: []TranscriptEntry{{Ts: time.Now(), Text: "hello"}}}
	rule := Rule{
		ID: "rule-none", Enabled: true,
		TriggerKind: "after_messages", TriggerConfig: map[string]any{"n": float64(1)},
		TargetScope: "project",
	}
	sess := SessionInfo{ID: "s-none", Cwd: "/x", Origin: "integration", IntegrationID: "intg-9"}

	r.runForceForSession(context.Background(), rule, sess)

	if prov.calls != 0 {
		t.Errorf("summarizer called %d times, want 0", prov.calls)
	}
	if len(mem.storeCalls) != 0 {
		t.Errorf("store calls = %d, want 0", len(mem.storeCalls))
	}
}

// TestOperatorSessionStaysDurable pins the no-regression case: the
// operator's own sessions keep writing durable memory.
func TestOperatorSessionStaysDurable(t *testing.T) {
	mem := &fakeMemory{}
	prov := &fakeProvider{
		name: "fake", kind: "anthropic",
		res: summarizer.SummarizeResult{
			Facts: []summarizer.Fact{{Text: "operator fact", Confidence: 0.9}},
		},
	}
	r := &runner{
		memory:         mem,
		workerProvider: prov,
		callLog:        &fakeCallLog{},
		state:          newStateMap(),
		historyLimit:   100,
		log:            newSilentLogger(),
		policy:         &fakePolicy{policy: "quarantine"}, // must not even be consulted
	}
	r.history = mockHistory{entries: []TranscriptEntry{{Ts: time.Now(), Text: "hello"}}}
	rule := Rule{
		ID: "rule-op", Enabled: true,
		TriggerKind: "after_messages", TriggerConfig: map[string]any{"n": float64(1)},
		TargetScope: "project",
	}
	sess := SessionInfo{ID: "s-op", Cwd: "/x", Origin: "operator"}

	r.runForceForSession(context.Background(), rule, sess)

	if len(mem.storeCalls) != 1 {
		t.Fatalf("store calls = %d, want 1", len(mem.storeCalls))
	}
	if got := mem.storeCalls[0].Tier; got != memory.TierDurable {
		t.Errorf("tier = %q, want durable", got)
	}
	if mem.storeCalls[0].QuarantineExpiresAt != nil {
		t.Errorf("operator fact must not carry a quarantine expiry")
	}
	if fp := r.policy.(*fakePolicy); fp.calls != 0 {
		t.Errorf("policy resolver consulted %d times for an operator session, want 0", fp.calls)
	}
}
