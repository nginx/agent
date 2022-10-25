package collector

import (
	"github.com/sirupsen/logrus"
)

// NAPConfig holds the config for NAPConfig Collector.
type NAPConfig struct {
	SyslogIP   string
	SyslogPort int
	Logger     *logrus.Entry
}
