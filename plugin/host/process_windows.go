//go:build windows

package host

import (
	"os/exec"
	"syscall"
)

// configureProcAttrs sets CREATE_NEW_PROCESS_GROUP so the sidecar is
// isolated from the host's console process group. Tree-kill uses
// taskkill /T (shelled out in killProcessGroup) to reap descendants.
func configureProcAttrs(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= 0x00000200 // CREATE_NEW_PROCESS_GROUP
}

// killProcessGroup shells out to taskkill /T /F /PID <n> which walks
// the process tree. Best-effort — failures are logged by the caller
// via the already-closed exited channel (the process is being torn
// down regardless).
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = exec.Command("taskkill", "/T", "/F", "/PID", itoa(cmd.Process.Pid)).Run()
}

// itoa avoids pulling strconv into this tiny file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
