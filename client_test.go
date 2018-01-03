package minerconfig

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
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

func TestBadBinaryPath(t *testing.T) {
	require := require.New(t)
	bp := "/tmp/123414"

	tmpConfig, err := generateConfigFile()
	require.Nil(err)
	defer os.Remove(tmpConfig)

	c, err := NewClient(bp, tmpConfig, "google.com")
	require.Nil(c, "Client should have been nil")
	require.NotNil(err, "Expected error")
}

func TestBadConfigPath(t *testing.T) {
	require := require.New(t)

	tmpConfig, err := generateConfigFile()
	require.Nil(err)
	defer os.Remove(tmpConfig)
	binaryPath := tmpConfig

	c, err := NewClient(binaryPath, "/tmp/1234", "google.com")
	require.Nil(c, "Client should have been nil")
	require.NotNil(err, "Expected error")
}

func TestBadWebserverAddress(t *testing.T) {
	require := require.New(t)

	tmpConfig, err := generateConfigFile()
	require.Nil(err)
	defer os.Remove(tmpConfig)
	binaryPath := tmpConfig

	c, err := NewClient(binaryPath, tmpConfig, "google.com")
	require.Nil(c, "Client should have been nil")
	require.NotNil(err, "Expected error")
	logrus.Debugf("Error: %v", err)
}

func TestRunServer(t *testing.T) {
	snl := RunServer("webserver/www", 61117)
	snl.Stop()
}

func TestConnect(t *testing.T) {
	require := require.New(t)

	tmpConfig, err := generateConfigFile()
	require.Nil(err)
	defer os.Remove(tmpConfig)
	binaryPath := tmpConfig

	// Start webserver
	snl := RunServer("webserver/www", 61117)
	defer snl.Stop()
	time.Sleep(300 * time.Millisecond)
	c, err := NewClient(binaryPath, tmpConfig, "localhost:61117")
	require.Nil(err, "Unexpected error", err)
	require.NotNil(c, "Client should not be nil")
	logrus.Debugf("Error: %v", err)
}

func TestSetPool(t *testing.T) {
	require := require.New(t)

	tmpConfig, err := generateConfigFile()
	require.Nil(err)
	defer os.Remove(tmpConfig)
	binaryPath := tmpConfig

	// Start webserver
	snl := RunServer("webserver/www", 61117)
	defer snl.Stop()
	time.Sleep(300 * time.Millisecond)
	c, err := NewClient(binaryPath, tmpConfig, "localhost:61117")
	require.Nil(err, "Unexpected error", err)
	require.NotNil(c, "Client should not be nil")
	logrus.Debugf("Error: %v", err)

	str := `
{"pools": [
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
        }
]}`

	wg := sync.WaitGroup{}
	wg.Add(1)
	testPools := func(w *websockets.WebsocketClient, data interface{}) {
		defer wg.Done()
		var expected map[string]interface{}
		err := json.Unmarshal([]byte(str), &expected)
		require.Nil(err)

		b := data.([]byte)
		s := string(b)
		log.Debugf("Got: \n%v\n", s)
		var got interface{}
		err = json.Unmarshal(b, &got)

		expectedStr, _ := json.Marshal(expected["pools"])
		gotStr, _ := json.Marshal(got)
		require.Equal(string(expectedStr), string(gotStr))
		log.Debugf("Pools match")
	}

	go c.ProcessMessages()
	c.On("get-pools", testPools)
	c.Emit("set-pools", str)
	c.Emit("get-pools", "{}")
	wg.Wait()
}

func TestMiner(t *testing.T) {
	require := require.New(t)

	tmpConfig, err := generateConfigFile()
	require.Nil(err)
	defer os.Remove(tmpConfig)
	binaryPath := "dummyminer/dummyminer"

	// Start webserver
	snl := RunServer("webserver/www", 61117)
	defer snl.Stop()
	time.Sleep(300 * time.Millisecond)
	c, err := NewClient(binaryPath, tmpConfig, "localhost:61117")
	require.Nil(err, "Unexpected error", err)
	require.NotNil(c, "Client should not be nil")
	logrus.Debugf("Error: %v", err)

	err = c.StartMiner()
	require.Nil(err)
	// FIXME: Add some real tests here
	time.Sleep(1 * time.Second)
	err = c.StopMiner()
	require.Nil(err)
}
