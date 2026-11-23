package codexcfg

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"switcher/internal/winfs"
)

type configFiles struct {
	root       *os.Root
	lock       *os.File
	base, path string
}
type expectedFile struct {
	digest string
	exists bool
}

func canonicalSystemAlias(path string) string { return path }
func openConfigFiles(path string, create, locked bool) (*configFiles, error) {
	if !filepath.IsAbs(path) || filepath.Base(path) != "config.toml" {
		return nil, ErrConfigConflict
	}
	_, before := os.Stat(filepath.Dir(path))
	root, err := winfs.OpenRoot(filepath.Dir(path), create, create && os.IsNotExist(before))
	if err != nil {
		return nil, err
	}
	f := &configFiles{root: root, base: filepath.Base(path), path: path}
	if locked {
		f.lock, err = winfs.Open(root, ".switcher.lock", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if os.IsExist(err) {
			f.lock, err = winfs.Open(root, ".switcher.lock", os.O_RDWR, 0)
		}
		if err == nil {
			err = windows.LockFileEx(windows.Handle(f.lock.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &windows.Overlapped{})
		}
		if err != nil {
			f.close()
			return nil, err
		}
	}
	return f, nil
}
func (f *configFiles) close() {
	if f.lock != nil {
		windows.UnlockFileEx(windows.Handle(f.lock.Fd()), 0, 1, 0, &windows.Overlapped{})
		f.lock.Close()
	}
	f.root.Close()
}
func (f *configFiles) read(name string, private bool) ([]byte, bool, error) {
	raw, err := winfs.Read(f.root, name, 4<<20)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return raw, err == nil, err
}
func (f *configFiles) temp(data []byte) (string, error) { return winfs.Temp(f.root, data) }
func (f *configFiles) replace(name string, data []byte, expected ...expectedFile) error {
	if _, exists, err := f.read(name, name != f.base); err != nil {
		return err
	} else if exists && name == ".switcher.lock" {
		return fmt.Errorf("%w: lock cannot be replaced", ErrConfigConflict)
	}
	tmp, err := f.temp(data)
	if err != nil {
		return err
	}
	defer f.root.Remove(tmp)
	if name == f.base {
		if err := codexFault("before_rename"); err != nil {
			return err
		}
	}
	if len(expected) > 0 {
		current, present, err := f.read(name, name != f.base)
		if err != nil || present != expected[0].exists || digest(current) != expected[0].digest {
			return fmt.Errorf("%w: %s changed before rename", ErrConfigConflict, name)
		}
	}
	if err := f.root.Rename(tmp, name); err != nil {
		return err
	}
	return nil
}

func (f *configFiles) remove(name string, expected ...expectedFile) error {
	if current, exists, err := f.read(name, name != f.base); err != nil {
		return err
	} else if !exists {
		if len(expected) > 0 && expected[0].exists {
			return fmt.Errorf("%w: %s disappeared", ErrConfigConflict, name)
		}
		return nil
	} else if len(expected) > 0 && (!expected[0].exists || digest(current) != expected[0].digest) {
		return fmt.Errorf("%w: %s changed before removal", ErrConfigConflict, name)
	}
	if len(expected) > 0 {
		if err := codexFault("before_unlink"); err != nil {
			return err
		}
		current, present, err := f.read(name, name != f.base)
		if err != nil || !present || digest(current) != expected[0].digest {
			return fmt.Errorf("%w: %s changed before unlink", ErrConfigConflict, name)
		}
	}
	if err := f.root.Remove(name); err != nil {
		return err
	}
	return nil
}

func (f *configFiles) backupOnce(data []byte) error {
	name := f.base + ".switcher-backup"
	if _, exists, err := f.read(name, true); err != nil || exists {
		return err
	}
	tmp, err := f.temp(data)
	if err != nil {
		return err
	}
	defer f.root.Remove(tmp)
	if err := f.root.Link(tmp, name); err != nil {
		return fmt.Errorf("%w: backup: %v", ErrConfigConflict, err)
	}
	return nil
}
