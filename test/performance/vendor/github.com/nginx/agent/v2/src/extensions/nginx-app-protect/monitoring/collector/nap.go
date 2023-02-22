/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/mcuadros/go-syslog.v2"

	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
)

const (
	napComponentName = "collector:nap"
)

var (
	// logging fields for the component
	componentLogFields = logrus.Fields{
		"component": napComponentName,
	}
)

// NAPCollector lets you to Collect log data on given port.
type NAPCollector struct {
	syslog *syslogServer
	logger *logrus.Entry
}

type syslogServer struct {
	channel syslog.LogPartsChannel
	handler *syslog.ChannelHandler
	server  *syslog.Server
}

// NewNAPCollector gives you a NAP collector for the syslog server.
func NewNAPCollector(cfg *NAPConfig) (napCollector *NAPCollector, err error) {
	napCollector = &NAPCollector{}

	napCollector.logger = logrus.StandardLogger().WithFields(componentLogFields)
	if cfg.Logger != nil {
		napCollector.logger = cfg.Logger.WithFields(componentLogFields)
	}
	napCollector.logger.Infof("Getting %s Collector", monitoring.NAP)

	napCollector.syslog, err = newSyslogServer(napCollector.logger, cfg.SyslogIP, cfg.SyslogPort)
	if err != nil {
		return nil, err
	}

	return napCollector, nil
}

func newSyslogServer(logger *logrus.Entry, ip string, port int) (*syslogServer, error) {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)
	server.SetHandler(handler)

	addr := fmt.Sprintf("%s:%d", ip, port)
	err := server.ListenTCP(addr)
	if err != nil {
		msg := fmt.Sprintf("error while configuring syslog server to listen on %s:\n %v", addr, err)
		logger.Error(msg)
		return nil, err
	}

	err = server.Boot()
	if err != nil {
		msg := fmt.Sprintf("error while booting the syslog server at %s:\n %v ", addr, err)
		logger.Error(msg)
		return nil, err
	}

	return &syslogServer{channel, handler, server}, nil
}

// Collect starts collecting on collect chan until done chan gets a signal.
func (nap *NAPCollector) Collect(ctx context.Context, wg *sync.WaitGroup, collect chan<- *monitoring.RawLog) {
	defer wg.Done()

	nap.logger.Infof("Starting collection for %s", monitoring.NAP)

	for {
		select {
		case logParts := <-nap.syslog.channel:
			line, ok := logParts["content"].(string)
			if !ok {
				nap.logger.Warnf("Noncompliant syslog message, got: %v", logParts)
				break
			}

			nap.logger.Tracef("collected log line succesfully: %v", line)
			collect <- &monitoring.RawLog{Origin: monitoring.NAP, Logline: line}
		case <-ctx.Done():
			nap.logger.Infof("Context cancellation, collector is wrapping up...")

			err := nap.syslog.server.Kill()
			if err != nil {
				nap.logger.Errorf("Error while killing syslog collector server: %v", err)
			}

			return
		}
	}
}
