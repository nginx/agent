// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package load

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
)

func TestMetric10kDPS(t *testing.T) {
	performanceResultsSummary := &testbed.PerformanceResults{}

	otelTestBedCollector, err := filepath.Abs("../../build/nginx-agent")
	require.NoError(t, err)

	testbed.GlobalConfig.DefaultAgentExeRelativeFile = otelTestBedCollector

	name := "OTLP"
	sender := testbed.NewOTLPMetricDataSender(testbed.DefaultHost, 4317)
	receiver := testbed.NewOTLPDataReceiver(5643)

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
			testbed.WithResourceLimits(
				testbed.ResourceSpec{
					ExpectedMaxCPU: 20,
					ExpectedMaxRAM: 200,
				}),
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
