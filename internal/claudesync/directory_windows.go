package claudesync

import (
	"errors"
	"os"
)

func openDirectory(string, bool) (*os.Root, error) {
	return nil, errors.New("Claude session synchronization is not available in the Windows preview")
}
