// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package syslogprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	syslog "github.com/leodido/go-syslog/v4"
	"github.com/leodido/go-syslog/v4/rfc3164"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	notAvailable  = "N/A"
	maxSplitParts = 2
)

// syslogProcessor parses syslog-formatted log records and annotates
// them with structured SecurityEvent attributes.
type syslogProcessor struct {
	nextConsumer consumer.Logs
	parser       syslog.Machine
	settings     processor.Settings
}

func newSyslogProcessor(next consumer.Logs, settings processor.Settings) *syslogProcessor {
	return &syslogProcessor{
		nextConsumer: next,
		parser:       rfc3164.NewParser(rfc3164.WithBestEffort()),
		settings:     settings,
	}
}

func (p *syslogProcessor) Start(ctx context.Context, _ component.Host) error {
	p.settings.Logger.Info("Starting syslog processor")
	return nil
}

func (p *syslogProcessor) Shutdown(ctx context.Context) error {
	p.settings.Logger.Info("Shutting down syslog processor")
	return nil
}

func (p *syslogProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *syslogProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	var errs error

	rl := ld.ResourceLogs()
	for _, sl := range rl.All() {
		for _, lr := range sl.ScopeLogs().All() {
			if err := p.processLogRecords(lr.LogRecords()); err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	if errs != nil {
		return fmt.Errorf("failed processing log records: %w", errs)
	}

	return p.nextConsumer.ConsumeLogs(ctx, ld)
}

func (p *syslogProcessor) processLogRecords(lrs plog.LogRecordSlice) error {
	// Drop anything that isn't a string-bodied log before processing.
	var skipped, errCount int
	var t pcommon.ValueType
	var errs error
	lrs.RemoveIf(func(lr plog.LogRecord) bool {
		t = lr.Body().Type()
		if t == pcommon.ValueTypeStr {
			return false
		}

		skipped++
		return true
	})
	if skipped > 0 {
		p.settings.Logger.Debug("Skipping log record with unsupported body type", zap.Any("type", t))
	}
	errCount = 0
	for _, lr := range lrs.All() {
		if err := p.processLogRecord(lr); err != nil {
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

func (p *syslogProcessor) processLogRecord(lr plog.LogRecord) error {
	// Read the string body once.
	bodyStr := lr.Body().Str()

	msg, err := p.parser.Parse([]byte(bodyStr))
	if err != nil {
		p.settings.Logger.Debug("failed to parse syslog message",
			zap.Error(err),
			zap.String("body", bodyStr))

		return err
	}

	m, ok := msg.(*rfc3164.SyslogMessage)
	if !ok || !m.Valid() {
		p.settings.Logger.Debug("Invalid syslog message")

		return errors.New("invalid syslog message")
	}

	p.setSyslogAttributes(lr, m)

	if m.Message != nil {
		return p.processAppProtectMessage(lr, *m.Message, m.Hostname)
	}

	return nil
}

func (p *syslogProcessor) setSyslogAttributes(lr plog.LogRecord, m *rfc3164.SyslogMessage) {
	attrs := lr.Attributes()
	if m.Timestamp != nil {
		attrs.PutStr("syslog.timestamp", m.Timestamp.Format(time.RFC3339))
	}
	if m.Appname != nil {
		attrs.PutStr("syslog.appname", *m.Appname)
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

func (p *syslogProcessor) processAppProtectMessage(lr plog.LogRecord, message string, hostname *string) error {
	appProtectLog := p.parseAppProtectLog(message, hostname)

	jsonData, marshalErr := json.Marshal(appProtectLog)
	if marshalErr != nil {
		p.settings.Logger.Debug("failed to marshal App Protect log to JSON", zap.Error(marshalErr))
		lr.Body().SetStr(message)

		return marshalErr
	}

	lr.Body().SetStr(string(jsonData))
	attrs := lr.Attributes()
	attrs.PutStr("app_protect.policy_name", appProtectLog.PolicyName)
	attrs.PutStr("app_protect.support_id", appProtectLog.SupportID)
	attrs.PutStr("app_protect.outcome", appProtectLog.Outcome)
	attrs.PutStr("app_protect.remote_addr", appProtectLog.RemoteAddr)

	return nil
}

func (p *syslogProcessor) parseAppProtectLog(message string, hostname *string) *SecurityViolationEvent {
	log := &SecurityViolationEvent{}

	p.assignHostnames(log, hostname)

	kvMap := p.parseCSVLog(message)

	p.mapKVToSecurityViolationEvent(log, kvMap)

	if log.ServerAddr == "" && hostname != nil {
		if ip := extractIPFromHostname(*hostname); ip != "" {
			log.ServerAddr = ip
		}
	}

	// Parse violations data from available fields
	log.ViolationsData = p.parseViolationsData(kvMap)

	return log
}

func (p *syslogProcessor) assignHostnames(log *SecurityViolationEvent, hostname *string) {
	if hostname == nil {
		return
	}
	log.SystemID = *hostname
	log.ParentHostname = *hostname

	if log.ServerAddr == "" {
		if ip := extractIPFromHostname(*hostname); ip != "" {
			log.ServerAddr = ip
		}
	}
}

// nolint: lll
// parseCSVLog parses comma-separated syslog messages where fields are in a
// order : blocking_exception_reason,dest_port,ip_client,is_truncated_bool,method,policy_name,protocol,request_status,response_code,severity,sig_cves,sig_set_names,src_port,sub_violations,support_id,threat_campaign_names,violation_rating,vs_name,x_forwarded_for_header_value,outcome,outcome_reason,violations,violation_details,bot_signature_name,bot_category,bot_anomalies,enforced_bot_anomalies,client_class,client_application,client_application_version,transport_protocol,uri,request (secops_dashboard-log profile format).
// versions when key-value logging isn't enabled.
func (p *syslogProcessor) parseCSVLog(message string) map[string]string {
	kvMap := make(map[string]string)

	// Remove the "ASM:" prefix if present so we only process the values
	if idx := strings.Index(message, ":"); idx >= 0 {
		message = message[idx+1:]
	}

	fields := strings.Split(message, ",")

	// Mapping of CSV field positions to their corresponding keys
	fieldOrder := []string{
		"blocking_exception_reason",
		"dest_port",
		"ip_client",
		"is_truncated_bool",
		"method",
		"policy_name",
		"protocol",
		"request_status",
		"response_code",
		"severity",
		"sig_cves",
		"sig_set_names",
		"src_port",
		"sub_violations",
		"support_id",
		"threat_campaign_names",
		"violation_rating",
		"vs_name",
		"x_forwarded_for_header_value",
		"outcome",
		"outcome_reason",
		"violations",
		"violation_details",
		"bot_signature_name",
		"bot_category",
		"bot_anomalies",
		"enforced_bot_anomalies",
		"client_class",
		"client_application",
		"client_application_version",
		"transport_protocol",
		"uri",
		"request",
	}

	for i, field := range fields {
		if i >= len(fieldOrder) {
			break
		}
		kvMap[fieldOrder[i]] = strings.TrimSpace(field)
	}

	// combine multiple values separated by '::'
	if combined, ok := kvMap["sig_cves"]; ok {
		parts := strings.SplitN(combined, "::", maxSplitParts)
		kvMap["sig_ids"] = parts[0]
		if len(parts) > 1 {
			kvMap["sig_names"] = parts[1]
		}
	}

	if combined, ok := kvMap["sig_set_names"]; ok {
		parts := strings.SplitN(combined, "::", maxSplitParts)
		kvMap["sig_set_names"] = parts[0]
		if len(parts) > 1 {
			kvMap["sig_cves"] = parts[1]
		}
	}

	return kvMap
}

func (p *syslogProcessor) mapKVToSecurityViolationEvent(log *SecurityViolationEvent, kvMap map[string]string) {
	log.PolicyName = kvMap["policy_name"]
	log.SupportID = kvMap["support_id"]
	log.Outcome = kvMap["outcome"]
	log.OutcomeReason = kvMap["outcome_reason"]
	log.BlockingExceptionReason = kvMap["blocking_exception_reason"]
	log.Method = kvMap["method"]
	log.Protocol = kvMap["protocol"]
	log.XForwardedForHeaderValue = kvMap["x_forwarded_for_header_value"]
	log.URI = kvMap["uri"]
	log.Request = kvMap["request"]
	log.IsTruncated = kvMap["is_truncated_bool"]
	log.RequestStatus = kvMap["request_status"]
	log.ResponseCode = kvMap["response_code"]
	log.ServerAddr = kvMap["server_addr"]
	log.VSName = kvMap["vs_name"]
	log.RemoteAddr = kvMap["ip_client"]
	log.RemotePort = kvMap["dest_port"]
	log.ServerPort = kvMap["src_port"]
	log.Violations = kvMap["violations"]
	log.SubViolations = kvMap["sub_violations"]
	log.ViolationRating = kvMap["violation_rating"]
	log.SigSetNames = kvMap["sig_set_names"]
	log.SigCVEs = kvMap["sig_cves"]
	log.ClientClass = kvMap["client_class"]
	log.ClientApplication = kvMap["client_application"]
	log.ClientApplicationVersion = kvMap["client_application_version"]
	log.Severity = kvMap["severity"]
	log.ThreatCampaignNames = kvMap["threat_campaign_names"]
	log.BotAnomalies = kvMap["bot_anomalies"]
	log.BotCategory = kvMap["bot_category"]
	log.EnforcedBotAnomalies = kvMap["enforced_bot_anomalies"]
	log.BotSignatureName = kvMap["bot_signature_name"]
	log.InstanceTags = kvMap["instance_tags"]
	log.InstanceGroup = kvMap["instance_group"]
	log.DisplayName = kvMap["display_name"]

	if log.RemoteAddr == "" {
		log.RemoteAddr = kvMap["remote_addr"]
	}
	if log.RemotePort == "" {
		log.RemotePort = kvMap["remote_port"]
	}
}

// parseViolationsData extracts violation data from the syslog key-value map
func (p *syslogProcessor) parseViolationsData(kvMap map[string]string) []ViolationData {
	var violationsData []ViolationData

	// Extract violation name from violation_details XML - this is the only source
	violationName := ""
	if violationDetails := kvMap["violation_details"]; violationDetails != "" {
		violNameRegex := regexp.MustCompile(`<viol_name>([^<]+)</viol_name>`)
		if matches := violNameRegex.FindStringSubmatch(violationDetails); len(matches) > 1 {
			violationName = matches[1]
		}
	}

	// Create violation data if we have violation information
	if violationName != "" || kvMap["violations"] != "" {
		signatures := p.extractSignatureData(kvMap)
		if signatures == nil {
			signatures = []SignatureData{}
		}

		violationData := ViolationData{
			Name:        violationName,
			Context:     p.extractViolationContext(kvMap),
			ContextData: p.extractContextData(kvMap),
			Signatures:  signatures,
		}
		violationsData = append(violationsData, violationData)
	}

	return violationsData
}

// extractViolationContext extracts the violation context from syslog data
func (p *syslogProcessor) extractViolationContext(kvMap map[string]string) string {
	if uri := kvMap["uri"]; uri != "" {
		return uri
	}
	if method := kvMap["method"]; method != "" {
		return method
	}

	return ""
}

// extractContextData extracts context data from syslog
func (p *syslogProcessor) extractContextData(kvMap map[string]string) ContextData {
	contextData := ContextData{}

	if paramName := kvMap["parameter_name"]; paramName != "" {
		contextData.Name = paramName
		contextData.Value = kvMap["parameter_value"]
	} else if uri := kvMap["uri"]; uri != "" {
		// Use URI as context if no specific parameter data
		contextData.Name = "uri"
		contextData.Value = uri
	} else if request := kvMap["request"]; request != "" {
		// Use request as context if no URI
		contextData.Name = "request"
		contextData.Value = request
	}

	return contextData
}

// extractSignatureData extracts signature data from syslog
func (p *syslogProcessor) extractSignatureData(kvMap map[string]string) []SignatureData {
	sigIDs := kvMap["sig_ids"]
	sigNames := kvMap["sig_names"]
	blockingMask := kvMap["blocking_mask"]
	sigOffset := kvMap["sig_offset"]
	sigLength := kvMap["sig_length"]

	if sigIDs == "" || sigIDs == notAvailable {
		return []SignatureData{}
	}

	ids := splitAndTrim(sigIDs)
	names := splitAndTrim(sigNames)

	return buildSignatures(ids, names, blockingMask, sigOffset, sigLength)
}

func splitAndTrim(value string) []string {
	if strings.TrimSpace(value) == "" || value == notAvailable {
		return nil
	}

	parts := strings.Split(value, ",")

	var trimmedParts []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			trimmedParts = append(trimmedParts, trimmed)
		}
	}

	return trimmedParts
}
func buildSignatures(ids, names []string, mask, offset, length string) []SignatureData {
	signatures := make([]SignatureData, 0, len(ids))
	for i, id := range ids {
		if id == "" || id == notAvailable {
			continue
		}
		signature := SignatureData{
			ID:           id,
			BlockingMask: mask,
			Offset:       offset,
			Length:       length,
		}
		if i < len(names) {
			signature.Buffer = names[i]
		}
		signatures = append(signatures, signature)
	}

	return signatures
}

func extractIPFromHostname(hostname string) string {
	if ip := net.ParseIP(hostname); ip != nil {
		return ip.String()
	}

	re := regexp.MustCompile(`^ip-([0-9-]+)`)
	if matches := re.FindStringSubmatch(hostname); len(matches) > 1 {
		candidate := strings.ReplaceAll(matches[1], "-", ".")
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}

	return ""
}
