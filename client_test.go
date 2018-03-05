package minerconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/gurupras/go-easyfiles"
	"github.com/homesound/simple-websockets"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var testConfig string = `
{
    "algo": "cryptonight",
    "background": false,
    "colors": true,
    "donate-level": 0,
    "log-file": null,
    "print-time": 60,
    "retries": 5,
    "retry-pause": 5,
    "syslog": false,
    "pools": [
		{
            "url": "mine.sumo.fairpool.xyz:5555",
			"user": "Sumoo3U9dFo2CtvGknjrupdw3p2FHqnhJDdqFeErUJLq2zPRMu2sdp1ZqHVooBpmYo9Co1f3xphLZ6jjX5XSuyW3PRMqMERhvuR",
            "pass": "x",
            "keepalive": true,
            "nicehash": false
        },
		{
            "url": "pool.sumokoin.hashvault.pro:5555",
			"user": "Sumoo3U9dFo2CtvGknjrupdw3p2FHqnhJDdqFeErUJLq2zPRMu2sdp1ZqHVooBpmYo9Co1f3xphLZ6jjX5XSuyW3PRMqMERhvuR",
            "pass": "x",
            "keepalive": true,
            "nicehash": false
        },
        {
            "url": "pool.minexmr.com:5555",
			"user": "42ZSsWcvppV1Ux3XDPvBoZ3Lt1MZdfG9tFAuvVwTsceFZYcYovTPAQLgi9c6ectzqiUSimgwkBscQA4RDfpgE5ieLQjYjAv.dileant",
            "pass": "x",
            "keepalive": true,
            "nicehash": false
        },
		{
            "url": "bytecoin.uk:7777",
			"user": "22vY5Fs6tFx2ubP8xosDiBfozkZTXZ4BFL2GQKAojrbLL45ruYiqqKBfNCezqRpKfLJf5dmANoy6uA2bGtZ3uT5fJNdo3G7",
            "pass": "x",
            "keepalive": true,
            "nicehash": false
        },
        {
            "url": "mine.moneropool.com:3333",
			"user": "42ZSsWcvppV1Ux3XDPvBoZ3Lt1MZdfG9tFAuvVwTsceFZYcYovTPAQLgi9c6ectzqiUSimgwkBscQA4RDfpgE5ieLQjYjAv.dileant",
            "pass": "x",
            "keepalive": true,
            "nicehash": false
        }
    ]
}
`

func generateConfigFile() (string, error) {
	tmpFile, err := easyfiles.TempFile(os.TempDir(), "tmpConfig", ".json")
	if err != nil {
		return "", err
	} else {
		tmpFile.Close()
		if err := ioutil.WriteFile(tmpFile.Name(), []byte(testConfig), 0666); err != nil {
			return "", err
		}
		return tmpFile.Name(), nil
	}
}

func checkJson(require *require.Assertions, expected interface{}, got interface{}) {
	switch expected.(type) {
	case string:
		require.Fail("Expected non-string interface")
	}
	switch got.(type) {
	case string:
		require.Fail("Expected non-string interface")
	}

	eb, err := json.Marshal(expected)
	require.Nil(err)
	expectedStr := string(eb)

	gb, err := json.Marshal(got)
	require.Nil(err)
	gotStr := string(gb)

	require.Nil(err)
	require.Equal(expectedStr, gotStr, fmt.Sprintf("expected: %v\ngot: %v\n", expectedStr, gotStr))
}

func generateValidClientConfig(require *require.Assertions) *ClientConfig {
	tmpConfig, err := generateConfigFile()
	require.Nil(err)
	binaryPath := tmpConfig

	return &ClientConfig{
		BinaryPath:       binaryPath,
		MinerConfigPath:  tmpConfig,
		WebserverAddress: "localhost:61118",
	}
}

func TestParseClientConfig(t *testing.T) {
	require := require.New(t)

	configStr := `
binary_path: test
binary_args: ["hello", 1, 4]
binary_is_script: false
miner_config_path: test
webserver_address: google.com
`

	var clientConfig ClientConfig

	err := yaml.Unmarshal([]byte(configStr), &clientConfig)
	require.Nil(err)
	require.Equal("test", clientConfig.BinaryPath)
	require.Equal([]interface{}{"hello", 1, 4}, clientConfig.BinaryArgs)
	require.False(clientConfig.BinaryIsScript)
	require.Equal("test", clientConfig.MinerConfigPath)
	require.Nil(clientConfig.MinerConfig)
	require.Equal("google.com", clientConfig.WebserverAddress)
}

func TestBadBinaryPath(t *testing.T) {
	require := require.New(t)
	bp := "/tmp/123414"

	clientConfig := generateValidClientConfig(require)
	defer os.Remove(clientConfig.MinerConfigPath)

	// Mess up the binary path
	clientConfig.BinaryPath = bp

	c, err := NewClient(clientConfig)
	require.Nil(c, "Client should have been nil")
	require.NotNil(err, "Expected error")
}

