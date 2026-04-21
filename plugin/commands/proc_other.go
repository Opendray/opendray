//go:build !unix

package commands

import "os/exec"

// setProcAttr is a no-op on non-Unix platforms.
func setProcAttr(_ *exec.Cmd) {}

// killProcGroup kills the process directly on non-Unix platforms.
func killProcGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
