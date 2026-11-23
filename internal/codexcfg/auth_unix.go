//go:build !windows

package codexcfg

import (
	"os"
	"syscall"
)

func openAuthFile(path string, flags int, mode os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flags|syscall.O_NOFOLLOW, mode)
}
