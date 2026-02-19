// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"encoding/csv"
	"strconv"
	"strings"

	events "github.com/nginx/agent/v3/api/grpc/events/v1"
)

// Mapping of CSV field positions to their corresponding keys
var fieldOrder = []string{
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

// parseCSVLog parses comma-separated syslog messages where fields are in a
// order : blocking_exception_reason,dest_port,ip_client,is_truncated_bool,method,policy_name,protocol,request_status,response_code,severity,sig_cves,sig_set_names,src_port,sub_violations,support_id,threat_campaign_names,violation_rating,vs_name,x_forwarded_for_header_value,outcome,outcome_reason,violations,violation_details,bot_signature_name,bot_category,bot_anomalies,enforced_bot_anomalies,client_class,client_application,client_application_version,transport_protocol,uri,request (secops_dashboard-log profile format).
// versions when key-value logging isn't enabled.
//
//nolint:lll //long test string kept for log profile readability
func parseCSVLog(message string) map[string]string {
	fieldValueMap := make(map[string]string, 33)

	// Remove the "ASM:" prefix if present so we only process the values
	message = strings.TrimPrefix(message, "ASM:")

	reader := csv.NewReader(strings.NewReader(message))
	reader.LazyQuotes = true
	fields, err := reader.Read()
	if err != nil {
		// fallback: return empty map if parsing fails
		return fieldValueMap
	}

	for i, field := range fields {
		if i >= len(fieldOrder) {
			break
		}
		fieldValueMap[fieldOrder[i]] = strings.TrimSpace(field)
	}

	// combine multiple values separated by '::'
	if combined, ok := fieldValueMap["sig_cves"]; ok {
		parts := strings.SplitN(combined, "::", maxSplitParts)
		fieldValueMap["sig_ids"] = parts[0]
		if len(parts) > 1 {
			fieldValueMap["sig_names"] = parts[1]
		}
	}

	if combined, ok := fieldValueMap["sig_set_names"]; ok {
		parts := strings.SplitN(combined, "::", maxSplitParts)
		fieldValueMap["sig_set_names"] = parts[0]
		if len(parts) > 1 {
			fieldValueMap["sig_cves"] = parts[1]
		}
	}

	return fieldValueMap
}

// parseOutcome converts string outcome to RequestOutcome enum
func parseOutcome(outcome string) events.RequestOutcome {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "passed":
		return events.RequestOutcome_REQUEST_OUTCOME_PASSED
	case "rejected":
		return events.RequestOutcome_REQUEST_OUTCOME_REJECTED
	default:
		return events.RequestOutcome_REQUEST_OUTCOME_UNKNOWN
	}
}

// parseIsTruncated converts string to boolean
func parseIsTruncated(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true
	default:
		return false
	}
}

// parseSeverity converts string severity to Severity enum
func parseSeverity(severity string) events.Severity {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "emergency":
		return events.Severity_SEVERITY_EMERGENCY
	case "alert":
		return events.Severity_SEVERITY_ALERT
	case "critical":
		return events.Severity_SEVERITY_CRITICAL
	case "error":
		return events.Severity_SEVERITY_ERROR
	case "warning":
		return events.Severity_SEVERITY_WARNING
	case "notice":
		return events.Severity_SEVERITY_NOTICE
	case "informational":
		return events.Severity_SEVERITY_INFORMATIONAL
	default:
		return events.Severity_SEVERITY_UNKNOWN
	}
}

// parseRequestOutcomeReason converts string outcome reason to RequestOutcomeReason enum
func parseRequestOutcomeReason(reason string) events.RequestOutcomeReason {
	switch strings.ToUpper(strings.TrimSpace(reason)) {
	case "SECURITY_WAF_OK":
		return events.RequestOutcomeReason_SECURITY_WAF_OK
	case "SECURITY_WAF_VIOLATION":
		return events.RequestOutcomeReason_SECURITY_WAF_VIOLATION
	case "SECURITY_WAF_FLAGGED":
		return events.RequestOutcomeReason_SECURITY_WAF_FLAGGED
	case "SECURITY_WAF_VIOLATION_TRANSPARENT":
		return events.RequestOutcomeReason_SECURITY_WAF_VIOLATION_TRANSPARENT
	default:
		return events.RequestOutcomeReason_SECURITY_WAF_UNKNOWN
	}
}

// parseRequestStatus converts string request status to RequestStatus enum
func parseRequestStatus(status string) events.RequestStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "blocked":
		return events.RequestStatus_REQUEST_STATUS_BLOCKED
	case "alerted":
		return events.RequestStatus_REQUEST_STATUS_ALERTED
	case "passed":
		return events.RequestStatus_REQUEST_STATUS_PASSED
	default:
		return events.RequestStatus_REQUEST_STATUS_UNKNOWN
	}
}

// parseUint32 converts string to uint32
func parseUint32(value string) uint32 {
	if val, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32); err == nil {
		return uint32(val)
	}
	return 0
}

func mapKVToSecurityViolationEvent(log *events.SecurityViolationEvent,
	kvMap map[string]string,
) {
	log.PolicyName = kvMap["policy_name"]
	log.SupportId = kvMap["support_id"]
	log.RequestOutcome = parseOutcome(kvMap["outcome"])
	log.RequestOutcomeReason = parseRequestOutcomeReason(kvMap["outcome_reason"])
	log.BlockingExceptionReason = kvMap["blocking_exception_reason"]
	log.Method = kvMap["method"]
	log.Protocol = kvMap["protocol"]
	log.XffHeaderValue = kvMap["x_forwarded_for_header_value"]
	log.Uri = kvMap["uri"]
	log.Request = kvMap["request"]
	log.IsTruncated = parseIsTruncated(kvMap["is_truncated_bool"])
	log.RequestStatus = parseRequestStatus(kvMap["request_status"])
	log.ResponseCode = parseUint32(kvMap["response_code"])
	log.ServerAddr = kvMap["server_addr"]
	log.VsName = kvMap["vs_name"]
	log.RemoteAddr = kvMap["ip_client"]
	log.DestinationPort = parseUint32(kvMap["dest_port"])
	log.ServerPort = parseUint32(kvMap["src_port"])
	log.Violations = kvMap["violations"]
	log.SubViolations = kvMap["sub_violations"]
	log.ViolationRating = parseUint32(kvMap["violation_rating"])
	log.SigSetNames = kvMap["sig_set_names"]
	log.SigCves = kvMap["sig_cves"]
	log.ClientClass = kvMap["client_class"]
	log.ClientApplication = kvMap["client_application"]
	log.ClientApplicationVersion = kvMap["client_application_version"]
	log.Severity = parseSeverity(kvMap["severity"])
	log.ThreatCampaignNames = kvMap["threat_campaign_names"]
	log.BotAnomalies = kvMap["bot_anomalies"]
	log.BotCategory = kvMap["bot_category"]
	log.EnforcedBotAnomalies = kvMap["enforced_bot_anomalies"]
	log.BotSignatureName = kvMap["bot_signature_name"]
	log.DisplayName = kvMap["display_name"]

	if log.GetRemoteAddr() == "" {
		log.RemoteAddr = kvMap["remote_addr"]
	}
	if log.GetDestinationPort() == 0 {
		log.DestinationPort = parseUint32(kvMap["remote_port"])
	}
}
