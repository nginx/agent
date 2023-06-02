/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package processor

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"golang.org/x/text/language"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"

	models "github.com/nginx/agent/sdk/v2/proto/events"
)

const (
	napDateTimeLayout       = "2006-01-02 15:04:05.000"
	listSeperator           = "::"
	defaultBlockedRespCode  = "0"
	defaultBlockedRespValue = "Blocked"

	decodedComma = ","
	encodedComma = "%2C"

	violationNameSeparator    = '_'
	violationNamePrefix       = "viol"
	violationContextRequest   = "request"
	violationContextHeader    = "header"
	violationContextParameter = "parameter"
	violationContextCookie    = "cookie"
	violationContextUrl       = "url"
	violationContextUri       = "uri"
)

// NGINX App Protect Logging Directives
const (
	blockingExceptionReason  = "blocking_exception_reason"
	protocol                 = "protocol"
	requestStatus            = "request_status"
	severity                 = "severity"
	sigSetNames              = "sig_set_names"
	threatCampaignNames      = "threat_campaign_names"
	violationDetails         = "violation_details"
	clientApplication        = "client_application"
	clientApplicationVersion = "client_application_version"
	transportProtocol        = "transport_protocol"
	httpRequestMethod        = "method"
	httpResponseCode         = "response_code"
	sigCVEs                  = "sig_cves"
	httpRemotePort           = "src_port"
	httpURI                  = "uri"
	httpHostname             = "vs_name"
	requestOutcome           = "outcome"
	requestOutcomeReason     = "outcome_reason"
	httpRemoteAddr           = "ip_client"
	httpServerPort           = "dest_port"
	isTruncated              = "is_truncated_bool"
	policyName               = "policy_name"
	request                  = "request"
	subViolations            = "sub_violations"
	supportID                = "support_id"
	violations               = "violations"
	violationRating          = "violation_rating"
	xForwardedForHeaderVal   = "x_forwarded_for_header_value"
	botAnomalies             = "bot_anomalies"
	botCategory              = "bot_category"
	clientClass              = "client_class"
	botSignatureName         = "bot_signature_name"
	enforcedBotAnomalies     = "enforced_bot_anomalies"
)

// NGINX App Protect Log Directives Order Per Config
var (
	logFormatKeys = []string{
		blockingExceptionReason,
		httpServerPort,
		httpRemoteAddr,
		isTruncated,
		httpRequestMethod,
		policyName,
		protocol,
		requestStatus,
		httpResponseCode,
		severity,
		sigCVEs,
		sigSetNames,
		httpRemotePort,
		subViolations,
		supportID,
		threatCampaignNames,
		violationRating,
		httpHostname,
		xForwardedForHeaderVal,
		requestOutcome,
		requestOutcomeReason,
		violations,
		violationDetails,
		botSignatureName,
		botCategory,
		botAnomalies,
		enforcedBotAnomalies,
		clientClass,
		clientApplication,
		clientApplicationVersion,
		transportProtocol,
		httpURI,
		request,
	}
)

type ParameterData struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"name"`
	Value           string `xml:"value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

type ParamData struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"param_name"`
	Value           string `xml:"param_value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

type Header struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"header_name"`
	Value           string `xml:"header_value"`
	ActualValue     string `xml:"header_actual_value"`
	MatchedValue    string `xml:"header_matched_value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

type Cookie struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"cookie_name"`
	Value           string `xml:"cookie_value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

type UriObjectData struct {
	Text   string `xml:",chardata"`
	Object string `xml:"object"`
}

