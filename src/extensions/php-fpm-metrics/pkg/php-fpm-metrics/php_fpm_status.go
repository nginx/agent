package php_fpm

import (
	"fmt"
	"os/exec"
	"strings"
)

// Status is an Enum that represents the status of PhpFpm.
type Status int

const (
	UNKNOWN Status = iota
	MISSING
	INSTALLED
	RUNNING
)

// String get the string representation of the enum
func (s Status) String() string {
	switch s {
	case MISSING:
		return "MISSING"
	case INSTALLED:
		return "INSTALLED"
	case RUNNING:
		return "RUNNING"
	}
	return "UNKNOWN"
}

func GetPhpFpmStatus() (Status, error) {
	output, err := exec.Command("bash", "-c", "ps aux | grep php-fpm").Output()
	if err != nil {
		return MISSING, fmt.Errorf("failed to retrieve ps info about php-fpm: %v", err)
	}

	outputSplit := strings.Split(string(output), "\n")
	for _, l := range outputSplit {
		if len(l) == 0 {
			continue
		}

		// master info, otherwise a pool worker
		if strings.Contains(l, "master process") {
			return RUNNING, nil
		}
	}

	// not running; maybe it's installed?
	output, err = exec.Command("ls", "/etc/php/").Output()
	if err != nil {
		return MISSING, err
	}

	installs := strings.Fields(string(output))
	if len(installs) > 0 {
		return INSTALLED, nil
	}

	return MISSING, nil
}
