// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/leodido/go-syslog/v4/rfc3164"
	events "github.com/nginx/agent/v3/api/grpc/events/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// loadTestData loads test data from testdata folder and wraps it in syslog format
func loadTestData(t *testing.T, filename string) string {
	t.Helper()
	data := loadRawTestData(t, filename)
	// Wrap in syslog format
	return "<130>Aug 22 03:28:35 ip-172-16-0-213 ASM:" + data
}

func loadRawTestData(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	require.NoError(t, err, "Failed to read test data file: %s", filename)

	return string(data)
}

// unmarshalEvent is a helper that extracts and unmarshals the security violation event from a log record
func unmarshalEvent(t *testing.T, lrOut plog.LogRecord) *events.SecurityViolationEvent {
	t.Helper()
	processedBody := lrOut.Body().Bytes().AsRaw()

	var actualEvent events.SecurityViolationEvent
	protoErr := proto.Unmarshal(processedBody, &actualEvent)
	require.NoError(t, protoErr, "Failed to unmarshal processed log body as SecurityViolationEvent")

	return &actualEvent
}

//nolint:lll,revive,maintidx // long test string kept for readability, table-driven test with many cases
func TestSecurityViolationsProcessor(t *testing.T) {
	testCases := []struct {
		expectAttrs   map[string]string
		body          any
		assertFunc    func(*testing.T, plog.LogRecord)
		name          string
		expectJSON    string
		expectRecords int
		expectError   bool
	}{
		{
			name: "Test 1: CSV NGINX App Protect syslog message",
			body: loadTestData(t, "csv_url_violations_bot_client.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "nms_app_protect_default_policy",
				"app_protect.support_id":  "5377540117854870581",
				"app_protect.outcome":     "REQUEST_OUTCOME_REJECTED",
				"app_protect.remote_addr": "127.0.0.1",
			},
			expectRecords: 1,
			assertFunc:    assertTest1Event,
		},
		{
			name: "Test 2: CSV NGINX App Protect with signatures",
			body: loadTestData(t, "csv_sql_injection_parameter_signatures.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "security_policy_01",
				"app_protect.support_id":  "9876543210123456789",
				"app_protect.outcome":     "REQUEST_OUTCOME_REJECTED",
				"app_protect.remote_addr": "10.0.1.50",
			},
			expectRecords: 1,
			assertFunc:    assertTest2Event,
		},
		{
			name:          "Test 3: Simple valid syslog message (non-App Protect)",
			body:          loadRawTestData(t, "syslog_non_app_protect.log.txt"),
			expectRecords: 1, // Processed successfully even though not App Protect format
		},
		{
			name:          "Test 4: Unsupported body type",
			body:          12345,
			expectRecords: 0,
		},
		{
			name:          "Test 5: Invalid syslog message",
			body:          loadRawTestData(t, "invalid_syslog_plain_text.log.txt"),
			expectRecords: 0,
			expectError:   true, // Error returned for invalid syslog
		},
		{
			name: "Test 6: Violation name parsing - VIOL_ASM_COOKIE_MODIFIED with cookie_name",
			body: loadTestData(t, "xml_violation_name.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592519",
			},
			expectRecords: 1,
			assertFunc:    assertTest6Event,
		},
		{
			name: "Test 7: Parameter data parsing with empty value_error",
			body: loadTestData(t, "xml_parameter_data_empty_context.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592517",
			},
			expectRecords: 1,
			assertFunc:    assertTest7Event,
		},
		{
			name: "Test 8: Header metachar with base64 text",
			body: loadTestData(t, "xml_header_text.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "3255056874564592516",
			},
			expectRecords: 1,
			assertFunc:    assertTest8Event,
		},
		{
			name: "Test 9: Cookie length violation",
			body: loadTestData(t, "xml_cookie_length.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "3255056874564592514",
			},
			expectRecords: 1,
			assertFunc:    assertTest9Event,
		},
		{
			name: "Test 10: Header length violation",
			body: loadTestData(t, "xml_header_length.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "3255056874564592515",
			},
			expectRecords: 1,
			assertFunc:    assertTest10Event,
		},
		{
			name: "Test 11: URL context with HeaderData",
			body: loadTestData(t, "xml_url_header_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "3255056874564592517",
			},
			expectRecords: 1,
			assertFunc:    assertTest11Event,
		},
		{
			name: "Test 12: Parameter value and name metachar violations",
			body: loadTestData(t, "xml_violation_parameter_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592511",
			},
			expectRecords: 1,
			assertFunc:    assertTest12Event,
		},
		{
			name: "Test 13: Request context violations (max length)",
			body: loadTestData(t, "xml_request_max_length.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592512",
			},
			expectRecords: 1,
			assertFunc:    assertTest13Event,
		},
		{
			name: "Test 14: URL metachar and length violations",
			body: loadTestData(t, "xml_url_metachar_length.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592513",
			},
			expectRecords: 1,
			assertFunc:    assertTest14Event,
		},
		{
			name: "Test 15: Parameter data parsing with signature",
			body: loadTestData(t, "xml_parameter_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592515",
			},
			expectRecords: 1,
			assertFunc:    assertTest15Event,
		},
		{
			name: "Test 16: Header data parsing with signature",
			body: loadTestData(t, "xml_header_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592514",
			},
			expectRecords: 1,
			assertFunc:    assertTest16Event,
		},
		{
			name: "Test 17: Signature data with multiple signatures in request",
			body: loadTestData(t, "xml_signature_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592518",
			},
			expectRecords: 1,
			assertFunc:    assertTest17Event,
		},
		{
			name: "Test 18: Cookie malformed violation",
			body: loadTestData(t, "xml_cookie_malformed.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592511",
			},
			expectRecords: 1,
			assertFunc:    assertTest18Event,
		},
		{
			name: "Test 19: Default context with no explicit context tag but HeaderData present",
			body: loadTestData(t, "xml_http_protocol_header_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592520",
			},
			expectRecords: 1,
			assertFunc:    assertTest19Event,
		},
		{
			name: "Test 20: Cookie violations - malformed, modified, expired",
			body: loadTestData(t, "xml_violation_cookie_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
			},
			expectRecords: 1,
			assertFunc:    assertTest20Event,
		},
		{
			name: "Test 21: URL violations - metachar, length, JSON malformed",
			body: loadTestData(t, "xml_violation_url_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
			},
			expectRecords: 1,
			assertFunc:    assertTest21Event,
		},
		{
			name: "Test 22: Request violations - max length, length",
			body: loadTestData(t, "xml_violation_request_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
			},
			expectRecords: 1,
			assertFunc:    assertTest22Event,
		},
		{
			name: "Test 23: Malformed XML with unclosed tag",
			body: loadTestData(t, "xml_malformed.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "5543056874564592513",
			},
			expectRecords: 1,
			assertFunc:    assertTest23Event,
		},
		{
			name: "Test 24: Parameter data with param_data structure",
			body: loadTestData(t, "xml_parameter_data_as_param_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592516",
			},
			expectRecords: 1,
			assertFunc:    assertTest24Event,
		},
		{
			name: "Test 25: Header violations - metachar and repeated",
			body: loadTestData(t, "xml_violation_header_data.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592511",
			},
			expectRecords: 1,
			assertFunc:    assertTest25Event,
		},
		{
			name: "Test 26: Unmatched XML structure",
			body: loadTestData(t, "xml_struct_unmatched.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "5543056874564592514",
			},
			expectRecords: 1,
			assertFunc:    assertTest26Event,
		},
		{
			name: "Test 27: Syslog with less fields than expected",
			body: loadTestData(t, "syslog_logline_less_fields.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "5543056874564592516",
			},
			expectRecords: 1,
			assertFunc:    assertTest27Event,
		},
		{
			name: "Test 28: Syslog with more fields than expected",
			body: loadTestData(t, "syslog_logline_more_fields.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "5543056874564592517",
			},
			expectRecords: 1,
			assertFunc:    assertTest28Event,
		},
		{
			name: "Test 29: URI and request with escaped commas",
			body: loadTestData(t, "uri_request_contain_escaped_comma.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592513",
			},
			expectRecords: 1,
			assertFunc:    assertTest29Event,
		},
		{
			name: "Test 30: Expanded NAP WAF log",
			body: loadTestData(t, "expanded_nap_waf.log.txt"),
			expectAttrs: map[string]string{
				"app_protect.policy_name": "app_protect_default_policy",
				"app_protect.support_id":  "4355056874564592513",
			},
			expectRecords: 1,
			assertFunc:    assertTest30Event,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			settings := processortest.NewNopSettings(processortest.NopType)
			settings.Logger = zap.NewNop()

			logs := plog.NewLogs()
			lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			switch v := tc.body.(type) {
			case string:
				lr.Body().SetStr(v)
			case int:
				lr.Body().SetInt(int64(v))
			case []byte:
				lr.Body().SetEmptyBytes().FromRaw(v)
			}

			sink := &consumertest.LogsSink{}
			p := newSecurityViolationsProcessor(sink, settings)
			require.NoError(t, p.Start(ctx, nil))

			err := p.ConsumeLogs(ctx, logs)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectRecords == 0 {
				assert.Equal(t, 0, sink.LogRecordCount(), "no logs should be produced")
				require.NoError(t, p.Shutdown(ctx))

				return
			}

			got := sink.AllLogs()[0]
			lrOut := got.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

			for k, v := range tc.expectAttrs {
				val, ok := lrOut.Attributes().Get(k)
				assert.True(t, ok, "attribute %s missing %v", k, v)
				assert.Equal(t, v, val.Str())
			}

			if tc.assertFunc != nil {
				tc.assertFunc(t, lrOut)
			}

			require.NoError(t, p.Shutdown(ctx))
		})
	}
}

