package settings

import (
	"os"
	"path/filepath"
	"switcher/internal/winfs"
)

func ensurePrivateDirectory(path string) error {
	root, err := winfs.OpenRoot(path, true, true)
	if err == nil {
		root.Close()
	}
	return err
}
func readPrivateFile(path string) ([]byte, error) {
	root, err := winfs.OpenRoot(filepath.Dir(path), false, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return winfs.Read(root, filepath.Base(path), 4<<20)
}
func writeAtomic(path string, raw []byte) error {
	root, err := winfs.OpenRoot(filepath.Dir(path), false, false)
	if err != nil {
		return err
	}
	defer root.Close()
	return winfs.Atomic(root, filepath.Base(path), raw)
}
func removePrivateFile(path string) error {
	root, err := winfs.OpenRoot(filepath.Dir(path), false, false)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := winfs.Open(root, filepath.Base(path), os.O_RDONLY, 0)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	f.Close()
	return root.Remove(filepath.Base(path))
}
