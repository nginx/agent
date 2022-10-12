package collector

import (
	"github.com/sirupsen/logrus"
)

// NAPWAFConfig holds the config for NAPWAFConfig Collector.
type NAPWAFConfig struct {
	SyslogIP   string
	SyslogPort int
	Logger     *logrus.Entry
}
