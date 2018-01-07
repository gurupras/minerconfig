package minerconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/gurupras/go-easyfiles"
	"github.com/homesound/simple-websockets"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	*websockets.WebsocketClient
	BinaryPath       string
	BinaryIsScript   bool
	Config           map[string]interface{}
	tmpConfigPath    string
	WebserverAddress string
	miner            *exec.Cmd
}

func NewClient(binaryPath, configPath, webserver string) (*Client, error) {
	// First, verify that binary and config paths are valid
	if !easyfiles.Exists(binaryPath) {
		return nil, fmt.Errorf("Binary path '%v' does not exist!", binaryPath)
	}
	if !easyfiles.Exists(configPath) {
		return nil, fmt.Errorf("Config path '%v' does not exist!", configPath)
	}
	var config map[string]interface{}

	if b, err := ioutil.ReadFile(configPath); err != nil {
		return nil, fmt.Errorf("Failed to read baseConfig file '%v': %v", configPath, err)
	} else {
		if err := json.Unmarshal(b, &config); err != nil {
			return nil, fmt.Errorf("Failed to unmarshal baseConfig to JSON: %v", err)
		}
	}

	tmpConfigFile, err := easyfiles.TempFile(os.TempDir(), "minerconfig", ".json")
	if err != nil {
		return nil, fmt.Errorf("Failed to create temporary config file: %v", err)

	}
	tmpConfigPath := tmpConfigFile.Name()
	tmpConfigFile.Close()

	c := &Client{}
	c.BinaryPath = binaryPath
	c.Config = config
	c.tmpConfigPath = tmpConfigPath
	c.WebserverAddress = webserver
	// Should we connect here?
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("Failed to connect to webserver: %v", err)
	}
	go c.ProcessMessages()
	return c, nil
}

func (c *Client) Connect() error {
	u := url.URL{
		Scheme: "ws",
		Host:   c.WebserverAddress,
		Path:   "/ws",
	}
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	client := websockets.NewClient(ws)
	c.WebsocketClient = client
	return nil
}

func (c *Client) HandlePoolInfo(w *websockets.WebsocketClient, data interface{}) {
	log.Infof("Received pool info from server")
	c.Config["pools"] = data
	log.Infof("data type: %t", data)
	// Override c.Config["url"] to deal with cpuminer-multi
	m := data.([]interface{})
	firstPool := m[0].(map[string]interface{})
	// TODO: Ideally, the following lines should have checks
	c.Config["url"] = firstPool["url"]
	c.Config["user"] = firstPool["user"]
	c.Config["pass"] = firstPool["pass"]

	if b, err := json.MarshalIndent(c.Config, "", "  "); err != nil {
		log.Errorf("Failed to marshal config: %v\n", err)
	} else {
		// Stop current miner if it exists
		// Overwrite tmpConfigPath file
		// Start miner with -c tmpConfigPath
		log.Infof("Received pools from server")
		if err := ioutil.WriteFile(c.tmpConfigPath, b, 0666); err != nil {
			log.Errorf("Failed to update config: %v", err)
			return
		}
		if err := c.ResetMiner(); err != nil {
			log.Errorf("Failed to reset miner: %v", err)
		}
	}
}

func (c *Client) ResetMiner() error {
	if c.miner != nil {
		if err := c.StopMiner(); err != nil {
			return fmt.Errorf("Failed to stop miner: %v", err)
		}
		c.miner = nil
	}
	log.Infof("Starting miner ...")
	if err := c.StartMiner(); err != nil {
		return fmt.Errorf("Failed to start miner: %v", err)
	}
	return nil
}

func (c *Client) AddPoolListeners() {
	c.On("pools-update", c.HandlePoolInfo)
	c.On("get-pools-result", c.HandlePoolInfo)
}

func (c *Client) UpdatePools() error {
	return c.Emit("get-pools", "{}")
}

func (c *Client) StartMiner() error {
	cmdline := fmt.Sprintf(`%v -c "%v"`, c.BinaryPath, c.tmpConfigPath)
	var miner *exec.Cmd
	if c.BinaryIsScript {
		cmdline = fmt.Sprintf("/bin/bash %v", cmdline)
		miner = exec.Command("/bin/bash", c.BinaryPath, "-c", c.tmpConfigPath)
	} else {
		miner = exec.Command(c.BinaryPath, "-c", c.tmpConfigPath)
	}
	log.Infof("cmdline: %v", cmdline)
	miner.Stdin = os.Stdin
	miner.Stdout = os.Stdout
	miner.Stderr = os.Stderr
	c.miner = miner
	return miner.Start()
}

func (c *Client) StopMiner() error {
	//return c.miner.Process.Kill()
	if runtime.GOOS == "windows" {
		if err := c.miner.Process.Kill(); err != nil {
			return err
		}
	} else {
		if err := c.miner.Process.Signal(syscall.SIGINT); err != nil {
			return err
		}
		c.miner.Wait()
	}
	return nil
}
