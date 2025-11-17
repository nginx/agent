// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

// SecurityViolationEvent represents the structured NGINX App Protect security violation data
type SecurityViolationEvent struct {
	PolicyName               string          `json:"policy_name"`
	SupportID                string          `json:"support_id"`
	Outcome                  string          `json:"outcome"`
	OutcomeReason            string          `json:"outcome_reason"`
	BlockingExceptionReason  string          `json:"blocking_exception_reason"`
	Method                   string          `json:"method"`
	Protocol                 string          `json:"protocol"`
	XForwardedForHeaderValue string          `json:"xff_header_value"`
	URI                      string          `json:"uri"`
	Request                  string          `json:"request"`
	IsTruncated              string          `json:"is_truncated"`
	RequestStatus            string          `json:"request_status"`
	ResponseCode             string          `json:"response_code"`
	ServerAddr               string          `json:"server_addr"`
	VSName                   string          `json:"vs_name"`
	RemoteAddr               string          `json:"remote_addr"`
	RemotePort               string          `json:"destination_port"`
	ServerPort               string          `json:"server_port"`
	Violations               string          `json:"violations"`
	SubViolations            string          `json:"sub_violations"`
	ViolationRating          string          `json:"violation_rating"`
	SigSetNames              string          `json:"sig_set_names"`
	SigCVEs                  string          `json:"sig_cves"`
	ClientClass              string          `json:"client_class"`
	ClientApplication        string          `json:"client_application"`
	ClientApplicationVersion string          `json:"client_application_version"`
	Severity                 string          `json:"severity"`
	ThreatCampaignNames      string          `json:"threat_campaign_names"`
	BotAnomalies             string          `json:"bot_anomalies"`
	BotCategory              string          `json:"bot_category"`
	EnforcedBotAnomalies     string          `json:"enforced_bot_anomalies"`
	BotSignatureName         string          `json:"bot_signature_name"`
	SystemID                 string          `json:"system_id"`
	InstanceTags             string          `json:"instance_tags"`
	InstanceGroup            string          `json:"instance_group"`
	ParentHostname           string          `json:"parent_hostname"`
	DisplayName              string          `json:"display_name"`
	ViolationsData           []ViolationData `json:"violations_data"`
}

type ViolationData struct {
	Name        string          `json:"violation_data_name"`
	Context     string          `json:"violation_data_context"`
	ContextData ContextData     `json:"violation_data_context_data"`
	Signatures  []SignatureData `json:"violation_data_signatures"`
}

// SignatureData represents signature data contained within each violation
type SignatureData struct {
	ID           string `json:"sig_data_id"`
	BlockingMask string `json:"sig_data_blocking_mask"`
	Buffer       string `json:"sig_data_buffer"`
	Offset       string `json:"sig_data_offset"`
	Length       string `json:"sig_data_length"`
}

// ContextData represents the context data of the violation
type ContextData struct {
	Name  string `json:"context_data_name"`
	Value string `json:"context_data_value"`
}
