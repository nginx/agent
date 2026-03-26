// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsfilterprocessor

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

const (
	csvSchemaName    = "secops-dashboard-log"
	csvSchemaVersion = "1.0"

	csvSchemaNameKey    = "csv.schema.name"
	csvSchemaVersionKey = "csv.schema.version"

	csvSeparator = "|"

	// expectedCSVFields is the number of pipe-separated fields in the
	// secops-dashboard-log profile format:
	// support_id|ip_client|src_port|dest_ip|dest_port|vs_name|policy_name|
	// method|uri|protocol|request_status|response_code|outcome|outcome_reason|
	// violation_rating|blocking_exception_reason|is_truncated_bool|sig_ids|
	// sig_names|sig_cves|sig_set_names|threat_campaign_names|sub_violations|
	// x_forwarded_for_header_value|violations|violation_details|request|geo_location
	expectedCSVFields = 28

	gatePending int32 = 0
	gateOpen    int32 = 1
	gateClosed  int32 = -1
)

// securityViolationsFilterProcessor passes log record bodies through untouched and sets
// resource-level schema attributes. It validates the first string body to confirm the
// NAP logging profile is producing pipe-separated CSV output. If validation fails, all
// subsequent messages are dropped until the OTel collector is restarted.
type securityViolationsFilterProcessor struct {
	nextConsumer consumer.Logs
	settings     processor.Settings
	gateOnce     sync.Once
	gateState    atomic.Int32 // gatePending → gateOpen | gateClosed
}

func newSecurityViolationsFilterProcessor(
	next consumer.Logs,
	settings processor.Settings,
) *securityViolationsFilterProcessor {
	return &securityViolationsFilterProcessor{
		nextConsumer: next,
		settings:     settings,
	}
}

func (p *securityViolationsFilterProcessor) Start(_ context.Context, _ component.Host) error {
	p.settings.Logger.Info("Starting security violations filter processor")
	return nil
}

func (p *securityViolationsFilterProcessor) Shutdown(_ context.Context) error {
	p.settings.Logger.Info("Shutting down security violations filter processor")
	return nil
}

func (p *securityViolationsFilterProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *securityViolationsFilterProcessor) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	// Gate already evaluated and closed — drop everything without further inspection.
	if p.gateState.Load() == gateClosed {
		return nil
	}

	p.filterLogRecords(logs)

	// Don't forward empty payloads when all records were dropped.
	if logs.LogRecordCount() == 0 {
		return nil
	}

	p.addSchemaAttributes(logs)

	return p.nextConsumer.ConsumeLogs(ctx, logs)
}

func (p *securityViolationsFilterProcessor) filterLogRecords(logs plog.Logs) {
	for _, rl := range logs.ResourceLogs().All() {
		for _, sl := range rl.ScopeLogs().All() {
			sl.LogRecords().RemoveIf(func(lr plog.LogRecord) bool {
				return p.shouldDropRecord(lr)
			})
		}
	}
}

func (p *securityViolationsFilterProcessor) shouldDropRecord(lr plog.LogRecord) bool {
	p.gateOnce.Do(func() {
		p.evaluateGate(lr)
	})

	return p.gateState.Load() != gateOpen
}

func (p *securityViolationsFilterProcessor) evaluateGate(lr plog.LogRecord) {
	if lr.Body().Type() != pcommon.ValueTypeStr {
		p.logNonStringBody(lr)
		p.gateState.Store(gateClosed)

		return
	}

	fieldCount := strings.Count(lr.Body().Str(), csvSeparator) + 1
	if fieldCount != expectedCSVFields {
		p.logInvalidCSVBody(fieldCount)
		p.gateState.Store(gateClosed)

		return
	}

	p.gateState.Store(gateOpen)
}

func (p *securityViolationsFilterProcessor) logNonStringBody(lr plog.LogRecord) {
	p.settings.Logger.Error(
		"Security violation log body is not a string. "+
			"All security violation logs will be dropped until the collector is restarted.",
		zap.String("type", lr.Body().Type().String()),
	)
}

func (p *securityViolationsFilterProcessor) logInvalidCSVBody(fieldCount int) {
	p.settings.Logger.Error(
		"Security violation log does not appear to be CSV format. "+
			"Ensure the NAP logging profile uses the secops-dashboard-log format. "+
			"All security violation logs will be dropped until the collector is restarted.",
		zap.Int("expected_fields", expectedCSVFields),
		zap.Int("actual_fields", fieldCount),
	)
}

func (p *securityViolationsFilterProcessor) addSchemaAttributes(logs plog.Logs) {
	for _, rl := range logs.ResourceLogs().All() {
		attrs := rl.Resource().Attributes()
		attrs.PutStr(csvSchemaNameKey, csvSchemaName)
		attrs.PutStr(csvSchemaVersionKey, csvSchemaVersion)
	}
}
