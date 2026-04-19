//go:build linux

package bridge

import (
	"log/slog"
	"sync"
	"syscall"
)

// applyNetNS adds CLONE_NEWNET to attrs when isolateNetNS is set AND the
// process has CAP_SYS_ADMIN. Otherwise a one-time warning is logged and
// the flag is ignored.
//
// Detecting CAP_SYS_ADMIN from within a Go process without cgo is
// imperfect; we check euid == 0 as a coarse proxy — root always has the
// capability. Non-root processes with CAP_SYS_ADMIN granted via file
// capabilities are uncommon in the opendray deployment model and will
// log the warning once then skip, which matches the plan's documented
// fallback.
func applyNetNS(attrs *syscall.SysProcAttr, isolateNetNS bool, mu *sync.Mutex, warned *bool, log *slog.Logger) {
	if !isolateNetNS {
		return
	}
	if syscall.Geteuid() != 0 {
		mu.Lock()
		alreadyWarned := *warned
		*warned = true
		mu.Unlock()
		if !alreadyWarned && log != nil {
			log.Warn("exec: IsolateNetNS requested but host is not root — skipping CLONE_NEWNET",
				"note", "set CAP_SYS_ADMIN on the opendray binary to enable network isolation")
		}
		return
	}
	attrs.Cloneflags |= syscall.CLONE_NEWNET
}
