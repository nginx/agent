package component

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/plugins"

	"encoding/json"

	"github.com/go-resty/resty/v2"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
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
		Features: config.Defaults.Features,
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
	agentAPI.Close()

}

func TestMetricsDisabled(t *testing.T) {
	port := 9090

	conf := &config.Config{
		AgentAPI: config.AgentAPI{
			Port: port,
		},
		Features: []string{
			"agent-api",
			"nginx-config-async",
		},
	}

	mockEnvironment := tutils.NewMockEnvironment()
	mockNginxBinary := tutils.NewMockNginxBinary()

	agentAPI := plugins.NewAgentAPI(conf, mockEnvironment, mockNginxBinary)
	agentAPI.Init(core.NewMockMessagePipe(context.TODO()))

	client := resty.New()

	url := fmt.Sprintf("http://localhost:%d/metrics", port)
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	response, err := client.R().EnableTrace().Get(url)
	responseData := tutils.ProcessResponse(response)

	for _, m := range responseData {
		metric := strings.Split(m, " ")
		assert.True(t, strings.HasPrefix(metric[0], "go_"))

	}
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode())
	agentAPI.Close()
}

func TestConfigApply(t *testing.T) {
	port := 9090

	conf := &proto.NginxConfig{}

	tests := []struct {
		name           string
		configUpdate   string
		expectedStatus int
		agentConf      *config.Config
		agentStatus    *proto.Command_NginxConfigResponse
	}{
		{
			name:           "successful config apply response",
			configUpdate:   tutils.GetDetailsNginxOssConfig(),
			expectedStatus: http.StatusOK,
			agentConf: &config.Config{
				AgentAPI: config.AgentAPI{
					Port: port,
				},
				Features: []string{
					"agent-api",
					"nginx-config-async",
				},
			},
			agentStatus: &proto.Command_NginxConfigResponse{
				NginxConfigResponse: &proto.NginxConfigResponse{
					Status: &proto.CommandStatusResponse{
						Status:  proto.CommandStatusResponse_CMD_OK,
						Message: "config applied successfully",
					},
					Action:     proto.NginxConfigAction_APPLY,
					ConfigData: conf.ConfigData,
				},
			},
		},
		{
			name:           "failed config apply disabled",
			configUpdate:   tutils.GetDetailsNginxOssConfig(),
			expectedStatus: http.StatusNotFound,
			agentConf: &config.Config{
				AgentAPI: config.AgentAPI{
					Port: port,
				},
				Features: []string{
					"agent-api",
				},
			},
			agentStatus: &proto.Command_NginxConfigResponse{
				NginxConfigResponse: &proto.NginxConfigResponse{
					Status: &proto.CommandStatusResponse{
						Status:  proto.CommandStatusResponse_CMD_ERROR,
						Message: "nginx-config-async feature is disabled",
					},
					Action:     proto.NginxConfigAction_APPLY,
					ConfigData: conf.ConfigData,
				},
			},
		},
		{
			name:           "failed config apply",
			configUpdate:   tutils.GetDetailsNginxOssConfig(),
			expectedStatus: http.StatusBadRequest,
			agentConf: &config.Config{
				AgentAPI: config.AgentAPI{
					Port: port,
				},
				Features: []string{
					"agent-api",
					"nginx-config-async",
				},
			},
			agentStatus: &proto.Command_NginxConfigResponse{
				NginxConfigResponse: &proto.NginxConfigResponse{
					Status: &proto.CommandStatusResponse{
						Status:  proto.CommandStatusResponse_CMD_ERROR,
						Message: "failed config apply",
					},
					Action:     proto.NginxConfigAction_APPLY,
					ConfigData: conf.ConfigData,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			confFile := createTempFile(t, tt.configUpdate, "nginx.conf")

			nginxDetails := &proto.NginxDetails{
				NginxId: "45d4sf5d4sf4e8s4f8es4564", Version: "21", ProcessId: "12345", ConfPath: confFile, StartTime: 1238043824,
				BuiltFromSource: false,
				Plus:            &proto.NginxPlusMetaData{Enabled: true},
			}

			mockEnvironment := tutils.NewMockEnvironment()
			mockEnvironment.On("Processes").Return([]core.Process{{Pid: 12345, IsMaster: true}})
			mockEnvironment.On("WriteFiles", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockNginxBinary := tutils.NewMockNginxBinary()
			mockNginxBinary.On("GetNginxDetailsFromProcess", mock.Anything).Return(nginxDetails)
			mockNginxBinary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(conf, nil)

			agentAPI := plugins.NewAgentAPI(tt.agentConf, mockEnvironment, mockNginxBinary)
			pipeline := core.NewMockMessagePipe(context.TODO())
			agentAPI.Init(pipeline)

			fileName := createTempFile(t, tt.configUpdate, "temp.conf")

			url := fmt.Sprintf("http://localhost:%d/nginx/config/", port)

			client := resty.New()
			client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

			go func() {
				time.Sleep(1 * time.Millisecond)
				message := core.NewMessage(core.AgentAPIConfigApplyResponse, tt.agentStatus)
				agentAPI.Process(message)
			}()

			resp, err := client.R().SetFile("file", fileName).EnableTrace().Put(url)

			assert.NoError(t, err)
			fmt.Println(resp)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode())
			agentAPI.Close()

		})
	}
}

func createTempFile(t *testing.T, configUpdate string, fileName string) string {
	confPath := t.TempDir()
	file, err := os.CreateTemp(confPath, fileName)
	assert.NoError(t, err)
	defer file.Close()

	err = os.WriteFile(file.Name(), []byte(configUpdate), fs.FileMode(os.O_RDWR))
	assert.NoError(t, err)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	assert.NoError(t, err)
	_, err = io.Copy(part, file)
	assert.NoError(t, err)
	writer.Close()

	return file.Name()
}
