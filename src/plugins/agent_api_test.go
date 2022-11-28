package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

			err := sendInstanceDetailsPayload(tt.nginxDetails, respRec, req)
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
	tests := []struct {
		name         string
		configUpdate string
	}{
		{
			name:         "update config",
			configUpdate: "# test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip()
			w := httptest.NewRecorder()
			path := "/nginx/config/"

			file, err := os.CreateTemp(t.TempDir(), "file")
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

			h := &NginxHandler{
				config:          config.Defaults,
				env:             &core.EnvironmentType{},
				pipeline:        core.NewMessagePipe(context.TODO()),
				nginxBinary:     tutils.NewMockNginxBinary(),
				responseChannel: make(chan *proto.Command_NginxConfigResponse),
			}

			err = h.updateConfig(w, r)
			assert.NoError(t, err)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, "# test", w.Body.String())
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

	agentAPI := NewAgentAPI(conf, mockEnvironment, mockNginxBinary)

	// Check that latest metric report isn't set
	assert.NotEqual(t, metricReport, agentAPI.exporter.GetLatestMetricReport())

	agentAPI.Process(core.NewMessage(core.MetricReport, metricReport))

	// Check that latest metric report matches the report that was processed
	assert.Equal(t, metricReport, agentAPI.exporter.GetLatestMetricReport())
}
