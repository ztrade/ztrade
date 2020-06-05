package common

import (
	"os/exec"
	"runtime"
)

func OpenURL(strURL string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, strURL)
	return exec.Command(cmd, args...).Start()
}
