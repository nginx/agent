package utils

import (
	"github.com/go-resty/resty/v2"
	"strings"
)

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
