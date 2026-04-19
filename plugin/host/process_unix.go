//go:build unix

package host

import (
	"os/exec"
	"syscall"
)

// configureProcAttrs applies Setpgid=true so the sidecar and any
// children share a process group. Kill-group on the sidecar then tears
// down the whole subtree without leaving orphans.
func configureProcAttrs(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGKILL to the whole group. Called after the
// graceful stdin-close + timeout elapsed without the process exiting.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil || pgid <= 0 {
		// Fall back to single-process kill.
		_ = cmd.Process.Kill()
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}
