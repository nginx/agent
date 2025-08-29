// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package metrics

import (
	"context"
	"os"
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/suite"
)

type MetricsTestSuite struct {
	suite.Suite
	ctx            context.Context
	teardownTest   func(testing.TB)
	metricFamilies map[string]*dto.MetricFamily
}

func (s *MetricsTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.teardownTest = utils.SetupMetricsTest(s.T())
	utils.WaitUntilNextScrapeCycle(s.T(), s.ctx)
}

func (s *MetricsTestSuite) SetupTest() {
	s.metricFamilies = utils.ScrapeCollectorMetricFamilies(s.T(), s.ctx, utils.MockCollectorStack.Otel)
}

func (s *MetricsTestSuite) TearDownTest() {
	if s.T().Skipped() {
		return
	}
	utils.WaitUntilNextScrapeCycle(s.T(), s.ctx)
}

func (s *MetricsTestSuite) TearDownSuite() {
	s.teardownTest(s.T())
}

// Check that the NGINX request count metric increases after generating some requests
func (s *MetricsTestSuite) TestNginx_TestRequestCount() {
	metricName := "nginx_http_request_count"
	family := s.metricFamilies[metricName]
	s.Require().NotNil(family)

	var baselineMetric []float64
	baselineMetric = append(baselineMetric, utils.SumMetricFamily(family))
	s.T().Logf("NGINX HTTP request count total: %v", baselineMetric[0])

	requestCount := 50
	utils.GenerateMetrics(s.ctx, s.T(), utils.MockCollectorStack.Agent, requestCount, "")

	got := utils.PollingForMetrics(s.T(), s.ctx, metricName, utils.LabelFilter{
		Key:    "",
		Values: []string{},
	}, baselineMetric)

	s.T().Logf("NGINX HTTP request count total: %v", got[0])
	s.Require().Greater(got[0], baselineMetric[0])
}

// Check that the NGINX response count metric increases after generating some requests for each response code
func (s *MetricsTestSuite) TestNginx_TestResponseCode() {
	if os.Getenv("IMAGE_PATH") != "/nginx/agent" {
		s.T().Skip("Skipping test for NGINX OSS specific metric")
	}

	metricName := "nginx_http_response_count"
	family := s.metricFamilies[metricName]
	s.Require().NotNil(family)

	responseCodes := []string{"1xx", "2xx", "3xx", "4xx", "5xx"}
	respBaseline := make([]float64, len(responseCodes))
	for code := range responseCodes {
		respBaseline[code] = utils.SumMetricFamilyLabel(family, "nginx_status_range", responseCodes[code])
		s.T().Logf("NGINX HTTP response code %s total: %v", responseCodes[code], respBaseline[code])
		s.Require().NotNil(respBaseline[code])
	}

	requestCount := 50
	for code := range responseCodes {
		utils.GenerateMetrics(s.ctx, s.T(), utils.MockCollectorStack.Agent, requestCount, responseCodes[code])
	}

	got := utils.PollingForMetrics(s.T(), s.ctx, metricName, utils.LabelFilter{
		Key:    "nginx_status_range",
		Values: responseCodes,
	}, respBaseline)
	for code := range responseCodes {
		s.T().Logf("NGINX HTTP response code %s total: %v", responseCodes[code], got[code])
		s.Require().Greater(got[code], respBaseline[code])
	}
}

// Check that the system CPU utilization metric increases after generating some requests
func (s *MetricsTestSuite) TestHostMetrics_TestSystemCPUUtilization() {
	family := s.metricFamilies["system_cpu_utilization"]
	s.Require().NotNil(family)

	states := []string{"system", "user"}
	respBaseline := make([]float64, len(states))
	for state := range states {
		respBaseline[state] = utils.SumMetricFamilyLabel(family, "state", states[state])
		s.T().Logf("CPU utilization for %s: %v", states[state], respBaseline[state])
		s.Require().NotNil(respBaseline[state])
	}

	utils.GenerateMetrics(s.ctx, s.T(), utils.MockCollectorStack.Agent, 20, "2xx")

	got := utils.PollingForMetrics(s.T(), s.ctx,
		"system_cpu_utilization", utils.LabelFilter{
			Key:    "state",
			Values: states,
		}, respBaseline)

	for state := range states {
		s.T().Logf("CPU utilization for %s: %v", states[state], got[state])
		s.Require().Greater(got[state], respBaseline[state])
	}
}

// Check that the system memory usage metric changes after generating some requests
func (s *MetricsTestSuite) TestHostMetrics_TestSystemMemoryUsage() {
	family := s.metricFamilies["system_memory_usage"]
	s.Require().NotNil(family)

	states := []string{"free", "used"}
	respBaseline := make([]float64, len(states))
	for state := range states {
		respBaseline[state] = utils.SumMetricFamilyLabel(family, "state", states[state])
		s.T().Logf("Memory %s: %v", states[state], respBaseline[state])
		s.Require().NotNil(respBaseline[state])
	}

	utils.GenerateMetrics(s.ctx, s.T(), utils.MockCollectorStack.Agent, 20, "2xx")

	got := utils.PollingForMetrics(s.T(), s.ctx, "system_memory_usage", utils.LabelFilter{
		Key:    "state",
		Values: states,
	}, respBaseline)

	for state := range states {
		s.T().Logf("Memory %s: %v", states[state], got[state])
	}
	s.Require().Less(got[0], respBaseline[0])
	s.Require().Greater(got[1], respBaseline[1])
}

func TestMetricsTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsTestSuite))
}
