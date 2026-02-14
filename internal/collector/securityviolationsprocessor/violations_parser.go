// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
	"unicode/utf8"

	events "github.com/nginx/agent/v3/api/grpc/events/v1"
	"go.uber.org/zap"
)

const (
	violationContextRequest   = "request"
	violationContextHeader    = "header"
	violationContextParameter = "parameter"
	violationContextCookie    = "cookie"
	violationContextUrl       = "url"
	violationContextUri       = "uri"
)

// isValidProtobufString checks if a byte slice can be safely used as a protobuf string
// Protobuf strings cannot contain null bytes
func isValidProtobufString(data []byte) bool {
	return !bytes.Contains(data, []byte{0})
}

// parseViolationsData extracts violation data from the syslog key-value map
func (p *securityViolationsProcessor) parseViolationsData(kvMap map[string]string) []*events.ViolationData {
	violationDetails := kvMap["violation_details"]
	if violationDetails == "" {
		return nil
	}

	// Parse XML violation details
	var xmlData BADMSG
	if err := xml.Unmarshal([]byte(violationDetails), &xmlData); err != nil {
		p.settings.Logger.Warn("Failed to parse XML violation details", zap.Error(err))
		return nil
	}

	// Extract context from violation names if not present
	p.extractViolationContext(&xmlData)

	// Process each violation
	violationsData := make([]*events.ViolationData, 0, len(xmlData.RequestViolations.Violations))
	for _, v := range xmlData.RequestViolations.Violations {
		violation := &events.ViolationData{
			ViolationDataName:    v.ViolName,
			ViolationDataContext: strings.ToLower(v.Context),
		}

		// Extract context data based on context type
		violation.ViolationDataContextData = p.extractContextDataFromXML(v, kvMap)

		// Extract signatures from XML
		violation.ViolationDataSignatures = p.extractSignaturesFromXML(v)

		violationsData = append(violationsData, violation)
	}

	return violationsData
}

// extractViolationContext extracts context from violation names if context is empty
func (p *securityViolationsProcessor) extractViolationContext(xmlData *BADMSG) {
	for i, v := range xmlData.RequestViolations.Violations {
		if strings.ToLower(v.Context) == violationContextUrl {
			xmlData.RequestViolations.Violations[i].Context = violationContextUri
		}
		if v.Context != "" {
			continue
		}
		// Extract context from violation name
		if v.ViolName != "" {
			xmlData.RequestViolations.Violations[i].Context = extractContextFromViolationName(v.ViolName)
		}
	}
}

// extractContextFromViolationName derives context from violation name
func extractContextFromViolationName(violationName string) string {
	lowerName := strings.ToLower(violationName)
	if strings.Contains(lowerName, violationContextParameter) {
		return violationContextParameter
	}
	if strings.Contains(lowerName, violationContextHeader) {
		return violationContextHeader
	}
	if strings.Contains(lowerName, violationContextCookie) {
		return violationContextCookie
	}
	if strings.Contains(lowerName, violationContextRequest) {
		return violationContextRequest
	}
	if strings.Contains(lowerName, violationContextUri) || strings.Contains(lowerName, violationContextUrl) {
		return violationContextUri
	}

	return ""
}

// contextExtractResult holds the extracted context data before final processing
type contextExtractResult struct {
	name         string
	value        string
	isB64Decoded bool
}

// extractContextDataFromXML extracts context data from XML based on violation context type
func (p *securityViolationsProcessor) extractContextDataFromXML(
	v *Violation, kvMap map[string]string,
) *events.ContextData {
	var result contextExtractResult

	switch strings.ToLower(v.Context) {
	case violationContextParameter:
		result = p.extractParameterContext(v)
	case violationContextHeader:
		result = p.extractHeaderContext(v)
	case violationContextCookie:
		result = p.extractCookieContext(v)
	case violationContextUri:
		result = p.extractUriContext(v)
	case violationContextRequest:
		result = p.extractRequestContext(v)
	default:
		result = p.extractDefaultContext(v)
	}

	return p.buildContextData(result, v, kvMap)
}

