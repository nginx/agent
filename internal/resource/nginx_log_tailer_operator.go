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
	// Line is over 120 characters long, regex needs to be on one line so needs to be ignored by linter
	//nolint:lll // needs to be on one line
	reloadErrorList = re.MustCompile(`\d{1,4}\/\d{1,2}\/\d{1,2} ([0-1][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9] ?(nginx\:|) (\[emerg\]|\[alert\]|\[crit\])`)
	//nolint:lll // needs to be on one line
	warningRegex    = re.MustCompile(`\d{1,4}\/\d{1,2}\/\d{1,2} ([0-1][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9] ?(nginx\:|) (\[warn\])`)
	ignoreErrorList = re.MustCompile(`.*(usage report| license expired).*`)
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
				errorChannel <- fmt.Errorf("%s", d)
				return
			}
		case <-ctxWithTimeout.Done():
			errorChannel <- nil
			return
		}
	}
}

func (l *NginxLogTailerOperator) doesLogLineContainError(line string) bool {
	if ignoreErrorList.MatchString(line) {
		return false
	} else if (l.agentConfig.DataPlaneConfig.Nginx.TreatWarningsAsErrors && warningRegex.MatchString(line)) ||
		reloadErrorList.MatchString(line) {
		return true
	}

	return false
}
