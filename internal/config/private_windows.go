package config

import "switcher/internal/winfs"

func protectDataDirectory(path string) error {
	r, e := winfs.OpenRoot(path, true, true)
	if e == nil {
		r.Close()
	}
	return e
}