type BADMSG struct {
	XMLName        xml.Name `xml:"BAD_MSG"`
	Text           string   `xml:",chardata"`
	ViolationMasks struct {
		Text    string `xml:",chardata"`
		Block   string `xml:"block"`
		Alarm   string `xml:"alarm"`
		Learn   string `xml:"learn"`
		Staging string `xml:"staging"`
	} `xml:"violation_masks"`
	RequestViolations struct {
		Text       string `xml:",chardata"`
		Violations []struct {
			Text      string `xml:",chardata"`
			ViolIndex string `xml:"viol_index"`
			ViolName  string `xml:"viol_name"`
			Context   string `xml:"context"`
			// ParameterData and ParamData are both received when context == "parameter" | ""
			// We receive either ParameterData or ParamData separately and not in the same XML message
			// ParameterData and ParamData semantically represent the same thing (with ParameterData having more fields).
			ParameterData     ParameterData `xml:"parameter_data"`
			ParamData         ParamData     `xml:"param_data"`
			ParamName         string        `xml:"param_name"`
			IsBase64Decoded   bool          `xml:"is_base64_decoded"`
			Header            Header        `xml:"header"`
			HeaderData        Header        `xml:"header_data"`
			HeaderName        string        `xml:"header_name"`
			HeaderLength      string        `xml:"header_len"`
			HeaderLengthLimit string        `xml:"header_len_limit"`
			Cookie            Cookie        `xml:"cookie"`
			CookieName        string        `xml:"cookie_name"`
			CookieLength      string        `xml:"cookie_len"`
			CookieLengthLimit string        `xml:"cookie_len_limit"`
			Buffer            string        `xml:"buffer"`
			SpecificDesc      string        `xml:"specific_desc"`
			Uri               string        `xml:"uri"`
			UriObjectData     UriObjectData `xml:"object_data"`
			UriLength         string        `xml:"uri_len"`
			UriLengthLimit    string        `xml:"uri_len_limit"`
			DefinedLength     string        `xml:"defined_length"`
			DetectedLength    string        `xml:"detected_length"`
			TotalLen          string        `xml:"total_len"`
			TotalLenLimit     string        `xml:"total_len_limit"`
			Staging           string        `xml:"staging"`
			SigData           []struct {
				Text         string `xml:",chardata"`
				SigID        string `xml:"sig_id"`
				BlockingMask string `xml:"blocking_mask"`
				KwData       struct {
					Text   string `xml:",chardata"`
					Buffer string `xml:"buffer"`
					Offset string `xml:"offset"`
					Length string `xml:"length"`
				} `xml:"kw_data"`
			} `xml:"sig_data"`
			HTTPSanityChecksStatus string   `xml:"http_sanity_checks_status"`
			HTTPSubViolationStatus string   `xml:"http_sub_violation_status"`
			HTTPSubViolation       string   `xml:"http_sub_violation"`
			WildcardEntity         string   `xml:"wildcard_entity"`
			LanguageType           string   `xml:"language_type"`
			MetacharIndex          []string `xml:"metachar_index"`
		} `xml:"violation"`
	} `xml:"request-violations"`
}

type NAPConfig struct {
	DateTime                 string
	BlockingExceptionReason  string
	HTTPServerPort           string
	HTTPRemoteAddr           string
	IsTruncated              string
	HTTPRequestMethod        string
	PolicyName               string
	Protocol                 string
	RequestStatus            string
	HTTPResponseCode         string
	Severity                 string
	SignatureCVEs            string
	SigSetNames              string
	HTTPRemotePort           string
	SubViolations            string
	SupportID                string
	ThreatCampaignNames      string
	UnitHostname             string
	HTTPURI                  string
	ViolationRating          string
	HTTPHostname             string
	XForwardedForHeaderVal   string
	RequestOutcome           string
	RequestOutcomeReason     string
	Violations               string
	ViolationDetailsXML      *BADMSG
	BotSignatureName         string
	BotCategory              string
	BotAnomalies             string
	EnforcedBotAnomalies     string
	ClientClass              string
	ClientApplication        string
	ClientApplicationVersion string
	Request                  string
	TransportProtocol        string
	ViolationContext         string
}

