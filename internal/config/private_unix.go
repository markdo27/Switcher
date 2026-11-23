//go:build !windows

package config

func protectDataDirectory(string) error { return nil }
