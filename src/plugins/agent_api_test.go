package plugins

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	// "os/exec"

	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
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

func TestMtlsForApi(t *testing.T) {
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
			dir:      t.TempDir(),
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
			dir:      t.TempDir(),
			conf: &config.Config{
				AgentAPI: config.AgentAPI{
					Port: 2345,
					Key:  "../../build/certs/server.key",
					Cert: "../../build/certs/server.crt",
				},
			},
			clientMTLS: true,
		},
		// {
		// 	name: 	  "mtls test, no client cert",
		// 	expected: tutils.GetDetailsMap()["12345"],
		// 	dir:      t.TempDir(),
		// 	conf:     &config.Config{
		// 		AgentAPI: config.AgentAPI{
		// 			Port: 2345,
		// 			Key:  "../../build/certs/server.key",
		// 			Cert: "../../build/certs/server.crt",
		// 		},
		// 	},
		// 	clientMTLS: false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%v", tt.dir)

			var url string

			if tt.conf.AgentAPI.Key != "" {
				url = fmt.Sprintf("https://127.0.0.1:%d/nginx", tt.conf.AgentAPI.Port)

				certsDir, err := os.MkdirTemp(tt.dir, "certs")
				if err != nil {
					t.Fail()
				}
				t.Logf("%s", certsDir)
			} else {
				url = fmt.Sprintf("http://localhost:%d/nginx", tt.conf.AgentAPI.Port)

			}

			pluginUnderTest := NewAgentAPI(tt.conf, tutils.GetMockEnvWithProcess(), tutils.GetMockNginxBinary())
			pluginUnderTest.Init(core.NewMockMessagePipe(context.TODO()))

			client := resty.New()

			client.SetDebug(true)

			if tt.clientMTLS {
				transport := &http.Transport{TLSClientConfig: getConfig(t)}
				client.SetTransport(transport)
			}
			// Set client timeout as per your need
			client.SetTimeout(1 * time.Minute)
			client.AddRetryCondition(
				// RetryConditionFunc type is for retry condition function
				// input: non-nil Response OR request execution error
				func(r *resty.Response, err error) bool {
					return r.StatusCode() == http.StatusTooManyRequests
				},
			)

			resp, err := client.R().EnableTrace().Get(url)

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
		})
	}
}

func getConfig(t *testing.T) *tls.Config {
	crt, err := ioutil.ReadFile("../../build/certs/client.crt")
	assert.NoError(t, err)
	key, err := ioutil.ReadFile("../../build/certs/client.key")
	assert.NoError(t, err)
	ca, err := ioutil.ReadFile("../../build/certs/ca.pem")
	assert.NoError(t, err)

	os.Setenv("SSL_CERT_FILE", "../../build/certs/ca.pem")

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
func printResult(resp *resty.Response, err error) (*resty.Response, error) {
	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()
	return resp, err
}
