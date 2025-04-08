package processpriority

// ProcessPriority is a universal type for process priorities. It used with the universal wrapper Set() to be platform agnostic.
type ProcessPriority int

const (
	// PriorityOSSpecific is only used on Get(), it indicates that the current level is not a universal one from this package.
	OSSpecific ProcessPriority = iota
	Idle
	BelowNormal
	Normal
	AboveNormal
	High
	RealTime
)

// String implements the fmt.Stringer interface
func (pp ProcessPriority) String() string {
	switch pp {
	case OSSpecific:
		return "OS Specific"
	case Idle:
		return "Idle"
	case BelowNormal:
		return "Below Normal"
	case Normal:
		return "Normal"
	case AboveNormal:
		return "Above Normal"
	case High:
		return "High"
	case RealTime:
		return "Real Time"
	default:
		return "<unknown>"
	}
}

// Set is an universal wrapper for setting process priority.
// It uses OS specific convertion and calls OS specific implementation.
func Set(pid int, priority ProcessPriority) (err error) {
	return setOS(pid, priority)
}

// Get is an universal wrapper for getting process priority.
// It uses OS specific convertion and calls OS specific implementation.
func Get(pid int) (priority ProcessPriority, rawOSPriority int, err error) {
	return getOS(pid)
}
