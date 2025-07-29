// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/nginx/agent/v3/internal/config"
)

// DialViaHTTPProxy establishes a tunnel via HTTP CONNECT and returns a net.Conn
func DialViaHTTPProxy(ctx context.Context, proxyConf *config.Proxy, targetAddr string) (net.Conn, error) {
	proxyURL, err := url.Parse(proxyConf.URL)
	if err != nil {
		return nil, wrapProxyError(ctx, "Invalid proxy URL", err, proxyConf.URL)
	}

	dialConn, err := dialProxy(ctx, proxyURL, proxyConf)
	if err != nil {
		return nil, err
	}

	if err = writeConnectRequest(dialConn, targetAddr, proxyConf); err != nil {
		dialConn.Close()
		return nil, wrapProxyError(ctx, "Failed to write CONNECT request", err, proxyConf.URL)
	}

	resp, err := readConnectResponse(dialConn)
	if err != nil {
		dialConn.Close()
		return nil, wrapProxyError(ctx, "Failed to read CONNECT response", err, proxyConf.URL)
	}

	if err = validateProxyResponse(ctx, resp, dialConn); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Established proxy tunnel", "proxy_url", proxyConf.URL, "target_addr", targetAddr)

	return dialConn, nil
}

func buildProxyTLSConfig(proxyConf *config.Proxy) (*tls.Config, error) {
	tlsConf := &tls.Config{}
	if proxyConf.TLS == nil {
		return tlsConf, nil
	}

	if err := addRootCAs(tlsConf, proxyConf.TLS.Ca); err != nil {
		return nil, err
	}
	if err := addCertKeyPair(tlsConf, proxyConf.TLS.Cert, proxyConf.TLS.Key); err != nil {
		return nil, err
	}
	setServerName(tlsConf, proxyConf.TLS.ServerName)
	tlsConf.InsecureSkipVerify = proxyConf.TLS.SkipVerify

	return tlsConf, nil
}

func dialToProxyTLS(proxyURL *url.URL, tlsConf *tls.Config, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	return tls.DialWithDialer(dialer, "tcp", proxyURL.Host, tlsConf)
}

func dialToProxyTCP(ctx context.Context, proxyURL *url.URL, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	return dialer.DialContext(ctx, "tcp", proxyURL.Host)
}

func writeConnectRequest(conn net.Conn, targetAddr string, proxyConf *config.Proxy) error {
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: targetAddr},
		Host:   targetAddr,
		Header: make(http.Header),
	}
	if proxyConf.AuthMethod == "basic" && proxyConf.Username != "" && proxyConf.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(proxyConf.Username + ":" + proxyConf.Password))
		req.Header.Set("Proxy-Authorization", "Basic "+auth)
	} else if proxyConf.AuthMethod == "bearer" && proxyConf.Token != "" {
		req.Header.Set("Proxy-Authorization", "Bearer "+proxyConf.Token)
	}

	return req.Write(conn)
}

func readConnectResponse(conn net.Conn) (*http.Response, error) {
	return http.ReadResponse(bufio.NewReader(conn), nil)
}

func wrapProxyError(ctx context.Context, msg string, err error, proxyURL string) error {
	slog.ErrorContext(ctx, "Failed to connect via proxy", "proxyurl", proxyURL, "error", err)
	return fmt.Errorf("%s: %w", msg, err)
}

func dialProxy(ctx context.Context, proxyURL *url.URL, proxyConf *config.Proxy) (net.Conn, error) {
	if proxyURL.Scheme == "https" {
		tlsConf, err := buildProxyTLSConfig(proxyConf)
		if err != nil {
			return nil, wrapProxyError(ctx, "Failed to build TLS config", err, proxyConf.URL)
		}

		return dialToProxyTLS(proxyURL, tlsConf, proxyConf.Timeout)
	}

	return dialToProxyTCP(ctx, proxyURL, proxyConf.Timeout)
}

func validateProxyResponse(ctx context.Context, resp *http.Response, dialConn net.Conn) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		slog.ErrorContext(ctx, "Failed to discard response body", "error", err)
	}
	resp.Body.Close()
	dialConn.Close()

	return errors.New("proxy CONNECT failed: " + resp.Status)
}

func addRootCAs(tlsConf *tls.Config, caPath string) error {
	if caPath == "" {
		return nil
	}

	return appendRootCAs(tlsConf, caPath)
}

func addCertKeyPair(tlsConf *tls.Config, certPath, keyPath string) error {
	if certPath == "" || keyPath == "" {
		return nil
	}

	return appendCertKeyPair(tlsConf, certPath, keyPath)
}

func setServerName(tlsConf *tls.Config, serverName string) {
	if serverName != "" {
		tlsConf.ServerName = serverName
	}
}
