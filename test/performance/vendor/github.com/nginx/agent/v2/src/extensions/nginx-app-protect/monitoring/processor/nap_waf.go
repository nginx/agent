package processor

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/sirupsen/logrus"

	models "github.com/nginx/agent/sdk/v2/proto/events"
)

const (
	napWAFDateTimeLayout = "2006-01-02 15:04:05.000"
	listSeperator        = "::"
)

const (
	// TODO: Identify the usage of the following new keys
	blockingExceptionReason  = "blocking_exception_reason"
	protocol                 = "protocol"
	requestStatus            = "request_status"
	severity                 = "severity"
	sigSetNames              = "sig_set_names"
	threatCampaignNames      = "threat_campaign_names"
	unitHostname             = "unit_hostname"
	violationDetails         = "violation_details"
	clientApplication        = "client_application"
	clientApplicationVersion = "client_application_version"
	transportProtocol        = "transport_protocol"

	// Using default values instead of overriden keys per older CAS policy
	// httpRequestMethod      = "http_request_method"
	httpRequestMethod = "method"
	// httpResponseCode       = "http_response_code"
	httpResponseCode = "response_code"
	// sigCVEs                = "signature_cves"
	sigCVEs = "sig_cves"
	// sigIds                 = "signature_ids"
	sigIds = "sig_ids"
	// sigNames               = "signature_names"
	sigNames = "sig_names"
	// httpRemotePort       = "http_remote_port"
	httpRemotePort = "src_port"
	// httpURI              = "http_uri"
	httpURI = "uri"
	// httpHostname   = "http_hostname"
	httpHostname = "vs_name"
	// requestOutcome       = "request_outcome"
	requestOutcome = "outcome"
	// requestOutcomeReason = "request_outcome_reason"
	requestOutcomeReason = "outcome_reason"
	// description = "description"
	// As -> description == "attack_type"
	// httpRemoteAddr         = "http_remote_addr"
	httpRemoteAddr = "ip_client"
	// httpServerPort         = "http_server_port"
	httpServerPort = "dest_port"
	isTruncated    = "is_truncated"
	// isTruncated = "is_truncated_bool"

	// Existing parsed keys from the log
	dateTime               = "date_time"
	policyName             = "policy_name"
	request                = "request"
	subViolations          = "sub_violations"
	supportID              = "support_id"
	violations             = "violations"
	violationRating        = "violation_rating"
	xForwardedForHeaderVal = "x_forwarded_for_header_value"
	botAnomalies           = "bot_anomalies"
	botCategory            = "bot_category"
	clientClass            = "client_class"
	botSignatureName       = "bot_signature_name"
	enforcedBotAnomalies   = "enforced_bot_anomalies"
)

