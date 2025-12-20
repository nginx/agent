// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"strings"

	events "github.com/nginx/agent/v3/api/grpc/events/v1"
)

// parseCSVLog parses comma-separated syslog messages where fields are in a
// order : blocking_exception_reason,dest_port,ip_client,is_truncated_bool,method,policy_name,protocol,request_status,response_code,severity,sig_cves,sig_set_names,src_port,sub_violations,support_id,threat_campaign_names,violation_rating,vs_name,x_forwarded_for_header_value,outcome,outcome_reason,violations,violation_details,bot_signature_name,bot_category,bot_anomalies,enforced_bot_anomalies,client_class,client_application,client_application_version,transport_protocol,uri,request (secops_dashboard-log profile format).
// versions when key-value logging isn't enabled.
//
//nolint:lll //long test string kept for log profile readability
func (p *securityViolationsProcessor) parseCSVLog(message string) map[string]string {
	fieldValueMap := make(map[string]string)

	// Remove the "ASM:" prefix if present so we only process the values
	message = strings.TrimPrefix(message, "ASM:")

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

func (p *securityViolationsProcessor) mapKVToSecurityViolationEvent(log *events.SecurityViolationEvent,
	kvMap map[string]string,
) {
	log.PolicyName = kvMap["policy_name"]
	log.SupportId = kvMap["support_id"]
	log.Outcome = kvMap["outcome"]
	log.OutcomeReason = kvMap["outcome_reason"]
	log.BlockingExceptionReason = kvMap["blocking_exception_reason"]
	log.Method = kvMap["method"]
	log.Protocol = kvMap["protocol"]
	log.XffHeaderValue = kvMap["x_forwarded_for_header_value"]
	log.Uri = kvMap["uri"]
	log.Request = kvMap["request"]
	log.IsTruncated = kvMap["is_truncated_bool"]
	log.RequestStatus = kvMap["request_status"]
	log.ResponseCode = kvMap["response_code"]
	log.ServerAddr = kvMap["server_addr"]
	log.VsName = kvMap["vs_name"]
	log.RemoteAddr = kvMap["ip_client"]
	log.DestinationPort = kvMap["dest_port"]
	log.ServerPort = kvMap["src_port"]
	log.Violations = kvMap["violations"]
	log.SubViolations = kvMap["sub_violations"]
	log.ViolationRating = kvMap["violation_rating"]
	log.SigSetNames = kvMap["sig_set_names"]
	log.SigCves = kvMap["sig_cves"]
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

	if log.GetRemoteAddr() == "" {
		log.RemoteAddr = kvMap["remote_addr"]
	}
	if log.GetDestinationPort() == "" {
		log.DestinationPort = kvMap["remote_port"]
	}
}
