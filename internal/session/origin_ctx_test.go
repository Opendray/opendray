package session

import (
	"context"
	"testing"
)

// WithOrigin/OriginFromContext thread the session provenance to the
// catalog adapter so integration sessions can be denied the auto-attached
// cross-project memory MCP. See provider.go and adapter.go.

func TestWithOrigin_RoundTrip(t *testing.T) {
	ctx := WithOrigin(context.Background(), OriginIntegration)
	if got := OriginFromContext(ctx); got != OriginIntegration {
		t.Fatalf("OriginFromContext = %q, want %q", got, OriginIntegration)
	}
}

func TestWithOrigin_EmptyIsNoOp(t *testing.T) {
	ctx := WithOrigin(context.Background(), "")
	if got := OriginFromContext(ctx); got != "" {
		t.Fatalf("empty origin must be a no-op, got %q", got)
	}
}

func TestOriginFromContext_DefaultEmpty(t *testing.T) {
	if got := OriginFromContext(context.Background()); got != "" {
		t.Fatalf("unset origin must return empty, got %q", got)
	}
}

func TestWithOrigin_OperatorAndCLI(t *testing.T) {
	for _, o := range []Origin{OriginOperator, OriginCLI} {
		ctx := WithOrigin(context.Background(), o)
		if got := OriginFromContext(ctx); got != o {
			t.Fatalf("OriginFromContext = %q, want %q", got, o)
		}
	}
}