type ParameterData struct {
	Text  string `xml:",chardata"`
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

type ParamData struct {
	Text  string `xml:",chardata"`
	Name  string `xml:"param_name"`
	Value string `xml:"param_value"`
}

type Header struct {
	Text  string `xml:",chardata"`
	Name  string `xml:"header_name"`
	Value string `xml:"header_value"`
}

type Cookie struct {
	Text  string `xml:",chardata"`
	Name  string `xml:"cookie_name"`
	Value string `xml:"cookie_value"`
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
		Text      string `xml:",chardata"`
		Violation []struct {
			Text      string `xml:",chardata"`
			ViolIndex string `xml:"viol_index"`
			ViolName  string `xml:"viol_name"`
			Context   string `xml:"context"`
			// ParameterData and ParamData are both received when context == "parameter" | ""
			// We receive either ParameterData or ParamData separately and not in the same XML message
			// ParameterData and ParamData semantically represent the same thing (with ParameterData having more fields).
			ParameterData ParameterData `xml:"parameter_data"`
			ParamData     ParamData     `xml:"param_data"`
			Header        Header        `xml:"header"`
			Cookie        Cookie        `xml:"cookie"`
			Staging       string        `xml:"staging"`
			SigData       []struct {
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

type NAPWAFConfig struct {
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
	SignatureIDs             string
	SignatureNames           string
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
func (f *NAPWAFConfig) GetEvent(hostPattern *regexp.Regexp, logger *logrus.Entry) (*models.Event, error) {
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

func (f *NAPWAFConfig) getSecurityViolation(logger *logrus.Entry) *models.SecurityViolationEvent {
	return &models.SecurityViolationEvent{
		DateTime:                 f.DateTime, // remove, metadata has it
		PolicyName:               f.PolicyName,
		SupportID:                f.SupportID,
		Outcome:                  f.RequestOutcome,       //rename the proto
		OutcomeReason:            f.RequestOutcomeReason, //rename the proto
		BlockingExceptionReason:  f.BlockingExceptionReason,
		Method:                   f.HTTPRequestMethod,
		Protocol:                 f.Protocol,
		XForwardedForHeaderValue: f.XForwardedForHeaderVal,
		URI:                      f.HTTPURI, // rename to HTTP URI?
		Request:                  f.Request,
		IsTruncated:              f.IsTruncated,
		RequestStatus:            f.RequestStatus,
		ResponseCode:             f.HTTPResponseCode,
		GeoIP:                    "blah", // to add
		Host:                     "blah", // to add
		UnitHostname:             f.UnitHostname,
		SourceHost:               "blah", // remove, this is the same as HTTPRemoteAddr
		VSName:                   f.HTTPHostname,
		IPClient:                 f.HTTPRemoteAddr,
		DestinationPort:          f.HTTPRemotePort,
		SourcePort:               f.HTTPServerPort,
		Violations:               f.Violations,
		SubViolations:            f.SubViolations,
		ViolationRating:          f.ViolationRating,
		SigID:                    f.SignatureIDs,
		SigNames:                 f.SignatureNames,
		SigSetNames:              f.SigSetNames,
		SigCVEs:                  f.SignatureCVEs,
		Severity:                 f.Severity,
		SeverityLabel:            "blah", // to do
		Priority:                 "blah", // to do
		ThreatCampaignNames:      f.ThreatCampaignNames,
		ClientClass:              f.ClientClass,
		ClientApplication:        f.ClientApplication,
		ClientApplicationVersion: f.ClientApplicationVersion,
		BotAnomalies:             f.BotAnomalies,
		BotCategory:              f.BotCategory,
		BotSignatureName:         f.BotSignatureName,
		EnforcedBotAnomalies:     f.EnforcedBotAnomalies,
		ViolationContexts:        f.getViolationContext(),
		ViolationsData:           f.getViolations(logger),
	}
}

func (f *NAPWAFConfig) getViolationContext() string {
	contexts := []string{}
	if f.ViolationDetailsXML != nil {
		for _, v := range f.ViolationDetailsXML.RequestViolations.Violation {
			if v.Context != "" {
				contexts = append(contexts, v.Context)
			}
		}
	}
	return strings.Join(contexts, ",")
}

func (f *NAPWAFConfig) getMetadata() (*models.Metadata, error) {
	// Set date time as current time with format YYYY-MM-DD HH:MM:SS.SSS
	// This is a temporary solution - https://nginxsoftware.atlassian.net/browse/IND-10651
	f.DateTime = time.Now().UTC().Format(napWAFDateTimeLayout)

	t, err := parseNAPDateTime(f.DateTime)
	if err != nil {
		return nil, err
	}

	// TODO: https://nginxsoftware.atlassian.net/browse/NMS-37563
	// set the correlation ID correctly
	return NewMetadata(t, "123")
}

// Parse the NAP WAF date time string into a Proto Time type.
func parseNAPDateTime(raw string) (*types.Timestamp, error) {
	t, err := time.Parse(napWAFDateTimeLayout, raw)
	if err != nil {
		return nil, err
	}

	return types.TimestampProto(t)
}

func parseNAPWAF(logEntry string, logger *logrus.Entry) (*NAPWAFConfig, error) {
	var waf NAPWAFConfig

	// Lasy reader
	// Assumptions:
	// 1. list values do not contain `commas`, rather have `::` as delimiter
	// 2. no json values
	// 3. no other comma exists in the response other than the delimiter comma
	logger.Infof("Entry: %s", logEntry)

	keys := []string{"date_time", "blocking_exception_reason", "dest_port", "ip_client", "is_truncated", "method", "policy_name", "protocol", "request_status", "response_code", "severity", "sig_cves", "sig_ids", "sig_names", "sig_set_names", "src_port", "sub_violations", "support_id", "threat_campaign_names", "unit_hostname", "uri", "violation_rating", "vs_name", "x_forwarded_for_header_value", "outcome", "outcome_reason", "violations", "violation_details", "bot_signature_name", "bot_category", "bot_anomalies", "enforced_bot_anomalies", "client_class", "client_application", "client_application_version", "transport_protocol"}
	values := strings.Split(logEntry, ",")

	for idx, key := range keys {
		err := setValue(&waf, key, values[idx], logger)
		if err != nil {
			return &NAPWAFConfig{}, err
		}
	}

	err := setValue(&waf, "request", strings.Join(values[len(keys):], ","), logger)
	if err != nil {
		return &NAPWAFConfig{}, err
	}

	return &waf, nil
}

func setValue(napWaf *NAPWAFConfig, key, value string, logger *logrus.Entry) error {
	switch key {
	case blockingExceptionReason:
		napWaf.BlockingExceptionReason = value
	case protocol:
		napWaf.Protocol = value
	case requestStatus:
		napWaf.RequestStatus = value
	case severity:
		napWaf.Severity = value
	case sigSetNames:
		napWaf.SigSetNames = replaceEncodedList(value, listSeperator)
	case threatCampaignNames:
		napWaf.ThreatCampaignNames = value
	case unitHostname:
		napWaf.UnitHostname = value
	case violationDetails:
		napWaf.ViolationDetailsXML = func(data string) *BADMSG {
			var xmlData BADMSG
			err := xml.Unmarshal([]byte(data), &xmlData)
			if err != nil {
				logger.Errorf("failed to parse XML message: %v", err)
				return nil
			}
			return &xmlData
		}(value)
	case clientApplication:
		napWaf.ClientApplication = value
	case clientApplicationVersion:
		napWaf.ClientApplicationVersion = value
	case transportProtocol:
		napWaf.TransportProtocol = value
	case dateTime:
		napWaf.DateTime = value
	case httpHostname:
		napWaf.HTTPHostname = value
	case httpRemoteAddr:
		napWaf.HTTPRemoteAddr = value
	case httpRemotePort:
		napWaf.HTTPRemotePort = value
	case httpRequestMethod:
		napWaf.HTTPRequestMethod = value
	case httpResponseCode:
		napWaf.HTTPResponseCode = value
	case httpServerPort:
		napWaf.HTTPServerPort = value
	case httpURI:
		napWaf.HTTPURI = value
	case isTruncated:
		napWaf.IsTruncated = value
	case policyName:
		napWaf.PolicyName = value
	case request:
		napWaf.Request = value
	case requestOutcome:
		napWaf.RequestOutcome = value
	case requestOutcomeReason:
		napWaf.RequestOutcomeReason = value
	case sigCVEs:
		napWaf.SignatureCVEs = replaceEncodedList(value, listSeperator)
	case sigIds:
		napWaf.SignatureIDs = replaceEncodedList(value, listSeperator)
	case sigNames:
		napWaf.SignatureNames = replaceEncodedList(value, listSeperator)
	case subViolations:
		napWaf.SubViolations = value
	case supportID:
		napWaf.SupportID = value
	case violations:
		napWaf.Violations = replaceEncodedList(value, listSeperator)
	case violationRating:
		napWaf.ViolationRating = value
	case xForwardedForHeaderVal:
		napWaf.XForwardedForHeaderVal = value
	case botAnomalies:
		napWaf.BotAnomalies = value
	case botCategory:
		napWaf.BotCategory = value
	case clientClass:
		napWaf.ClientClass = value
	case botSignatureName:
		napWaf.BotSignatureName = value
	case enforcedBotAnomalies:
		napWaf.EnforcedBotAnomalies = value
	default:
		msg := fmt.Sprintf("Invalid field for NAPWAFConfig - %s", key)
		return errors.New(msg)
	}
	return nil
}

func replaceEncodedList(entry, decoder string) string {
	return strings.ReplaceAll(entry, decoder, ",")
}

const (
	parameterCtx = "parameter"
	headerCtx    = "header"
	cookieCtx    = "cookie"
)

func (f *NAPWAFConfig) getViolations(logger *logrus.Entry) []*models.ViolationData {
	violations := []*models.ViolationData{}

	if f.ViolationDetailsXML == nil {
		return violations
	}

	for _, v := range f.ViolationDetailsXML.RequestViolations.Violation {
		violation := models.ViolationData{
			Name:    v.ViolName,
			Context: v.Context,
		}

		switch v.Context {
		case parameterCtx, "":
			if v.ParameterData != (ParameterData{}) {
				decodedName, err := base64.StdEncoding.DecodeString(v.ParameterData.Name)
				if err != nil {
					logger.Errorf("could not decode the Paramater Name %s for %v", v.ParameterData.Name, f.SupportID)
					break
				}
				decodedValue, err := base64.StdEncoding.DecodeString(v.ParameterData.Value)
				if err != nil {
					logger.Errorf("could not decode the Paramater Value %s for %v", v.ParameterData.Value, f.SupportID)
					break
				}

				violation.ContextData = &models.ContextData{
					Name:  string(decodedName),
					Value: string(decodedValue),
				}
			} else if v.ParamData != (ParamData{}) {
				decodedName, err := base64.StdEncoding.DecodeString(v.ParamData.Name)
				if err != nil {
					logger.Errorf("could not decode the Paramater Name %s for %v", v.ParamData.Name, f.SupportID)
					break
				}
				decodedValue, err := base64.StdEncoding.DecodeString(v.ParamData.Value)
				if err != nil {
					logger.Errorf("could not decode the Paramater Value %s for %v", v.ParamData.Value, f.SupportID)
					break
				}

				violation.ContextData = &models.ContextData{
					Name:  string(decodedName),
					Value: string(decodedValue),
				}
			} else if v.Context == parameterCtx {
				logger.Warn("context is parameter but no Parameter data received")
			}
		case headerCtx:
			if v.Header == (Header{}) {
				logger.Warn("context is header but no Header data received")
				break
			}

			decodedName, err := base64.StdEncoding.DecodeString(v.Header.Name)
			if err != nil {
				logger.Errorf("could not decode the Header Name %s for %v", v.Header.Name, f.SupportID)
				break
			}
			decodedValue, err := base64.StdEncoding.DecodeString(v.Header.Value)
			if err != nil {
				logger.Errorf("could not decode the Header Value %s for %v", v.Header.Value, f.SupportID)
				break
			}

			violation.ContextData = &models.ContextData{
				Name:  string(decodedName),
				Value: string(decodedValue),
			}
		case cookieCtx:
			if v.Cookie == (Cookie{}) {
				logger.Warn("context is cookie but no Cookie data received")
				break
			}

			decodedName, err := base64.StdEncoding.DecodeString(v.Cookie.Name)
			if err != nil {
				logger.Errorf("could not decode the Cookie Name %s for %v", v.Cookie.Name, f.SupportID)
				break
			}
			decodedValue, err := base64.StdEncoding.DecodeString(v.Cookie.Value)
			if err != nil {
				logger.Errorf("could not decode the Cookie Value %s for %v", v.Cookie.Value, f.SupportID)
				break
			}

			violation.ContextData = &models.ContextData{
				Name:  string(decodedName),
				Value: string(decodedValue),
			}
		default:
			logger.Warnf("Got an invalid context %v while parsing ViolationDetails for %v", v.Context, f.SupportID)
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
