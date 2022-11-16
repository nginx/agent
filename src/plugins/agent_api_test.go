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
			assert.Nil(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.True(t, json.Valid(respRec.Body.Bytes()))
			assert.Equal(t, tt.nginxDetails, nginxDetailsResponse)
		})
	}
}
