package system

import (
	"syscall"
)

// getProcessPriority returns the nice value of the process with the given pid.
func getProcessPriority(pid int) (int, error) {
	knice, err := syscall.Getpriority(syscall.PRIO_PROCESS, pid)
	if err != nil {
		return 0, err
	}

	// Linux: convert syscall.Getpriority()'s "kernel nice" to "user nice"
	//	=> unice = 20 - knice
	// See https://linux.die.net/man/2/getpriority
	return ((knice - 20) * -1), nil
}
