package minerconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/gorilla/websocket"
	"github.com/gurupras/go-easyfiles"
	gpureset "github.com/gurupras/minerconfig/gpu-reset"
	gputool "github.com/gurupras/minerconfig/gpu-tool"
	"github.com/homesound/simple-websockets"
	log "github.com/sirupsen/logrus"
)

// Client structure represents a minerconfig client
// minerconfig Clients are responsible for:
// 1) Connect to webserver
// 2) Listen for changes to selected pools
// 3) Start/Stop the miner
type Client struct {
	*ClientConfig
	*websockets.WebsocketClient
	MinerConfig    *Config
	TempConfigPath string
	miner          *exec.Cmd
}

// ClientConfig structure representing the configuration parameters for a
// minerconfig client
type ClientConfig struct {
	BinaryPath       string  `json:"binary_path" yaml:"binary_path"`
	BinaryIsScript   bool    `json:"binary_is_script" yaml:"binary_is_script"`
	MinerConfigPath  string  `json:"miner_config_path" yaml:"miner_config_path"`
	MinerConfig      *Config `json:"miner_config" yaml:"miner_config"`
	WebserverAddress string  `json:"webserver_address" yaml:"webserver_address"`
}

// NewClient creates a new minerconfig client
func NewClient(clientConfig *ClientConfig) (*Client, error) {
	// First, verify that binary and config paths are valid
	binaryPath := clientConfig.BinaryPath

	if !easyfiles.Exists(binaryPath) {
		return nil, fmt.Errorf("Binary path '%v' does not exist!", binaryPath)
	}

	// Read from MinerConfigPath if specified
	if strings.Compare(clientConfig.MinerConfigPath, "") != 0 {
		log.Debugf("Reading miner-config from file: '%v'", clientConfig.MinerConfigPath)
		if b, err := ioutil.ReadFile(clientConfig.MinerConfigPath); err != nil {
			return nil, fmt.Errorf("Failed to read miner-config file: '%v': %v", clientConfig.MinerConfigPath, err)
		} else {
			if err := yaml.Unmarshal(b, &clientConfig.MinerConfig); err != nil {
				return nil, fmt.Errorf("Failed to parse miner-config file into Config struct: %v", err)
			}
		}
	}

	tmpConfigFile, err := easyfiles.TempFile(os.TempDir(), "minerconfig", ".json")
	if err != nil {
		return nil, fmt.Errorf("Failed to create temporary config file: %v", err)

	}
	tmpConfigPath := tmpConfigFile.Name()
	tmpConfigFile.Close()

	c := &Client{}
	c.ClientConfig = clientConfig
	c.MinerConfig = c.ClientConfig.MinerConfig
	c.TempConfigPath = tmpConfigPath
	// Should we connect here?
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("Failed to connect to webserver: %v", err)
	}
	go c.ProcessMessages()
	return c, nil
}

// Connect to the remote webserver
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

	// Periodically send messages to server for keepalive
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if err := c.Emit("keepalive", "{}"); err != nil {
				if err = c.Connect(); err != nil {
					log.Errorf("Failed to re-connect to server: %v", err)
				}
			}
		}
	}()
	return nil
}

