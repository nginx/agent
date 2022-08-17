package nap

// Enums for Status
const (
	UNDEFINED Status = iota
	MISSING
	INSTALLED
	RUNNING
)

// String get the string representation of the enum
func (s Status) String() string {
	switch s {
	case MISSING:
		return "missing"
	case INSTALLED:
		return "installed"
	case RUNNING:
		return "running"
	}
	return "unknown"
}
