package gputool

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/google/shlex"
)

type OverdriveNTool struct {
}

func (odnt *OverdriveNTool) Run(path string, args map[string]interface{}) error {
	gpuID := args["gpu-id"]
	profile := args["profile"]
	cmdArgs, err := shlex.Split(fmt.Sprintf("-consoleonly -p%v%v", gpuID, profile))
	if err != nil {
		return err
	}
	cmd := exec.Command(path, cmdArgs...)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}
