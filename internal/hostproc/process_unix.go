//go:build !windows

package hostproc

import (
	"os/exec"
	"syscall"
)

func Detach(cmd *exec.Cmd)      { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func Writable(path string) bool { return syscall.Access(path, 2) == nil }