func assertTest1Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	assert.Equal(t, "nms_app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "5377540117854870581", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, "N/A", actualEvent.GetXffHeaderValue())
	assert.Equal(t, "/<><script>", actualEvent.GetUri())
	assert.Equal(t, false, actualEvent.GetIsTruncated())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.Equal(t, uint32(0), actualEvent.GetResponseCode())
	assert.Equal(t, "172.16.0.213", actualEvent.GetServerAddr())
	assert.Equal(t, "1-localhost:1-/", actualEvent.GetVsName())
	assert.Equal(t, "127.0.0.1", actualEvent.GetRemoteAddr())
	assert.Equal(t, uint32(80), actualEvent.GetDestinationPort())
	assert.Equal(t, uint32(56064), actualEvent.GetServerPort())
	assert.Equal(t,
		"Illegal meta character in URL::Attack signature detected::"+
			"Violation Rating Threat detected::Bot Client Detected",
		actualEvent.GetViolations())
	assert.Equal(t, "N/A", actualEvent.GetSubViolations())
	assert.Equal(t, uint32(5), actualEvent.GetViolationRating())
	assert.Equal(t, "{High Accuracy Signatures;Cross Site Scripting Signatures}",
		actualEvent.GetSigSetNames())
	assert.Equal(t, "{High Accuracy Signatures; Cross Site Scripting Signatures}",
		actualEvent.GetSigCves())
	assert.Equal(t, "Untrusted Bot", actualEvent.GetClientClass())
	assert.Equal(t, "N/A", actualEvent.GetClientApplication())
	assert.Equal(t, "N/A", actualEvent.GetClientApplicationVersion())
	assert.Equal(t, events.Severity_SEVERITY_UNKNOWN, actualEvent.GetSeverity())
	assert.Equal(t, "N/A", actualEvent.GetThreatCampaignNames())
	assert.Equal(t, "N/A", actualEvent.GetBotAnomalies())
	assert.Equal(t, "HTTP Library", actualEvent.GetBotCategory())
	assert.Equal(t, "N/A", actualEvent.GetEnforcedBotAnomalies())
	assert.Equal(t, "curl", actualEvent.GetBotSignatureName())
	assert.Equal(t, "ip-172-16-0-213", actualEvent.GetSystemId())
	assert.Empty(t, actualEvent.GetDisplayName())

	// Test 1 has 5 violations in the XML:
	// VIOL_ATTACK_SIGNATURE, 2x VIOL_URL_METACHAR, VIOL_BOT_CLIENT, VIOL_RATING_THREAT
	require.Len(t, actualEvent.GetViolationsData(), 5)

	// First violation: VIOL_ATTACK_SIGNATURE with signatures
	assert.Equal(t, "VIOL_ATTACK_SIGNATURE", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "uri", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "uri",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "/<><script>",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())

	// Verify signatures from XML
	require.Len(t, actualEvent.GetViolationsData()[0].GetViolationDataSignatures(), 2)
	assert.Equal(t, uint32(200000099),
		actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataId())
	assert.Equal(t, "/<><script>",
		actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataBuffer())
	assert.Equal(t, uint32(200000093), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataId())
}

