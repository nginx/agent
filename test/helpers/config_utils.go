// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	headerTemplate = `worker_processes  1;

error_log  /var/log/nginx/error.log;

events {
    worker_connections  1024;
}


http {
    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                  '$status $body_bytes_sent "$http_referer" '
                  '"$http_user_agent" "$http_x_forwarded_for" '
                  '"$bytes_sent" "$request_length" "$request_time" '
                  '"$gzip_ratio" $server_protocol ';

    access_log  /var/log/nginx/access.log main;

    sendfile        on;

    keepalive_timeout  65;
`
	serverBlockTemplate = `server {
    listen %d;
    server_name %s;
    location / {
        proxy_pass %s;
    }
}
`
	footerTemplate = `}`
)

func GenerateConfig(t testing.TB, outputFile string, targetSize int64) (fs.FileInfo, error) {
	t.Helper()

	file, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("error creating file %w", err)
	}
	defer file.Close()

	// Write the header
	if _, headerErr := file.WriteString(headerTemplate); headerErr != nil {
		return nil, fmt.Errorf("error writing to file %w", headerErr)
	}

	const serverLength = 12
	const proxyLength = 8
	// Write server blocks until the file size reaches the target size
	for {
		// Generate random values
		port := generateRandomPort()

		server, serverErr := generateRandomString(serverLength)
		require.NoError(t, serverErr)

		serverName := server + ".com"

		proxy, proxyErr := generateRandomString(proxyLength)
		require.NoError(t, proxyErr)

		proxyPass := fmt.Sprintf("http://%s.com", proxy)

		// Write the server block to the file
		block := fmt.Sprintf(serverBlockTemplate, port, serverName, proxyPass)
		if _, blockErr := file.WriteString(block); blockErr != nil {
			return nil, fmt.Errorf("error writing server block to file %w", blockErr)
		}

		// Check the file size
		info, fileStatErr := file.Stat()
		if fileStatErr != nil {
			return nil, fmt.Errorf("error getting file info %w", fileStatErr)
		}
		if info.Size() >= targetSize {
			break
		}
	}

	if _, footerErr := file.WriteString(footerTemplate); footerErr != nil {
		return nil, fmt.Errorf("error writing to file %w", footerErr)
	}

	// Verify the file size and adjust if necessary
	return file.Stat()
}

func generateRandomPort() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(65535-49152+1))
	const minPort = 49152

	return int(n.Int64()) + minPort
}

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b)[:length], err
}
