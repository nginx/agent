/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nginx

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/nginx/agent/v3/internal/util"
)

type NginxInfo struct {
	Version       string
	PlusVersion   string
	Prefix        string
	ConfPath      string
	Cfgf          map[string]interface{}
	ConfigureArgs []string
}

type NginxProcess struct {
	helper util.HelperInterface
}

func NewNginxProcess(helper util.HelperInterface) *NginxProcess {
	return &NginxProcess{helper: helper}
}

func (np *NginxProcess) GetNginxInfo(pid int32, exe string) (*NginxInfo, error) {
	var err error
	var nginxInfo *NginxInfo

	if exe == "" {
		exe = np.getExe()
	}

	if exe == "" {
		return nil, fmt.Errorf("unable to find NGINX exe for pid %d", pid)
	} else {
		outputBuffer, err := np.helper.RunCmd(exe, "-V")
		if err != nil {
			return nil, err
		} else {
			nginxInfo = np.parseNginxVersionCommandOutput(outputBuffer)
		}
	}

	return nginxInfo, err
}

func (np *NginxProcess) getExe() string {
	exe := ""

	out, commandErr := np.helper.RunCmd("sh", "-c", "command -v nginx")
	if commandErr == nil {
		exe = strings.TrimSuffix(out.String(), "\n")
	}

	if exe == "" {
		exe = np.defaultToNginxCommandForProcessPath()
	}

	if strings.Contains(exe, "(deleted)") {
		exe = np.sanitizeExeDeletedPath(exe)
	}

	return exe
}

func (np *NginxProcess) parseNginxVersionCommandOutput(output *bytes.Buffer) *NginxInfo {
	nginxInfo := &NginxInfo{}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "nginx version"):
			nginxInfo.Version, nginxInfo.PlusVersion = np.parseNginxVersion(line)
		case strings.HasPrefix(line, "configure arguments"):
			nginxInfo.Cfgf, nginxInfo.ConfigureArgs = np.parseConfigureArguments(line)
		}
	}

	if nginxInfo.Cfgf["prefix"] != nil {
		nginxInfo.Prefix = nginxInfo.Cfgf["prefix"].(string)
	} else {
		nginxInfo.Prefix = "/usr/local/nginx"
	}

	if nginxInfo.Cfgf["conf-path"] != nil {
		nginxInfo.ConfPath = nginxInfo.Cfgf["conf-path"].(string)
	} else {
		nginxInfo.ConfPath = path.Join(nginxInfo.Prefix, "/conf/nginx.conf")
	}

	return nginxInfo
}

func (np *NginxProcess) defaultToNginxCommandForProcessPath() string {
	path, err := np.helper.FindExecutable("nginx")
	if err != nil {
		return ""
	}
	return path
}

func (np *NginxProcess) sanitizeExeDeletedPath(exe string) string {
	firstSpace := strings.Index(exe, "(deleted)")
	if firstSpace != -1 {
		return strings.TrimSpace(exe[0:firstSpace])
	}
	return strings.TrimSpace(exe)
}

func (np *NginxProcess) parseNginxVersion(line string) (version, plusVersion string) {
	matches := re.FindStringSubmatch(line)
	plusMatches := plusre.FindStringSubmatch(line)

	if len(plusMatches) > 0 {
		subNames := plusre.SubexpNames()
		for i, v := range plusMatches {
			switch subNames[i] {
			case "plus":
				plusVersion = v
			case "version":
				version = v
			}
		}
		return version, plusVersion
	}

	if len(matches) > 0 {
		for i, key := range re.SubexpNames() {
			val := matches[i]
			if key == "version" {
				version = val
			}
		}
	}

	return version, plusVersion
}

func (np *NginxProcess) parseConfigureArguments(line string) (result map[string]interface{}, flags []string) {
	// need to check for empty strings
	flags = strings.Split(line[len("configure arguments:"):], " --")
	result = map[string]interface{}{}
	for _, flag := range flags {
		vals := strings.Split(flag, "=")
		switch len(vals) {
		case 1:
			if vals[0] != "" {
				result[vals[0]] = true
			}
		case 2:
			result[vals[0]] = vals[1]
		}
	}
	return result, flags
}