// extractParameterContext extracts parameter context data from violation
func (p *securityViolationsProcessor) extractParameterContext(v *Violation) contextExtractResult {
	if v.ContextDataWrap != nil && v.ContextDataWrap.ParamData != nil && v.ContextDataWrap.ParamData.Name != "" {
		return contextExtractResult{
			name:         v.ContextDataWrap.ParamData.Name,
			value:        v.ContextDataWrap.ParamData.Value,
			isB64Decoded: v.ContextDataWrap.ParamData.IsBase64Decoded,
		}
	}
	if v.ParameterData != nil {
		return contextExtractResult{
			name:         v.ParameterData.Name,
			value:        v.ParameterData.Value,
			isB64Decoded: v.ParameterData.IsBase64Decoded,
		}
	}
	if v.ParamData != nil {
		return contextExtractResult{
			name:         v.ParamData.Name,
			value:        v.ParamData.Value,
			isB64Decoded: v.ParamData.IsBase64Decoded,
		}
	}
	if v.ParamName != "" {
		return contextExtractResult{
			name:         v.ParamName,
			isB64Decoded: v.IsBase64Decoded,
		}
	}

	return contextExtractResult{}
}

// extractHeaderContext extracts header context data from violation
func (p *securityViolationsProcessor) extractHeaderContext(v *Violation) contextExtractResult {
	if v.Header != nil {
		if v.Header.Name != "" || v.Header.Value != "" {
			return contextExtractResult{
				name:         v.Header.Name,
				value:        v.Header.Value,
				isB64Decoded: v.Header.IsBase64Decoded,
			}
		}
		// Fallback to Header.Text when Name and Value are empty
		return contextExtractResult{
			value:        v.Header.Text,
			isB64Decoded: v.Header.IsBase64Decoded,
		}
	}
	if v.HeaderData != nil {
		return contextExtractResult{
			name:         v.HeaderData.Name,
			value:        v.HeaderData.Value,
			isB64Decoded: v.HeaderData.IsBase64Decoded,
		}
	}
	if v.HeaderLength != "" {
		decodedName := v.HeaderName
		if decoded, err := p.tryDecodeBase64(v.HeaderName); err == nil {
			decodedName = decoded
		} else {
			p.settings.Logger.Warn("Failed to decode header name",
				zap.String("header_name", v.HeaderName), zap.Error(err))
		}

		return contextExtractResult{
			name: decodedName,
			value: fmt.Sprintf("Header length: %s, exceeds Header length limit: %s",
				v.HeaderLength, v.HeaderLengthLimit),
			isB64Decoded: true,
		}
	}

	return contextExtractResult{}
}

// extractCookieContext extracts cookie context data from violation
func (p *securityViolationsProcessor) extractCookieContext(v *Violation) contextExtractResult {
	if v.Cookie != nil && v.CookieLength == "" {
		return contextExtractResult{
			name:         v.Cookie.Name,
			value:        v.Cookie.Value,
			isB64Decoded: v.Cookie.IsBase64Decoded,
		}
	}
	if v.CookieName != "" {
		return contextExtractResult{
			name:         v.CookieName,
			isB64Decoded: v.IsBase64Decoded,
		}
	}
	if v.Buffer != "" {
		decodedBuffer, bufferErr := p.tryDecodeBase64(v.Buffer)
		if bufferErr == nil {
			return contextExtractResult{
				name:         v.SpecificDesc,
				value:        decodedBuffer,
				isB64Decoded: true,
			}
		}
		p.settings.Logger.Warn("Failed to decode cookie buffer", zap.String("buffer", v.Buffer), zap.Error(bufferErr))
	}
	if v.CookieLength != "" && v.Cookie != nil {
		decodedValue, valueErr := p.tryDecodeBase64(v.Cookie.Text)
		if valueErr == nil {
			return contextExtractResult{
				name: fmt.Sprintf("Cookie length: %s, exceeds Cookie length limit: %s",
					v.CookieLength, v.CookieLengthLimit),
				value:        decodedValue,
				isB64Decoded: true,
			}
		}
		p.settings.Logger.Warn("Failed to decode cookie text",
			zap.String("cookie_text", v.Cookie.Text), zap.Error(valueErr))
	}

	return contextExtractResult{}
}

