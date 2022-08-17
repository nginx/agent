package nap

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	// package logger
	logger = logrus.New()
)

func init() {
	// Initial logger values
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	logger.WithField("package", "nginx-security")
}

// SetPackageLoggingValues sets the values of the logging done within this package to
// the values of the parameters passed in.
func SetPackageLoggingValues(output io.Writer, level logrus.Level) {
	logger.SetLevel(level)
	logger.SetOutput(output)
}
