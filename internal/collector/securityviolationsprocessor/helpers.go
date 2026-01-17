// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"net"
	"regexp"
	"strings"

	events "github.com/nginx/agent/v3/api/grpc/events/v1"
)

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

func buildSignatures(ids, names []string, mask, offset, length string) []*events.SignatureData {
	signatures := make([]*events.SignatureData, 0, len(ids))
	for i, id := range ids {
		if id == "" || id == notAvailable {
			continue
		}
		signature := &events.SignatureData{
			SigDataId:           id,
			SigDataBlockingMask: mask,
			SigDataOffset:       offset,
			SigDataLength:       length,
		}
		if i < len(names) {
			signature.SigDataBuffer = names[i]
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
