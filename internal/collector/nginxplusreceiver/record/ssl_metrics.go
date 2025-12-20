// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package record

import (
	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginx/nginx-plus-go-client/v3/client"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func RecordSSLMetrics(mb *metadata.MetricsBuilder, now pcommon.Timestamp, stats *plusapi.Stats) {
	// SSL Handshake
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.HandshakesFailed),
		0,
		metadata.AttributeNginxSslStatusFAILED,
	)
	mb.RecordNginxSslHandshakesDataPoint(now, int64(stats.SSL.Handshakes), 0, 0)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.SessionReuses),
		0,
		metadata.AttributeNginxSslStatusREUSE,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.NoCommonProtocol),
		metadata.AttributeNginxSslHandshakeReasonNOCOMMONPROTOCOL,
		metadata.AttributeNginxSslStatusFAILED,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.NoCommonCipher),
		metadata.AttributeNginxSslHandshakeReasonNOCOMMONCIPHER,
		metadata.AttributeNginxSslStatusFAILED,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.HandshakeTimeout),
		metadata.AttributeNginxSslHandshakeReasonTIMEOUT,
		metadata.AttributeNginxSslStatusFAILED,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.PeerRejectedCert),
		metadata.AttributeNginxSslHandshakeReasonCERTREJECTED,
		metadata.AttributeNginxSslStatusFAILED,
	)

	// SSL Certificate
	mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.NoCert),
		metadata.AttributeNginxSslVerifyFailureReasonNOCERT,
	)
	mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.ExpiredCert),
		metadata.AttributeNginxSslVerifyFailureReasonEXPIREDCERT,
	)
	mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.RevokedCert),
		metadata.AttributeNginxSslVerifyFailureReasonREVOKEDCERT,
	)
	mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.HostnameMismatch),
		metadata.AttributeNginxSslVerifyFailureReasonHOSTNAMEMISMATCH,
	)
	mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.Other),
		metadata.AttributeNginxSslVerifyFailureReasonOTHER,
	)
}
