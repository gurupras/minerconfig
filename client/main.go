package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	yaml "gopkg.in/yaml.v2"

	"github.com/alecthomas/kingpin"
	easyfiles "github.com/gurupras/go-easyfiles"
	"github.com/gurupras/minerconfig"
	log "github.com/sirupsen/logrus"
)

var (
	app        = kingpin.New("miner-client", "Miner client")
	configPath = app.Arg("config-path", "Path to YAML configuration").Required().String()
	verbose    = app.Flag("verbose", "Enable verbose messages").Short('v').Default("false").Bool()
)

func main() {
	log.Infof("Starting ...")
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	if !easyfiles.Exists(*configPath) {
		log.Errorf("configPath '%v' does not exist!", *configPath)
		os.Exit(-1)
	}

	var clientConfig minerconfig.ClientConfig
	if b, err := ioutil.ReadFile(*configPath); err != nil {
		log.Errorf("Failed to read config file: '%v': %v", *configPath, err)
		os.Exit(-1)
	} else {
		if err := yaml.Unmarshal(b, &clientConfig); err != nil {
			log.Errorf("Failed to parse config file into ClientConfig: %v", err)
			os.Exit(-1)
		}
	}

	client, err := minerconfig.NewClient(&clientConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(-1)
	}
	log.Infof("Connected to server")

	client.AddPoolListeners()
	log.Infof("Finished setting up listeners")

	client.UpdatePools()
	log.Infof("Requested get-pools")

	// Wait forever
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
