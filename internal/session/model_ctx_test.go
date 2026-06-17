package session

import (
	"context"
	"testing"
)

// WithModel/ModelFromContext thread a session's pinned model to the
// catalog adapter so it renders the provider's model flag. See
// provider.go and adapter.go.

func TestWithModel_RoundTrip(t *testing.T) {
	ctx := WithModel(context.Background(), "opus")
	if got := ModelFromContext(ctx); got != "opus" {
		t.Fatalf("ModelFromContext = %q, want %q", got, "opus")
	}
}

func TestWithModel_EmptyIsNoOp(t *testing.T) {
	ctx := WithModel(context.Background(), "")
	if got := ModelFromContext(ctx); got != "" {
		t.Fatalf("empty model must be a no-op, got %q", got)
	}
}

func TestModelFromContext_DefaultEmpty(t *testing.T) {
	if got := ModelFromContext(context.Background()); got != "" {
		t.Fatalf("unset model must return empty, got %q", got)
	}
}
