package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/alecthomas/kingpin"
	"github.com/gurupras/minerconfig"
	log "github.com/sirupsen/logrus"
)

var (
	app        = kingpin.New("miner-client", "Miner client")
	binaryPath = app.Arg("binary-path", "Path to binary to execute").String()
	baseConfig = app.Arg("base-config", "Base config file to use").String()
	webserver  = app.Arg("server-address", "Address of server to fetch pool information from").String()
	verbose    = app.Flag("verbose", "Enable verbose messages").Short('v').Default("false").Bool()
)

func main() {
	log.Infof("Starting ...")
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	client, err := minerconfig.NewClient(*binaryPath, *baseConfig, *webserver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(-1)
	}
	client.AddPoolListeners()
	client.UpdatePools()

	// Wait forever
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