// GetEvent will generate a protobuf Security Event.
func (f *NAPConfig) GetEvent(hostPattern *regexp.Regexp, logger *logrus.Entry) (*models.Event, error) {
	var (
		event  models.Event
		secevt *models.SecurityViolationEvent
		err    error
	)

	if logger == nil {
		logger = logrus.StandardLogger().WithFields(componentLogFields)
	}

	metadata, err := f.getMetadata()
	if err != nil {
		return nil, err
	}
	event.Metadata = metadata

	secevt = f.getSecurityViolation(logger)
	if err != nil {
		return nil, err
	}
	event.Data = &models.Event_SecurityViolationEvent{
		SecurityViolationEvent: secevt,
	}

	return &event, err
}

func (f *NAPConfig) getSecurityViolation(logger *logrus.Entry) *models.SecurityViolationEvent {
	sve := &models.SecurityViolationEvent{
		PolicyName:               f.PolicyName,
		SupportID:                f.SupportID,
		BlockingExceptionReason:  f.BlockingExceptionReason,
		Method:                   f.HTTPRequestMethod,
		Protocol:                 f.Protocol,
		XForwardedForHeaderValue: f.XForwardedForHeaderVal,
		Request:                  f.Request,
		IsTruncated:              f.IsTruncated,
		RequestStatus:            f.RequestStatus,
		ResponseCode:             f.HTTPResponseCode,
		VSName:                   f.HTTPHostname,
		RemoteAddr:               f.HTTPRemoteAddr,
		RemotePort:               f.HTTPRemotePort,
		ServerPort:               f.HTTPServerPort,
		Violations:               f.Violations,
		SubViolations:            f.SubViolations,
		ViolationRating:          f.ViolationRating,
		SigSetNames:              f.SigSetNames,
		SigCVEs:                  f.SignatureCVEs,
		Severity:                 f.Severity,
		ThreatCampaignNames:      f.ThreatCampaignNames,
		ClientClass:              f.ClientClass,
		ClientApplication:        f.ClientApplication,
		ClientApplicationVersion: f.ClientApplicationVersion,
		BotAnomalies:             f.BotAnomalies,
		BotCategory:              f.BotCategory,
		BotSignatureName:         f.BotSignatureName,
		EnforcedBotAnomalies:     f.EnforcedBotAnomalies,
		Outcome:                  f.RequestOutcome,
		OutcomeReason:            f.RequestOutcomeReason,
		URI:                      f.HTTPURI,
	}

	sve.ViolationsData = f.getViolations(logger)

	return sve
}

func (f *NAPConfig) getMetadata() (*models.Metadata, error) {
	f.DateTime = time.Now().UTC().Format(napDateTimeLayout)

	t, err := parseNAPDateTime(f.DateTime)
	if err != nil {
		return nil, err
	}

	return NewMetadata(t, f.SupportID)
}

func (f *NAPConfig) extractViolationContext() {
	if f.ViolationDetailsXML != nil {
		for i, v := range f.ViolationDetailsXML.RequestViolations.Violations {
			if strings.ToLower(v.Context) == violationContextUrl {
				f.ViolationDetailsXML.RequestViolations.Violations[i].Context = violationContextUri
			}
			if v.Context != "" {
				continue
			}
			if v.ViolName != "" {
				f.ViolationDetailsXML.RequestViolations.Violations[i].Context = extractContextFromViolationName(v.ViolName)
			}
		}
	}
}

func extractContextFromViolationName(violationName string) string {
	if strings.Contains(strings.ToLower(violationName), violationContextParameter) {
		return violationContextParameter
	}
	if strings.Contains(strings.ToLower(violationName), violationContextHeader) {
		return violationContextHeader
	}
	if strings.Contains(strings.ToLower(violationName), violationContextCookie) {
		return violationContextCookie
	}
	if strings.Contains(strings.ToLower(violationName), violationContextRequest) {
		return violationContextRequest
	}
	if strings.Contains(strings.ToLower(violationName), violationContextUri) ||
		strings.Contains(strings.ToLower(violationName), violationContextUrl) {
		return violationContextUri
	}

	return ""
}

