package plugins

import (
	"context"
	"crypto/tls"
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

// const (
// 	GEN_CNF  = "../../%s"
// 	GEN_CERT = "../../%s"
// )

func TestMtlsForApi(t *testing.T) {
	tests := []struct {
		name       string
		expected   *proto.NginxDetails
		dir        string
		conf       *config.Config
		clientMTLS bool
	}{
		{
			name: 	  "no tls test",
			expected: tutils.GetDetailsMap()["12345"],
			dir:      t.TempDir(),
			conf:     &config.Config{
				AgentAPI: config.AgentAPI{ 
					Port: 2345,
					Key:  "",
					Cert: "",
				},
			},
			clientMTLS: false,
		},
		{
			name: 	  "mtls test",
			expected: tutils.GetDetailsMap()["12345"],
			dir:      t.TempDir(),
			conf:     &config.Config{
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
		// 			Key:  "../../build/certs/server/ca.key",
		// 			Cert: "../../build/certs/server/ca.crt",
		// 		},
		// 	},
		// 	clientMTLS: false,
		// }, 
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%v", tt.dir)

			if (tt.conf.AgentAPI.Key != "") {
				certsDir, err := os.MkdirTemp(tt.dir, "certs")
				if err != nil {
					t.Fail()
				}
				t.Logf("%s", certsDir)

				// commands := []string{
				// 	fmt.Sprintf("%s ca --cn 'client-ca.local' --state Cork --locality Cork --org NGINX --country IE --out %s/client/conf", GEN_CNF, certsDir),
				// 	fmt.Sprintf("%s ca --config %s/client/conf/ca.cnf --out %s/client", GEN_CERT, certsDir, certsDir),
				// 	fmt.Sprintf("cp %s/client/ca.crt %s/client-ca.crt", certsDir, certsDir),
				// 	fmt.Sprintf("cp %s/client/ca.key %s/client-ca.key", certsDir, certsDir),

				// 	fmt.Sprintf("%s ca --cn 'server-ca.local' --state Cork --locality Cork --org NGINX --country IE --out %s/server/conf", GEN_CNF, certsDir),
				// 	fmt.Sprintf("%s ca --config %s/server/conf/ca.cnf --out %s/server", GEN_CERT, certsDir, certsDir),
				// 	fmt.Sprintf("cp %s/server/ca.crt %s/server-ca.pem", certsDir, certsDir),
				// 	fmt.Sprintf("cp %s/server/ca.key %s/server-ca.key", certsDir, certsDir),
				// }
			
				// for _, command := range commands {
				// 	cmd := exec.Command(command)
				// 	stdout, err := cmd.Output()

				// 	if err != nil {
				// 		t.Fatal(err)
				// 	}
			
				// 	t.Log(string(stdout))
				// } 

				// openssl req -new -nodes -x509 -out certs/server.pem -keyout certs/server.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Random Company/OU=IT/CN=www.random.com/emailAddress=$1"
				// echo "make client cert"
				// openssl req -new -nodes -x509 -out certs/client.pem -keyout certs/client.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Random Company/OU=IT/CN=www.random.com/emailAddress=$1"
			}

	
			// generateCertificate(t, dir)
			pluginUnderTest := NewAgentAPI(tt.conf, tutils.GetMockEnvWithProcess(), tutils.GetMockNginxBinary())
			pluginUnderTest.Init(core.NewMockMessagePipe(context.TODO()))
		
			client := resty.New()

			client.SetDebug(true)
			var url string
			if (tt.conf.AgentAPI.Key != "") {
				url = fmt.Sprintf("https://localhost:%d/nginx", tt.conf.AgentAPI.Port)
				
				if (tt.clientMTLS) {
					// Assign Client TLSClientConfig
					// One can set custom root-certificate. Refer: http://golang.org/pkg/crypto/tls/#example_Dial
					crt, err := ioutil.ReadFile("../../build/certs/client.crt")
					assert.NoError(t, err)
					key, err := ioutil.ReadFile("../../build/certs/client.key")
					assert.NoError(t, err)
					tlsConfig := &tls.Config{
						// ServerName: "localhost", // Optional
						MaxVersion: tls.VersionTLS13, // Optional
						GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
							if cert, err := tls.X509KeyPair(crt, key); err != nil {						
								return nil, err
			
							} else {
								return &cert, nil
							}
						},
					}
			
					transport := &http.Transport{ TLSClientConfig: tlsConfig }
					client.SetTransport(transport)
					// client.SetTLSClientConfig(&tls.Config{ RootCAs: roots })

					// or One can disable security check (https)
					//client.SetTLSClientConfig(&tls.Config{ InsecureSkipVerify: true })
				}
			} else {
				url = fmt.Sprintf("http://localhost:%d/nginx", tt.conf.AgentAPI.Port)
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
			if (len(details) < 1) {
				assert.Fail(t, "No data returned")
			} else {
				assert.Equal(t, expected, details[0])
			}
			pluginUnderTest.Close()
		})
	}
}

// explore response object for debugging
func printResult( resp *resty.Response, err error) (*resty.Response, error) {
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
