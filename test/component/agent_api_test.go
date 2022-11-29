package component

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

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

			response, err := http.Get(fmt.Sprintf("http://localhost:%d/nginx/", port))
			assert.Nil(t, err)

			responseData, err := io.ReadAll(response.Body)
			assert.Nil(t, err)

			var nginxDetailsResponse []*proto.NginxDetails
			err = json.Unmarshal(responseData, &nginxDetailsResponse)
			assert.Nil(t, err)

			assert.Equal(t, http.StatusOK, response.StatusCode)
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

	response, err := http.Get(fmt.Sprintf("http://localhost:%d/invalid/", port))
	assert.Nil(t, err)

	agentAPI.Close()

	assert.Equal(t, http.StatusNotFound, response.StatusCode)
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
	agentAPI.Process(core.NewMessage(core.MetricReport, &proto.MetricsReport{
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
	}))

	response, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	assert.Nil(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	responseData, err := io.ReadAll(response.Body)
	assert.Nil(t, err)

	agentAPI.Close()

	assert.Contains(t, string(responseData), "# HELP system_cpu_idle")
	assert.Contains(t, string(responseData), "# TYPE system_cpu_idle gauge")
	assert.Contains(t, string(responseData), "system_cpu_idle{hostname=\"example.com\",system_tags=\"\"} 12")
}
