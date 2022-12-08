package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	tc "github.com/testcontainers/testcontainers-go"
)

func TestAPI_setupTestContainer (t *testing.T){
	compose, err := tc.NewDockerCompose("docker-compose.yml")
    assert.NoError(t, err, "NewDockerComposeAPI()")

	t.Cleanup(func() {
        assert.NoError(t, compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesLocal), "compose.Down()")
    })

	ctx, cancel := context.WithCancel(context.Background())
    t.Cleanup(cancel)

    assert.NoError(t, compose.Up(ctx, tc.Wait(true)), "compose.Up()")
}

func TestAPI_Nginx (t *testing.T){

	TestAPI_setupTestContainer(t)

	port := 9091

	time.Sleep(15 * time.Second)

    client := resty.New()

    url := fmt.Sprintf("http://localhost:%d/nginx", port)

	resp, err := client.R().EnableTrace().Get(url)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Contains(t, string(resp.String()), "nginx_id")

	nginxDetails := strings.Split(resp.String(), " ")

	for _, detail := range nginxDetails{
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

	printResult(resp,err)
 
	
}

func TestAPI_Metrics (t *testing.T) {

	TestAPI_setupTestContainer(t)

	port := 9091

	time.Sleep(15 * time.Second)

    client := resty.New()

    url := fmt.Sprintf("http://localhost:%d/metrics", port)

    resp, err := client.R().EnableTrace().Get(url)

    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode())
    assert.Contains(t, string(resp.String()), "system_cpu_system")
    assert.NoError(t, err)

	printResult(resp, err)

	metrics := ProcessResponse(resp)

	for _, m := range metrics{
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

