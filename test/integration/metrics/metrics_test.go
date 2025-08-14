// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package metrics

import (
	"context"
	"testing"
	"time"

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
	time.Sleep(30 * time.Second)
	s.metricFamilies = utils.ScrapeCollectorMetricFamilies(s.T(), s.ctx, utils.MockCollectorStack.Otel)
}

func (s *MetricsTestSuite) TearDownSuite() {
	s.teardownTest(s.T())
}

func (s *MetricsTestSuite) TestNginxOSS_Test1_TestRequestCount() {
	family := s.metricFamilies["nginx_http_request_count"]
	s.T().Logf("nginx_http_request_count metric family: %v", family)
	s.Require().NotNil(family)

	baselineMetric := utils.SumMetricFamily(family)
	s.T().Logf("NGINX HTTP request count total: %v", baselineMetric)

	requestCount := 5
	for range requestCount {
		url := "http://127.0.0.1/"
		_, _, err := utils.MockCollectorStack.AgentOSS.Exec(
			s.ctx,
			[]string{"curl", "-s", url},
		)
		s.Require().NoError(err)
	}

	time.Sleep(65 * time.Second)

	s.metricFamilies = utils.ScrapeCollectorMetricFamilies(s.T(), s.ctx, utils.MockCollectorStack.Otel)
	family = s.metricFamilies["nginx_http_request_count"]
	s.T().Logf("nginx_http_request_count metric family: %v", family)
	s.Require().NotNil(family)

	got := utils.SumMetricFamily(family)

	s.T().Logf("NGINX HTTP request count total: %v", got)
	s.Require().GreaterOrEqual(got, baselineMetric+float64(requestCount))
}

func (s *MetricsTestSuite) TestNginxOSS_Test2_TestResponseCode() {
	family := s.metricFamilies["nginx_http_response_count"]
	s.T().Logf("nginx_http_response_count family: %v", family)
	s.Require().NotNil(family)

	responseCodes := []string{"1xx", "2xx", "3xx", "4xx"}
	codeRes := make([]float64, 0, len(responseCodes))
	for code := range responseCodes {
		codeRes = append(codeRes, utils.SumMetricFamilyLabel(family, "nginx_status_range", responseCodes[code]))
		s.T().Logf("NGINX HTTP response code %s total: %v", responseCodes[code], codeRes[code])
		s.Require().NotNil(codeRes[code])
	}
}

func (s *MetricsTestSuite) TestHostMetrics_Test1_TestSystemCPUUtilization() {
	family := s.metricFamilies["system_cpu_utilization"]
	s.T().Logf("system_cpu_utilization metric family: %v", family)
	s.Require().NotNil(family)

	cpuUtilizationSystem := utils.SumMetricFamilyLabel(family, "state", "system")
	cpuUtilizationUser := utils.SumMetricFamilyLabel(family, "state", "user")

	s.T().Logf("System cpu utilization: %v", cpuUtilizationSystem)
	s.T().Logf("System cpu utilization: %v", cpuUtilizationUser)
	s.Require().NotNil(cpuUtilizationSystem)
	s.Require().NotNil(cpuUtilizationUser)
}

func (s *MetricsTestSuite) TestHostMetrics_Test2_TestSystemMemoryUsage() {
	family := s.metricFamilies["system_memory_usage"]
	s.T().Logf("system_memory_usage metric family: %v", family)
	s.Require().NotNil(family)

	memoryUsageFree := utils.SumMetricFamilyLabel(family, "state", "free")
	memoryUsageUsed := utils.SumMetricFamilyLabel(family, "state", "used")

	s.T().Logf("System memory usage: %v", memoryUsageFree)
	s.T().Logf("System memory usage: %v", memoryUsageUsed)
	s.Require().NotNil(memoryUsageFree)
	s.Require().NotNil(memoryUsageUsed)
}

func TestMetricsTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsTestSuite))
}
