// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package accesslog

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNginxConfParsing(t *testing.T) {
	tests := []struct {
		name      string
		confPath  string
		expOutput string
		expErrMsg string
		shouldErr bool
	}{
		{
			name:     "basic NGINX config",
			confPath: filepath.Join("testdata", "basic.conf"),
			expOutput: `$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer"` +
				` "$http_user_agent" "$http_x_forwarded_for" "$bytes_sent" "$request_length" "$request_time"` +
				`"$gzip_ratio" $server_protocol `,
		},
		{
			name:      "no log_format NGINX config",
			confPath:  filepath.Join("testdata", "no-log-format.conf"),
			shouldErr: true,
			expErrMsg: "no log_format directive found",
		},
		{
			name:      "path to non-existent file",
			confPath:  filepath.Join("testdata", "does-not-exist"),
			shouldErr: true,
			expErrMsg: "NGINX config path [testdata/does-not-exist]",
		},
		{
			name:      "path to directory",
			confPath:  "testdata",
			shouldErr: true,
			expErrMsg: "NGINX config path argument is a directory",
		},
		{
			name:      "invalid NGINX conf",
			confPath:  filepath.Join("testdata", "invalid.conf"),
			shouldErr: true,
			expErrMsg: "parse NGINX config",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actualLogFormat, err := logFormatFromNginxConf(test.confPath)
			if test.shouldErr {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), test.expErrMsg)
			} else {
				require.NoError(tt, err)
				assert.Equal(tt, test.expOutput, actualLogFormat)
			}
		})
	}
}
