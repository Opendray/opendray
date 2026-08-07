//go:build !darwin

package keepawake

import (
	"context"
	"log/slog"
	"os/exec"
)

// newInhibitCmd is deliberately nil off macOS: a Linux or BSD host
// running a gateway does not idle-suspend under any default server
// install, so there is nothing to inhibit. Keeper.Run sees the nil and
// returns immediately.
var newInhibitCmd func(ctx context.Context, mode Mode) *exec.Cmd

// wakePreflight has nothing to verify off macOS.
var wakePreflight func(ctx context.Context, log *slog.Logger)
