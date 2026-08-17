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

	var priorityClassName string
	switch priorityClass {
	case windows.NORMAL_PRIORITY_CLASS:
		priorityClassName = "NORMAL_PRIORITY_CLASS"
	case windows.IDLE_PRIORITY_CLASS:
		priorityClassName = "IDLE_PRIORITY_CLASS"
	case windows.HIGH_PRIORITY_CLASS:
		priorityClassName = "HIGH_PRIORITY_CLASS"
	case windows.REALTIME_PRIORITY_CLASS:
		priorityClassName = "REALTIME_PRIORITY_CLASS"
	case windows.BELOW_NORMAL_PRIORITY_CLASS:
		priorityClassName = "BELOW_NORMAL_PRIORITY_CLASS"
	case windows.ABOVE_NORMAL_PRIORITY_CLASS:
		priorityClassName = "ABOVE_NORMAL_PRIORITY_CLASS"
	default:
		// Fallback for any unknown values
		priorityClassName = fmt.Sprintf("UNKNOWN_PRIORITY_CLASS (%d)", priorityClass)
	}

	// Prints the Windows priority class name
	fmt.Printf("%s\r\n", priorityClassName)
}
