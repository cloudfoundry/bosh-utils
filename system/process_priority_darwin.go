//go:build darwin

package system

import (
	"syscall"
)

// getProcessPriority returns the nice value of the process with the given pid.
func getProcessPriority(pid int) (int, error) {
	return syscall.Getpriority(syscall.PRIO_PROCESS, pid)
}
