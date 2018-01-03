package main

import (
	"os"
	"os/exec"
)

func main() {
	cmd := exec.Command("./dummyminer")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	cmd.Wait()
}
