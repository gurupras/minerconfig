package gputool

import (
	"os/exec"
	"path/filepath"
)

type ScriptTool struct {
}

func (script *ScriptTool) Run(path string, args map[string]interface{}) error {
	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}
