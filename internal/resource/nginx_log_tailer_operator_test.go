// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/require"
)

const (
	errorLogLine   = "2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)"
	warningLogLine = "2023/03/14 14:16:23 nginx: [warn] 2048 worker_connections exceed open file resource limit: 1024"
)

func TestLogOperator_Tail(t *testing.T) {
	ctx := context.Background()

	errorLogFile := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLogFile.Name())

	tests := []struct {
		out              *bytes.Buffer
		err              error
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
			err:              nil,
			expected:         nil,
		},
		{
			name:             "Test 2: Error in error logs",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: errorLogLine,
			err:              nil,
			expected:         errors.Join(fmt.Errorf(errorLogLine)),
		},
		{
			name:             "Test 3: Warning in error logs",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: warningLogLine,
			err:              nil,
			expected:         errors.Join(fmt.Errorf(warningLogLine)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operator := NewLogTailerOperator(types.AgentConfig())

			logErrorChannel := make(chan error, len(test.errorLogs))
			defer close(logErrorChannel)

			var wg sync.WaitGroup
			wg.Add(1)
			operator.Tail(ctx, test.errorLogs, logErrorChannel)
			go func(testErr error) {
				defer wg.Done()
				err := <-logErrorChannel
				assert.Equal(t, testErr, err)
			}(test.err)

			time.Sleep(200 * time.Millisecond)

			if test.errorLogContents != "" {
				_, err := errorLogFile.WriteString(test.errorLogContents)
				require.NoError(t, err, "Error writing data to error log file")
			}

			wg.Wait()
		})
	}
}
