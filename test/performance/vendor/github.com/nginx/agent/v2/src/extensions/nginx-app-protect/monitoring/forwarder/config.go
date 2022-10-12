package forwarder

import "github.com/sirupsen/logrus"

// Config holds the config for Forwarder.
type Config struct {
	Logger *logrus.Entry
}
