/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/nginx/agent/v2/src/core/metrics"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"time"

	"os"
	"os/exec"

	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	nginxConfigContent = tutils.GetDetailsNginxOssConfig()
)

func TestAgentAPI_Info(t *testing.T) {
	agentAPI := AgentAPI{}
	info := agentAPI.Info()

	assert.Equal(t, "Agent API Plugin", info.Name())
	assert.Equal(t, "v0.0.1", info.Version())
}

func TestAgentAPI_Subscriptions(t *testing.T) {
	expectedSubscriptions := []string{
		core.AgentAPIConfigApplyResponse,
		core.MetricReport,
		core.NginxConfigValidationPending,
		core.NginxConfigApplyFailed,
		core.NginxConfigApplySucceeded,
	}

	agentAPI := AgentAPI{}
	subscriptions := agentAPI.Subscriptions()

	assert.Equal(t, expectedSubscriptions, subscriptions)
}

func TestNginxHandler_sendInstanceDetailsPayload(t *testing.T) {
	tests := []struct {
		name         string
		nginxDetails []*proto.NginxDetails
	}{
		{
			name:         "no instances",
			nginxDetails: []*proto.NginxDetails{},
		},
		{
			name: "single instance",
			nginxDetails: []*proto.NginxDetails{
				{
					NginxId: "1", Version: "21", ConfPath: "/etc/yo", ProcessId: "123", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
			},
		},
		{
			name: "multi instance",
			nginxDetails: []*proto.NginxDetails{
				{
					NginxId: "1", Version: "21", ConfPath: "/etc/yo", ProcessId: "123", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "2", Version: "21", ConfPath: "/etc/yo", ProcessId: "123", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respRec := httptest.NewRecorder()
			path := "/nginx/"
			req := httptest.NewRequest(http.MethodGet, path, nil)

			env := tutils.GetMockEnv()
			mockNginxBinary := tutils.NewMockNginxBinary()
			processes := []core.Process{}

			for _, nginxDetail := range tt.nginxDetails {
				mockNginxBinary.On("GetNginxDetailsFromProcess", mock.Anything).Return(nginxDetail).Once()
				processes = append(processes, core.Process{Pid: 1, Name: "12345", IsMaster: true})
			}

			env.On("Processes").Return(processes)

			nginxHandler := NginxHandler{env: env, nginxBinary: mockNginxBinary}
			err := nginxHandler.sendInstanceDetailsPayload(respRec, req)
			assert.NoError(t, err)

			resp := respRec.Result()
			defer resp.Body.Close()

			var nginxDetailsResponse []*proto.NginxDetails
			err = json.Unmarshal(respRec.Body.Bytes(), &nginxDetailsResponse)
			assert.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.True(t, json.Valid(respRec.Body.Bytes()))
			assert.Equal(t, tt.nginxDetails, nginxDetailsResponse)
		})
	}
}

func TestNginxHandler_updateConfig(t *testing.T) {
	conf := &proto.NginxConfig{}

	tests := []struct {
		name                  string
		configUpdate          string
		validationTimeout     time.Duration
		response              *proto.Command_NginxConfigResponse
		nginxInstancesPresent bool
		expectedStatusCode    int
		expectedMessage       string
		expectedStatus        string
	}{
		{
			name:                  "no nginx instances",
			configUpdate:          nginxConfigContent,
			validationTimeout:     15 * time.Second,
			response:              nil,
			nginxInstancesPresent: false,
			expectedStatusCode:    500,
			expectedMessage:       "No NGINX instances found",
			expectedStatus:        "",
		},
		{
			name:                  "no config apply response",
			configUpdate:          nginxConfigContent,
			validationTimeout:     1 * time.Millisecond,
			response:              nil,
			nginxInstancesPresent: true,
			expectedStatusCode:    408,
			expectedMessage:       "pending config apply",
			expectedStatus:        "PENDING",
		},
		{
			name:              "pending config apply response",
			configUpdate:      nginxConfigContent,
			validationTimeout: 15 * time.Second,
			response: &proto.Command_NginxConfigResponse{
				NginxConfigResponse: &proto.NginxConfigResponse{
					Status: &proto.CommandStatusResponse{
						Status:  proto.CommandStatusResponse_CMD_OK,
						Message: configAppliedProcessedResponse,
					},
					Action:     proto.NginxConfigAction_APPLY,
					ConfigData: conf.GetConfigData(),
				},
			},
			nginxInstancesPresent: true,
			expectedStatusCode:    408,
			expectedMessage:       "config apply request successfully processed",
			expectedStatus:        "PENDING",
		},
		{
			name:              "successful config apply response",
			configUpdate:      nginxConfigContent,
			validationTimeout: 15 * time.Second,
			response: &proto.Command_NginxConfigResponse{
				NginxConfigResponse: &proto.NginxConfigResponse{
					Status: &proto.CommandStatusResponse{
						Status:  proto.CommandStatusResponse_CMD_OK,
						Message: configAppliedResponse,
					},
					Action:     proto.NginxConfigAction_APPLY,
					ConfigData: conf.GetConfigData(),
				},
			},
			nginxInstancesPresent: true,
			expectedStatusCode:    200,
			expectedMessage:       "config applied successfully",
			expectedStatus:        "OK",
		},
		{
			name:              "failed config apply response",
			configUpdate:      nginxConfigContent,
			validationTimeout: 15 * time.Second,
			response: &proto.Command_NginxConfigResponse{
				NginxConfigResponse: &proto.NginxConfigResponse{
					Status: &proto.CommandStatusResponse{
						Status:  proto.CommandStatusResponse_CMD_ERROR,
						Message: "config applied failed",
					},
					Action:     proto.NginxConfigAction_APPLY,
					ConfigData: conf.GetConfigData(),
				},
			},
			nginxInstancesPresent: true,
			expectedStatusCode:    400,
			expectedMessage:       "config applied failed",
			expectedStatus:        "ERROR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationTimeout = tt.validationTimeout
			w := httptest.NewRecorder()
			path := "/nginx/config/"

			file, err := os.CreateTemp(t.TempDir(), "nginx.conf")
			require.NoError(t, err)
			defer file.Close()

			err = os.WriteFile(file.Name(), []byte(tt.configUpdate), fs.FileMode(os.O_RDWR))
			require.NoError(t, err)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
			require.NoError(t, err)
			_, err = io.Copy(part, file)
			require.NoError(t, err)
			writer.Close()

			r := httptest.NewRequest(http.MethodPut, path, body)
			r.Header.Set("Content-Type", writer.FormDataContentType())

			nginxDetail := &proto.NginxDetails{
				NginxId: "1", Version: "21", ConfPath: file.Name(), ProcessId: "123", StartTime: 1238043824,
				BuiltFromSource: false,
				LoadableModules: []string{},
				RuntimeModules:  []string{},
				Plus:            &proto.NginxPlusMetaData{Enabled: true},
				ConfigureArgs:   []string{},
			}

			var env *tutils.MockEnvironment
			if tt.nginxInstancesPresent {
				env = tutils.GetMockEnvWithProcess()
			} else {
				env = tutils.GetMockEnv()
				env.On("Processes", mock.Anything).Return([]core.Process{})
			}
			env.On("WriteFiles", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockNginxBinary := tutils.NewMockNginxBinary()
			mockNginxBinary.On("GetNginxDetailsFromProcess", mock.Anything).Return(nginxDetail)
			mockNginxBinary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(conf, nil)

			pipeline := core.NewMessagePipe(context.TODO())

			h := &NginxHandler{
				config:          config.Defaults,
				env:             env,
				pipeline:        pipeline,
				nginxBinary:     mockNginxBinary,
				responseChannel: make(chan *proto.Command_NginxConfigResponse),
			}

			if tt.response != nil {
				go func() { h.responseChannel <- tt.response }()
			}

			err = h.updateConfig(w, r)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedStatusCode, w.Result().StatusCode)

			resp := w.Result()
			defer resp.Body.Close()

			if tt.response == nil {
				result := &AgentAPIConfigApplyStatusResponse{}
				err = json.NewDecoder(w.Body).Decode(result)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMessage, result.Message)
				assert.Equal(t, tt.expectedStatus, result.Status)
			} else {
				result := &AgentAPIConfigApplyResponse{}
				err = json.NewDecoder(w.Body).Decode(result)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMessage, result.NginxInstances[0].Message)
				assert.Equal(t, tt.expectedStatus, result.NginxInstances[0].Status)
			}

		})
	}
}