func assertTest2Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	assert.Equal(t, "security_policy_01", actualEvent.GetPolicyName())
	assert.Equal(t, "9876543210123456789", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "POST", actualEvent.GetMethod())
	assert.Equal(t, "HTTPS", actualEvent.GetProtocol())
	assert.Equal(t, "/api/users", actualEvent.GetUri())
	assert.Equal(t, "10.0.1.50", actualEvent.GetRemoteAddr())
	assert.Equal(t, uint32(443), actualEvent.GetDestinationPort())
	assert.Equal(t, uint32(8080), actualEvent.GetServerPort())

	require.Len(t, actualEvent.GetViolationsData(), 1)
	assert.Equal(t, "VIOL_ATTACK_SIGNATURE", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "parameter", actualEvent.GetViolationsData()[0].GetViolationDataContext())

	// Context data should be extracted from XML parameter_data
	assert.Equal(t, "id",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "1' OR '1'='1",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())

	// Signatures array should be extracted from XML sig_data blocks
	require.NotNil(t, actualEvent.GetViolationsData()[0].GetViolationDataSignatures())
	require.Len(t, actualEvent.GetViolationsData()[0].GetViolationDataSignatures(), 2)

	// Verify first signature
	assert.Equal(t, uint32(200001475), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataId())
	assert.Equal(t, "7",
		actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataBlockingMask())
	assert.Equal(t, "1' OR '1'='1",
		actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataBuffer())
	assert.Equal(t, uint32(0), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataOffset())
	assert.Equal(t, uint32(15), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataLength())

	// Verify second signature
	assert.Equal(t, uint32(200001476), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataId())
	assert.Equal(t, "7",
		actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataBlockingMask())
	assert.Equal(t, "1' OR '1'='1",
		actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataBuffer())
	assert.Equal(t, uint32(5), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataOffset())
	assert.Equal(t, uint32(10), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataLength())
}