// extractUriContext extracts URI context data from violation
func (p *securityViolationsProcessor) extractUriContext(v *Violation) contextExtractResult {
	if v.Uri != "" {
		return contextExtractResult{
			name:         violationContextUri,
			value:        v.Uri,
			isB64Decoded: true,
		}
	}
	if v.UriObjectData != nil {
		return contextExtractResult{
			name:         violationContextUri,
			value:        v.UriObjectData.Object,
			isB64Decoded: true,
		}
	}
	if v.UriLength != "" {
		return contextExtractResult{
			name:         "URI length: " + v.UriLength,
			value:        "URI length limit: " + v.UriLengthLimit,
			isB64Decoded: true,
		}
	}
	if v.HeaderData != nil &&
		(v.HeaderData.Name != "" || v.HeaderData.ActualValue != "" || v.HeaderData.MatchedValue != "") {
		return p.extractHeaderDataWithValues(v.HeaderData, "URI context")
	}

	return contextExtractResult{}
}

// extractRequestContext extracts request context data from violation
func (p *securityViolationsProcessor) extractRequestContext(v *Violation) contextExtractResult {
	if v.DefinedLength != "" {
		return contextExtractResult{
			name:         "Defined length: " + v.DefinedLength,
			value:        "Detected length: " + v.DetectedLength,
			isB64Decoded: true,
		}
	}
	if v.TotalLen != "" {
		return contextExtractResult{
			name:         "Total length: " + v.TotalLen,
			value:        "Total length limit: " + v.TotalLenLimit,
			isB64Decoded: true,
		}
	}

	return contextExtractResult{isB64Decoded: true}
}

// extractDefaultContext handles cases where context is empty but HeaderData exists
func (p *securityViolationsProcessor) extractDefaultContext(v *Violation) contextExtractResult {
	if v.HeaderData != nil &&
		(v.HeaderData.Name != "" || v.HeaderData.ActualValue != "" || v.HeaderData.MatchedValue != "") {
		return p.extractHeaderDataWithValues(v.HeaderData, "Default context")
	}

	return contextExtractResult{}
}

// extractHeaderDataWithValues extracts and decodes header data with actual/matched values
func (p *securityViolationsProcessor) extractHeaderDataWithValues(
	headerData *Header, contextLabel string,
) contextExtractResult {
	decodedName, err := p.tryDecodeBase64(headerData.Name)
	if err != nil {
		p.settings.Logger.Warn(contextLabel+": failed to decode header name",
			zap.String("header_name", headerData.Name), zap.Error(err))
	}
	decodedActualValue, err := p.tryDecodeBase64(headerData.ActualValue)
	if err != nil {
		p.settings.Logger.Warn(contextLabel+": failed to decode actual header value",
			zap.String("actual_value", headerData.ActualValue), zap.Error(err))
	}
	decodedMatchedValue, err := p.tryDecodeBase64(headerData.MatchedValue)
	if err != nil {
		p.settings.Logger.Warn(contextLabel+": failed to decode matched header value",
			zap.String("matched_value", headerData.MatchedValue), zap.Error(err))
	}

	return contextExtractResult{
		name: decodedName,
		value: fmt.Sprintf("actual header value: %s. matched header value: %s",
			decodedActualValue, decodedMatchedValue),
		isB64Decoded: true,
	}
}

