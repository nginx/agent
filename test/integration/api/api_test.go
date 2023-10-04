package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/test/integration/utils"
	tutils "github.com/nginx/agent/v2/test/utils"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

const (
	API_PORT = 9091
	API_HOST = "0.0.0.0"
)

func TestAPI_Nginx(t *testing.T) {
	testContainer := utils.SetupTestContainerWithAgent(
		t,
		"api-nginx",
		"./nginx-agent.conf:/etc/nginx-agent/nginx-agent.conf",
		"Starting Agent API HTTP server with port from config and TLS disabled",
	)

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s:%d/nginx/", API_HOST, API_PORT)
	resp, err := client.R().EnableTrace().Get(url)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Contains(t, resp.String(), "nginx_id")
	assert.NotContains(t, resp.String(), "test_fail_nginx")

	var nginxDetailsResponse []*proto.NginxDetails

	responseData := resp.Body()
	err = json.Unmarshal(responseData, &nginxDetailsResponse)

	assert.Nil(t, err)
	assert.True(t, json.Valid(responseData))

	assert.NotNil(t, nginxDetailsResponse[0].NginxId)
	assert.NotNil(t, nginxDetailsResponse[0].Version)
	assert.Contains(t, nginxDetailsResponse[0].RuntimeModules, "http_stub_status_module")
	assert.Equal(t, "/etc/nginx/nginx.conf", nginxDetailsResponse[0].ConfPath)

	utils.TestAgentHasNoErrorLogs(t, testContainer)
}

func TestAPI_Metrics(t *testing.T) {
	testContainer := utils.SetupTestContainerWithAgent(
		t,
		"api-metrics",
		"./nginx-agent.conf:/etc/nginx-agent/nginx-agent.conf",
		"Starting Agent API HTTP server with port from config and TLS disabled",
	)

	client := resty.New()

	url := fmt.Sprintf("http://%s:%d/metrics/", API_HOST, API_PORT)
	client.SetRetryCount(5).SetRetryWaitTime(5 * time.Second).SetRetryMaxWaitTime(5 * time.Second)
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return len(r.String()) < 22000
		},
	)

	resp, err := client.R().EnableTrace().Get(url)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Contains(t, resp.String(), "system_cpu_system")
	assert.NotContains(t, resp.String(), "test_fail_metric")
	// Validate that the agent can call the stub status API
	assert.Contains(t, resp.String(), "nginx_http_request_count")
	// Validate that the agent can read the NGINX access logs
	assert.Contains(t, resp.String(), "nginx_http_status_2xx")

	metrics := tutils.ProcessResponse(resp)

	for _, m := range metrics {
		metric := strings.Split(m, " ")
		switch {
		case strings.Contains(metric[0], "system_cpu_system"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))

		case strings.Contains(metric[0], "system_mem_used_all"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))

		case strings.Contains(metric[0], "container_cpu_cores"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))

		case strings.Contains(metric[0], "nginx_workers_count"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))
		}
	}

	utils.TestAgentHasNoErrorLogs(t, testContainer)
}