func assertTest6Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "4355056874564592519", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, "/", actualEvent.GetUri())
	assert.Equal(t, "127.0.0.1", actualEvent.GetRemoteAddr())
	assert.Equal(t, uint32(80), actualEvent.GetDestinationPort())
	assert.Equal(t, uint32(61478), actualEvent.GetServerPort())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.Equal(t, uint32(0), actualEvent.GetResponseCode())
	assert.Equal(t, events.Severity_SEVERITY_CRITICAL, actualEvent.GetSeverity())
	assert.Equal(t, uint32(5), actualEvent.GetViolationRating())
	assert.Equal(t, "Untrusted Bot", actualEvent.GetClientClass())
	assert.Equal(t, "HTTP Library", actualEvent.GetBotCategory())
	assert.Equal(t, "curl", actualEvent.GetBotSignatureName())

	// Verify we have 4 violations of type VIOL_ASM_COOKIE_MODIFIED
	require.Len(t, actualEvent.GetViolationsData(), 4)

	// Verify first violation has cookie name decoded from base64
	assert.Equal(t, "VIOL_ASM_COOKIE_MODIFIED", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "cookie", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "Asm Cookie Modified",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "TS0144e914_1",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())

	// Verify second violation
	assert.Equal(t, "TS0144e914_31",
		actualEvent.GetViolationsData()[1].GetViolationDataContextData().GetContextDataValue())
}

func assertTest7Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "4355056874564592517", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	// Verify we have 2 violations of type VIOL_PARAMETER
	require.Len(t, actualEvent.GetViolationsData(), 2)

	// Verify first parameter violation with empty value_error field
	assert.Equal(t, "VIOL_PARAMETER", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "parameter", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "x", actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "1", actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())

	// Verify second parameter violation
	assert.Equal(t, "y", actualEvent.GetViolationsData()[1].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "%", actualEvent.GetViolationsData()[1].GetViolationDataContextData().GetContextDataValue())
}

