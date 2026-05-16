package prwatcher

import (
	"testing"

	"github.com/opendray/opendray-v2/internal/githost"
)

func TestAggregate_AllPassing(t *testing.T) {
	checks := []githost.CheckRun{
		{Name: "lint", Status: "completed", Conclusion: "success"},
		{Name: "build", Status: "completed", Conclusion: "success"},
		{Name: "test", Status: "completed", Conclusion: "neutral"},
	}
	got := aggregate(checks)
	if !got.terminal || got.conclusion != "success" {
		t.Errorf("got %+v, want terminal=true conclusion=success", got)
	}
	if got.totalChecks != 3 {
		t.Errorf("totalChecks = %d, want 3", got.totalChecks)
	}
}

func TestAggregate_AnyFailedIsFailure(t *testing.T) {
	checks := []githost.CheckRun{
		{Status: "completed", Conclusion: "success"},
		{Status: "completed", Conclusion: "failure"},
	}
	got := aggregate(checks)
	if !got.terminal || got.conclusion != "mixed" {
		t.Errorf("got %+v, want terminal=true conclusion=mixed (partial pass + failure)", got)
	}
}

func TestAggregate_AllFailedIsFailure(t *testing.T) {
	checks := []githost.CheckRun{
		{Status: "completed", Conclusion: "failure"},
		{Status: "completed", Conclusion: "cancelled"},
	}
	got := aggregate(checks)
	if !got.terminal || got.conclusion != "failure" {
		t.Errorf("got %+v, want terminal=true conclusion=failure", got)
	}
}

func TestAggregate_AnyPendingMeansNotTerminal(t *testing.T) {
	checks := []githost.CheckRun{
		{Status: "completed", Conclusion: "success"},
		{Status: "in_progress"},
	}
	got := aggregate(checks)
	if got.terminal {
		t.Errorf("got terminal=true, want false (one check still running)")
	}
	if got.conclusion != "pending" {
		t.Errorf("conclusion = %q, want pending", got.conclusion)
	}
}

func TestAggregate_EmptyChecks(t *testing.T) {
	if got := aggregate(nil); got.terminal || got.conclusion != "" {
		t.Errorf("nil input: got %+v", got)
	}
}

func TestUniqueLiveCwds_DedupesAndFiltersTerminated(t *testing.T) {
	sessions := []SessionInfo{
		{ID: "1", Cwd: "/repo/a", State: "running"},
		{ID: "2", Cwd: "/repo/a", State: "idle"}, // duplicate cwd
		{ID: "3", Cwd: "/repo/b", State: "running"},
		{ID: "4", Cwd: "/repo/c", State: "ended"},   // filtered: terminated
		{ID: "5", Cwd: "/repo/d", State: "stopped"}, // filtered: terminated
		{ID: "6", Cwd: "", State: "running"},        // filtered: no cwd
		{ID: "7", Cwd: "/repo/e", State: "pending"},
	}
	got := uniqueLiveCwds(sessions)
	// Expect /repo/a, /repo/b, /repo/e in some order; capacity 3.
	if len(got) != 3 {
		t.Fatalf("got %d cwds, want 3: %v", len(got), got)
	}
	want := map[string]bool{"/repo/a": true, "/repo/b": true, "/repo/e": true}
	for _, c := range got {
		if !want[c] {
			t.Errorf("unexpected cwd %q in result", c)
		}
		delete(want, c)
	}
	if len(want) != 0 {
		t.Errorf("missing expected cwds: %v", want)
	}
}
