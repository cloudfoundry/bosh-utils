//go:build !windows

package processpriority

import (
	"fmt"
	"syscall"
)

// Opiniated nice values
const (
	UnixPriorityIdle        = 19
	UnixPriorityBelowNormal = 5
	UnixPriorityNormal      = 0
	UnixPriorityAboveNormal = -5
	UnixPriorityHigh        = -10
	UnixPriorityRealTime    = -20
)

func getOS(pid int) (priority ProcessPriority, rawPriority int, err error) {
	if rawPriority, err = GetRAW(pid); err != nil {
		return
	}
	switch rawPriority {
	case UnixPriorityIdle:
		priority = Idle
	case UnixPriorityBelowNormal:
		priority = BelowNormal
	case UnixPriorityNormal:
		priority = Normal
	case UnixPriorityAboveNormal:
		priority = AboveNormal
	case UnixPriorityHigh:
		priority = High
	case UnixPriorityRealTime:
		priority = RealTime
	default:
		priority = OSSpecific
	}
	return
}

func setOS(pid int, priority ProcessPriority) error {
	var unixPriority int
	switch priority {
	case Idle:
		unixPriority = UnixPriorityIdle
	case BelowNormal:
		unixPriority = UnixPriorityBelowNormal
	case Normal:
		unixPriority = UnixPriorityNormal
	case AboveNormal:
		unixPriority = UnixPriorityAboveNormal
	case High:
		unixPriority = UnixPriorityHigh
	case RealTime:
		unixPriority = UnixPriorityRealTime
	default:
		return fmt.Errorf("unknown universal priority: %d", priority)
	}
	return SetRAW(pid, unixPriority)
}

// GetRAW is an OS specific function to get the priority of a process.
// As priority values are not the same on all OSes, you should use the universal function Get() instead to be platform agnostic.
func GetRAW(pid int) (priority int, err error) {
	if priority, err = syscall.Getpriority(syscall.PRIO_PROCESS, pid); err != nil {
		return
	}
	priority = (priority - 20) * -1 // Convert knice to unice, see notes on https://linux.die.net/man/2/getpriority
	return
}

// SetRAW is an OS specific function to set the priority of a process.
// As priority values are not the same on all OSes, you should use the universal function Set() instead to be platform agnostic.
func SetRAW(pid, priority int) (err error) {
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, priority) // it seems there is no need to convert nice value to knice value here
}
