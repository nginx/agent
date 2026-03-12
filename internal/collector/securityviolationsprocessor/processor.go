// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	syslog "github.com/leodido/go-syslog/v4"
	"github.com/leodido/go-syslog/v4/rfc3164"
	events "github.com/nginx/agent/v3/api/grpc/events/v1"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	maxSplitParts = 2
)

var ipRegex = regexp.MustCompile(`^ip-([0-9-]+)`)

// securityViolationsProcessor parses syslog-formatted log records and annotates
// them with structured SecurityEvent attributes.
type securityViolationsProcessor struct {
	nextConsumer consumer.Logs
	parser       syslog.Machine
	settings     processor.Settings
}

func newSecurityViolationsProcessor(next consumer.Logs, settings processor.Settings) *securityViolationsProcessor {
	return &securityViolationsProcessor{
		nextConsumer: next,
		parser:       rfc3164.NewParser(rfc3164.WithBestEffort()),
		settings:     settings,
	}
}

func (p *securityViolationsProcessor) Start(ctx context.Context, _ component.Host) error {
	p.settings.Logger.Info("Starting securityviolations processor")
	return nil
}

func (p *securityViolationsProcessor) Shutdown(ctx context.Context) error {
	p.settings.Logger.Info("Shutting down securityviolations processor")
	return nil
}

func (p *securityViolationsProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *securityViolationsProcessor) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	var errs error

	resourceLogs := logs.ResourceLogs()
	for _, scopeLog := range resourceLogs.All() {
		for _, logRecord := range scopeLog.ScopeLogs().All() {
			if err := p.processLogRecords(logRecord.LogRecords()); err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	if errs != nil {
		return fmt.Errorf("failed processing log records: %w", errs)
	}

	return p.nextConsumer.ConsumeLogs(ctx, logs)
}

func (p *securityViolationsProcessor) processLogRecords(logRecordSlice plog.LogRecordSlice) error {
	// Drop anything that isn't a string-bodied log before processing.
	var skipped, errCount int
	var logType pcommon.ValueType
	var errs error
	logRecordSlice.RemoveIf(func(lr plog.LogRecord) bool {
		logType = lr.Body().Type()
		if logType == pcommon.ValueTypeStr {
			return false
		}

		skipped++

		return true
	})
	if skipped > 0 {
		p.settings.Logger.Debug("Skipping log record with unsupported body type", zap.Any("type", logType))
	}
	errCount = 0
	for _, logRecord := range logRecordSlice.All() {
		if err := p.processLogRecord(logRecord); err != nil {
			errs = multierr.Append(errs, err)
			errCount++
		}
	}
	if errCount > 0 {
		p.settings.Logger.Debug("Some log records failed to process", zap.Int("count", errCount))
		return errs
	}

	return nil
}

func (p *securityViolationsProcessor) processLogRecord(lr plog.LogRecord) error {
	// Read the string body once.
	bodyStr := lr.Body().Str()

	msg, err := p.parser.Parse([]byte(bodyStr))
	if err != nil {
		return err
	}

	m, ok := msg.(*rfc3164.SyslogMessage)
	if !ok || !m.Valid() {
		return errors.New("invalid syslog message")
	}

	p.setSyslogAttributes(lr, m)

	if m.Message != nil {
		return p.processAppProtectMessage(lr, *m.Message, m.Hostname)
	}

	return nil
}

func (p *securityViolationsProcessor) setSyslogAttributes(lr plog.LogRecord, m *rfc3164.SyslogMessage) {
	attrs := lr.Attributes()
	if m.Timestamp != nil {
		attrs.PutStr("syslog.timestamp", m.Timestamp.Format(time.RFC3339))
	}
	if m.ProcID != nil {
		attrs.PutStr("syslog.procid", *m.ProcID)
	}
	if sev := m.SeverityLevel(); sev != nil {
		attrs.PutStr("syslog.severity", *sev)
	}
	if fac := m.FacilityLevel(); fac != nil {
		attrs.PutStr("syslog.facility", *fac)
	}
}

func (p *securityViolationsProcessor) processAppProtectMessage(lr plog.LogRecord,
	message string,
	hostname *string,
) error {
	appProtectLog := p.parseAppProtectLog(message, hostname)

	protoData, marshalErr := proto.Marshal(appProtectLog)
	if marshalErr != nil {
		return marshalErr
	}
	lr.Body().SetEmptyBytes().FromRaw(protoData)
	attrs := lr.Attributes()
	attrs.PutStr("app_protect.policy_name", appProtectLog.GetPolicyName())
	attrs.PutStr("app_protect.support_id", appProtectLog.GetSupportId())
	attrs.PutStr("app_protect.outcome", appProtectLog.GetRequestOutcome().String())
	attrs.PutStr("app_protect.remote_addr", appProtectLog.GetRemoteAddr())

	return nil
}

func (p *securityViolationsProcessor) parseAppProtectLog(
	message string, hostname *string,
) *events.SecurityViolationEvent {
	log := &events.SecurityViolationEvent{}

	assignHostnames(log, hostname)

	kvMap := parseCSVLog(message)

	mapKVToSecurityViolationEvent(log, kvMap)

	if log.GetServerAddr() == "" && hostname != nil {
		if ip := extractIPFromHostname(*hostname); ip != "" {
			log.ServerAddr = ip
		}
	}

	// Parse violations data from available fields
	log.ViolationsData = p.parseViolationsData(kvMap)

	return log
}

func assignHostnames(log *events.SecurityViolationEvent, hostname *string) {
	if hostname == nil {
		return
	}
	log.SystemId = *hostname

	if log.GetServerAddr() == "" {
		if ip := extractIPFromHostname(*hostname); ip != "" {
			log.ServerAddr = ip
		}
	}
}

func extractIPFromHostname(hostname string) string {
	if ip := net.ParseIP(hostname); ip != nil {
		return ip.String()
	}

	if matches := ipRegex.FindStringSubmatch(hostname); len(matches) > 1 {
		candidate := strings.ReplaceAll(matches[1], "-", ".")
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}

	return ""
}
