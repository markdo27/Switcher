package store

import (
	"os"
	"path/filepath"
	"strings"
	"switcher/internal/winfs"
)

const cacheAccounts = false

func readStoreFile(path string) ([]byte, error) {
	root, err := winfs.OpenRoot(filepath.Dir(path), false, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return winfs.Read(root, filepath.Base(path), 32<<20)
}
func writeAtomic(path string, raw []byte, _ os.FileMode) error {
	root, err := winfs.OpenRoot(filepath.Dir(path), false, false)
	if err != nil {
		return err
	}
	defer root.Close()
	return winfs.Atomic(root, filepath.Base(path), raw)
}

func prepareStoreRoot(path string) error {
	r, e := winfs.OpenRoot(path, true, true)
	if e == nil {
		r.Close()
	}
	return e
}

func storeTemporaryName(name string) bool {
	return strings.Contains(name, ".json") || strings.HasPrefix(name, ".switcher-")
}