func (f *NAPConfig) getViolations(logger *logrus.Entry) []*models.ViolationData {
	violations := []*models.ViolationData{}

	if f.ViolationDetailsXML == nil {
		return violations
	}

	f.extractViolationContext()

	for _, v := range f.ViolationDetailsXML.RequestViolations.Violations {
		violation := models.ViolationData{
			Name:    v.ViolName,
			Context: strings.ToLower(v.Context),
		}

		contextualized := true
		var name, value string
		var isB64Decoded bool
		switch strings.ToLower(v.Context) {
		case violationContextParameter:
			if v.ParameterData != (ParameterData{}) {
				isB64Decoded = v.ParameterData.IsBase64Decoded
				name = v.ParameterData.Name
				value = v.ParameterData.Value
			} else if v.ParamData != (ParamData{}) {
				isB64Decoded = v.ParamData.IsBase64Decoded
				name = v.ParamData.Name
				value = v.ParamData.Value
			} else if v.ParamName != "" {
				isB64Decoded = v.IsBase64Decoded
				name = v.ParamName
			} else {
				logger.Warn("context is parameter but no Parameter data received")
			}
		case violationContextHeader:
			if v.Header != (Header{}) {
				isB64Decoded = v.Header.IsBase64Decoded
				if v.Header.Name != "" || v.Header.Value != "" {
					name = v.Header.Name
					value = v.Header.Value
				} else {
					value = v.Header.Text
				}
			} else if v.HeaderData != (Header{}) {
				isB64Decoded = v.HeaderData.IsBase64Decoded
				name = v.HeaderData.Name
				value = v.HeaderData.Value
			} else if v.HeaderLength != "" {
				isB64Decoded = true
				decodedName, err := base64.StdEncoding.DecodeString(v.HeaderName)
				if err != nil {
					logger.Errorf("could not decode the Header %s for %v", v.HeaderName, f.SupportID)
					break
				}
				name = string(decodedName)
				value = fmt.Sprintf("Header length: %s, exceeds Header length limit: %s", v.HeaderLength, v.HeaderLengthLimit)
			}
		case violationContextCookie:
			if v.Cookie != (Cookie{}) && v.CookieLength == "" {
				isB64Decoded = v.Cookie.IsBase64Decoded
				name = v.Cookie.Name
				value = v.Cookie.Value
			} else if v.CookieName != "" {
				isB64Decoded = v.IsBase64Decoded
				name = v.CookieName
			} else if v.Buffer != "" {
				// `buffer` is base64 encoded, while `specific_desc` is not.
				isB64Decoded = true
				decodedBuffer, err := base64.StdEncoding.DecodeString(v.Buffer)
				if err != nil {
					logger.Errorf("could not decode the Buffer %s for %v", v.Buffer, f.SupportID)
					break
				}
				name = v.SpecificDesc
				value = string(decodedBuffer)
			} else if v.CookieLength != "" {
				isB64Decoded = true
				decodedValue, err := base64.StdEncoding.DecodeString(v.Cookie.Text)
				if err != nil {
					logger.Errorf("could not decode the Cookie %s for %v", v.Cookie.Text, f.SupportID)
					break
				}
				name = fmt.Sprintf("Cookie length: %s, exceeds Cookie length limit: %s", v.CookieLength, v.CookieLengthLimit)
				value = string(decodedValue)
			}
		case violationContextUri:
			if v.Uri != "" {
				name = base64.StdEncoding.EncodeToString([]byte(violationContextUri))
				value = v.Uri
			} else if v.UriObjectData != (UriObjectData{}) {
				name = base64.StdEncoding.EncodeToString([]byte(violationContextUri))
				value = v.UriObjectData.Object
			} else if v.UriLength != "" {
				isB64Decoded = true
				name = fmt.Sprintf("URI length: %s", v.UriLength)
				value = fmt.Sprintf("URI length limit: %s", v.UriLengthLimit)
			} else if v.HeaderData != (Header{}) {
				isB64Decoded = true
				decodedName, err := base64.StdEncoding.DecodeString(v.HeaderData.Name)
				if err != nil {
					logger.Errorf("uri context could not decode the Header name %s for %v", v.HeaderData.Name, f.SupportID)
					break
				}
				decodedActualValue, err := base64.StdEncoding.DecodeString(v.HeaderData.ActualValue)
				if err != nil {
					logger.Errorf("uri context could not decode the Actual Header value %s for %v", v.HeaderData.ActualValue, f.SupportID)
					break
				}
				decodedMatchedValue, err := base64.StdEncoding.DecodeString(v.HeaderData.MatchedValue)
				if err != nil {
					logger.Errorf("uri context could not decode the Matched Header value %s for %v", v.HeaderData.MatchedValue, f.SupportID)
					break
				}
				name = string(decodedName)
				value = fmt.Sprintf("actual header value: %s. matched header value: %s", string(decodedActualValue), string(decodedMatchedValue))
			}
		case violationContextRequest:
			isB64Decoded = true
			if v.DefinedLength != "" {
				name = fmt.Sprintf("Defined length: %s", v.DefinedLength)
				value = fmt.Sprintf("Detected length: %s", v.DetectedLength)
			} else if v.TotalLen != "" {
				name = fmt.Sprintf("Total length: %s", v.TotalLen)
				value = fmt.Sprintf("Total length limit: %s", v.TotalLenLimit)
			}
		default:
			contextualized = false
		}

		if contextualized {
			if isB64Decoded {
				name, value = populateNameValue(violation.Name, name, value)
				violation.ContextData = &models.ContextData{
					Name:  name,
					Value: value,
				}
			} else {
				decodedName, err := base64.StdEncoding.DecodeString(name)
				if err != nil {
					logger.Errorf("could not decode the %s Name %s for %v", v.Context, name, f.SupportID)
				} else {
					decodedValue, err := base64.StdEncoding.DecodeString(value)
					if err != nil {
						logger.Errorf("could not decode the %s Value %s for %v", v.Context, value, f.SupportID)
					} else {
						name, value = populateNameValue(violation.Name, string(decodedName), string(decodedValue))
						violation.ContextData = &models.ContextData{Name: name, Value: value}
					}
				}
			}
		}

		for _, s := range v.SigData {
			buf, err := base64.StdEncoding.DecodeString(s.KwData.Buffer)
			if err != nil {
				logger.Errorf("could not decode the Buffer value %s for %v", s, f.SupportID)
				continue
			}

			violation.Signatures = append(violation.Signatures, &models.SignatureData{
				ID:           s.SigID,
				BlockingMask: s.BlockingMask,
				Buffer:       string(buf),
				Offset:       s.KwData.Offset,
				Length:       s.KwData.Length,
			})
		}

		violations = append(violations, &violation)
	}

	return violations
}

