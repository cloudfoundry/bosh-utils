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

	done := make(chan struct{})
	var exitStatus int
	go func() {
		defer close(done)
		sig := <-sigCh
		switch s := sig.String(); s {
		case "SIGTERM":
			fmt.Println("Received SIGTERM")
			exitStatus = 13
		default:
			fmt.Printf("Received unhandled signal: %s\n", s)
			exitStatus = 17
		}
	}()

	<-done
	os.Exit(exitStatus)
}
