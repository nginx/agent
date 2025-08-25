// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"bufio"
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDialViaHTTPProxy_NoProxy(t *testing.T) {
	// This test attempts to connect directly to a known open port (localhost:80)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	proxyConf := &config.Proxy{
		URL:     "",
		Timeout: 2 * time.Second,
	}
	_, err := DialViaHTTPProxy(ctx, proxyConf, "localhost:80")
	require.Error(t, err, "expected failure with empty proxy URL")
}

func TestDialViaHTTPProxy_InvalidProxy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	proxyConf := &config.Proxy{
		URL:     "http://invalid:9999",
		Timeout: 2 * time.Second,
	}
	_, err := DialViaHTTPProxy(ctx, proxyConf, "localhost:80")
	require.Error(t, err, "expected failure with invalid proxy")
}

// To fully test with a real proxy, set the env var TEST_HTTP_PROXY_URL and have a proxy listening.
func TestDialViaHTTPProxy_RealProxy(t *testing.T) {
	proxyURL := os.Getenv("TEST_HTTP_PROXY_URL")
	if proxyURL == "" {
		t.Skip("TEST_HTTP_PROXY_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proxyConf := &config.Proxy{
		URL:     proxyURL,
		Timeout: 3 * time.Second,
	}
	conn, err := DialViaHTTPProxy(ctx, proxyConf, "example.com:80")
	require.NoError(t, err, "failed to connect via proxy")
	defer conn.Close()

	// Basic write/read to check tunnel
	if _, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")); err != nil {
		t.Errorf("failed to write to tunnel: %v", err)
	}
	buf := make([]byte, 128)
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)), "failed to set read deadline")

	_, err = conn.Read(buf)
	if err != nil && err != context.DeadlineExceeded && !os.IsTimeout(err) {
		t.Errorf("failed to read from tunnel: %v", err)
	}
}

//nolint:noctx,revive //No need for ctx in test cases.
func TestDialViaHTTPProxy_BearerTokenHeader(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen")
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			t.Errorf("Failed to accept connection: %v", acceptErr)
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		headerLines := readHeaders(reader)

		if !hasBearerHeader(headerLines, "testtoken") {
			_, writeErr := conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n" +
				"Proxy-Authenticate: Bearer realm=\"nginx-agent\"\r\n\r\n"))
			if writeErr != nil {
				t.Errorf("Warning: mock proxy failed to write 407 response: %v", writeErr)
			}
			t.Errorf("Proxy-Authorization Bearer header with token 'testtoken' was not found in received request")

			return
		}

		_, writeErr := conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if writeErr != nil {
			t.Errorf("mock proxy failed to write 200 OK response: %v", writeErr)
			return
		}
		close(done)
	}()

	proxyConf := &config.Proxy{
		URL:        "http://" + ln.Addr().String(),
		AuthMethod: "bearer",
		Token:      "testtoken",
		Timeout:    2 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = DialViaHTTPProxy(ctx, proxyConf, "example.com:443")

	if err != nil && err != context.DeadlineExceeded && !os.IsTimeout(err) {
		require.NoError(t, err, "DialViaHTTPProxy returned an unexpected non-timeout error")
	}

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		if err == context.DeadlineExceeded {
			t.Fatalf("Test timed out (DialViaHTTPProxy context deadline exceeded): %v", err)
		}
		t.Fatalf("Test timed out: Proxy-Authorization Bearer header was not sent or verified within %v", err)
	}
}

func readHeaders(reader *bufio.Reader) []string {
	var headerLines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
		headerLines = append(headerLines, line)
	}

	return headerLines
}

func hasBearerHeader(headerLines []string, token string) bool {
	expected := "Proxy-Authorization: Bearer " + token
	for _, h := range headerLines {
		if strings.HasPrefix(h, expected) {
			return true
		}
	}

	return false
}

func TestDialViaHTTPProxy_InvalidCAPath(t *testing.T) {
	proxyConf := &config.Proxy{
		URL:     "https://localhost:9999",
		TLS:     &config.TLSConfig{Ca: "/invalid/path/to/ca.pem"},
		Timeout: 1 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := DialViaHTTPProxy(ctx, proxyConf, "example.com:443")
	require.Error(t, err, "expected error for invalid CA path")
}

func TestDialViaHTTPProxy_MissingCertKey(t *testing.T) {
	proxyConf := &config.Proxy{
		URL:     "https://localhost:9999",
		TLS:     &config.TLSConfig{}, // No cert/key
		Timeout: 1 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := DialViaHTTPProxy(ctx, proxyConf, "example.com:443")
	require.Error(t, err, "expected error for missing cert")
}

func TestDialViaHTTPProxy_InvalidProxyURL(t *testing.T) {
	proxyConf := &config.Proxy{
		URL:     "://bad-url",
		Timeout: 1 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := DialViaHTTPProxy(ctx, proxyConf, "example.com:443")
	require.Error(t, err, "expected error for invalid proxy URL")
}
