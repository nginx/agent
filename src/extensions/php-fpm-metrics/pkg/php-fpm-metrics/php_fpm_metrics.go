package php_fpm

import (
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/reader"
)

type PhpFpmMetrics struct {
	reader *reader.StatusPageReader
}

func NewPhpFpmMetrics() (*PhpFpmMetrics, error) {
	return &PhpFpmMetrics{}, nil
}
