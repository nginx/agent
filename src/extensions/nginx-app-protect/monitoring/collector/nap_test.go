/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collector_test

import (
	"context"
	"fmt"
	"log/syslog"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/mcuadros/go-syslog.v2/format"

	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/collector"
)

func TestNAPCollect(t *testing.T) {
	var logwriter *syslog.Writer
	var testPort = 22514
	var testIP = "127.0.0.1"

	waf, err := collector.NewNAPCollector(&collector.NAPConfig{
		SyslogIP:   testIP,
		SyslogPort: testPort,
	})
	require.NoError(t, err, "Error while setting up syslog: %v", err)

	collect := make(chan *monitoring.RawLog, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go waf.Collect(ctx, wg, collect)

	logwriter, err = syslog.Dial("tcp4", testIP+":"+strconv.Itoa(testPort), syslog.LOG_INFO|syslog.LOG_USER, "test")
	if err != nil {
		t.Errorf("Error while dialing into syslog: %v", err)
	}

	msg := "Hello Logs!"

	err = logwriter.Info(msg)
	if err != nil {
		t.Errorf("Error while writing to syslog: %v", err)
	}

	readline := <-collect
	assert.True(t, strings.Contains(readline.Logline, msg), fmt.Sprintf("Wrote `%s`, got `%s` \n", msg, readline.Logline))
}

func TestNAPSyslogParser(t *testing.T) {
	find := `<131>Oct  7 18:19:38 9a44c32084d2 ASM:unit_hostname="itay-108-117.f5net.com""`
	expected := `unit_hostname="itay-108-117.f5net.com""`

	f := format.RFC3164{}
	parser := f.GetParser([]byte(find))
	err := parser.Parse()

	if err != nil {
		t.Errorf("Error while parsing syslog: %v", err)
	}

	assert.Equal(t, expected, parser.Dump()["content"], "Could not parse syslog message")
}
