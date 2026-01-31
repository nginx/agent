// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
)

//nolint:lll // long test string kept for readability
const (
	sampleAppProtectSyslog = `<130>Aug 22 03:28:35 ip-172-16-0-213 ASM:N/A,80,127.0.0.1,false,GET,nms_app_protect_default_policy,HTTP,blocked,0,N/A,N/A::N/A,{High Accuracy Signatures;Cross Site Scripting Signatures}::{High Accuracy Signatures; Cross Site Scripting Signatures},56064,N/A,5377540117854870581,N/A,5,1-localhost:1-/,N/A,REJECTED,SECURITY_WAF_VIOLATION,Illegal meta character in URL::Attack signature detected::Violation Rating Threat detected::Bot Client Detected,<?xml version='1.0' encoding='UTF-8'?><BAD_MSG><violation_masks><block>414000000200c00-3a03030c30000072-8000000000000000-0</block><alarm>475f0ffcbbd0fea-befbf35cb000007e-f400000000000000-0</alarm><learn>0-0-0-0</learn><staging>0-0-0-0</staging></violation_masks><request-violations><violation><viol_index>42</viol_index><viol_name>VIOL_ATTACK_SIGNATURE</viol_name><context>url</context><sig_data><sig_id>200000099</sig_id><blocking_mask>3</blocking_mask><kw_data><buffer>Lzw+PHNjcmlwdD4=</buffer><offset>3</offset><length>7</length></kw_data></sig_data><sig_data><sig_id>200000093</sig_id><blocking_mask>3</blocking_mask><kw_data><buffer>Lzw+PHNjcmlwdD4=</buffer><offset>4</offset><length>7</length></kw_data></sig_data></violation><violation><viol_index>26</viol_index><viol_name>VIOL_URL_METACHAR</viol_name><uri>Lzw+PHNjcmlwdD4=</uri><metachar_index>60</metachar_index><wildcard_entity>*</wildcard_entity><staging>0</staging></violation><violation><viol_index>26</viol_index><viol_name>VIOL_URL_METACHAR</viol_name><uri>Lzw+PHNjcmlwdD4=</uri><metachar_index>62</metachar_index><wildcard_entity>*</wildcard_entity><staging>0</staging></violation><violation><viol_index>122</viol_index><viol_name>VIOL_BOT_CLIENT</viol_name></violation><violation><viol_index>93</viol_index><viol_name>VIOL_RATING_THREAT</viol_name></violation></request-violations></BAD_MSG>,curl,HTTP Library,N/A,N/A,Untrusted Bot,N/A,N/A,HTTP/1.1,/<><script>,GET /<><script> HTTP/1.1\\r\\nHost: localhost\\r\\nUser-Agent: curl/7.81.0\\r\\nAccept: */*\\r\\n\\r\\n`
)

//nolint:lll,revive // long test string kept for readability
func generateSecurityViolationLogs(numRecords int, message string) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()

	for range numRecords {
		lr := sl.LogRecords().AppendEmpty()
		lr.Body().SetStr(message)
	}

	return logs
}

func newBenchmarkProcessor() *securityViolationsProcessor {
	settings := processortest.NewNopSettings(processortest.NopType)
	return newSecurityViolationsProcessor(consumertest.NewNop(), settings)
}

func BenchmarkSecurityViolationsProcessor(b *testing.B) {
	benchmarks := []struct {
		name       string
		message    string
		numRecords int
	}{
		{name: "AppProtect_1", message: sampleAppProtectSyslog, numRecords: 1},
		{name: "AppProtect_10", message: sampleAppProtectSyslog, numRecords: 10},
		{name: "AppProtect_100", message: sampleAppProtectSyslog, numRecords: 100},
		{name: "AppProtect_1000", message: sampleAppProtectSyslog, numRecords: 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			p := newBenchmarkProcessor()
			logs := generateSecurityViolationLogs(bm.numRecords, bm.message)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				logsCopy := plog.NewLogs()
				logs.CopyTo(logsCopy)
				_ = p.ConsumeLogs(context.Background(), logsCopy)
			}
		})
	}
}

func BenchmarkSecurityViolationsProcessor_Concurrent(b *testing.B) {
	p := newBenchmarkProcessor()
	logs := generateSecurityViolationLogs(1000, sampleAppProtectSyslog)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logsCopy := plog.NewLogs()
			logs.CopyTo(logsCopy)
			_ = p.ConsumeLogs(context.Background(), logsCopy)
		}
	})
}
