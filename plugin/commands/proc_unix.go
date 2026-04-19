//go:build unix

package commands

import (
	"os/exec"
	"syscall"
)

// setProcAttr sets the process group attribute on the command so that
// when we kill it via the process group we also take down any child
// processes the shell may have spawned. This prevents zombie children.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcGroup sends SIGKILL to the entire process group of cmd.
// cmd.Process must be non-nil (the process was started successfully).
func killProcGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// Negative PID targets the whole process group.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
