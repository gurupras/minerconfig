package gpureset

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"

	easyfiles "github.com/gurupras/go-easyfiles"
	log "github.com/sirupsen/logrus"
)

func ResetGPU(ids []string) error {
	if runtime.GOOS == "windows" {
		tmpBatchFile, err := easyfiles.TempFile(os.TempDir(), "gpu-reset", ".bat")
		if err != nil {
			return err
		}
		fileContents := make([]string, 0)
		for _, instanceID := range ids {
			// This is expected to be a device instance ID
			fileContents = append(fileContents, fmt.Sprintf(`
powershell.exe -NoProfile -ExecutionPolicy Bypass "disable-pnpdevice '%v' -ErrorAction Ignore -Confirm:$false"
	    `, instanceID))
		}
		fileContents = append(fileContents, `powershell.exe -NoProfile -ExecutionPolicy Bypass "start-sleep -s 2"`)
		for _, instanceID := range ids {
			fileContents = append(fileContents, fmt.Sprintf(`
powershell.exe -NoProfile -ExecutionPolicy Bypass "enable-pnpdevice '%v' -ErrorAction Ignore -Confirm:$false"
		`, instanceID))
		}
		fileContents = append(fileContents, `powershell.exe -NoProfile -ExecutionPolicy Bypass "start-sleep -s 2"`)
		tmpBatchPath := tmpBatchFile.Name()
		tmpBatchFile.Close()
		defer os.Remove(tmpBatchPath)
		fileContentsStr := strings.Join(fileContents, "\n") // FIXME: This should be cross-platform
		if err := ioutil.WriteFile(tmpBatchPath, []byte(fileContentsStr), 0666); err != nil {
			return err
		}
		log.Infof("cmdline: %v", tmpBatchPath)
		cmd := exec.Command(tmpBatchPath)
		return cmd.Run()
	} else {
		return fmt.Errorf("Unimplemented")
	}
}