func TestBadConfigPath(t *testing.T) {
	require := require.New(t)

	clientConfig := generateValidClientConfig(require)
	// Remove the file immediately thereby messing up MinerConfigPath
	defer os.Remove(clientConfig.MinerConfigPath)

	clientConfig.MinerConfigPath = "/badpath"

	c, err := NewClient(clientConfig)
	require.Nil(c, "Client should have been nil")
	require.NotNil(err, "Expected error")
	fmt.Printf("Error: %v\n", err)
}

func TestBadWebserverAddress(t *testing.T) {
	require := require.New(t)

	clientConfig := generateValidClientConfig(require)
	defer os.Remove(clientConfig.MinerConfigPath)

	c, err := NewClient(clientConfig)
	require.Nil(c, "Client should have been nil")
	require.NotNil(err, "Expected error")
	log.Debugf("Error: %v", err)
}

func TestRunServer(t *testing.T) {
	snl := RunServer("webserver/www", 61118)
	snl.Stop()
}

func TestConnect(t *testing.T) {
	require := require.New(t)

	clientConfig := generateValidClientConfig(require)
	defer os.Remove(clientConfig.MinerConfigPath)

	// Start webserver
	snl := RunServer("webserver/www", 61118)
	defer snl.Stop()
	time.Sleep(300 * time.Millisecond)
	c, err := NewClient(clientConfig)
	require.Nil(err, "Unexpected error", err)
	require.NotNil(c, "Client should not be nil")
	log.Debugf("Error: %v", err)
}

func TestAddPool(t *testing.T) {
	require := require.New(t)

	clientConfig := generateValidClientConfig(require)
	defer os.Remove(clientConfig.MinerConfigPath)

	// Start webserver
	snl := RunServer("webserver/www", 61118)
	defer snl.Stop()
	time.Sleep(300 * time.Millisecond)
	c, err := NewClient(clientConfig)
	require.Nil(err, "Unexpected error", err)
	require.NotNil(c, "Client should not be nil")
	log.Debugf("Error: %v", err)

	str := `
{
  "url": "mine.sumo.fairpool.xyz:5555",
  "user": "Sumoo3U9dFo2CtvGknjrupdw3p2FHqnhJDdqFeErUJLq2zPRMu2sdp1ZqHVooBpmYo9Co1f3xphLZ6jjX5XSuyW3PRMqMERhvuR",
  "pass": "x",
  "keepalive": true,
  "nicehash": false
}`

	wg := sync.WaitGroup{}
	wg.Add(1)

	var got interface{}

	testAvailablePools := func(w *websockets.WebsocketClient, data interface{}) {
		defer wg.Done()
		got = data
	}

	var expected interface{}
	err = json.Unmarshal([]byte(str), &expected)
	require.Nil(err)

	go c.ProcessMessages()
	c.On("new-pool", testAvailablePools)
	c.Emit("add-pool", str)
	c.Emit("get-available-pools", nil)

	wg.Wait()

	checkJson(require, expected, got)
}

func TestSetPool(t *testing.T) {
	require := require.New(t)

	clientConfig := generateValidClientConfig(require)
	defer os.Remove(clientConfig.MinerConfigPath)

	// Start webserver
	snl := RunServer("webserver/www", 61118)
	defer snl.Stop()
	time.Sleep(300 * time.Millisecond)
	c, err := NewClient(clientConfig)
	require.Nil(err, "Unexpected error", err)
	require.NotNil(c, "Client should not be nil")
	log.Debugf("Error: %v", err)

	str := `
{
  "url": "mine.sumo.fairpool.xyz:5555",
  "user": "Sumoo3U9dFo2CtvGknjrupdw3p2FHqnhJDdqFeErUJLq2zPRMu2sdp1ZqHVooBpmYo9Co1f3xphLZ6jjX5XSuyW3PRMqMERhvuR",
  "pass": "x",
  "keepalive": true,
  "nicehash": false
}`

	wg := sync.WaitGroup{}
	wg.Add(1)

	var got interface{}

	testPools := func(w *websockets.WebsocketClient, data interface{}) {
		defer wg.Done()
		got = data
	}

	var p interface{}
	err = json.Unmarshal([]byte(str), &p)
	require.Nil(err)
	expected := make([]interface{}, 1)
	expected[0] = p

	b, _ := json.Marshal(expected)

	go c.ProcessMessages()
	c.On("update-selected-pools", testPools)
	c.Emit("update-selected-pools", string(b))

	wg.Wait()

	checkJson(require, expected, got)
}

func TestMiner(t *testing.T) {
	t.Skip()
	require := require.New(t)

	clientConfig := generateValidClientConfig(require)
	defer os.Remove(clientConfig.MinerConfigPath)

	// Start webserver
	snl := RunServer("webserver/www", 61118)
	defer snl.Stop()
	time.Sleep(300 * time.Millisecond)
	c, err := NewClient(clientConfig)
	require.Nil(err, "Unexpected error", err)
	require.NotNil(c, "Client should not be nil")
	log.Debugf("Error: %v", err)

	err = c.StartMiner()
	require.Nil(err)
	// FIXME: Add some real tests here
	time.Sleep(1 * time.Second)
	err = c.StopMiner()
	require.Nil(err)
}

func TestMain(m *testing.M) {
	log.SetLevel(log.WarnLevel)
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}
