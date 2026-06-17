package session

import (
	"testing"

	"github.com/opendray/opendray-v2/internal/integration"
)

// visibleSessions hides third-party integration sessions from the operator
// console and scopes an integration token to its own sessions.

func sample() []Session {
	return []Session{
		{ID: "op1", Origin: OriginOperator},
		{ID: "cli1", Origin: OriginCLI},
		{ID: "intA", Origin: OriginIntegration, IntegrationID: "int_A"},
		{ID: "intB", Origin: OriginIntegration, IntegrationID: "int_B"},
	}
}

func ids(list []Session) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.ID
	}
	return out
}

func TestVisibleSessions_OperatorHidesIntegration(t *testing.T) {
	got := ids(visibleSessions(integration.Principal{Kind: integration.KindAdmin, ID: "admin"}, true, sample()))
	want := []string{"op1", "cli1"}
	if len(got) != len(want) {
		t.Fatalf("operator visible = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("operator visible = %v, want %v", got, want)
		}
	}
}

func TestVisibleSessions_NoPrincipalTreatedAsOperator(t *testing.T) {
	got := ids(visibleSessions(integration.Principal{}, false, sample()))
	for _, id := range got {
		if id == "intA" || id == "intB" {
			t.Fatalf("no-principal must not expose integration sessions, got %v", got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("expected only operator/cli sessions, got %v", got)
	}
}

func TestVisibleSessions_IntegrationSeesOnlyOwn(t *testing.T) {
	got := ids(visibleSessions(integration.Principal{Kind: integration.KindIntegration, ID: "int_A"}, true, sample()))
	if len(got) != 1 || got[0] != "intA" {
		t.Fatalf("integration int_A visible = %v, want [intA]", got)
	}
}

func TestVisibleSessions_IntegrationWithNoOwnSessions(t *testing.T) {
	got := visibleSessions(integration.Principal{Kind: integration.KindIntegration, ID: "int_Z"}, true, sample())
	if len(got) != 0 {
		t.Fatalf("integration with no own sessions must see none, got %v", ids(got))
	}
}
