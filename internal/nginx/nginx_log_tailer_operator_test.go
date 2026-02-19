// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/types"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/require"
)

const (
	errorLogLine = "2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)"
	critLogLine  = "2023/07/07 11:30:00 [crit] 123456#123456: *1 connect() to unix:/test/test/test/test.sock failed" +
		" (2: No such file or directory) while connecting to upstream, client: 0.0.0.0, server: _, request:" +
		" \"POST /test HTTP/2.0\", upstream: \"grpc://unix:/test/test/test/test.sock:\", host: \"0.0.0.0:0\""
	alertLogLine = "2023/06/20 11:01:56 [alert] 4138#4138: open() \"/var/log/nginx/error.log\" failed (13: Permission" +
		" denied)"
	warningLogLine = "2023/03/14 14:16:23 nginx: [warn] 2048 worker_connections exceed open file resource limit: 1024"
)

func TestLogOperator_Tail(t *testing.T) {
	ctx := context.Background()

	errorLogFile := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLogFile.Name())

	tests := []struct {
		out              *bytes.Buffer
		expected         error
		name             string
		errorLogs        string
		errorLogContents string
	}{
		{
			name:             "Test 1: No errors in logs",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: "",
			expected:         nil,
		},
		{
			name:             "Test 2: Error in error logs",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: errorLogLine,
			expected:         fmt.Errorf("%s", errorLogLine),
		},
		{
			name:             "Test 3: Warning in error logs",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: warningLogLine,
			expected:         fmt.Errorf("%s", warningLogLine),
		},
		{
			name:      "Test 4: ignore error log: usage report ",
			out:       bytes.NewBufferString(""),
			errorLogs: errorLogFile.Name(),
			errorLogContents: "2025/06/25 15:08:04 [error] 123456#123456: certificate verify error: " +
				"(10:certificate has expired) during usage report",
			expected: nil,
		},
		{
			name:      "Test 5: ignore error log: license expired ",
			out:       bytes.NewBufferString(""),
			errorLogs: errorLogFile.Name(),
			errorLogContents: "2025/06/25 15:07:24 [alert] 123456#123456: license expired; the grace period " +
				"will end in 71 days",
			expected: nil,
		},
		{
			name:             "Test 6: ignore error log: license expired ",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: "2024/12/25 15:00:04 [error] 123456#123456: server returned 400 during usage report",
			expected:         nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			agentConfig := types.AgentConfig()
			agentConfig.DataPlaneConfig.Nginx.ReloadMonitoringPeriod = 5 * time.Second
			operator := NewLogTailerOperator(agentConfig)

			logErrorChannel := make(chan error, len(test.errorLogs))
			defer close(logErrorChannel)

			var wg sync.WaitGroup
			wg.Add(1)
			go func(testErr error) {
				defer wg.Done()
				operator.Tail(ctx, test.errorLogs, logErrorChannel)
				err := <-logErrorChannel
				assert.Equal(tt, testErr, err)
			}(test.expected)

			time.Sleep(100 * time.Millisecond)

			if test.errorLogContents != "" {
				_, err := errorLogFile.WriteString(test.errorLogContents)
				require.NoError(tt, err, "Error writing data to error log file")
			}

			wg.Wait()
		})
	}
}

func TestLogOperator_doesLogLineContainError(t *testing.T) {
	tests := []struct {
		name             string
		line             string
		treatWarnAsError bool
		expected         bool
	}{
		{
			name:             "Test 1: no error in line",
			line:             "",
			treatWarnAsError: false,
			expected:         false,
		},
		{
			name:             "Test 2: emerg in line",
			line:             errorLogLine,
			treatWarnAsError: false,
			expected:         true,
		},
		{
			name:             "Test 3: crit in line",
			line:             critLogLine,
			treatWarnAsError: false,
			expected:         true,
		},
		{
			name:             "Test 4: alert in line",
			line:             alertLogLine,
			treatWarnAsError: false,
			expected:         true,
		},
		{
			name:             "Test 5: warn in line, treat warn as error off",
			line:             warningLogLine,
			treatWarnAsError: false,
			expected:         false,
		},
		{
			name:             "Test 6: warn in line, treat warn as error on",
			line:             warningLogLine,
			treatWarnAsError: true,
			expected:         true,
		},
		{
			name: "Test 7: ignore error, usage report",
			line: "2025/06/25 15:08:04 [error] 123456#123456: certificate verify error: " +
				"(10:certificate has expired) during usage report",
			treatWarnAsError: false,
			expected:         false,
		},
		{
			name: "Test 8: ignore error, license expired",
			line: "2025/06/25 15:07:24 [alert] 123456#123456: license expired; the grace period " +
				"will end in 71 days",
			treatWarnAsError: false,
			expected:         false,
		},
		{
			name:             "Test 9: check log is ignored, [emerg] in log but not at emerg level",
			line:             "2025/06/25 15:07:24 [info] 123456#123456: checking [emerg] is ignored",
			treatWarnAsError: false,
			expected:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			agentConfig := types.AgentConfig()
			agentConfig.DataPlaneConfig.Nginx.TreatWarningsAsErrors = test.treatWarnAsError
			operator := NewLogTailerOperator(agentConfig)

			result := operator.doesLogLineContainError(test.line)
			assert.Equal(tt, test.expected, result)
		})
	}
}
