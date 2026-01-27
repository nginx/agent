// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"context"
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

//nolint:lll,revive // long test string kept for readability
func TestSecurityViolationsProcessor(t *testing.T) {
	testCases := []struct {
		expectAttrs   map[string]string
		body          any
		name          string
		expectJSON    string
		expectRecords int
		expectError   bool
	}{
		{
			name: "Test 1: CSV NGINX App Protect syslog message",
			body: `<130>Aug 22 03:28:35 ip-172-16-0-213 ASM:N/A,80,127.0.0.1,false,GET,nms_app_protect_default_policy,HTTP,blocked,0,N/A,N/A::N/A,{High Accuracy Signatures;Cross Site Scripting Signatures}::{High Accuracy Signatures; Cross Site Scripting Signatures},56064,N/A,5377540117854870581,N/A,5,1-localhost:1-/,N/A,REJECTED,SECURITY_WAF_VIOLATION,Illegal meta character in URL::Attack signature detected::Violation Rating Threat detected::Bot Client Detected,<?xml version='1.0' encoding='UTF-8'?><BAD_MSG><violation_masks><block>414000000200c00-3a03030c30000072-8000000000000000-0</block><alarm>475f0ffcbbd0fea-befbf35cb000007e-f400000000000000-0</alarm><learn>0-0-0-0</learn><staging>0-0-0-0</staging></violation_masks><request-violations><violation><viol_index>42</viol_index><viol_name>VIOL_ATTACK_SIGNATURE</viol_name><context>url</context><sig_data><sig_id>200000099</sig_id><blocking_mask>3</blocking_mask><kw_data><buffer>Lzw+PHNjcmlwdD4=</buffer><offset>3</offset><length>7</length></kw_data></sig_data><sig_data><sig_id>200000093</sig_id><blocking_mask>3</blocking_mask><kw_data><buffer>Lzw+PHNjcmlwdD4=</buffer><offset>4</offset><length>7</length></kw_data></sig_data></violation><violation><viol_index>26</viol_index><viol_name>VIOL_URL_METACHAR</viol_name><uri>Lzw+PHNjcmlwdD4=</uri><metachar_index>60</metachar_index><wildcard_entity>*</wildcard_entity><staging>0</staging></violation><violation><viol_index>26</viol_index><viol_name>VIOL_URL_METACHAR</viol_name><uri>Lzw+PHNjcmlwdD4=</uri><metachar_index>62</metachar_index><wildcard_entity>*</wildcard_entity><staging>0</staging></violation><violation><viol_index>122</viol_index><viol_name>VIOL_BOT_CLIENT</viol_name></violation><violation><viol_index>93</viol_index><viol_name>VIOL_RATING_THREAT</viol_name></violation></request-violations></BAD_MSG>,curl,HTTP Library,N/A,N/A,Untrusted Bot,N/A,N/A,HTTP/1.1,/<><script>,GET /<><script> HTTP/1.1\\r\\nHost: localhost\\r\\nUser-Agent: curl/7.81.0\\r\\nAccept: */*\\r\\n\\r\\n`,
			expectAttrs: map[string]string{
				"app_protect.policy_name": "nms_app_protect_default_policy",
				"app_protect.support_id":  "5377540117854870581",
				"app_protect.outcome":     "REQUEST_OUTCOME_REJECTED",
				"app_protect.remote_addr": "127.0.0.1",
			},
			expectRecords: 1,
		},
		{
			name: "Test 2: Simple valid syslog message",
			body: "<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8",
			expectAttrs: map[string]string{
				"syslog.facility": "auth",
			},
			expectRecords: 1,
		},
		{
			name:          "Test 3: Unsupported body type",
			body:          12345,
			expectRecords: 0,
		},
		{
			name:          "Test 4: Invalid syslog message",
			body:          "not a syslog line",
			expectRecords: 0,
			expectError:   true,
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

			if tc.name == "Test 1: CSV NGINX App Protect syslog message" {
				processedBody := lrOut.Body().Bytes().AsRaw()

				var actualEvent events.SecurityViolationEvent
				jsonErr := proto.Unmarshal(processedBody, &actualEvent)
				require.NoError(t, jsonErr, "Failed to unmarshal processed log body as SecurityViolationEvent")

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
				assert.Equal(t, "Illegal meta character in URL::Attack signature detected::Violation Rating Threat detected::Bot Client Detected", actualEvent.GetViolations())
				assert.Equal(t, "N/A", actualEvent.GetSubViolations())
				assert.Equal(t, uint32(5), actualEvent.GetViolationRating())
				assert.Equal(t, "{High Accuracy Signatures;Cross Site Scripting Signatures}", actualEvent.GetSigSetNames())
				assert.Equal(t, "{High Accuracy Signatures; Cross Site Scripting Signatures}", actualEvent.GetSigCves())
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

				require.Len(t, actualEvent.ViolationsData, 5)
				assert.Equal(t, "VIOL_ATTACK_SIGNATURE", actualEvent.ViolationsData[0].GetViolationDataName())
				assert.Equal(t, "uri", actualEvent.ViolationsData[0].GetViolationDataContextData().GetContextDataName())
				assert.Equal(t, "/<><script>", actualEvent.ViolationsData[0].GetViolationDataContextData().GetContextDataValue())

				assert.NotNil(t, actualEvent.ViolationsData[0].GetViolationDataSignatures())
				assert.NotEmpty(t, actualEvent.ViolationsData[0].GetViolationDataSignatures())
				assert.IsType(t, []*events.SignatureData{}, actualEvent.ViolationsData[0].GetViolationDataSignatures())
			}
			require.NoError(t, p.Shutdown(ctx))
		})
	}
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