func assertTest8Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "3255056874564592516", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	// Verify we have 1 violation of type VIOL_HEADER_METACHAR
	require.Len(t, actualEvent.GetViolationsData(), 1)

	// Verify header violation with base64 decoded text
	assert.Equal(t, "VIOL_HEADER_METACHAR", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "header", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "Header Metachar",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "Referer: aa'bbb'",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())
}

func assertTest9Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "3255056874564592514", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	// Verify we have 1 violation of type VIOL_COOKIE_LENGTH
	require.Len(t, actualEvent.GetViolationsData(), 1)

	// Verify cookie length violation
	assert.Equal(t, "VIOL_COOKIE_LENGTH", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "cookie", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "Cookie length: 28, exceeds Cookie length limit: 10",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "Cookie: dfdfdfdfdf=dfdfdfdf;",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())
}

func assertTest10Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "3255056874564592515", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	// Verify we have 1 violation of type VIOL_HEADER_LENGTH
	require.Len(t, actualEvent.GetViolationsData(), 1)

	// Verify header length violation
	assert.Equal(t, "VIOL_HEADER_LENGTH", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "header", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "Host: dflkdjfldkfldkfldkflkdflkdflkdlfkdlf",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "Header length: 42, exceeds Header length limit: 10",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())
}

func assertTest11Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "3255056874564592517", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	// Verify we have 1 violation of type VIOL_URL_CONTENT_TYPE
	require.Len(t, actualEvent.GetViolationsData(), 1)

	// Verify URL violation with HeaderData
	assert.Equal(t, "VIOL_URL_CONTENT_TYPE", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "uri", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "$", actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "actual header value: beni. matched header value: beni",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())
}

func assertTest12Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "4355056874564592511", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	require.GreaterOrEqual(t, len(actualEvent.GetViolationsData()), 2)
	// xml_violation_parameter_data.log.txt has: VIOL_PARAMETER_VALUE_METACHAR, VIOL_PARAMETER_NAME_METACHAR,
	// VIOL_PARAMETER_VALUE_LENGTH, VIOL_PARAMETER_VALUE_BASE64
	assert.Equal(t, "VIOL_PARAMETER_VALUE_METACHAR", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "parameter", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "VIOL_PARAMETER_NAME_METACHAR", actualEvent.GetViolationsData()[1].GetViolationDataName())
	assert.Equal(t, "parameter", actualEvent.GetViolationsData()[1].GetViolationDataContext())
}

func assertTest13Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "4355056874564592512", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	require.Len(t, actualEvent.GetViolationsData(), 2)
	assert.Equal(t, "VIOL_REQUEST_MAX_LENGTH", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "request", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "Defined length: 10000000",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "Detected length: 11000125",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())

	assert.Equal(t, "VIOL_REQUEST_LENGTH", actualEvent.GetViolationsData()[1].GetViolationDataName())
	assert.Equal(t, "Total length: 11000000",
		actualEvent.GetViolationsData()[1].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "Total length limit: 10000000",
		actualEvent.GetViolationsData()[1].GetViolationDataContextData().GetContextDataValue())
}

func assertTest14Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Validate basic fields
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "4355056874564592513", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestOutcome_REQUEST_OUTCOME_REJECTED, actualEvent.GetRequestOutcome())
	assert.Equal(t, events.RequestOutcomeReason_SECURITY_WAF_VIOLATION, actualEvent.GetRequestOutcomeReason())
	assert.Equal(t, "GET", actualEvent.GetMethod())
	assert.Equal(t, "HTTP", actualEvent.GetProtocol())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
	assert.NotEmpty(t, actualEvent.GetRemoteAddr())

	require.GreaterOrEqual(t, len(actualEvent.GetViolationsData()), 2)
	assert.Equal(t, "VIOL_URL_METACHAR", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "uri", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	// URI field is not decoded in this case
	uriValue := actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue()
	assert.Contains(t, []string{"LztzaHV0ZG93bg==", "/;shutdown"}, uriValue)

	assert.Equal(t, "VIOL_URL_LENGTH", actualEvent.GetViolationsData()[1].GetViolationDataName())
	assert.Equal(t, "URI length: 30",
		actualEvent.GetViolationsData()[1].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "URI length limit: 20",
		actualEvent.GetViolationsData()[1].GetViolationDataContextData().GetContextDataValue())
}