func TestNginxHandler_getConfigStatus(t *testing.T) {
	tests := []struct {
		name                   string
		url                    string
		configResponseStatuses map[string]*proto.NginxConfigStatus
		expectedStatusCode     int
		expectedMessage        string
		expectedStatus         string
	}{
		{
			name:                   "no query parameter",
			url:                    "/nginx/config/status/",
			configResponseStatuses: make(map[string]*proto.NginxConfigStatus),
			expectedStatusCode:     400,
			expectedMessage:        "Missing required query parameter correlation_id",
			expectedStatus:         "UNKNOWN",
		},
		{
			name:                   "no matching correlation_id",
			url:                    "/nginx/config/status/?correlation_id=123",
			configResponseStatuses: make(map[string]*proto.NginxConfigStatus),
			expectedStatusCode:     404,
			expectedMessage:        "Unable to find a config apply request with the correlation_id 123",
			expectedStatus:         "UNKNOWN",
		},
		{
			name: "found matching correlation_id",
			url:  "/nginx/config/status/?correlation_id=123",
			configResponseStatuses: map[string]*proto.NginxConfigStatus{
				"12345": {
					CorrelationId: "123",
					Status:        proto.NginxConfigStatus_OK,
					Message:       "config applied successfully",
					NginxId:       "12345",
				},
			},
			expectedStatusCode: 200,
			expectedMessage:    "config applied successfully",
			expectedStatus:     "OK",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)
			h := &NginxHandler{
				config:                 config.Defaults,
				env:                    tutils.GetMockEnv(),
				pipeline:               core.NewMessagePipe(context.TODO()),
				nginxBinary:            tutils.NewMockNginxBinary(),
				responseChannel:        make(chan *proto.Command_NginxConfigResponse),
				configResponseStatuses: tt.configResponseStatuses,
			}

			err := h.getConfigStatus(w, r)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedStatusCode, w.Result().StatusCode)

			resp := w.Result()
			defer resp.Body.Close()

			if len(tt.configResponseStatuses) > 0 {
				result := &AgentAPIConfigApplyResponse{}
				err = json.NewDecoder(w.Body).Decode(result)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMessage, result.NginxInstances[0].Message)
				assert.Equal(t, tt.expectedStatus, result.NginxInstances[0].Status)
				assert.Equal(t, "12345", result.NginxInstances[0].NginxId)

			} else {
				result := &AgentAPIConfigApplyStatusResponse{}
				err = json.NewDecoder(w.Body).Decode(result)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMessage, result.Message)
				assert.Equal(t, tt.expectedStatus, result.Status)
			}
		})
	}
}

