package minerconfig

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestParseHIPGPUThread(t *testing.T) {
	require := require.New(t)

	testConfig := `
threads: 8
blocks: 224
index: 0
device_index: 0`

	index := 0
	expected := GPUThread{
		StandardGPUThread{},
		HIPGPUThread{
			Threads: 8,
			Blocks:  224,
		},
		&index,
		&index,
		false,
	}

	var got GPUThread
	err := yaml.Unmarshal([]byte(testConfig), &got)
	if err != nil {
		log.Fatalf("Failed to unmarshal into GPUThread: %v", err)
	}
	require.Equal(expected, got)
}