func populateNameValue(violationName, dataName, dataValue string) (name string, value string) {
	if dataName != "" && dataValue != "" {
		name = dataName
		value = dataValue
		return
	}
	name = violationNameToDataName(violationName)
	if dataName == "" && dataValue != "" {
		value = dataValue
		return
	}
	if dataName != "" && dataValue == "" {
		value = dataName
		return
	}
	return
}

func violationNameToDataName(violationName string) string {
	c := cases.Title(language.English)
	parts := strings.Split(strings.ToLower(violationName), string(violationNameSeparator))
	if parts[0] != violationNamePrefix {
		return c.String(strings.Join(parts, " "))
	}
	return c.String(strings.Join(parts[1:], " "))
}

// Parse the NAP date time string into a Proto Time type.
func parseNAPDateTime(raw string) (*types.Timestamp, error) {
	t, err := time.Parse(napDateTimeLayout, raw)
	if err != nil {
		return nil, err
	}

	return types.TimestampProto(t)
}

func parseNAP(logEntry string, logger *logrus.Entry) (*NAPConfig, error) {
	logger.Tracef("Parsing log entry: %s", logEntry)

	var waf NAPConfig

	values := strings.Split(logEntry, ",")

	lenV := len(values)
	lenFK := len(logFormatKeys)
	if lenV != lenFK {
		return nil, fmt.Errorf("log line values does not match expected values. expecting %d values got %d values", lenFK, lenV)
	}

	for idx, key := range logFormatKeys {
		err := setValue(&waf, key, values[idx], logger)
		if err != nil {
			return &NAPConfig{}, err
		}
	}

	return &waf, nil
}

