//go:build !linux

package app

// canMapExecutable is a no-op off Linux: MemoryDenyWriteExecute is a
// systemd/Linux concern, and macOS (launchd) has no equivalent that
// would block the spawned CLIs' JIT.
func canMapExecutable() error { return nil }
