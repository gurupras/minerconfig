package gpureset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResetGPU(t *testing.T) {
	require := require.New(t)

	instanceID := `PCI\VEN_1002&DEV_687F&SUBSYS_0B361002&REV_C1\6&17A7B5E1&0&00000008`
	err := ResetGPU(instanceID)
	require.Nil(err)
}
