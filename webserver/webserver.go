package main

import (
	"sync"

	"github.com/gurupras/minerconfig"
)

func main() {
	minerconfig.RunServer("www", 61117)
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