func setValue(napConfig *NAPConfig, key, value string, logger *logrus.Entry) error {
	switch key {
	case blockingExceptionReason:
		napConfig.BlockingExceptionReason = value
	case protocol:
		napConfig.Protocol = value
	case requestStatus:
		napConfig.RequestStatus = value
	case severity:
		napConfig.Severity = strings.ToLower(value)
	case sigSetNames:
		napConfig.SigSetNames = replaceEncodedList(value, listSeperator)
	case threatCampaignNames:
		napConfig.ThreatCampaignNames = replaceEncodedList(value, listSeperator)
	case violationDetails:
		napConfig.ViolationDetailsXML = func(data string) *BADMSG {
			var xmlData BADMSG
			if data != "" {
				err := xml.Unmarshal([]byte(data), &xmlData)
				if err != nil {
					logger.Errorf("failed to parse XML message: %v", err)
					return nil
				}
			}
			return &xmlData
		}(value)
	case clientApplication:
		napConfig.ClientApplication = value
	case clientApplicationVersion:
		napConfig.ClientApplicationVersion = value
	case transportProtocol:
		napConfig.TransportProtocol = value
	case httpHostname:
		napConfig.HTTPHostname = value
	case httpRemoteAddr:
		napConfig.HTTPRemoteAddr = value
	case httpRemotePort:
		napConfig.HTTPRemotePort = value
	case httpRequestMethod:
		napConfig.HTTPRequestMethod = value
	case httpResponseCode:
		napConfig.HTTPResponseCode = value
		if value == defaultBlockedRespCode {
			napConfig.HTTPResponseCode = defaultBlockedRespValue
		}
	case httpServerPort:
		napConfig.HTTPServerPort = value
	case httpURI:
		napConfig.HTTPURI = strings.ReplaceAll(value, encodedComma, decodedComma)
	case isTruncated:
		napConfig.IsTruncated = value
	case policyName:
		napConfig.PolicyName = value
	case request:
		napConfig.Request = strings.ReplaceAll(value, encodedComma, decodedComma)
	case requestOutcome:
		napConfig.RequestOutcome = value
	case requestOutcomeReason:
		napConfig.RequestOutcomeReason = value
	case sigCVEs:
		napConfig.SignatureCVEs = replaceEncodedList(value, listSeperator)
	case subViolations:
		napConfig.SubViolations = replaceEncodedList(value, listSeperator)
	case supportID:
		napConfig.SupportID = value
	case violations:
		napConfig.Violations = replaceEncodedList(value, listSeperator)
	case violationRating:
		napConfig.ViolationRating = value
	case xForwardedForHeaderVal:
		napConfig.XForwardedForHeaderVal = value
	case botAnomalies:
		napConfig.BotAnomalies = value
	case botCategory:
		napConfig.BotCategory = value
	case clientClass:
		napConfig.ClientClass = value
	case botSignatureName:
		napConfig.BotSignatureName = value
	case enforcedBotAnomalies:
		napConfig.EnforcedBotAnomalies = value
	default:
		msg := fmt.Sprintf("Invalid field for NAP Config - %s", key)
		return errors.New(msg)
	}
	return nil
}

func replaceEncodedList(entry, decoder string) string {
	return strings.ReplaceAll(entry, decoder, ",")
}
