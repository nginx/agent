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
)

const (
	headerTemplate = `worker_processes  1;

error_log  /opt/homebrew/var/log/nginx/error.log;

events {
    worker_connections  1024;
}


http {
    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                  '$status $body_bytes_sent "$http_referer" '
                  '"$http_user_agent" "$http_x_forwarded_for" '
                  '"$bytes_sent" "$request_length" "$request_time" '
                  '"$gzip_ratio" $server_protocol ';

    access_log  /opt/homebrew/var/log/nginx/access.log main;

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
	if _, err := file.WriteString(headerTemplate); err != nil {
		return nil, fmt.Errorf("error writing to file %w", err)
	}

	// Write server blocks until the file size reaches the target size
	for {
		// Generate random values
		port := generateRandomPort()
		serverName := fmt.Sprintf("%s.com", generateRandomString(12))
		proxyPass := fmt.Sprintf("http://%s.com", generateRandomString(8))

		// Write the server block to the file
		block := fmt.Sprintf(serverBlockTemplate, port, serverName, proxyPass)
		if _, err := file.WriteString(block); err != nil {
			return nil, fmt.Errorf("error writing server block to file %w", err)
		}

		// Check the file size
		info, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("error getting file info %w", err)
		}
		if info.Size() >= targetSize {
			break
		}
	}

	if _, err := file.WriteString(footerTemplate); err != nil {
		return nil, fmt.Errorf("error writing to file %w", err)
	}

	// Verify the file size and adjust if necessary
	return file.Stat()
	
}

func generateRandomPort() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(65535-49152+1))
	return int(n.Int64()) + 49152
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}
