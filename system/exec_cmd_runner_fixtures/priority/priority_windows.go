//go:build windows

package main

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func main() {
	time.Sleep(100 * time.Millisecond)

	// Open a handle to the current process with permission to query its info
	pid := uint32(os.Getpid())
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		fmt.Printf("error opening process: %s\n", err)
		os.Exit(1)
	}
	// Ensure the handle is closed when we're done
	defer windows.CloseHandle(handle) //nolint:errcheck

	// Get the priority class
	priorityClass, err := windows.GetPriorityClass(handle)
	if err != nil {
		fmt.Printf("error getting priority: %s\n", err)
		os.Exit(1)
	}

	// Prints the raw Windows priority class integer
	fmt.Printf("%d\n", priorityClass)
}
