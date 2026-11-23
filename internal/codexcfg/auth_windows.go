package codexcfg

import (
	"os"
	"path/filepath"
	"switcher/internal/winfs"
)

func openAuthFile(path string, flags int, mode os.FileMode) (*os.File, error) {
	root, err := winfs.OpenRoot(filepath.Dir(path), false, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return winfs.Open(root, filepath.Base(path), flags, mode)
}
