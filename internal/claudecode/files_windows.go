package claudecode

import "errors"

type fileValue struct {
	Data   []byte `json:"data"`
	Exists bool   `json:"exists"`
}

const nativeFileLimit = 2 << 20
const recoveryFileLimit = 32 << 20

var errWindowsNative = errors.New("native Claude credential switching is not available in the Windows preview; proxy accounts remain separate")

func safeDirectory(string, bool) error       { return errWindowsNative }
func syncPrivateDirectory(string) error      { return errWindowsNative }
func readPrivate(string) (fileValue, error)  { return fileValue{}, errWindowsNative }
func readRecovery(string) (fileValue, error) { return fileValue{}, errWindowsNative }
func writePrivate(string, fileValue) error   { return errWindowsNative }
func writeRecovery(string, fileValue) error  { return errWindowsNative }
