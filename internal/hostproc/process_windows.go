package hostproc

import (
	"os"
	"os/exec"
	"syscall"
)

func Detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200, HideWindow: true}
}
func Writable(path string) bool {
	f, err := os.CreateTemp(path, ".switcher-write-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	return os.Remove(name) == nil
}
