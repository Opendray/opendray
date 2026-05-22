package app

import "testing"

// In an unrestricted environment (normal CI/dev, no MemoryDenyWriteExecute)
// the probe must succeed. The negative case (W^X active) can't be set up
// from a plain unit test, but asserting the positive case guards against a
// broken probe — e.g. wrong mmap flags — that would false-alarm every
// operator on every start.
func TestCanMapExecutable(t *testing.T) {
	if err := canMapExecutable(); err != nil {
		t.Fatalf("canMapExecutable() = %v; want nil in an unrestricted environment", err)
	}
}
