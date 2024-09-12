// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"fmt"
	"log/slog"
	re "regexp"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
)

const logTailerChannelSize = 1024

type NginxLogTailerOperator struct {
	agentConfig *config.Config
}

var _ logTailerOperator = (*NginxLogTailerOperator)(nil)

var (
	reloadErrorList = []*re.Regexp{
		re.MustCompile(`.*\[emerg\].*`),
		re.MustCompile(`.*\[alert\].*`),
		re.MustCompile(`.*\[crit\].*`),
	}
	warningRegex = re.MustCompile(`.*\[warn\].*`)
)

func NewLogTailerOperator(agentConfig *config.Config) *NginxLogTailerOperator {
	return &NginxLogTailerOperator{
		agentConfig: agentConfig,
	}
}

func (l *NginxLogTailerOperator) Tail(ctx context.Context, errorLog string, errorChannel chan error) {
	t, err := nginx.NewTailer(errorLog)
	if err != nil {
		slog.ErrorContext(ctx, "Unable to tail error log after NGINX reload", "log_file", errorLog, "error", err)
		// this is not an error in the logs, ignoring tailing
		errorChannel <- nil

		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, l.agentConfig.DataPlaneConfig.Nginx.ReloadMonitoringPeriod)
	defer cancel()

	slog.DebugContext(ctxWithTimeout, "Monitoring NGINX error log file for any errors", "file", errorLog)

	data := make(chan string, logTailerChannelSize)
	go t.Tail(ctxWithTimeout, data)

	for {
		select {
		case d := <-data:
			if l.doesLogLineContainError(d) {
				errorChannel <- fmt.Errorf(d)
				return
			}
		case <-ctxWithTimeout.Done():
			errorChannel <- nil
			return
		}
	}
}

func (l *NginxLogTailerOperator) doesLogLineContainError(line string) bool {
	if l.agentConfig.DataPlaneConfig.Nginx.TreatWarningsAsErrors && warningRegex.MatchString(line) {
		return true
	}

	for _, errorRegex := range reloadErrorList {
		if errorRegex.MatchString(line) {
			return true
		}
	}

	return false
}
