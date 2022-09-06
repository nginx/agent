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
	napDateTimeLayout = "2006-01-02 15:04:05.000"
	listSeperator     = "::"
	parameterCtx      = "parameter"
	headerCtx         = "header"
	cookieCtx         = "cookie"
)

var (
	logFormatKeys = []string{
		"date_time",
		"blocking_exception_reason",
		"dest_port",
		"ip_client",
		"is_truncated",
		"method",
		"policy_name",
		"protocol",
		"request_status",
		"response_code",
		"severity",
		"sig_cves",
		"sig_ids",
		"sig_names",
		"sig_set_names",
		"src_port",
		"sub_violations",
		"support_id",
		"threat_campaign_names",
		"unit_hostname",
		"uri",
		"violation_rating",
		"vs_name",
		"x_forwarded_for_header_value",
		"outcome",
		"outcome_reason",
		"violations",
		"violation_details",
		"bot_signature_name",
		"bot_category",
		"bot_anomalies",
		"enforced_bot_anomalies",
		"client_class",
		"client_application",
		"client_application_version",
		"transport_protocol",
	}
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
	httpRequestMethod    = "method"
	httpResponseCode     = "response_code"
	sigCVEs              = "sig_cves"
	sigIds               = "sig_ids"
	sigNames             = "sig_names"
	httpRemotePort       = "src_port"
	httpURI              = "uri"
	httpHostname         = "vs_name"
	requestOutcome       = "outcome"
	requestOutcomeReason = "outcome_reason"
	httpRemoteAddr       = "ip_client"
	httpServerPort       = "dest_port"
	isTruncated          = "is_truncated"

	// Older CAS Naming (this needs to be removed)
	// httpRequestMethod    = "http_request_method"
	// httpResponseCode     = "http_response_code"
	// sigCVEs              = "signature_cves"
	// sigIds               = "signature_ids"
	// sigNames             = "signature_names"
	// httpRemotePort       = "http_remote_port"
	// httpURI              = "http_uri"
	// httpHostname         = "http_hostname"
	// requestOutcome       = "request_outcome"
	// requestOutcomeReason = "request_outcome_reason"
	// description          = "description"
	// As -> description    == "attack_type"
	// httpRemoteAddr       = "http_remote_addr"
	// httpServerPort       = "http_server_port"
	// isTruncated          = "is_truncated_bool"

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
	return &models.SecurityViolationEvent{
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
		UnitHostname:             f.UnitHostname,
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
		// The following items needs to be fixed before release
		// TODO: https://nginxsoftware.atlassian.net/browse/NMS-38119
		DateTime:      f.DateTime,             // remove, metadata has it
		Outcome:       f.RequestOutcome,       //rename the proto
		OutcomeReason: f.RequestOutcomeReason, //rename the proto
		URI:           f.HTTPURI,              // rename to HTTP URI?
		GeoIP:         "blah",                 // to add
		Host:          "blah",                 // to add
		SourceHost:    "blah",                 // remove, this is the same as HTTPRemoteAddr
		SeverityLabel: "blah",                 // to do
		Priority:      "blah",                 // to do
	}
}

func (f *NAPConfig) getMetadata() (*models.Metadata, error) {
	// Set date time as current time with format YYYY-MM-DD HH:MM:SS.SSS
	// This is a temporary solution - https://nginxsoftware.atlassian.net/browse/IND-10651
	f.DateTime = time.Now().UTC().Format(napDateTimeLayout)

	t, err := parseNAPDateTime(f.DateTime)
	if err != nil {
		return nil, err
	}

	// set the correlation ID correctly
	// TODO: https://nginxsoftware.atlassian.net/browse/NMS-37563
	return NewMetadata(t, "123")
}

func (f *NAPConfig) getViolationContext() string {
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

func (f *NAPConfig) getViolations(logger *logrus.Entry) []*models.ViolationData {
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

// Parse the NAP date time string into a Proto Time type.
func parseNAPDateTime(raw string) (*types.Timestamp, error) {
	t, err := time.Parse(napDateTimeLayout, raw)
	if err != nil {
		return nil, err
	}

	return types.TimestampProto(t)
}

// Assumptions while parsing the NAP Syslog data:
// 1. list values do not contain `commas`, rather have `::` as delimiter
// 2. no json values
// 3. no other comma exists in the response other than the delimiter comma
// TODO: https://nginxsoftware.atlassian.net/browse/NMS-38118
func parseNAP(logEntry string, logger *logrus.Entry) (*NAPConfig, error) {
	var waf NAPConfig

	logger.Debugf("Got log entry: %s", logEntry)

	values := strings.Split(logEntry, ",")

	for idx, key := range logFormatKeys {
		err := setValue(&waf, key, values[idx], logger)
		if err != nil {
			return &NAPConfig{}, err
		}
	}

	err := setValue(&waf, "request", strings.Join(values[len(logFormatKeys):], ","), logger)
	if err != nil {
		return &NAPConfig{}, err
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
		napConfig.Severity = value
	case sigSetNames:
		napConfig.SigSetNames = replaceEncodedList(value, listSeperator)
	case threatCampaignNames:
		napConfig.ThreatCampaignNames = value
	case unitHostname:
		napConfig.UnitHostname = value
	case violationDetails:
		napConfig.ViolationDetailsXML = func(data string) *BADMSG {
			var xmlData BADMSG
			err := xml.Unmarshal([]byte(data), &xmlData)
			if err != nil {
				logger.Errorf("failed to parse XML message: %v", err)
				return nil
			}
			return &xmlData
		}(value)
	case clientApplication:
		napConfig.ClientApplication = value
	case clientApplicationVersion:
		napConfig.ClientApplicationVersion = value
	case transportProtocol:
		napConfig.TransportProtocol = value
	case dateTime:
		napConfig.DateTime = value
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
	case httpServerPort:
		napConfig.HTTPServerPort = value
	case httpURI:
		napConfig.HTTPURI = value
	case isTruncated:
		napConfig.IsTruncated = value
	case policyName:
		napConfig.PolicyName = value
	case request:
		napConfig.Request = value
	case requestOutcome:
		napConfig.RequestOutcome = value
	case requestOutcomeReason:
		napConfig.RequestOutcomeReason = value
	case sigCVEs:
		napConfig.SignatureCVEs = replaceEncodedList(value, listSeperator)
	case sigIds:
		napConfig.SignatureIDs = replaceEncodedList(value, listSeperator)
	case sigNames:
		napConfig.SignatureNames = replaceEncodedList(value, listSeperator)
	case subViolations:
		napConfig.SubViolations = value
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
