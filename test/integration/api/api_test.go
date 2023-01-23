package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
    "github.com/testcontainers/testcontainers-go"
	wait "github.com/testcontainers/testcontainers-go/wait"
	dc "github.com/testcontainers/testcontainers-go/modules/compose"
)

const (
	port = 9091
)

func TestAPI_setupTestContainer(t *testing.T) {
	compose, err := dc.NewDockerCompose("docker-compose.yml")
	assert.NoError(t, err, "NewDockerComposeAPI()")

	t.Cleanup(func() {
		assert.NoError(t, compose.Down(context.Background(), dc.RemoveOrphans(true), dc.RemoveImagesLocal), "compose.Down()")
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	assert.NoError(t, compose.
		WaitForService("agent", wait.ForLog("OneTimeRegistration completed")).WithEnv(map[string]string{
		"PACKAGE": os.Getenv("PACKAGE"),
	}).
		Up(ctx, dc.Wait(true)), "compose.Up()")
}

func TestAPI_Nginx(t *testing.T) {

	TestAPI_setupTestContainer(t)

	client := resty.New()

	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/nginx", port)

	resp, err := client.R().EnableTrace().Get(url)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Contains(t, string(resp.String()), "nginx_id")
	assert.NotContains(t, string(resp.String()), "test_fail_nginx")

	nginxDetails := strings.Split(resp.String(), " ")

	for _, detail := range nginxDetails {
		detail := strings.Split(detail, ":")

		switch {
		case strings.Contains(detail[0], "nginx_id"):
			assert.NotNil(t, detail[1])

		case strings.Contains(detail[0], "version"):
			assert.NotNil(t, detail[1])

		case strings.Contains(detail[0], "runtime_modules"):
			assert.Equal(t, detail[1], "http_stub_status_module")

		case strings.Contains(detail[0], "conf_path"):
			assert.Equal(t, detail[1], "/usr/local/nginx/conf/nginx.conf")
		}
	}

}

func TestAPI_Metrics(t *testing.T) {

	TestAPI_setupTestContainer(t)
	client := resty.New()

	url := fmt.Sprintf("http://localhost:%d/metrics", port)
	client.SetRetryCount(5).SetRetryWaitTime(5 * time.Second).SetRetryMaxWaitTime(5 * time.Second)
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return len(r.String()) < 22000
		})

	resp, err := client.R().EnableTrace().Get(url)
	metrics := ProcessResponse(resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Contains(t, string(resp.String()), "system_cpu_system")
	assert.NotContains(t, string(resp.String()), "test_fail_metric")

	for _, m := range metrics {
		metric := strings.Split(m, " ")

		switch {
		case strings.Contains(metric[0], "system_cpu_system"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))

		case strings.Contains(metric[0], "system_mem_used_all"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))

		case strings.Contains(metric[0], "container_cpu_cores"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))

		case strings.Contains(metric[0], "nginx_workers_count"):
			value, _ := strconv.ParseFloat(metric[1], 64)
			assert.Greater(t, value, float64(0))
		}

	}

}

func ProcessResponse(resp *resty.Response) []string {
	metrics := strings.Split(resp.String(), "\n")

	i := 0

	for _, metric := range metrics {
		if metric[0:1] != "#" {
			metrics[i] = metric
			i++
		}
	}

	metrics = metrics[:i]

	return metrics

}
