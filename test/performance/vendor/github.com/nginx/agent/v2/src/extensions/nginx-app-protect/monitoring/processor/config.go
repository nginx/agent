package processor

import (
	"github.com/sirupsen/logrus"
)

// Config holds the config for Processor.
type Config struct {
	Logger  *logrus.Entry
	Workers int
}
