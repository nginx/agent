package utils

import (
	"github.com/go-resty/resty/v2"
	"strings"
)

func ProcessApiMetricResponse(resp *resty.Response) []string {
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

func ProcessApiNginxInstanceResponse(resp *resty.Response) []string {
	details := strings.ReplaceAll(resp.String(), "\"", "")
	details = strings.ReplaceAll(details, "\\", "")

	detail := strings.Split(details, ",")

	return detail

}
