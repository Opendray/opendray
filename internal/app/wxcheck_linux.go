//go:build linux

package app

import "syscall"

// canMapExecutable reports whether the process can flip a freshly-written
// (writable) page to executable — the operation V8/Node's JIT performs at
// runtime. codex and antigravity are V8-based and crash with a fatal
// SetPermissions error the moment this is blocked; Claude survives via an
// interpreter-only fallback, which is why "codex/antigravity die, Claude works"
// is the tell-tale signature.
//
// Two things break it, both inherited by the CLIs we spawn:
//   - systemd MemoryDenyWriteExecute=true (W^X seccomp filter → EPERM)
//   - an exhausted vm.max_map_count or a tight memory cgroup (→ ENOMEM)
//
// The opendray daemon itself is AOT-compiled Go and never needs this, so it
// runs fine under the restriction and can detect+warn on behalf of the
// children it spawns.
func canMapExecutable() error {
	const size = 4096
	p, err := syscall.Mmap(-1, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS)
	if err != nil {
		return err
	}
	defer func() { _ = syscall.Munmap(p) }()
	p[0] = 0xc3 // x86 RET — make the page look like real code
	return syscall.Mprotect(p, syscall.PROT_READ|syscall.PROT_EXEC)
}