// HandlePoolInfo handles the selected-pools data from the server
func (c *Client) HandlePoolInfo(w *websockets.WebsocketClient, data interface{}) {
	log.Infof("Received pool info from server")
	var b []byte
	var err error
	switch data.(type) {
	case string:
		b = []byte(data.(string))
	default:
		if b, err = json.Marshal(data); err != nil {
			log.Errorf("Failed to unmarshal JSON data: %v", err)
			return
		}
	}
	// Now, we have b representing the bytes of data regardless of type
	// Convert this into []Pool
	var poolData []Pool
	if err = json.Unmarshal(b, &poolData); err != nil {
		log.Errorf("Failed to convert data into []Pool: %v", err)
		return
	}
	if len(poolData) == 0 {
		// The server has no selected pools..wait for it to inform us
		log.Infof("Server has no selected pool information. Waiting for server to inform us")
		return
	}

	c.MinerConfig.Pools = poolData
	// Override c.MinerConfig.Url to deal with cpuminer-multi

	firstPool := poolData[0]
	// TODO: Ideally, the following lines should have checks
	// XXX: Currently, this adds a stratum+tcp:// prefix..this needs to be fixed somehow
	c.MinerConfig.Url = fmt.Sprintf("stratum+tcp://%v", firstPool.Url)
	c.MinerConfig.User = firstPool.User
	c.MinerConfig.Pass = firstPool.Pass

	if b, err := json.MarshalIndent(c.MinerConfig, "", "  "); err != nil {
		log.Errorf("Failed to marshal config: %v\n", err)
	} else {
		// Stop current miner if it exists
		// Overwrite TempConfigPath file
		// Start miner with -c TempConfigPath
		log.Infof("Received pools from server")
		if err := ioutil.WriteFile(c.TempConfigPath, b, 0666); err != nil {
			log.Errorf("Failed to update config: %v", err)
			return
		}
		if err := c.ResetMiner(); err != nil {
			log.Errorf("Failed to reset miner: %v", err)
		}
	}
}

// ResetMiner stops current miner (if exists) and starts a new instance
func (c *Client) ResetMiner() error {
	if c.miner != nil {
		if err := c.StopMiner(); err != nil {
			return fmt.Errorf("Failed to stop miner: %v", err)
		}
		c.miner = nil
		// Remove tmpConfigPath
		if strings.Compare(c.TempConfigPath, "") != 0 {
			os.Remove(c.TempConfigPath)
		}
	}

	// Check for extra reset logic
	if c.MinerConfig.Reset != nil {
		if strings.Compare(c.MinerConfig.Reset.ScriptPath, "") != 0 {
			cmdline := c.MinerConfig.Reset.ScriptPath
			cmd := exec.Command(cmdline)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("Failed to run reset script: %v", err)
			}
			if err := cmd.Wait(); err != nil {
				return fmt.Errorf("Failed to wait for reset script to complete: %v", err)
			}
		} else {
			if runtime.GOOS != "windows" {
				log.Fatalf("Sorry! This is unimeplemented on your OS")
			}
			// Get pre-determined properties and figure out how to run them
			instanceID := c.MinerConfig.Reset.DeviceInstanceID
			gpuToolConf := c.MinerConfig.Reset.GPUTool
			gpureset.ResetGPU(instanceID)
			// Now, we need to run the gpu tool to configure the GPU
			gpuTool, err := gputool.ParseGPUTool(gpuToolConf.Type)
			if err != nil {
				return err
			}
			if err := gpuTool.Run(gpuToolConf.Path, gpuToolConf.Args); err != nil {
				return fmt.Errorf("Failed to run gpu-tool: %v", err)
			}
		}
	}

	log.Infof("Starting miner ...")
	if err := c.StartMiner(); err != nil {
		return fmt.Errorf("Failed to start miner: %v", err)
	}
	return nil
}

// AddPoolListeners adds the default listeners
func (c *Client) AddPoolListeners() {
	c.On("update-selected-pools", c.HandlePoolInfo)
	c.On("get-selected-pools-result", c.HandlePoolInfo)
}

// UpdatePools requests the server to send back the current set of selected pools
func (c *Client) UpdatePools() error {
	return c.Emit("get-selected-pools", "{}")
}

// StartMiner starts the miner
func (c *Client) StartMiner() error {
	cmdline := fmt.Sprintf(`%v -c "%v"`, c.BinaryPath, c.TempConfigPath)
	var miner *exec.Cmd
	if c.BinaryIsScript {
		cmdline = fmt.Sprintf("/bin/bash %v", cmdline)
		miner = exec.Command("/bin/bash", c.BinaryPath, "-c", c.TempConfigPath)
	} else {
		miner = exec.Command(c.BinaryPath, "-c", c.TempConfigPath)
	}
	log.Infof("cmdline: %v", cmdline)
	miner.Stdin = os.Stdin
	miner.Stdout = os.Stdout
	miner.Stderr = os.Stderr
	c.miner = miner
	return miner.Start()
}

// StopMiner stops the miner
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
