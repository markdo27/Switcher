//go:build !windows

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const cacheAccounts = true

func readStoreFile(path string) ([]byte, error) { return os.ReadFile(path) }
func writeAtomic(path string, raw []byte, mode os.FileMode) error {
	name := filepath.Base(path)
	f, err := os.CreateTemp(filepath.Dir(path), name+".*.tmp")
	if err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	tmp := f.Name()
	fail := func(step string, err error) error {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("%s %s: %w", step, name, err)
	}
	if err := f.Chmod(mode); err != nil {
		return fail("write", err)
	}
	if _, err := f.Write(raw); err != nil {
		return fail("write", err)
	}
	// Sync before rename so a crash never leaves a truncated file behind.
	if err := f.Sync(); err != nil {
		return fail("sync", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close %s: %w", name, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s: %w", name, err)
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func prepareStoreRoot(string) error { return nil }

func storeTemporaryName(name string) bool { return strings.Contains(name, ".json") }
