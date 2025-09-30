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
		metadata.AttributeNginxSslStatusFAILED,
		0,
	)
	mb.RecordNginxSslHandshakesDataPoint(now, int64(stats.SSL.Handshakes), 0, 0)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.SessionReuses),
		metadata.AttributeNginxSslStatusREUSE,
		0,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.NoCommonProtocol),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonNOCOMMONPROTOCOL,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.NoCommonCipher),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonNOCOMMONCIPHER,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.HandshakeTimeout),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonTIMEOUT,
	)
	mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.PeerRejectedCert),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonCERTREJECTED,
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
