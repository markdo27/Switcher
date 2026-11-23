package reexec

import (
	"errors"
	"net"
)

func Keep(net.Listener) {}
func BeforeExec(func()) {}
func Exec(string, []string, []string) error {
	return errors.New("automatic restart is not available in the Windows preview; restart Switcher manually")
}
func ExecSame(p string, a, e []string) error { return Exec(p, a, e) }
func Inherited(string) net.Listener          { return nil }
