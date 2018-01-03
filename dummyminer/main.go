package main

import (
	"fmt"
	"time"

	"github.com/eiannone/keyboard"
	log "github.com/sirupsen/logrus"
)

func main() {
	// We're going to be dumping random text until close
	shouldQuit := false

	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()

	go func() {
		for {
			if shouldQuit {
				break
			}
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("%v: Hash rate: ...\n", time.Now())
		}
	}()
	for {
		char, key, err := keyboard.GetSingleKey()
		if err != nil {
			log.Errorf("Failed to get key from keyboard: %v", err)
		}
		_ = key
		_ = err
		switch char {
		case 'h':
			fmt.Printf("Hash Rate: ...\n")
		case 'p':
			fmt.Printf("Paused\n")
		case 'r':
			fmt.Printf("Resumed\n")
		case 'q':
			fmt.Printf("Quit..\n")
			shouldQuit = true
		}
		if shouldQuit {
			break
		}
	}
}
