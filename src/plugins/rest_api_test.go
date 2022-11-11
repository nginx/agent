package plugins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

func TestNginxHandler_sendInstanceDetailsPayload(t *testing.T) {
	tests := []struct {
		name         string
		nginxDetails []*proto.NginxDetails
		expectedJSON string
	}{
		{
			name:         "no instances",
			nginxDetails: []*proto.NginxDetails{},
			expectedJSON: "[]",
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
			expectedJSON: `[{"nginx_id":"1","version":"21","conf_path":"/etc/yo","process_id":"123","process_path":"","start_time":1238043824,"built_from_source":false,"loadable_modules":[],"runtime_modules":[],"plus":{"enabled":true,"release":""},"ssl":null,"status_url":"","configure_args":[]}]` + "\n",
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
			expectedJSON: `[{"nginx_id":"1","version":"21","conf_path":"/etc/yo","process_id":"123","process_path":"","start_time":1238043824,"built_from_source":false,"loadable_modules":[],"runtime_modules":[],"plus":{"enabled":true,"release":""},"ssl":null,"status_url":"","configure_args":[]},{"nginx_id":"2","version":"21","conf_path":"/etc/yo","process_id":"123","process_path":"","start_time":1238043824,"built_from_source":false,"loadable_modules":[],"runtime_modules":[],"plus":{"enabled":true,"release":""},"ssl":null,"status_url":"","configure_args":[]}]` + "\n",
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

			jsonStr := respRec.Body.String()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.True(t, json.Valid([]byte(jsonStr)))
			assert.Equal(t, tt.expectedJSON, jsonStr)
		})
	}
}
