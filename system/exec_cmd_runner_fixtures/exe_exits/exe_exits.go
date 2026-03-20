package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)

	go func() {
		for {
			switch <-sigCh {
			case syscall.SIGTERM:
				fmt.Printf("Exe received SIGTERM\n")
			}
		}
	}()

	// Exit immediately
	os.Exit(0)
}
