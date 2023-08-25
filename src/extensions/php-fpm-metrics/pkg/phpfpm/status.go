package phpfpm

import (
	"fmt"
	"strings"

	"github.com/nginx/agent/v2/src/core"
	log "github.com/sirupsen/logrus"
)

var shell core.Shell = core.ExecShellCommand{}

// Status is an Enum that represents the status of PhpFpm.
type Status int

const (
	UNKNOWN Status = iota
	INSTALLED
	RUNNING
)

// String get the string representation of the enum
func (s Status) String() string {
	switch s {
	case INSTALLED:
		return "INSTALLED"
	case RUNNING:
		return "RUNNING"
	}
	return "UNKNOWN"
}

// GetStatus returns phpfpm process status
func GetStatus(pid, version string) Status {
	output, err := shell.Exec("ps xao pid,ppid,command | grep 'php-fpm[:]'")
	if err != nil {
		log.Warnf("failed to retrieve ps info about php-fpm: %v for pid %s", err, pid)
		return UNKNOWN
	}

	outputSplit := strings.Split(string(output), "\n")
	for _, l := range outputSplit {
		if len(l) == 0 {
			continue
		}

		// master info, otherwise a pool worker
		if strings.Contains(l, "master process") {
			parsed := strings.Fields(l)
			if parsed[1] == pid {
				return RUNNING
			}
		}
	}

	// not running; maybe it's installed
	output, err = shell.Exec("ls", fmt.Sprintf("/etc/php/%s", version))
	if err != nil {
		log.Warnf("failed to retrieve ps info about php: %v for pid %s", err, pid)
		return UNKNOWN
	}

	installs := strings.Fields(string(output))
	if len(installs) > 0 {
		return INSTALLED
	}

	return UNKNOWN
}
