package metrics

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/integration/utils"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
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
	s.metricFamilies = scrapeCollectorMetricFamilies(s.T(), s.ctx, utils.MockCollectorStack.Otel)
}

func (s *MetricsTestSuite) TearDownSuite() {
	s.teardownTest(s.T())
}

func (s *MetricsTestSuite) TestNginxOSS_Test1_TestRequestCount() {
	family := s.metricFamilies["nginx_http_request_count"]
	s.Require().NotNil(family, "nginx_http_requests_count metric family should not be nil")

	baselineMetric := sumMetricFamily(family)
	s.T().Logf("nginx http requests count observed total: %v", baselineMetric)

	for i := 0; i < 5; i++ {
		utils.MockCollectorStack.AgentOSS.Exec(s.ctx, []string{
			"curl", "-s", "http://127.0.0.1/",
		})
	}

	time.Sleep(30 * time.Second)

	s.metricFamilies = scrapeCollectorMetricFamilies(s.T(), s.ctx, utils.MockCollectorStack.Otel)

	family = s.metricFamilies["nginx_http_request_count"]
	s.Require().NotNil(family, "nginx_http_requests_count metric family should not be nil")

	got := sumMetricFamily(family)

	s.T().Logf("nginx http requests observed total: %f", got)

	// expected request total should be 'want' + the 5 curl requests above + 1 health check request
	s.Require().GreaterOrEqual(got, baselineMetric+5, "nginx http requests count should increase by at least 5")
}

func (s *MetricsTestSuite) TestNginxOSS_Test2_TestResponseCode() {
	family := s.metricFamilies["nginx_http_response_count"]
	s.T().Logf("nginx_http_response_count family: %v", family)
	s.Require().NotNil(family, "nginx_http_response_count metric family should not be nil")

	for i := 1; i < 5; i++ {
		code := fmt.Sprintf("%dxx", i)
		s.T().Logf("nginx http response code %s total: %v", code, sumMetricFamilyLabel(family, "nginx_status_range", code))
	}

	s.Require().Greater(sumMetricFamily(family), 0.0, "expected some nginx http response codes")
}

func (s *MetricsTestSuite) TestHostMetrics_Test1_TestSystemCPUUtilization() {
	family := s.metricFamilies["system_cpu_utilization"]
	s.T().Logf("system cpu utilization metric family: %v", family)
	s.Require().NotNil(family, "system_cpu_utilization metric family should not be nil")

	cpuUtilization := sumMetricFamily(family)

	s.T().Logf("system cpu utilization: %v", cpuUtilization)
	s.Require().Greater(cpuUtilization, 0.0, "expected some system cpu utilization")
}

func (s *MetricsTestSuite) TestHostMetrics_Test2_TestSystemMemoryUsage() {
	family := s.metricFamilies["system_memory_usage"]
	s.T().Logf("system memory usage metric family: %v", family)
	s.Require().NotNil(family, "system_memory_usage metric family should not be nil")

	memoryUsage := sumMetricFamily(family)

	s.T().Logf("system memory usage: %v", memoryUsage)
	s.Require().Greater(memoryUsage, 0.0, "expected some system memory usage")
}

func TestMetricsTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsTestSuite))
}

func scrapeCollectorMetricFamilies(t *testing.T, ctx context.Context, otelContainer testcontainers.Container) map[string]*dto.MetricFamily {
	t.Helper()

	host, _ := otelContainer.Host(ctx)
	port, _ := otelContainer.MappedPort(ctx, "9775")

	resp, err := http.Get(fmt.Sprintf("http://%s:%s/metrics", host, port.Port()))
	if err != nil {
		t.Fatalf("failed to get response from Otel Collector: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Fatalf("failed to parse metrics: %v", err)
	}

	return metricFamilies
}

func sumMetricFamily(metricFamily *dto.MetricFamily) float64 {
	var total float64
	for _, metric := range metricFamily.Metric {
		if value, ok := metricValue(metricFamily, metric); ok {
			total += value
		}
	}
	return total
}

func sumMetricFamilyLabel(metricFamily *dto.MetricFamily, key, val string) float64 {
	var total float64
	for _, metric := range metricFamily.Metric {
		labels := map[string]string{}
		for _, labelPair := range metric.Label {
			labels[labelPair.GetName()] = labelPair.GetValue()
		}
		if labels[key] != val {
			continue
		}
		if value, ok := metricValue(metricFamily, metric); ok {
			total += value
		}
	}
	return total
}

func metricValue(metricFamily *dto.MetricFamily, metric *dto.Metric) (float64, bool) {
	switch metricFamily.GetType() {
	case dto.MetricType_COUNTER:
		if counter := metric.GetCounter(); counter != nil {
			return counter.GetValue(), true
		}
	case dto.MetricType_GAUGE:
		if gauge := metric.GetGauge(); gauge != nil {
			return gauge.GetValue(), true
		}
	}
	return 0, false
}

//func matchLabels(labels map[string]string, filter string) bool {
//	if filter == "" {
//		return true
//	}
//	if i := strings.IndexByte(filter, '='); i >= 0 {
//		key := filter[:i]
//		val := filter[i+1:]
//		return labels[key] == val
//	}
//	_, ok := labels[filter]
//	return ok
//}
