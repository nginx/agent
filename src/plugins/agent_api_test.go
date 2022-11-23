package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	// "os/exec"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/go-resty/resty/v2"
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

func TestMtlsForApi(t *testing.T) {
	dir := t.TempDir()
	t.Logf("%v", dir)
	conf := &config.Config{
		AgentAPI: config.AgentAPI{ 
			Port: 2345,
		},
	}

	// generateCertificate(t, dir)
	pluginUnderTest := NewAgentAPI(conf, tutils.GetMockEnvWithProcess(), tutils.GetMockNginxBinary())
	pluginUnderTest.Init(core.NewMockMessagePipe(context.TODO()))

	client := resty.New()

	resp, err := client.R().EnableTrace().Get(fmt.Sprintf("http://localhost:%d/nginx", conf.AgentAPI.Port))

	// Explore response object
	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()

	var details []*proto.NginxDetails
	err = json.Unmarshal(resp.Body(),&details)

	assert.NoError(t, err)
	// var responseItems map[string]*proto.NginxDetails 
	expected := tutils.GetDetailsMap()["12345"]
	assert.Len(t, details, 1)
	assert.Equal(t, expected, details[0])
}

// func generateCertificate(t *testing.T, dir string) error {
// 	cmd := exec.Command("../../scripts/mtls/gen_cnf.sh", "ca", "--cn", "'ca.local'", "--state", "Cork", "--locality", "Cork", "--org", "NGINX", "--country", "IE", "--out", dir)

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Logf("%v", err)
// 		t.Fail()
// 	}

// 	cmd1 := exec.Command("../../scripts/mtls/gen_cert.sh", "ca", "--config", "certs/conf/ca.cnf", "--out", dir)

// 	err = cmd1.Run()
// 	if err != nil {
// 		t.Logf("%v", err)
// 		t.Fail()
// 	}

// 	// scripts/mtls/gen_cnf.sh ca --cn '${CERT_CLIENT_CA_CN}' --state Cork --locality Cork --org NGINX --country IE --out ${CERTS_DIR}/client/conf
// 	// scripts/mtls/gen_cert.sh ca --config ${CERTS_DIR}/client/conf/ca.cnf --out ${CERTS_DIR}/client

// 	return nil
// }
