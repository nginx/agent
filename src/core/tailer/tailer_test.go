/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package tailer

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trivago/grok"
)

const (
	NGINX_ACCESS = "%{NGINX_ACCESS}"
)

var defaultPatterns = map[string]string{
	"DEFAULT": `%{IPORHOST:remote_addr} - %{USERNAME:remote_user} \[%{HTTPDATE:time_local}\] \"%{DATA:request}\" %{INT:status} %{NUMBER:body_bytes_sent} \"%{DATA:http_referer}\" \"%{DATA:http_user_agent}\"`,
}

func TestNewNginxAccessItem(t *testing.T) {
	actual, err := NewNginxAccessItem(map[string]string{"bytes_sent": "456"})
	assert.Nil(t, err)
	assert.Equal(t, "456", actual.BytesSent)
}

func TestGrok(t *testing.T) {
	g, err := grok.New(grok.Config{
		NamedCapturesOnly: true,
		Patterns:          defaultPatterns,
	})
	require.Nil(t, err)

	parsed, err := g.ParseString(
		"%{DEFAULT}",
		`127.0.0.1 - - [04/Nov/2020:19:40:38 +0000] "GET /500 HTTP/1.1" 500 4 "-" "curl/7.64.1"`,
	)
	require.Nil(t, err)
	assert.Equal(
		t,
		map[string]string{
			"body_bytes_sent": "4",
			"http_referer":    "-",
			"http_user_agent": "curl/7.64.1",
			"remote_addr":     "127.0.0.1",
			"remote_user":     "-",
			"request":         "GET /500 HTTP/1.1",
			"status":          "500",
			"time_local":      "04/Nov/2020:19:40:38 +0000",
		},
		parsed,
	)
}

func TestTailer(t *testing.T) {
	errorLogFile, _ := os.CreateTemp(os.TempDir(), "error.log")
	logLine := `2015/07/15 05:56:30 [info] 28386#28386: *94160 client 10.196.158.41 closed keepalive connection`

	tailer, err := NewTailer(errorLogFile.Name())
	require.Nil(t, err)

	timeoutDuration := time.Millisecond * 300

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)

	data := make(chan string, 100)
	
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func () {
		tailer.Tail(ctx, data)
		wg.Done()
	}()
	
	time.Sleep(time.Millisecond * 100)
	_, err = errorLogFile.WriteString(logLine)
	if err != nil {
		t.Fatalf("Error writing data to error log")
	}
	errorLogFile.Close()

	var count int
T:
	for {
		select {
		case d := <-data:
			assert.Equal(t, logLine, d)
			count++
		case <-time.After(timeoutDuration):
			break T
		case <-ctx.Done():
			break T
		}
	}
	cancel()

	wg.Wait()

	os.Remove(errorLogFile.Name())
	assert.Equal(t, 1, count)

	time.Sleep(500 * time.Millisecond)
}

func TestPatternTailer(t *testing.T) {
	accessLogFile, _ := os.CreateTemp(os.TempDir(), "access.log")
	logLine := "127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 500 98 \"-\" \"Go-http-client/1.1\" \"-\"\n"

	tailer, err := NewPatternTailer(accessLogFile.Name(), defaultPatterns)
	require.Nil(t, err)

	timeoutDuration := time.Millisecond * 300
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	data := make(chan map[string]string, 100)
	go tailer.Tail(ctx, data)

	time.Sleep(time.Millisecond * 100)
	_, err = accessLogFile.WriteString(logLine)
	if err != nil {
		t.Fatalf("Error writing data to access log")
	}
	accessLogFile.Close()

	var count int
T:
	for {
		select {
		case <-data:
			count++
		case <-time.After(timeoutDuration):
			break T
		case <-ctx.Done():
			break T
		}
	}

	os.Remove(accessLogFile.Name())
	assert.Equal(t, 1, count)
}

func TestLTSVTailer(t *testing.T) {
	accessLogFile, _ := os.CreateTemp(os.TempDir(), "access.log")
	logLine := "remote_addr:127.0.0.1\t remote_user:-\t time_local:04/Nov/2020:19:40:38 +0000\t request:GET /500 HTTP/1.1\t status:500\t body_bytes_sent:4\t http_referer:-\t http_user_agent:curl/7.64.1\n"

	tailer, err := NewLTSVTailer(accessLogFile.Name())
	require.Nil(t, err)

	timeoutDuration := time.Millisecond * 300
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	data := make(chan map[string]string, 100)
	go tailer.Tail(ctx, data)

	time.Sleep(time.Millisecond * 100)
	_, err = accessLogFile.WriteString(logLine)
	if err != nil {
		t.Fatalf("Error writing data to access log")
	}
	accessLogFile.Close()

	var count int
	var res map[string]string
T:
	for {
		select {
		case r := <-data:
			res = r
			count++
		case <-time.After(timeoutDuration):
			break T
		case <-ctx.Done():
			break T
		}
	}

	os.Remove(accessLogFile.Name())
	assert.Equal(t, 1, count)
	assert.Equal(
		t,
		map[string]string{
			"body_bytes_sent": "4",
			"http_referer":    "-",
			"http_user_agent": "curl/7.64.1",
			"remote_addr":     "127.0.0.1",
			"remote_user":     "-",
			"request":         "GET /500 HTTP/1.1",
			"status":          "500",
			"time_local":      "04/Nov/2020:19:40:38 +0000",
		},
		res,
	)
}
