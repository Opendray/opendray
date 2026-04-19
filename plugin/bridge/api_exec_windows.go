//go:build windows

package bridge

import (
	"log/slog"
	"os/exec"
	"sync"
	"syscall"
)

// setProcAttrs on Windows: there is no Setpgid equivalent in a portable
// form. We set CREATE_NEW_PROCESS_GROUP so Console-Ctrl signals can be
// aimed at the child and its descendants. Network-namespace isolation is
// Linux-specific — warn-once and skip.
func setProcAttrs(c *exec.Cmd, isolateNetNS bool, mu *sync.Mutex, warned *bool, log *slog.Logger) {
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	if isolateNetNS {
		mu.Lock()
		alreadyWarned := *warned
		*warned = true
		mu.Unlock()
		if !alreadyWarned && log != nil {
			log.Warn("exec: IsolateNetNS requested on Windows — CLONE_NEWNET is Linux-only, ignoring")
		}
	}
}
