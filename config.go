package minerconfig

import (
	"crypto/md5"
	"fmt"

	gputool "github.com/gurupras/minerconfig/gpu-tool"
)

// Config structure representing config JSON file
// Add any relevant fields here
type Config struct {
	Algorithm         string      `json:"algo" yaml:"algo"`
	Background        bool        `json:"background" yaml:"background"`
	Colors            bool        `json:"colors" yaml:"colors"`
	DonateLevel       float64     `json:"donate-level" yaml:"donate-level"`
	LogFile           *string     `json:"log-file" yaml:"log-file"`
	PrintTime         int         `json:"print-time" yaml:"print-time"`
	Retries           int         `json:"retries" yaml:"retries"`
	RetryPause        int         `json:"retry-pause" yaml:"retry-pause"`
	Syslog            bool        `json:"syslog" yaml:"syslog"`
	OpenCLPlatform    int         `json:"opencl-platform" yaml:"opencl-platform"`
	CPUThreads        int         `json:"cpu_threads" yaml:"cpu_threads"`
	DeviceInstanceIDs []string    `json:"device_instance_ids" yaml:"device_instance_ids"`
	Threads           []GPUThread `json:"threads" yaml:"threads"`
	Pools             []Pool      `json:"pools" yaml:"pools"`
	// Arguments to support miners like cpuminer-multi
	Url   string `json:"url" yaml:"url"`
	User  string `json:"user" yaml:"user"`
	Pass  string `json:"pass" yaml:"pass"`
	Proxy string `json:"proxy" yaml:"proxy"`
	// Custom arguments
	Reset *Reset `json:"reset" yaml:"reset"`
}

func (c *Config) Clone() *Config {
	ret := &Config{}
	*ret = *c
	// TODO: LogFile

	if c.Reset != nil {
		ret.Reset = &Reset{}
		*ret.Reset = *c.Reset
	}

	ret.DeviceInstanceIDs = make([]string, len(c.DeviceInstanceIDs))
	if ret.Reset != nil {
		ret.Reset.DeviceInstanceIDs = ret.DeviceInstanceIDs
	}
	for idx, val := range c.DeviceInstanceIDs {
		ret.DeviceInstanceIDs[idx] = val
	}

	if c.Threads != nil {
		ret.Threads = make([]GPUThread, len(c.Threads))
		for idx := range c.Threads {
			t := &GPUThread{}
			*t = c.Threads[idx]
			ret.Threads[idx] = *t
		}
	}
	return ret
}

// GPUThread structure representing a GPU thread
type GPUThread struct {
	Index       *int `json:"index" yaml:"index"`
	DeviceIndex *int `json:"device_index" yaml:"device_index"`
	Intensity   int  `json:"intensity" yaml:"intensity"`
	WorkSize    int  `json:"worksize" yaml:"worksize"`
	AffineToCPU bool `json:"affine_to_cpu" yaml:"affine_to_cpu"`
}

// Pool structure representing a pool
type Pool struct {
	Algorithm  string  `json:"algorithm" yaml:"algorithm"`
	Url        string  `json:"url" yaml:"url"`
	User       string  `json:"user" yaml:"user"`
	Pass       string  `json:"pass" yaml:"pass"`
	Keepalive  bool    `json:"keepalive" yaml:"keepalive"`
	Nicehash   bool    `json:"nicehash" yaml:"nicehash"`
	Coin       *string `json:"coin" yaml:"coin"`
	PoolName   *string `json:"pool_name" yaml:"pool_name"`
	WalletName *string `json:"wallet_name" yaml:"wallet_name"`
	Label      *string `json:"label" yaml:"label"`
}

type Reset struct {
	ScriptPath        string   `json:"script_path" yaml:"script_path"`
	DeviceInstanceIDs []string `json:"device_instance_ids" yaml:"device_instance_ids"`
	GPUTool           *GPUTool `json:"gpu_tool" yaml:"gpu_tool"`
}

type GPUTool struct {
	Type gputool.GPUToolType    `json:"type" yaml:"type"`
	Path string                 `json:"path" yaml:"path"`
	Args map[string]interface{} `json:"args" yaml:"args"`
}

// Hash of this Pool
func (p *Pool) Hash() string {
	return fmt.Sprintf("%X", md5.Sum([]byte(fmt.Sprintf("%v-%v", p.Url, p.User))))
}
