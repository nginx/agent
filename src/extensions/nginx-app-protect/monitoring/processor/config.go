package processor

import (
	"github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core/metrics"
)

// Config holds the config for Processor.
type Config struct {
	Logger     *logrus.Entry
	Workers    int
	CommonDims *metrics.CommonDim
}
