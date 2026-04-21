//go:build unix && !linux

package bridge

import (
	"log/slog"
	"sync"
	"syscall"
)

// applyNetNS on non-Linux unix (Darwin, *BSD): CLONE_NEWNET is a Linux
// concept, so we log once and skip.
func applyNetNS(_ *syscall.SysProcAttr, isolateNetNS bool, mu *sync.Mutex, warned *bool, log *slog.Logger) {
	if !isolateNetNS {
		return
	}
	mu.Lock()
	alreadyWarned := *warned
	*warned = true
	mu.Unlock()
	if !alreadyWarned && log != nil {
		log.Warn("exec: IsolateNetNS requested but CLONE_NEWNET is Linux-only — ignoring")
	}
}
