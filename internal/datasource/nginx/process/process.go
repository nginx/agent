/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package process

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"path"
	"regexp"
	"strings"

	"github.com/nginx/agent/v3/internal/datasource/os/exec"
)

var (
	re     = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+)`)
	plusre = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+).\((?P<plus>\S+plus\S+)\)`)
)

type Info struct {
	Version       string
	PlusVersion   string
	Prefix        string
	ConfPath      string
	Cfgf          map[string]interface{}
	ConfigureArgs []string
	ExePath       string
}

type Process struct {
	exec exec.ExecInterface
}

func New(exec exec.ExecInterface) *Process {
	return &Process{exec: exec}
}

func (np *Process) GetInfo(pid int32, exePath string) (*Info, error) {
	var err error
	var nginxInfo *Info

	slog.Info("Exe Path", "exe", exePath)
	if exePath == "" {
		exePath = np.getExe()
	}

	if exePath == "" {
		return nil, fmt.Errorf("unable to find NGINX exe for pid %d", pid)
	} else {
		outputBuffer, err := np.exec.RunCmd(exePath, "-V")
		if err != nil {
			return nil, err
		} else {
			nginxInfo = np.parseNginxVersionCommandOutput(outputBuffer)
		}
	}

	nginxInfo.ExePath = exePath

	return nginxInfo, err
}

func (np *Process) getExe() string {
	exePath := ""

	out, commandErr := np.exec.RunCmd("sh", "-c", "command -v nginx")
	slog.Info("Command Out", "out", out)
	if commandErr == nil {
		exePath = strings.TrimSuffix(out.String(), "\n")
		slog.Info("Exe Trim", "exepath", exePath)
	}

	if exePath == "" {
		slog.Info("Exe Empty", "exepath", exePath)
		exePath = np.defaultToNginxCommandForProcessPath()
	}

	if strings.Contains(exePath, "(deleted)") {
		exePath = np.sanitizeExeDeletedPath(exePath)
	}

	slog.Info("End of GetEXe", "exepath", exePath)
	return exePath
}

func (np *Process) parseNginxVersionCommandOutput(output *bytes.Buffer) *Info {
	nginxInfo := &Info{}

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

func (np *Process) defaultToNginxCommandForProcessPath() string {
	path, err := np.exec.FindExecutable("nginx")
	if err != nil {
		return ""
	}
	return path
}

func (np *Process) sanitizeExeDeletedPath(exePath string) string {
	firstSpace := strings.Index(exePath, "(deleted)")
	if firstSpace != -1 {
		return strings.TrimSpace(exePath[0:firstSpace])
	}
	return strings.TrimSpace(exePath)
}

func (np *Process) parseNginxVersion(line string) (version, plusVersion string) {
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

func (np *Process) parseConfigureArguments(line string) (result map[string]interface{}, flags []string) {
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