func TestProcess_metricReport(t *testing.T) {
	conf := &config.Config{
		AgentAPI: config.AgentAPI{
			Port: 9090,
		},
	}

	mockEnvironment := tutils.NewMockEnvironment()
	mockNginxBinary := tutils.NewMockNginxBinary()

	metricReport := &proto.MetricsReport{Meta: &proto.Metadata{MessageId: "123"}}
	metricReportBundle := &metrics.MetricsReportBundle{Data: []*proto.MetricsReport{metricReport}}

	agentAPI := NewAgentAPI(conf, mockEnvironment, mockNginxBinary)

	// Check that latest metric report isn't set
	assert.NotEqual(t, metricReport, agentAPI.exporter.GetLatestMetricReports()[0])

	agentAPI.Process(core.NewMessage(core.MetricReport, metricReportBundle))

	// Check that latest metric report matches the report that was processed
	assert.Equal(t, metricReport, agentAPI.exporter.GetLatestMetricReports()[0])
}

func TestMtls_forApi(t *testing.T) {
	tests := []struct {
		name       string
		expected   *proto.NginxDetails
		dir        string
		conf       *config.Config
		clientMTLS bool
	}{
		{
			name:     "no tls test",
			expected: tutils.GetDetailsMap()["12345"],
			conf: &config.Config{
				AgentAPI: config.AgentAPI{
					Port: 2345,
					Key:  "",
					Cert: "",
				},
			},
			clientMTLS: false,
		},
		{
			name:     "mtls test",
			expected: tutils.GetDetailsMap()["12345"],
			conf: &config.Config{
				AgentAPI: config.AgentAPI{
					Port: 2345,
					Key:  "../../build/certs/server.key",
					Cert: "../../build/certs/server.crt",
				},
			},
			clientMTLS: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			ctx := context.Background()

			if tt.conf.AgentAPI.Key != "" {
				url = fmt.Sprintf("https://127.0.0.1:%d/nginx", tt.conf.AgentAPI.Port)
			} else {
				url = fmt.Sprintf("http://localhost:%d/nginx", tt.conf.AgentAPI.Port)

			}
			client := resty.New()

			if tt.clientMTLS {
				output, err := exec.Command("../../scripts/mtls/make_certs.sh").CombinedOutput()
				if err != nil {
					t.Errorf("make_certs.sh output: \n%s \n", output)
					os.RemoveAll("../../build/certs/")
					t.FailNow()
				}

				backoffSetting := backoff.BackoffSettings{
					InitialInterval: 100 * time.Millisecond,
					MaxInterval:     100 * time.Millisecond,
					MaxElapsedTime:  1 * time.Second,
					Jitter:          backoff.BACKOFF_JITTER,
					Multiplier:      backoff.BACKOFF_MULTIPLIER,
				}
				err = backoff.WaitUntil(ctx, backoffSetting, func() error {
					_, err := os.ReadFile("../../build/certs/server.crt")
					return err
				})

				assert.NoError(t, err)
				transport := &http.Transport{TLSClientConfig: getConfig(t)}
				client.SetTransport(transport)
			}

			pluginUnderTest := NewAgentAPI(tt.conf, tutils.GetMockEnvWithProcess(), tutils.GetMockNginxBinary())
			pluginUnderTest.Init(core.NewMockMessagePipe(ctx))

			client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

			resp, err := client.R().EnableTrace().Get(url)

			assert.NoError(t, err)

			printResult(resp, err)

			var details []*proto.NginxDetails
			err = json.Unmarshal(resp.Body(), &details)

			assert.NoError(t, err)

			expected := tutils.GetDetailsMap()["12345"]
			assert.Len(t, details, 1)
			if len(details) < 1 {
				assert.Fail(t, "No data returned")
			} else {
				assert.Equal(t, expected, details[0])
			}

			pluginUnderTest.Close()
			if tt.clientMTLS {
				os.RemoveAll("../../build/certs/")
			}
		})
	}
}

func getConfig(t *testing.T) *tls.Config {
	crt, err := os.ReadFile("../../build/certs/client.crt")
	assert.NoError(t, err)
	key, err := os.ReadFile("../../build/certs/client.key")
	assert.NoError(t, err)
	ca, err := os.ReadFile("../../build/certs/ca.pem")
	assert.NoError(t, err)

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		assert.Fail(t, "error reading cert")

	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	caPool := tlsConfig.RootCAs
	if caPool == nil {
		caPool = x509.NewCertPool()
	}

	if !caPool.AppendCertsFromPEM(ca) {
		assert.Fail(t, "Can't append cert")
	}

	tlsConfig.RootCAs = caPool
	return tlsConfig
}

// explore response object for debugging
func printResult(resp *resty.Response, err error) *resty.Response {
	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()
	return resp
}
