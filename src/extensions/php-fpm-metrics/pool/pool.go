package pool

import (
	"fmt"
	"strings"

	"github.com/nginx/agent/v2/src/core"
)

// Todo: Leverage gopsutil
var Shell core.Shell = core.ExecShellCommand{}

type Pool struct {
	dir string
}

func New(dir string) *Pool {
	return &Pool{
		dir: dir,
	}
}

// GetConfigs returns workers configuration in dir
func (p *Pool) GetConfigs(dir string) ([]string, error) {
	output, err := Shell.Exec("ls", dir)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pool conf files in dir %s: %v", dir, err)
	}

	files := strings.Fields(string(output))
	if len(files) == 0 {
		return nil, fmt.Errorf("no conf files in dir %s. pool configurations must be located in this dir. Err: %v", dir, err)
	}

	return files, nil
}
