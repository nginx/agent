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
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	wait "github.com/testcontainers/testcontainers-go/wait"
)

const (
	PORT = 9091
)

func setupTestContainer(t *testing.T) {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	t.Cleanup(func() {
		assert.NoError(t, comp.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t,
		comp.WaitForService("agent", wait.ForLog("OneTimeRegistration completed")).WithEnv(
			map[string]string{
				"PACKAGE_NAME": os.Getenv("PACKAGE_NAME"),
				"BASE_IMAGE":   os.Getenv("BASE_IMAGE"),
			},
		).Up(ctx, compose.Wait(true)), "compose.Up()")
}

func TestAPI_Nginx(t *testing.T) {
	setupTestContainer(t)

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/nginx", PORT)
	resp, err := client.R().EnableTrace().Get(url)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Contains(t, resp.String(), "nginx_id")
	assert.NotContains(t, resp.String(), "test_fail_nginx")

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
	setupTestContainer(t)
	client := resty.New()

	url := fmt.Sprintf("http://localhost:%d/metrics", PORT)
	client.SetRetryCount(5).SetRetryWaitTime(5 * time.Second).SetRetryMaxWaitTime(5 * time.Second)
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return len(r.String()) < 22000
		},
	)

	resp, err := client.R().EnableTrace().Get(url)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Contains(t, resp.String(), "system_cpu_system")
	assert.NotContains(t, resp.String(), "test_fail_metric")

	metrics := processResponse(resp)

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

func processResponse(resp *resty.Response) []string {
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
