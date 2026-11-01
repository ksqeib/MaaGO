package app

import "os/exec"

func openPath(path string) error {
	cmd := exec.Command("explorer.exe", path)
	return cmd.Start()
}
