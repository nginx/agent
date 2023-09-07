package phpfpm

import (
	"github.com/nginx/agent/v2/src/core"
)

var Shell core.Shell = core.ExecShellCommand{}

// Status is an Enum that represents the status of PhpFpm.
type Status int

const (
	UNKNOWN Status = iota
	INSTALLED
	RUNNING
	MISSING
)

// String get the string representation of the enum
func (s Status) String() string {
	switch s {
	case INSTALLED:
		return "INSTALLED"
	case RUNNING:
		return "RUNNING"
	case MISSING:
		return "MISSING"
	}
	return "UNKNOWN"
}
