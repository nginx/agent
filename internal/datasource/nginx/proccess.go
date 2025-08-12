// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path"
	"regexp"
	"strings"

	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
)

const (
	keyValueLen = 2
	flagLen     = 1
)

var versionRegex = regexp.MustCompile(`(?P<name>\S+)\/(?P<version>.*)`)

func ProcessInfo(ctx context.Context, proc *nginxprocess.Process,
	executer exec.ExecInterface,
) (*model.ProcessInfo, error) {
	exePath := proc.Exe

	if exePath == "" {
		exePath = Exe(ctx, executer)
		if exePath == "" {
			return nil, fmt.Errorf("unable to find NGINX exe for process %d", proc.PID)
		}
	}

	confPath := ConfPathFromCommand(proc.Cmd)

	var nginxInfo *model.ProcessInfo

	outputBuffer, err := executer.RunCmd(ctx, exePath, "-V")
	if err != nil {
		return nil, err
	}

	nginxInfo = ParseNginxVersionCommandOutput(ctx, outputBuffer)

	nginxInfo.ExePath = exePath
	nginxInfo.ProcessID = proc.PID

	if nginxInfo.ConfPath = NginxConfPath(ctx, nginxInfo); confPath != "" {
		nginxInfo.ConfPath = confPath
	}

	return nginxInfo, err
}

func Exe(ctx context.Context, executer exec.ExecInterface) string {
	exePath := ""

	out, commandErr := executer.RunCmd(ctx, "sh", "-c", "command -v nginx")
	if commandErr == nil {
		exePath = strings.TrimSuffix(out.String(), "\n")
	}

	if exePath == "" {
		exePath = defaultToNginxCommandForProcessPath(executer)
	}

	if strings.Contains(exePath, "(deleted)") {
		exePath = sanitizeExeDeletedPath(exePath)
	}

	return exePath
}

func defaultToNginxCommandForProcessPath(executer exec.ExecInterface) string {
	exePath, err := executer.FindExecutable("nginx")
	if err != nil {
		return ""
	}

	return exePath
}

func sanitizeExeDeletedPath(exe string) string {
	firstSpace := strings.Index(exe, "(deleted)")
	if firstSpace != -1 {
		return strings.TrimSpace(exe[0:firstSpace])
	}

	return strings.TrimSpace(exe)
}

func ConfPathFromCommand(command string) string {
	commands := strings.Split(command, " ")

	for i, command := range commands {
		if command == "-c" {
			if i < len(commands)-1 {
				return commands[i+1]
			}
		}
	}

	return ""
}

func NginxConfPath(ctx context.Context, nginxInfo *model.ProcessInfo) string {
	var confPath string

	if nginxInfo.ConfigureArgs["conf-path"] != nil {
		var ok bool
		confPath, ok = nginxInfo.ConfigureArgs["conf-path"].(string)
		if !ok {
			slog.DebugContext(ctx, "failed to cast nginxInfo conf-path to string")
		}
	} else {
		confPath = path.Join(nginxInfo.Prefix, "/conf/nginx.conf")
	}

	return confPath
}

func ParseNginxVersionCommandOutput(ctx context.Context, output *bytes.Buffer) *model.ProcessInfo {
	nginxInfo := &model.ProcessInfo{}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "nginx version"):
			nginxInfo.Version = parseNginxVersion(line)
		case strings.HasPrefix(line, "configure arguments"):
			nginxInfo.ConfigureArgs = parseConfigureArguments(line)
		}
	}

	nginxInfo.Prefix = nginxPrefix(ctx, nginxInfo)

	return nginxInfo
}

func parseNginxVersion(line string) string {
	return strings.TrimPrefix(versionRegex.FindString(line), "nginx/")
}

func parseConfigureArguments(line string) map[string]interface{} {
	// need to check for empty strings
	flags := strings.Split(line[len("configure arguments:"):], " --")
	result := make(map[string]interface{})

	for _, flag := range flags {
		vals := strings.Split(flag, "=")
		if isFlag(vals) {
			result[vals[0]] = true
		} else if isKeyValueFlag(vals) {
			result[vals[0]] = vals[1]
		}
	}

	return result
}

func nginxPrefix(ctx context.Context, nginxInfo *model.ProcessInfo) string {
	var prefix string

	if nginxInfo.ConfigureArgs["prefix"] != nil {
		var ok bool
		prefix, ok = nginxInfo.ConfigureArgs["prefix"].(string)
		if !ok {
			slog.DebugContext(ctx, "Failed to cast nginxInfo prefix to string")
		}
	} else {
		prefix = "/usr/local/nginx"
	}

	return prefix
}

func isFlag(vals []string) bool {
	return len(vals) == flagLen && vals[0] != ""
}

func isKeyValueFlag(vals []string) bool {
	return len(vals) == keyValueLen
}
