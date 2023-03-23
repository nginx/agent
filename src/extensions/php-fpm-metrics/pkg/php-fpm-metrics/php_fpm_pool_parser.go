package php_fpm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nginx/agent/v2/src/core"
)

func ParsePools(rootUUID, agent, parentLocalId, version, hostName string) ([]*PhpFpmPool, error) {
	// find the conf path file

	poolDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d/", version)
	output, err := exec.Command("ls", poolDir).Output()
	if err != nil {
		return []*PhpFpmPool{}, fmt.Errorf("failed to retrieve pool conf files in dir %s: %v", poolDir, err)
	}

	files := strings.Fields(string(output))
	if len(files) == 0 {
		return []*PhpFpmPool{}, fmt.Errorf("no conf files in dir %s. pool configurations must be located in this dir. Err: %v", poolDir, err)
	}

	children := []*PhpFpmPool{}
	for _, file := range files {
		b, err := os.ReadFile(fmt.Sprintf("%s/%s", poolDir, file))
		if err != nil {
			return []*PhpFpmPool{}, fmt.Errorf("error reading file to get phpfpm pool info: %v", err)
		}
		pool := parsePhpPoolConf(string(b), hostName)
		pool.CanHaveChildren = false
		pool.Agent = agent
		pool.RootUUID = rootUUID
		pool.ParentLocalId = parentLocalId
		pool.Type = "phpfpm_pool"
		children = append(children, pool)
	}

	return children, nil
}

func parsePhpPoolConf(poolConf, host string) *PhpFpmPool {
	pools := []*PhpFpmPool{}
	var currentPool *PhpFpmPool

	poolConfLines := strings.Split(poolConf, "\n")
	for _, line := range poolConfLines {
		// skip comments, empty lines
		if len(line) == 0 || string(line[0]) == ";" {
			continue
		}

		// strip
		line = strings.ReplaceAll(line, " ", "")

		// first check if this is a new pool declaration
		if string(line[0]) == "[" {
			// found a new context
			if currentPool != nil {
				// if this is not the first pool, append
				pools = append(pools, currentPool)
			}
			// and then reset struct
			currentPool = &PhpFpmPool{
				Includes: []string{},
				Name:     getPoolName(line),
			}
			// and then move to the next line
			continue
		}

		// otherwise, it is a directive within a pool
		var directive = strings.Split(line, "=")
		// this is a malformed directive and should be ignored
		if len(directive) != 2 {
			continue
		}
		if directive[0] == "include" {
			// found an include
			currentPool.Includes = append(currentPool.Includes, directive[1])
		} else if directive[0] == "listen" {
			// found listen socket
			currentPool.Listen = directive[1]
			currentPool.Flisten = directive[1]
		} else if directive[0] == "pm.status_path" {
			// found status page
			currentPool.StatusPath = directive[1]
		}
	}

	// append the last active pool we were collecting info on
	pools = append(pools, currentPool)

	pools[0].DisplayName = hostName(pools[0].Name, host)
	pools[0].LocalId = core.GenerateNginxID("%s_%s", pools[0].Name, pools[0].Listen)

	return pools[0]
}

func hostName(name, host string) string {
	return fmt.Sprintf("phpfpm %s @ %s", name, host)
}

func getPoolName(rawPoolName string) (poolName string) {
	poolName = strings.ReplaceAll(rawPoolName, "[", "")
	poolName = strings.ReplaceAll(poolName, "]", "")
	return poolName
}
