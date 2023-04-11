/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package processor

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	pb "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
)

const (
	// in seconds
	eventWaitTimeout = 5
	numWorkers       = 4
)

func TestNAPProcess(t *testing.T) {
	testCases := []struct {
		testName   string
		testFile   string
		expected   *pb.SecurityViolationEvent
		isNegative bool
		fileExists bool
	}{
		{
			testName: "PassedEvent",
			testFile: "./testdata/expanded_nap_waf.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592513",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "parameter,,parameter",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_ATTACK_SIGNATURE",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "a",
							Value: "<script>",
						},
						Signatures: []*pb.SignatureData{
							{
								ID:           "200001475",
								BlockingMask: "3",
								Buffer:       "a=<script>",
								Offset:       "3",
								Length:       "7",
							},
							{
								ID:           "200000098",
								BlockingMask: "3",
								Buffer:       "a=<script>",
								Offset:       "2",
								Length:       "7",
							},
						},
					},
					{
						Name: "VIOL_HTTP_PROTOCOL",
					},
					{
						Name:    "VIOL_PARAMETER_VALUE_METACHAR",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "a",
							Value: "<script>",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		// XML Parsing
		{
			testName: "violation name parsing",
			testFile: "./testdata/xml_violation_name.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592519",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "cookie,cookie,cookie,cookie",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_ASM_COOKIE_MODIFIED",
						Context: "cookie",
						ContextData: &pb.ContextData{
							Name: "TS0144e914_1",
						},
					},
					{
						Name:    "VIOL_ASM_COOKIE_MODIFIED",
						Context: "cookie",
						ContextData: &pb.ContextData{
							Name: "TS0144e914_31",
						},
					},
					{
						Name:    "VIOL_ASM_COOKIE_MODIFIED",
						Context: "cookie",
						ContextData: &pb.ContextData{
							Name: "TS0144e914_1",
						},
					},
					{
						Name:    "VIOL_ASM_COOKIE_MODIFIED",
						Context: "cookie",
						ContextData: &pb.ContextData{
							Name: "TS0144e914_31",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "parameter data parsing",
			testFile: "./testdata/xml_parameter_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592515",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "parameter",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_ATTACK_SIGNATURE",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Value: "f5paramautotest>",
						},
						Signatures: []*pb.SignatureData{
							{
								ID:           "300000110",
								BlockingMask: "7",
								Buffer:       "=f5paramautotest>",
								Offset:       "1",
								Length:       "15",
							},
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "parameter data parsing with empty context key",
			testFile: "./testdata/xml_parameter_data_empty_context.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592517",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "parameter,parameter",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_PARAMETER",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "x",
							Value: "1",
						},
					},
					{
						Name:    "VIOL_PARAMETER",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "y",
							Value: "%",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "parameter data parsing as param_data",
			testFile: "./testdata/xml_parameter_data_as_param_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592516",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "parameter",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_JSON_MALFORMED",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "json",
							Value: "{ \"a\": \"\326\326\"}",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "header data parsing",
			testFile: "./testdata/xml_header_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592514",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "header",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_ATTACK_SIGNATURE",
						Context: "header",
						ContextData: &pb.ContextData{
							Name:  "Foo",
							Value: "echo<!-- #echo",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "signature data parsing",
			testFile: "./testdata/xml_signature_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592518",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "",
				ViolationsData: []*pb.ViolationData{
					{
						Name: "VIOL_ATTACK_SIGNATURE",
						Signatures: []*pb.SignatureData{
							{
								ID:           "200021094",
								BlockingMask: "4",
								Buffer:       "Connection: keep-alive\r\nHost: a.com\r\nUser-Agent: Java/1.7.0_51\r\nAccept: text/html, image/gif, image/jpeg, *; q=.2, */*; q=.2\r\n\r\n",
								Offset:       "37",
								Length:       "16",
							},
							{
								ID:           "200011034",
								BlockingMask: "2",
								Buffer:       "pt: */*\r\nFoo: Authorization: %n%n%n%n\r\nFoo: echo<!-- #echo\r\nX18",
								Offset:       "29",
								Length:       "6",
							},
							{
								ID:           "200000179",
								BlockingMask: "3",
								Buffer:       "7\r\nAccept: */*\r\nFoo: Authorization: %n%n%n%n\r\nFoo: echo<!-- #ec",
								Offset:       "21",
								Length:       "23",
							},
							{
								ID:           "200004106",
								BlockingMask: "2",
								Buffer:       "zation: %n%n%n%n\r\nFoo: echo<!-- #echo\r\nX1892: echo\r\nX1893: '<!-",
								Offset:       "27",
								Length:       "10",
							},
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "violation contexts - type 'cookie'",
			testFile: "./testdata/xml_violation_cookie_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592511",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "cookie,cookie",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_COOKIE_MALFORMED",
						Context: "cookie",
						ContextData: &pb.ContextData{
							Name:  "Invalid quotation mark sign",
							Value: "ost\r\nCookie: TS0\"1c60e7b=013059d",
						},
					},
					{
						Name:    "VIOL_COOKIE_MODIFIED",
						Context: "cookie",
						ContextData: &pb.ContextData{
							Name:  "yummy_cookie",
							Value: "choco",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "violation contexts - type 'parameter'",
			testFile: "./testdata/xml_violation_parameter_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592511",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "parameter,parameter,parameter,header",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_PARAMETER_VALUE_METACHAR",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "y",
							Value: "%ggb",
						},
					},
					{
						Name:    "VIOL_PARAMETER_NAME_METACHAR",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "x@",
							Value: "",
						},
					},
					{
						Name:    "VIOL_PARAMETER_VALUE_LENGTH",
						Context: "parameter",
						ContextData: &pb.ContextData{
							Name:  "atype_2",
							Value: "Loop2<script>",
						},
					},
					{
						Name:    "VIOL_PARAMETER_VALUE_BASE64",
						Context: "header",
						ContextData: &pb.ContextData{
							Name:  "isbase64-test-bin",
							Value: "plaintext",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "violation contexts - type 'request'",
			testFile: "./testdata/xml_violation_request_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592511",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "request,request",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_REQUEST_MAX_LENGTH",
						Context: "request",
						ContextData: &pb.ContextData{
							Name:  "Defined length: 10000000",
							Value: "Detected length: 11000125",
						},
					},
					{
						Name:    "VIOL_REQUEST_LENGTH",
						Context: "request",
						ContextData: &pb.ContextData{
							Name:  "Total length: 316",
							Value: "Total length limit: 0",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
		{
			testName: "violation contexts - type 'url'",
			testFile: "./testdata/xml_violation_url_data.log.txt",
			expected: &pb.SecurityViolationEvent{
				PolicyName:               "app_protect_default_policy",
				SupportID:                "4355056874564592511",
				Outcome:                  "REJECTED",
				OutcomeReason:            "SECURITY_WAF_VIOLATION",
				BlockingExceptionReason:  "N/A",
				Method:                   "GET",
				Protocol:                 "HTTP",
				XForwardedForHeaderValue: "N/A",
				URI:                      "/",
				Request:                  "GET /?a=<script> HTTP/1.1\\r\\nHost: 127.0.0.1\\r\\nUser-Agent: curl/7.64.1\\r\\nAccept: */*\\r\\n\\r\\n",
				IsTruncated:              "",
				RequestStatus:            "blocked",
				ResponseCode:             "Blocked",
				ServerAddr:               "",
				VSName:                   "1-localhost:1-/",
				RemoteAddr:               "127.0.0.1",
				RemotePort:               "61478",
				ServerPort:               "80",
				Violations:               "HTTP protocol compliance failed,Illegal meta character in value,Attack signature detected,Violation Rating Threat detected,Bot Client Detected",
				SubViolations:            "HTTP protocol compliance failed:Host header contains IP address,HTTP protocol compliance failed:Evasion technique",
				ViolationRating:          "5",
				SigSetNames:              "{Cross Site Scripting Signatures;High Accuracy Signatures},{Cross Site Scripting Signatures;High Accuracy Signatures}",
				SigCVEs:                  ",",
				ClientClass:              "Untrusted Bot",
				ClientApplication:        "N/A",
				ClientApplicationVersion: "N/A",
				Severity:                 "critical",
				ThreatCampaignNames:      "campaign1,campaign2",
				BotAnomalies:             "N/A",
				BotCategory:              "HTTP Library",
				EnforcedBotAnomalies:     "N/A",
				BotSignatureName:         "curl",
				ViolationContexts:        "uri,uri,url",
				ViolationsData: []*pb.ViolationData{
					{
						Name:    "VIOL_URL_METACHAR",
						Context: "uri",
						ContextData: &pb.ContextData{
							Name:  "",
							Value: "/;shutdown",
						},
					},
					{
						Name:    "VIOL_URL_LENGTH",
						Context: "uri",
						ContextData: &pb.ContextData{
							Name:  "URI length: 18",
							Value: "URI length limit: 0",
						},
					},
					{
						Name:    "VIOL_JSON_MALFORMED",
						Context: "url",
						ContextData: &pb.ContextData{
							Name:  "",
							Value: "/",
						},
					},
				},
				SystemID:       "",
				InstanceTags:   "",
				InstanceGroup:  "",
				DisplayName:    "",
				ParentHostname: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			collect := make(chan *monitoring.RawLog, 2)
			processed := make(chan *pb.Event, 2)

			log := logrus.New()
			log.SetLevel(logrus.DebugLevel)

			p, err := GetClient(&Config{
				Logger:  log.WithField("extension", "test"),
				Workers: numWorkers,
			})
			if err != nil {
				t.Fatalf("Could not get a Processor Client: %s", err)
			}

			wg := &sync.WaitGroup{}

			wg.Add(1)
			// Start Processor
			go p.Process(ctx, wg, collect, processed)

			// Briefly sleep so map can be reconciled before event is collected
			// and processed
			if !tc.fileExists {
				time.Sleep(2 * time.Second)
			}

			input, err := os.ReadFile(tc.testFile)
			if err != nil {
				t.Fatalf("Error while reading the logfile %s: %v", tc.testFile, err)
			}

			collect <- &monitoring.RawLog{Origin: monitoring.NAP, Logline: string(input)}

			select {
			case event := <-processed:
				t.Logf("Got event: %v", event)
				se := event.GetSecurityViolationEvent()
				require.NotNil(t, se)

				require.Equal(t, tc.expected.PolicyName, se.PolicyName)
				require.Equal(t, tc.expected.SupportID, se.SupportID)
				require.Equal(t, tc.expected.Outcome, se.Outcome)
				require.Equal(t, tc.expected.OutcomeReason, se.OutcomeReason)
				require.Equal(t, tc.expected.BlockingExceptionReason, se.BlockingExceptionReason)
				require.Equal(t, tc.expected.Method, se.Method)
				require.Equal(t, tc.expected.Protocol, se.Protocol)
				require.Equal(t, tc.expected.XForwardedForHeaderValue, se.XForwardedForHeaderValue)
				require.Equal(t, tc.expected.URI, se.URI)
				require.Equal(t, tc.expected.Request, se.Request)
				require.Equal(t, tc.expected.IsTruncated, se.IsTruncated)
				require.Equal(t, tc.expected.RequestStatus, se.RequestStatus)
				require.Equal(t, tc.expected.ResponseCode, se.ResponseCode)
				require.Equal(t, tc.expected.ServerAddr, se.ServerAddr)
				require.Equal(t, tc.expected.VSName, se.VSName)
				require.Equal(t, tc.expected.RemoteAddr, se.RemoteAddr)
				require.Equal(t, tc.expected.RemotePort, se.RemotePort)
				require.Equal(t, tc.expected.ServerPort, se.ServerPort)
				require.Equal(t, tc.expected.Violations, se.Violations)
				require.Equal(t, tc.expected.SubViolations, se.SubViolations)
				require.Equal(t, tc.expected.ViolationRating, se.ViolationRating)
				require.Equal(t, tc.expected.SigSetNames, se.SigSetNames)
				require.Equal(t, tc.expected.ClientClass, se.ClientClass)
				require.Equal(t, tc.expected.ClientApplication, se.ClientApplication)
				require.Equal(t, tc.expected.ClientApplicationVersion, se.ClientApplicationVersion)
				require.Equal(t, tc.expected.Severity, se.Severity)
				require.Equal(t, tc.expected.ThreatCampaignNames, se.ThreatCampaignNames)
				require.Equal(t, tc.expected.BotAnomalies, se.BotAnomalies)
				require.Equal(t, tc.expected.BotCategory, se.BotCategory)
				require.Equal(t, tc.expected.EnforcedBotAnomalies, se.EnforcedBotAnomalies)
				require.Equal(t, tc.expected.BotSignatureName, se.BotSignatureName)
				require.Equal(t, tc.expected.ViolationContexts, se.ViolationContexts)
				require.Equal(t, tc.expected.ViolationsData, se.ViolationsData)
				require.Equal(t, tc.expected.SystemID, se.SystemID)
				require.Equal(t, tc.expected.InstanceTags, se.InstanceTags)
				require.Equal(t, tc.expected.InstanceGroup, se.InstanceGroup)
				require.Equal(t, tc.expected.DisplayName, se.DisplayName)
				require.Equal(t, tc.expected.ParentHostname, se.ParentHostname)
			case <-time.After(eventWaitTimeout * time.Second):
				// for negative test, there should not be an event generated.
				if !tc.isNegative {
					t.Error("Should receive security violation event, and should not be timeout.")
				}
			}
		})
	}
}
