//go:build !windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
)

func main() {
	time.Sleep(100 * time.Millisecond)

	pid := os.Getpid()
	niceValue, err := syscall.Getpriority(syscall.PRIO_PROCESS, pid)
	if err != nil {
		fmt.Printf("error getting priority: %s\n", err)
		os.Exit(1)
	}

	// Linux: convert knice => unice see https://linux.die.net/man/2/getpriority
	if runtime.GOOS == "linux" {
		fmt.Printf("%d\n", (niceValue-20)*-1)
	} else {
		fmt.Printf("%d\n", niceValue)
	}
}
