package gputool

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

type MSIAfterBurner struct {
}

func (ab *MSIAfterBurner) Run(path string, args map[string]interface{}) error {
	profile := args["profile"]

	cmdArgs := fmt.Sprintf("-Profile%v", profile)
	cmd := exec.Command(path, []string{cmdArgs}...)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}
