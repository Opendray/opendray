//go:build !windows

package bridge

import (
	"log/slog"
	"os/exec"
	"sync"
	"syscall"
)

// setProcAttrs applies Setpgid=true so the exec API can signal the
// process group on kill. When isolateNetNS is requested the platform
// branch may attach CLONE_NEWNET — only Linux honours that; other unix
// systems log once and skip.
//
// warnedLatchMu guards warnedLatch (a one-time warning flag stored on the
// ExecAPI instance so we warn per-instance, not per-call).
func setProcAttrs(c *exec.Cmd, isolateNetNS bool, warnedLatchMu *sync.Mutex, warnedLatch *bool, log *slog.Logger) {
	attrs := &syscall.SysProcAttr{Setpgid: true}
	applyNetNS(attrs, isolateNetNS, warnedLatchMu, warnedLatch, log)
	c.SysProcAttr = attrs
}
