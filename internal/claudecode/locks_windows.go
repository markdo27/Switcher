package claudecode

import (
	"context"
	"errors"
)

var ErrConflict = errors.New("Claude native credentials changed during switching")
var ErrNativeOwned = errors.New("Claude Code manages this login")

func checkLocks(ctx context.Context) error                 { return ctx.Err() }
func writeLocked(context.Context, string, fileValue) error { return errWindowsNative }
func (m *Manager) locks(context.Context) (context.Context, func(), error) {
	return nil, nil, errWindowsNative
}
