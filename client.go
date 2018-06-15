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
	mineros "github.com/gurupras/go-cryptonight-miner/miner-os"
	"github.com/gurupras/go-easyfiles"
	amdconfig "github.com/gurupras/minerconfig/amd"
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
	MinerConfig     *Config
	origMinerConfig *Config
	TempConfigPath  string
	miner           *exec.Cmd
}

// ClientConfig structure representing the configuration parameters for a
// minerconfig client
type ClientConfig struct {
	BinaryPath       string        `json:"binary_path" yaml:"binary_path"`
	BinaryArgs       []interface{} `json:"binary_args" yaml:"binary_args"`
	BinaryIsScript   bool          `json:"binary_is_script" yaml:"binary_is_script"`
	MinerConfigPath  string        `json:"miner_config_path" yaml:"miner_config_path"`
	MinerConfig      *Config       `json:"miner_config" yaml:"miner_config"`
	WebserverAddress string        `json:"webserver_address" yaml:"webserver_address"`
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
	c.origMinerConfig = clientConfig.MinerConfig
	c.MinerConfig = c.origMinerConfig.Clone()
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
			log.Errorf("Failed to marshal JSON data: %v", err)
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

	c.MinerConfig = c.origMinerConfig.Clone()
	minerConfig := c.MinerConfig
	minerConfig.Pools = poolData
	// Override c.MinerConfig.Url to deal with cpuminer-multi

	firstPool := poolData[0]
	// TODO: Ideally, the following lines should have checks
	// XXX: Currently, this adds a stratum+tcp:// prefix..this needs to be fixed somehow
	minerConfig.Url = fmt.Sprintf("stratum+tcp://%v", firstPool.Url)
	minerConfig.User = firstPool.User
	minerConfig.Pass = firstPool.Pass

	// Stop current miner if it exists
	// Overwrite TempConfigPath file
	// Start miner with -c TempConfigPath
	if err := c.ResetMiner(); err != nil {
		log.Errorf("Failed to reset miner: %v", err)
	}
	// Check to see if every thread has an Index. If not, then we need to parse
	// DeviceIndex into Index
	if minerConfig.Threads != nil {
		for threadIdx := 0; threadIdx < len(minerConfig.Threads); threadIdx++ {
			threadInfo := minerConfig.Threads[threadIdx]
			if threadInfo.Index == nil && threadInfo.DeviceIndex == nil {
				log.Fatalf("Either 'index' or 'device_index' must be present for every thread")
			}
			if threadInfo.Index == nil {
				// We need to convert the DeviceIndex into an OpenCL index
				instanceId := minerConfig.DeviceInstanceIDs[*threadInfo.DeviceIndex]
				topology, err := mineros.GetPCITopology(instanceId)
				if err != nil {
					log.Fatalf("Failed to get topology for device instance ID '%v': %v\n", instanceId, err)
				}
				var openclIdx int
				openclIdx, err = amdconfig.FindIndexMatchingTopology(topology)
				if err != nil {
					log.Fatalf("%v", err)
				}
				minerConfig.Threads[threadIdx].Index = &openclIdx
				log.Infof("Thread-%d: OpenCL index=%d", threadIdx, *minerConfig.Threads[threadIdx].Index)
			}
		}
	}

	b, err = json.MarshalIndent(minerConfig, "", "  ")
	if err != nil {
		log.Errorf("Failed to marshal config: %v\n", err)
		return
	}

	log.Infof("Received pools from server")
	if err := ioutil.WriteFile(c.TempConfigPath, b, 0666); err != nil {
		log.Errorf("Failed to update config: %v", err)
		return
	}

	log.Infof("Starting miner ...")
	if err := c.StartMiner(); err != nil {
		log.Errorf("Failed to start miner: %v", err)
		return
	}
}

// ResetMiner stops current miner (if exists) and starts a new instance
func (c *Client) ResetMiner() error {
	if c.miner != nil {
		if err := c.StopMiner(); err != nil {
			return fmt.Errorf("Failed to stop miner: %v", err)
		}
		c.miner = nil
		// // Remove tmpConfigPath
		// if strings.Compare(c.TempConfigPath, "") != 0 {
		// 	os.Remove(c.TempConfigPath)
		// }
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
			instanceIDs := c.MinerConfig.Reset.DeviceInstanceIDs
			gpuToolConf := c.MinerConfig.Reset.GPUTool
			gpureset.ResetGPU(instanceIDs)
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

	var miner *exec.Cmd
	args := make([]string, 0)

	var binaryArgsStr []string
	if c.BinaryArgs != nil {
		binaryArgsStr = make([]string, len(c.BinaryArgs))
		for i := 0; i < len(c.BinaryArgs); i++ {
			binaryArgsStr[i] = fmt.Sprintf("%v", c.BinaryArgs[i])
		}
	}

	cmdline := fmt.Sprintf(`%v %v -c "%v"`, c.BinaryPath, strings.Join(binaryArgsStr, " "), c.TempConfigPath)

	if c.BinaryIsScript {
		cmdline = fmt.Sprintf("/bin/bash %v", cmdline)
		args = append(args, c.BinaryPath)
		if c.BinaryArgs != nil {
			args = append(args, binaryArgsStr...)
		}
		args = append(args, []string{"-c", c.TempConfigPath}...)
		miner = exec.Command("/bin/bash", args...)
	} else {
		if c.BinaryArgs != nil {
			args = append(args, binaryArgsStr...)
		}
		args = append(args, []string{"-c", c.TempConfigPath}...)
		log.Infof("args: %v", args)
		miner = exec.Command(c.BinaryPath, args...)
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
