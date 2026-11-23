package main

import (
	"fmt"
	"os"
)

func runTray(int) int {
	fmt.Fprintln(os.Stderr, "The Windows preview uses the browser dashboard; a tray icon is not available yet.")
	return 1
}