func assertTest15Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Expected violations from xml_parameter_data.log.txt
	expectedViolations := []string{
		"VIOL_ATTACK_SIGNATURE",
		"VIOL_PARAMETER_REPEATED",
		"VIOL_PARAMETER_MULTIPART_NULL_VALUE",
		"VIOL_PARAMETER_EMPTY_VALUE",
		"VIOL_PARAMETER_VALUE_REGEXP",
		"VIOL_PARAMETER_NUMERIC_VALUE",
		"VIOL_PARAMETER_DATA_TYPE",
		"VIOL_PARAMETER_VALUE_LENGTH",
		"VIOL_PARAMETER_DYNAMIC_VALUE",
		"VIOL_PARAMETER_STATIC_VALUE",
		"VIOL_PARAMETER_ARRAY_VALUE",
		"VIOL_PARAMETER_LOCATION",
	}

	require.Len(t, actualEvent.GetViolationsData(), len(expectedViolations),
		"Should have all violations from test data")

	// Verify we have exactly the expected number of violations
	assert.Len(t, actualEvent.GetViolationsData(), len(expectedViolations), "Should have all expected violations")

	// Check all violations are present
	actualViolationNames := make([]string, len(actualEvent.GetViolationsData()))
	for i, v := range actualEvent.GetViolationsData() {
		actualViolationNames[i] = v.GetViolationDataName()
	}
	assert.ElementsMatch(t, expectedViolations, actualViolationNames, "All expected violations should be present")

	// Verify first violation details
	assert.Equal(t, "VIOL_ATTACK_SIGNATURE", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "parameter", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "Attack Signature",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "f5paramautotest>",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())

	require.GreaterOrEqual(t, len(actualEvent.GetViolationsData()[0].GetViolationDataSignatures()), 1,
		"Should have at least one signature")
	assert.Equal(t, uint32(300000110), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataId())
	assert.Equal(t, "7", actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataBlockingMask())
}

func assertTest16Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Verify we have exactly 1 violation
	require.Len(t, actualEvent.GetViolationsData(), 1, "Should have exactly 1 violation")

	// Verify all expected violations are present
	expectedViolations := []string{"VIOL_ATTACK_SIGNATURE"}
	actualViolationNames := []string{actualEvent.GetViolationsData()[0].GetViolationDataName()}
	assert.ElementsMatch(t, expectedViolations, actualViolationNames, "All expected violations should be present")

	assert.Equal(t, "VIOL_ATTACK_SIGNATURE", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "header", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	assert.Equal(t, "Foo", actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "echo<!-- #echo",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())
}

func assertTest17Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	require.Len(t, actualEvent.GetViolationsData(), 1)
	assert.Equal(t, "VIOL_ATTACK_SIGNATURE", actualEvent.GetViolationsData()[0].GetViolationDataName())

	// xml_signature_data.log.txt has 4 signatures
	require.GreaterOrEqual(t, len(actualEvent.GetViolationsData()[0].GetViolationDataSignatures()), 2)
	assert.Equal(t, uint32(200021094), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataId())
	assert.Equal(t, "4", actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[0].GetSigDataBlockingMask())
	assert.Equal(t, uint32(200011034), actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataId())
	assert.Equal(t, "2", actualEvent.GetViolationsData()[0].GetViolationDataSignatures()[1].GetSigDataBlockingMask())
}

