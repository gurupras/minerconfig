package gpureset

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	easyfiles "github.com/gurupras/go-easyfiles"
	log "github.com/sirupsen/logrus"
)

func ResetGPU(arg string) error {
	if runtime.GOOS == "windows" {
		// This is expected to be a device instance ID
		instanceID := arg
		tmpBatchFile, err := easyfiles.TempFile(os.TempDir(), "gpu-reset", ".bat")
		if err != nil {
			return err
		}
		fileContents := fmt.Sprintf(`
powershell.exe -NoProfile -ExecutionPolicy Bypass "disable-pnpdevice '%v' -ErrorAction Ignore -Confirm:$false"
powershell.exe -NoProfile -ExecutionPolicy Bypass "start-sleep -s 2"
powershell.exe -NoProfile -ExecutionPolicy Bypass "enable-pnpdevice '%v' -ErrorAction Ignore -Confirm:$false"
    `, instanceID, instanceID)
		tmpBatchPath := tmpBatchFile.Name()
		tmpBatchFile.Close()
		defer os.Remove(tmpBatchPath)
		if err := ioutil.WriteFile(tmpBatchPath, []byte(fileContents), 0666); err != nil {
			return err
		}
		log.Infof("cmdline: %v", tmpBatchPath)
		cmd := exec.Command(tmpBatchPath)
		return cmd.Run()
	} else {
		return fmt.Errorf("Unimplemented")
	}
}
