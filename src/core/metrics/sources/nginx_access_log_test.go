package sources

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	testutils "github.com/nginx/agent/v2/test/utils"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestAccessLogUpdate(t *testing.T) {
	binary := testutils.NewMockNginxBinary()
	binary.On("UpdatedAccessLogs").Return(true, map[string]string{"/tmp/access.log": ""}).Once()
	binary.On("UpdatedAccessLogs").Return(true, map[string]string{"/tmp/new_access.log": ""}).Once()

	collectionDuration, _ := time.ParseDuration("300ms")
	newCollectionDuration, _ := time.ParseDuration("500ms")
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	assert.Equal(t, "", nginxAccessLog.baseDimensions.InstanceTags)
	assert.Equal(t, collectionDuration, nginxAccessLog.collectionInterval)
	_, ok := nginxAccessLog.logs["/tmp/access.log"]
	assert.True(t, ok)

	nginxAccessLog.Update(
		&metrics.CommonDim{
			InstanceTags: "new-tag",
		},
		&metrics.NginxCollectorConfig{
			CollectionInterval: newCollectionDuration,
		},
	)

	assert.Equal(t, "new-tag", nginxAccessLog.baseDimensions.InstanceTags)
	assert.Equal(t, newCollectionDuration, nginxAccessLog.collectionInterval)
	_, ok = nginxAccessLog.logs["/tmp/new_access.log"]
	assert.True(t, ok)
}

func TestAccessLogStop(t *testing.T) {
	binary := testutils.NewMockNginxBinary()
	binary.On("UpdatedAccessLogs").Return(true, map[string]string{"/tmp/access.log": ""}).Once()

	collectionDuration, _ := time.ParseDuration("300ms")
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	_, ok := nginxAccessLog.logs["/tmp/access.log"]
	assert.True(t, ok)

	nginxAccessLog.Stop()

	assert.Equal(t, 0, len(nginxAccessLog.logs))
}

func TestAccessLogStats(t *testing.T) {
	tests := []struct {
		name          string
		logFormat     string
		logLines      []string
		m             chan *proto.StatsEntity
		expectedStats *proto.StatsEntity
	}{
		{
			"default_access_log_test",
			`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for"`,
			[]string{
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\"\n",
			},
			make(chan *proto.StatsEntity, 1),
			&proto.StatsEntity{
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "nginx.http.gzip.ratio",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.body_bytes_sent",
						Value: 196,
					},
					{
						Name:  "nginx.http.request.bytes_sent",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.length",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.malformed",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.post",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.get",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.delete",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.put",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.head",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.options",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.others",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.2xx",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.4xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.5xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.403",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.404",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.500",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.502",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.503",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.504",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.discarded",
						Value: 0,
					},
					{
						Name:  "nginx.http.v0_9",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_0",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_1",
						Value: 2,
					},
					{
						Name:  "nginx.http.v2",
						Value: 0,
					},
				},
			},
		},
		{
			"full_access_log_test",
			`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for" "$bytes_sent" "$request_length" "$request_time" "$gzip_ratio" "$server_protocol"`,
			[]string{
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"10\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"POST /nginx_status HTTP/1.1\" 201 98 \"-\" \"Go-http-client/1.1\" \"-\" \"250\" \"110\" \"0.300\" \"20\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"DELETE /nginx_status HTTP/1.1\" 400 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"DELETE /nginx_status HTTP/1.1\" 403 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"HEAD /nginx_status HTTP/1.1\" 404 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"PUT /nginx_status HTTP/1.1\" 499 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"PUT /nginx_status HTTP/1.1\" 500 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/1.0\" 502 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.0\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/2\" 503 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/2\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/0.9\" 504 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/0.9\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/1.1\" 502 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"TRACE /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"-\" \"HTTP/1.1\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"TRACE /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"-\" \"HTTP/1.1\"\n",
			},
			make(chan *proto.StatsEntity, 1),
			&proto.StatsEntity{
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "nginx.http.gzip.ratio",
						Value: 15,
					},
					{
						Name:  "nginx.http.request.time",
						Value: 0.1857142857142857,
					},
					{
						Name:  "nginx.http.request.time.count",
						Value: 14,
					},
					{
						Name:  "nginx.http.request.time.median",
						Value: 0.2,
					},
					{
						Name:  "nginx.http.request.time.max",
						Value: 0.3,
					},
					{
						Name:  "nginx.http.request.time.pctl95",
						Value: 0.2,
					},
					{
						Name:  "nginx.http.request.body_bytes_sent",
						Value: 1372,
					},
					{
						Name:  "nginx.http.request.bytes_sent",
						Value: 2700,
					},
					{
						Name:  "nginx.http.request.length",
						Value: 101.78571428571429,
					},
					{
						Name:  "nginx.http.request.malformed",
						Value: 1,
					},
					{
						Name:  "nginx.http.method.post",
						Value: 1,
					},
					{
						Name:  "nginx.http.method.get",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.delete",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.put",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.head",
						Value: 1,
					},
					{
						Name:  "nginx.http.method.options",
						Value: 4,
					},
					{
						Name:  "nginx.http.method.others",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.2xx",
						Value: 5,
					},
					{
						Name:  "nginx.http.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.4xx",
						Value: 4,
					},
					{
						Name:  "nginx.http.status.5xx",
						Value: 5,
					},
					{
						Name:  "nginx.http.status.403",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.404",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.500",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.502",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.503",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.504",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.discarded",
						Value: 1,
					},
					{
						Name:  "nginx.http.v0_9",
						Value: 1,
					},
					{
						Name:  "nginx.http.v1_0",
						Value: 1,
					},
					{
						Name:  "nginx.http.v1_1",
						Value: 11,
					},
					{
						Name:  "nginx.http.v2",
						Value: 1,
					},
				},
			},
		},
	}

	binary := core.NewNginxBinary(tutils.NewMockEnvironment(), &config.Config{})
	collectionDuration, _ := time.ParseDuration("300ms")
	sleepDuration, _ := time.ParseDuration("100ms")

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			context := context.TODO()
			accessLogFile, _ := ioutil.TempFile(os.TempDir(), "access.log")

			nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)
			go nginxAccessLog.logStats(context, accessLogFile.Name(), test.logFormat)

			time.Sleep(sleepDuration)
			for _, logLine := range test.logLines {
				_, err := accessLogFile.WriteString(logLine)
				if err != nil {
					tt.Fatalf("Error writing data to access log")
				}
			}

			time.Sleep(collectionDuration)

			accessLogFile.Close()
			os.Remove(accessLogFile.Name())

			// Sort metrics before doing comparison
			sort.SliceStable(test.expectedStats.GetSimplemetrics(), func(i, j int) bool {
				return test.expectedStats.GetSimplemetrics()[i].Name < test.expectedStats.GetSimplemetrics()[j].Name
			})
			sort.SliceStable(nginxAccessLog.buf[0].GetSimplemetrics(), func(i, j int) bool {
				return nginxAccessLog.buf[0].GetSimplemetrics()[i].Name < nginxAccessLog.buf[0].GetSimplemetrics()[j].Name
			})

			assert.Equal(tt, test.expectedStats.GetSimplemetrics(), nginxAccessLog.buf[0].GetSimplemetrics())
		})
	}
}
