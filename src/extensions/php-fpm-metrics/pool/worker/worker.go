package worker

import (
	"fmt"
	"path/filepath"
	"strings"

	re "regexp"

	"github.com/nginx/agent/v2/src/core"
)

// Todo: Leverage gopsutil
var Shell core.Shell = core.ExecShellCommand{}

type Worker struct {
	cfg, dir, host string
}

func New(config, dir, host string) *Worker {
	return &Worker{
		cfg:  config,
		dir:  dir,
		host: host,
	}
}

var listen_re, _ = re.Compile(`(\$\w+)`)

// GetConfigs returns workers configuration in dir
func GetConfigs(dir string) ([]string, error) {
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

// GetMetaData returns phpfpm worker meta-data
func (w *Worker) GetMetaData() *MetaData {
	pools := []*MetaData{}
	var currentPool *MetaData

	poolConfLines := strings.Split(w.cfg, "\n")
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
			currentPool = &MetaData{
				Includes: []string{},
				Name:     getName(line),
			}
			// and then move to the next line
			continue
		}

		// otherwise, it is a directive within a pool
		directive := strings.Split(line, "=")
		// this is a malformed directive and should be ignored
		if len(directive) != 2 {
			continue
		}
		if directive[0] == "include" {
			// found an include
			currentPool.Includes = getIncludes(w.dir, directive[1])
		} else if directive[0] == "listen" {
			// found listen socket
			currentPool.Listen = directive[1]
			currentPool.Flisten = getFlisten(currentPool, directive[1], line)
		} else if directive[0] == "pm.status_path" {
			// found status page
			currentPool.StatusPath = directive[1]
		}
	}

	// append the last active pool that has info collected on
	pools = append(pools, currentPool)

	pools[0].DisplayName = hostName(pools[0].Name, w.host)
	return pools[0]
}

func hostName(name, host string) string {
	return fmt.Sprintf("phpfpm %s @ %s", name, host)
}

func getIncludes(dir, includes string) []string {
	var result []string
	m := make(map[string]struct{})
	// Example: includes = site1,site2,*site3*
	include_rule := strings.Split(includes, ",")
	for _, include := range include_rule {
		relative_rule := resolveLocalPath(dir, include)
		if strings.Contains(relative_rule, "*") {
			files, err := filepath.Glob(relative_rule)
			if err == nil {
				for _, file := range files {
					if _, ok := m[file]; !ok {
						m[file] = struct{}{}
						result = append(result, file)
					}
				}
			}
		} else if _, ok := m[relative_rule]; !ok {
			m[relative_rule] = struct{}{}
			result = append(result, relative_rule)
		}
	}
	return result
}

func resolveLocalPath(dir, include string) string {
	result := strings.ReplaceAll(include, "\"", "")
	if strings.HasPrefix(result, "/") {
		return result
	}
	return fmt.Sprintf("%s/%s", dir, result)
}

func getName(rawName string) string {
	// Example: rawName = "[sample-site]"
	name := strings.ReplaceAll(rawName, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	return name
}

func getFlisten(pool *MetaData, value, line string) string {
	pool.Flisten = value
	match := listen_re.FindStringSubmatch(line)
	// Example: match = "/var/run/php/php7.4-fpm-$pool.sock"
	if found, _ := core.SliceContainsString(match, "$pool"); found {
		return strings.ReplaceAll(pool.Flisten, "$pool", pool.Name)
	}
	return pool.Flisten
}
