package utils

import (
	"os/exec"
)

// OpenURL opens the specified URL in the system default browser
// OS X specific for now
func OpenURL(url string) error {
	cmd := "open"
	args := []string{url}

	return exec.Command(cmd, args...).Start()
}
