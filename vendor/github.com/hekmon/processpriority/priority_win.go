//go:build windows

package processpriority

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// https://learn.microsoft.com/fr-fr/dotnet/api/system.diagnostics.process.priorityclass?view=net-8.0#remarques
// https://learn.microsoft.com/fr-fr/dotnet/api/system.diagnostics.processpriorityclass?view=net-8.0#champs
const (
	WinPriorityIdle        = 64
	WinPriorityBelowNormal = 16384
	WinPriorityNormal      = 32
	WinPriorityAboveNormal = 32768
	WinPriorityHigh        = 128
	WinPriorityRealTime    = 256
)

func getOS(pid int) (priority ProcessPriority, rawPriority int, err error) {
	if rawPriority, err = GetRAW(pid); err != nil {
		return
	}
	switch rawPriority {
	case WinPriorityIdle:
		priority = Idle
	case WinPriorityBelowNormal:
		priority = BelowNormal
	case WinPriorityNormal:
		priority = Normal
	case WinPriorityAboveNormal:
		priority = AboveNormal
	case WinPriorityHigh:
		priority = High
	case WinPriorityRealTime:
		priority = RealTime
	default:
		priority = OSSpecific
	}
	return
}

func setOS(pid int, priority ProcessPriority) error {
	var winPriority int
	switch priority {
	case Idle:
		winPriority = WinPriorityIdle
	case BelowNormal:
		winPriority = WinPriorityBelowNormal
	case Normal:
		winPriority = WinPriorityNormal
	case AboveNormal:
		winPriority = WinPriorityAboveNormal
	case High:
		winPriority = WinPriorityHigh
	case RealTime:
		winPriority = WinPriorityRealTime
	default:
		return fmt.Errorf("unknown universal priority: %d", priority)
	}
	return SetRAW(pid, winPriority)
}

// https://learn.microsoft.com/en-us/windows/win32/procthread/process-security-and-access-rights
const processAllAccess = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0xffff

// GetRAW is an OS specific function to get the priority of a process.
// As priority values are not the same on all OSes, you should use the universal function Get() instead to be platform agnostic.
func GetRAW(pid int) (priority int, err error) {
	handle, err := windows.OpenProcess(processAllAccess, false, uint32(pid))
	if err != nil {
		return 0, fmt.Errorf("failed to open process: %w", err)
	}
	defer windows.CloseHandle(handle)
	rawPriority, err := windows.GetPriorityClass(handle)
	if err != nil {
		return 0, fmt.Errorf("failed to get priority class: %w", err)
	}
	priority = int(rawPriority)
	return
}

// SetRAW is an OS specific function to set the priority of a process.
// As priority values are not the same on all OSes, you should use the universal function Set() instead to be platform agnostic.
func SetRAW(pid, priority int) error {
	handle, err := windows.OpenProcess(processAllAccess, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("failed to open process: %w", err)
	}
	defer windows.CloseHandle(handle)
	if err = windows.SetPriorityClass(handle, uint32(priority)); err != nil {
		return fmt.Errorf("failed to set priority class: %w", err)
	}
	return nil
}
