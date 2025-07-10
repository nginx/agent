// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package load

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
)

func TestMetric10kDPS(t *testing.T) {
	performanceResultsSummary := &testbed.PerformanceResults{}

	binary := parseBinary(os.Getenv("PACKAGE_NAME"))

	otelTestBedCollector, err := filepath.Abs("../../" + binary)
	require.NoError(t, err)

	t.Logf("Absolute path is %s", otelTestBedCollector)

	testbed.GlobalConfig.DefaultAgentExeRelativeFile = otelTestBedCollector

	name := fmt.Sprintf("OTLP-%s-%s", runtime.GOOS, binary)
	sender := testbed.NewOTLPMetricDataSender(testbed.DefaultHost, 4317)
	receiver := testbed.NewOTLPDataReceiver(5643)
	receiver = receiver.WithCompression("none")

	t.Run(name, func(t *testing.T) {
		require.NoError(t, err)

		options := testbed.LoadOptions{
			DataItemsPerSecond: 10_000,
			ItemsPerBatch:      100,
			Parallel:           1,
		}

		agentProc := NewNginxAgentProcessCollector(WithEnvVar("GOMAXPROCS", "10"))

		dataProvider := testbed.NewPerfTestDataProvider(options)
		tc := testbed.NewTestCase(
			t,
			dataProvider,
			sender,
			receiver,
			agentProc,
			&testbed.PerfTestValidator{},
			performanceResultsSummary,
			// this resource spec is overwritten in the agent process collector
			testbed.WithResourceLimits(testbed.ResourceSpec{}),
		)

		t.Cleanup(tc.Stop)

		tc.StartBackend()
		tc.StartAgent()

		tc.StartLoad(options)

		tc.WaitFor(func() bool { return tc.LoadGenerator.IsReady() }, "load generator ready")

		tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() > 0 }, "load generator started")

		tc.Sleep(tc.Duration)

		tc.StopLoad()

		tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
			"all data items received")

		tc.ValidateData()
	})

	defer testbed.SaveResults(performanceResultsSummary)
}

func parseBinary(s string) string {
	if s == "" {
		return "build/nginx-agent"
	}

	prefixes := []string{"./agent", "./", "agent"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimPrefix(s, prefix)
		}
	}

	return s
}