func assertTest18Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	require.Len(t, actualEvent.GetViolationsData(), 1)
	assert.Equal(t, "VIOL_COOKIE_MALFORMED", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "cookie", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	// Cookie malformed uses buffer and specific_desc which are decoded
	assert.NotEmpty(t, actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.NotEmpty(t, actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())
}

func assertTest19Event(t *testing.T, record plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, record)

	require.Len(t, actualEvent.GetViolationsData(), 1)
	assert.Equal(t, "VIOL_HTTP_PROTOCOL", actualEvent.GetViolationsData()[0].GetViolationDataName())
	// Context should be empty (default case)
	assert.Empty(t, actualEvent.GetViolationsData()[0].GetViolationDataContext())
	// Context data should be extracted from HeaderData fields
	assert.Equal(t, "Content-Type",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Equal(t, "actual header value: application/json. matched header value: text/html",
		actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())
}

func assertTest20Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Expected violations from xml_violation_cookie_data.log.txt
	expectedViolations := []string{
		"VIOL_COOKIE_MALFORMED",
		"VIOL_COOKIE_MODIFIED",
		"VIOL_COOKIE_EXPIRED",
	}

	// Verify we have exactly the expected number of violations
	assert.Len(t, actualEvent.GetViolationsData(), len(expectedViolations),
		"Should have all expected cookie violations")

	// Check all violations are present
	actualViolationNames := make([]string, len(actualEvent.GetViolationsData()))
	for i, v := range actualEvent.GetViolationsData() {
		actualViolationNames[i] = v.GetViolationDataName()
		assert.Equal(t, "cookie", v.GetViolationDataContext(),
			"All violations should have cookie context")
	}
	assert.ElementsMatch(t, expectedViolations, actualViolationNames,
		"All expected cookie violations should be present")
}

func assertTest21Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Expected violations from xml_violation_url_data.log.txt
	expectedViolations := []string{
		"VIOL_URL_METACHAR",
		"VIOL_URL_LENGTH",
		"VIOL_JSON_MALFORMED",
		"VIOL_URL",
	}

	// Verify we have exactly the expected number of violations
	assert.Len(t, actualEvent.GetViolationsData(), len(expectedViolations), "Should have all expected URL violations")

	// Check all violations are present
	actualViolationNames := make([]string, len(actualEvent.GetViolationsData()))
	for i, v := range actualEvent.GetViolationsData() {
		actualViolationNames[i] = v.GetViolationDataName()
	}
	assert.ElementsMatch(t, expectedViolations, actualViolationNames, "All expected URL violations should be present")
}

func assertTest22Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// Expected violations from xml_violation_request_data.log.txt
	expectedViolations := []string{
		"VIOL_REQUEST_MAX_LENGTH",
		"VIOL_REQUEST_LENGTH",
	}

	// Verify we have exactly the expected number of violations
	assert.Len(t, actualEvent.GetViolationsData(), len(expectedViolations),
		"Should have all expected request violations")

	// Check all violations are present
	actualViolationNames := make([]string, len(actualEvent.GetViolationsData()))
	for i, v := range actualEvent.GetViolationsData() {
		actualViolationNames[i] = v.GetViolationDataName()
		assert.Equal(t, "request", v.GetViolationDataContext(),
			"All violations should have request context")
	}
	assert.ElementsMatch(t, expectedViolations, actualViolationNames,
		"All expected request violations should be present")
}

func assertTest23Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// xml_malformed.log.txt has malformed XML with unclosed sig_id tag
	// Processor handles malformed XML gracefully by logging warning and returning empty violations_data
	assert.Empty(t, actualEvent.GetViolationsData(), "Malformed XML should result in empty violations_data")
	// But other fields should still be populated from CSV
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "5543056874564592513", actualEvent.GetSupportId())
	assert.Equal(t, events.RequestStatus_REQUEST_STATUS_BLOCKED, actualEvent.GetRequestStatus())
}

func assertTest24Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// xml_parameter_data_as_param_data.log.txt uses param_data with param_name/param_value tags
	// Parser successfully parses the violation but doesn't extract context_data
	// because ParamData struct expects name/value tags, not param_name/param_value
	require.Len(t, actualEvent.GetViolationsData(), 1)
	assert.Equal(t, "VIOL_JSON_MALFORMED", actualEvent.GetViolationsData()[0].GetViolationDataName())
	assert.Equal(t, "parameter", actualEvent.GetViolationsData()[0].GetViolationDataContext())
	// Context data is empty because param_name/param_value tags aren't mapped in ParamData struct
	assert.NotNil(t, actualEvent.GetViolationsData()[0].GetViolationDataContextData())
	assert.Empty(t, actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataName())
	assert.Empty(t, actualEvent.GetViolationsData()[0].GetViolationDataContextData().GetContextDataValue())

	// Basic fields should still be populated from CSV
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "4355056874564592516", actualEvent.GetSupportId())
}

