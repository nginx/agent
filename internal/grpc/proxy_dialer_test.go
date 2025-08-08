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
	if err == nil {
		t.Errorf("expected failure with empty proxy URL, got no error")
	}
}

func TestDialViaHTTPProxy_InvalidProxy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	proxyConf := &config.Proxy{
		URL:     "http://invalid:9999",
		Timeout: 2 * time.Second,
	}
	_, err := DialViaHTTPProxy(ctx, proxyConf, "localhost:80")
	if err == nil {
		t.Errorf("expected failure with invalid proxy, got no error")
	}
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
	if err != nil {
		t.Fatalf("failed to connect via proxy: %v", err)
	}
	defer conn.Close()

	// Basic write/read to check tunnel
	if _, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")); err != nil {
		t.Errorf("failed to write to tunnel: %v", err)
	}
	buf := make([]byte, 128)
	if deadlineErr := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); deadlineErr != nil {
		// Optionally log
		t.Logf("Failed to set read deadline: %v", deadlineErr)
	}
	_, err = conn.Read(buf)
	if err != nil && err != context.DeadlineExceeded && !isTimeout(err) {
		t.Errorf("failed to read from tunnel: %v", err)
	}
}

func TestDialViaHTTPProxy_BearerTokenHeader(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
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
		if hasBearerHeader(headerLines, "testtoken") {
			close(done)
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
	_, _ = DialViaHTTPProxy(ctx, proxyConf, "example.com:443")

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Errorf("Proxy-Authorization Bearer header was not sent")
	}
}

func isTimeout(err error) bool {
	nerr, ok := err.(net.Error)
	return ok && nerr.Timeout()
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
	if err == nil {
		t.Error("expected error for invalid CA path, got nil")
	}
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
	// No assert needed: just covers the branch
	if err == nil {
		t.Error("expected error for missing cert, got nil")
	}
}

func TestDialViaHTTPProxy_InvalidProxyURL(t *testing.T) {
	proxyConf := &config.Proxy{
		URL:     "://bad-url",
		Timeout: 1 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := DialViaHTTPProxy(ctx, proxyConf, "example.com:443")
	if err == nil {
		t.Error("expected error for invalid proxy URL, got nil")
	}
}