// buildContextData constructs the final ContextData from extracted result
func (p *securityViolationsProcessor) buildContextData(
	result contextExtractResult, v *Violation, kvMap map[string]string,
) *events.ContextData {
	ctxData := &events.ContextData{}

	// Handle already decoded data
	if result.isB64Decoded {
		name, value := populateNameValue(v.ViolName, result.name, result.value)
		ctxData.ContextDataName = name
		ctxData.ContextDataValue = value

		return ctxData
	}

	// Decode base64 if not already decoded
	if result.name != "" || result.value != "" {
		decodedName := p.decodeStringOrWarn(result.name, v.Context, "name")
		decodedValue := p.decodeStringOrWarn(result.value, v.Context, "value")
		name, value := populateNameValue(v.ViolName, decodedName, decodedValue)
		ctxData.ContextDataName = name
		ctxData.ContextDataValue = value

		return ctxData
	}

	// Fallback: use CSV URI for URI context
	if strings.ToLower(v.Context) == violationContextUri || strings.ToLower(v.Context) == violationContextUrl {
		if uri := kvMap["uri"]; uri != "" {
			ctxData.ContextDataName = violationContextUri
			ctxData.ContextDataValue = uri
		}
	}

	return ctxData
}

// tryDecodeBase64 attempts to decode a base64 string, returning the decoded string or error
func (p *securityViolationsProcessor) tryDecodeBase64(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

// decodeStringOrWarn decodes a base64 string or logs a warning and returns the original
func (p *securityViolationsProcessor) decodeStringOrWarn(encoded, context, fieldType string) string {
	if encoded == "" {
		return ""
	}
	decoded, err := p.tryDecodeBase64(encoded)
	if err != nil {
		p.settings.Logger.Warn("Failed to decode context field",
			zap.String("context", context), zap.String("field_type", fieldType), zap.String("value", encoded), zap.Error(err))

		return encoded
	}

	return decoded
}

// extractSignaturesFromXML extracts signature data from XML violation
func (p *securityViolationsProcessor) extractSignaturesFromXML(v *Violation) []*events.SignatureData {
	if len(v.SigData) == 0 {
		return nil
	}
	signatures := make([]*events.SignatureData, len(v.SigData))

	for i, s := range v.SigData {
		// Decode base64 buffer
		buf, err := base64.StdEncoding.DecodeString(s.KwData.Buffer)
		bufferStr := s.KwData.Buffer // Default to base64 string
		if err == nil {
			// Only use decoded buffer if it's valid UTF-8 and doesn't contain null bytes
			if utf8.ValidString(string(buf)) && isValidProtobufString(buf) {
				bufferStr = string(buf)
			}
		}

		signatures[i] = &events.SignatureData{
			SigDataId:           parseUint32(s.SigID),
			SigDataBlockingMask: s.BlockingMask,
			SigDataBuffer:       bufferStr,
			SigDataOffset:       parseUint32(s.KwData.Offset),
			SigDataLength:       parseUint32(s.KwData.Length),
		}
	}

	return signatures
}

// populateNameValue provides smart fallback logic for name and value extraction
// matching the behavior from dev-v2 implementation
func populateNameValue(violationName, dataName, dataValue string) (name, value string) {
	if dataName != "" && dataValue != "" {
		name = dataName
		value = dataValue

		return name, value
	}
	name = violationNameToDataName(violationName)
	if dataName == "" && dataValue != "" {
		value = dataValue
		return name, value
	}
	if dataName != "" && dataValue == "" {
		value = dataName
		return name, value
	}

	return name, value
}

// violationNameToDataName converts a violation name (e.g., "VIOL_ATTACK_SIGNATURE")
// to a readable data name (e.g., "Attack Signature")
func violationNameToDataName(violationName string) string {
	parts := strings.Split(strings.ToLower(violationName), "_")
	if len(parts) > 0 && parts[0] == "viol" {
		parts = parts[1:]
	}

	// Capitalize each word
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	return strings.Join(parts, " ")
}
