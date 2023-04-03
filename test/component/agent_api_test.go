package component

import (
	"context"
	"fmt"
	"github.com/nginx/agent/v2/src/core/metrics"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"encoding/json"
	"github.com/go-resty/resty/v2"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/plugins"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetNginxInstances(t *testing.T) {
	port := 9090
	processID := "12345"
	var pid int32 = 12345

	tests := []struct {
		name         string
		nginxDetails *proto.NginxDetails
		expectedJSON string
	}{
		{
			name:         "no instances",
			nginxDetails: nil,
		},
		{
			name: "single instance",
			nginxDetails: &proto.NginxDetails{
				NginxId: "45d4sf5d4sf4e8s4f8es4564", Version: "21", ConfPath: "/etc/nginx/conf", ProcessId: processID, StartTime: 1238043824,
				BuiltFromSource: false,
				Plus:            &proto.NginxPlusMetaData{Enabled: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config.Config{
				AgentAPI: config.AgentAPI{
					Port: port,
				},
			}

			mockEnvironment := tutils.NewMockEnvironment()
			if tt.nginxDetails == nil {
				mockEnvironment.On("Processes").Return([]core.Process{{Pid: pid, IsMaster: false}})
			} else {
				mockEnvironment.On("Processes").Return([]core.Process{{Pid: pid, IsMaster: true}})
			}

			mockNginxBinary := tutils.NewMockNginxBinary()
			mockNginxBinary.On("GetNginxDetailsFromProcess", mock.Anything).Return(tt.nginxDetails)

			agentAPI := plugins.NewAgentAPI(conf, mockEnvironment, mockNginxBinary)
			agentAPI.Init(core.NewMockMessagePipe(context.TODO()))

			client := resty.New()
			client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

			url := fmt.Sprintf("http://localhost:%d/nginx", port)
			response, err := client.R().EnableTrace().Get(url)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode())

			var nginxDetailsResponse []*proto.NginxDetails
			responseData := response.Body()
			err = json.Unmarshal(responseData, &nginxDetailsResponse)
			assert.Nil(t, err)
			assert.True(t, json.Valid(responseData))

			if tt.nginxDetails == nil {
				assert.Equal(t, 0, len(nginxDetailsResponse))
			} else {
				assert.Equal(t, 1, len(nginxDetailsResponse))
				assert.Equal(t, tt.nginxDetails, nginxDetailsResponse[0])
			}

			agentAPI.Close()
		})
	}
}

func TestInvalidPath(t *testing.T) {
	port := 9090

	conf := &config.Config{
		AgentAPI: config.AgentAPI{
			Port: port,
		},
	}

	mockEnvironment := tutils.NewMockEnvironment()
	mockNginxBinary := tutils.NewMockNginxBinary()

	agentAPI := plugins.NewAgentAPI(conf, mockEnvironment, mockNginxBinary)
	agentAPI.Init(core.NewMockMessagePipe(context.TODO()))

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/invalid/", port)
	response, err := client.R().EnableTrace().Get(url)

	assert.Nil(t, err)

	agentAPI.Close()

	assert.Equal(t, http.StatusNotFound, response.StatusCode())
}

func TestMetrics(t *testing.T) {
	port := 9090

	conf := &config.Config{
		AgentAPI: config.AgentAPI{
			Port: port,
		},
	}

	mockEnvironment := tutils.NewMockEnvironment()
	mockNginxBinary := tutils.NewMockNginxBinary()

	agentAPI := plugins.NewAgentAPI(conf, mockEnvironment, mockNginxBinary)
	agentAPI.Init(core.NewMockMessagePipe(context.TODO()))
	agentAPI.Process(core.NewMessage(core.MetricReport, &metrics.MetricsReportBundle{Data: []*proto.MetricsReport{
		{
			Type: proto.MetricsReport_SYSTEM,
			Meta: &proto.Metadata{
				MessageId: "123",
			},
			Data: []*proto.StatsEntity{
				{
					Dimensions: []*proto.Dimension{
						{
							Name:  "hostname",
							Value: "example.com",
						},
						{
							Name:  "system.tags",
							Value: "",
						},
					},
					Simplemetrics: []*proto.SimpleMetric{
						{
							Name:  "system.cpu.idle",
							Value: 12,
						},
					},
				},
			},
		},
		{
			Type: proto.MetricsReport_INSTANCE,
			Meta: &proto.Metadata{
				MessageId: "456",
			},
			Data: []*proto.StatsEntity{
				{
					Dimensions: []*proto.Dimension{
						{
							Name:  "hostname",
							Value: "example.com",
						},
						{
							Name:  "system.tags",
							Value: "",
						},
					},
					Simplemetrics: []*proto.SimpleMetric{
						{
							Name:  "nginx.workers.count",
							Value: 6,
						},
					},
				},
			},
		},
	}}))

	client := resty.New()

	url := fmt.Sprintf("http://localhost:%d/metrics", port)
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	response, err := client.R().EnableTrace().Get(url)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode())
	assert.Contains(t, response.String(), "# HELP system_cpu_idle")
	assert.Contains(t, response.String(), "# TYPE system_cpu_idle gauge")
	agentAPI.Close()

	responseData := tutils.ProcessResponse(response)

	for _, m := range responseData {
		metric := strings.Split(m, " ")
		switch {
		case strings.Contains(metric[0], "system_cpu_idle"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Equal(t, float64(12), value)
		case strings.Contains(metric[0], "nginx_workers_count"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Equal(t, float64(6), value)
		}

	}

}
