package main

import (
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/alecthomas/kingpin"
	"github.com/gurupras/minerconfig"
)

var (
	app     = kingpin.New("minerconfig-webserver", "Miner webserver")
	verbose = app.Flag("verbose", "Enable verbose messages").Short('v').Default("false").Bool()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	minerconfig.RunServer("www", 61117)
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