func assertTest25Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// xml_violation_header_data.log.txt has header violations
	expectedViolations := []string{
		"VIOL_HEADER_METACHAR",
		"VIOL_HEADER_REPEATED",
	}

	assert.Len(t, actualEvent.GetViolationsData(), len(expectedViolations),
		"Should have all expected header violations")

	actualViolationNames := make([]string, len(actualEvent.GetViolationsData()))
	for i, v := range actualEvent.GetViolationsData() {
		actualViolationNames[i] = v.GetViolationDataName()
		assert.Equal(t, "header", v.GetViolationDataContext(),
			"All violations should have header context")
	}
	assert.ElementsMatch(t, expectedViolations, actualViolationNames,
		"All expected header violations should be present")
}

func assertTest26Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// xml_struct_unmatched.log.txt has <UNMATCHED_STRUCT> instead of <BAD_MSG>
	// Should still process the CSV fields but violations_data should be empty
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "5543056874564592514", actualEvent.GetSupportId())
	// Violations data may be empty or minimal since XML structure is unmatched
}

func assertTest27Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// syslog_logline_less_fields.log.txt has fewer fields (missing last field)
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "5543056874564592516", actualEvent.GetSupportId())
	// Should still parse violations successfully
	require.GreaterOrEqual(t, len(actualEvent.GetViolationsData()), 1, "Should have at least one violation")
}

func assertTest28Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// syslog_logline_more_fields.log.txt has extra field at the end
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "5543056874564592517", actualEvent.GetSupportId())
	// Should still parse violations successfully, ignoring extra field
	require.GreaterOrEqual(t, len(actualEvent.GetViolationsData()), 1, "Should have at least one violation")
}

func assertTest29Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// uri_request_contain_escaped_comma.log.txt has %2C (escaped comma) in URI and request
	assert.Equal(t, "app_protect_default_policy", actualEvent.GetPolicyName())
	assert.Equal(t, "4355056874564592513", actualEvent.GetSupportId())
	assert.Contains(t, actualEvent.GetUri(), "comma", "URI should contain 'comma'")
	assert.Contains(t, actualEvent.GetRequest(), "%2C", "Request should contain escaped comma")
	require.GreaterOrEqual(t, len(actualEvent.GetViolationsData()), 1, "Should have at least one violation")
}

func assertTest30Event(t *testing.T, lrOut plog.LogRecord) {
	t.Helper()
	actualEvent := unmarshalEvent(t, lrOut)

	// expanded_nap_waf.log.txt is standard format, similar to other tests
	expectedViolations := []string{
		"VIOL_ATTACK_SIGNATURE",
		"VIOL_HTTP_PROTOCOL",
		"VIOL_PARAMETER_VALUE_METACHAR",
	}

	assert.Len(t, actualEvent.GetViolationsData(), len(expectedViolations), "Should have all expected violations")

	actualViolationNames := make([]string, len(actualEvent.GetViolationsData()))
	for i, v := range actualEvent.GetViolationsData() {
		actualViolationNames[i] = v.GetViolationDataName()
	}
	assert.ElementsMatch(t, expectedViolations, actualViolationNames, "All expected violations should be present")
}

func TestSecurityViolationsProcessor_ExtractIPFromHostname(t *testing.T) {
	assert.Equal(t, "127.0.0.1", extractIPFromHostname("127.0.0.1"))
	assert.Equal(t, "172.16.0.213", extractIPFromHostname("ip-172-16-0-213"))
	assert.Empty(t, extractIPFromHostname("not-an-ip"))
}

func TestSetSyslogAttributesNilFields(t *testing.T) {
	lr := plog.NewLogRecord()
	m := &rfc3164.SyslogMessage{}
	p := newSecurityViolationsProcessor(&consumertest.LogsSink{}, processortest.NewNopSettings(processortest.NopType))
	p.setSyslogAttributes(lr, m)
	attrs := lr.Attributes()
	assert.Equal(t, 0, attrs.Len())
}
