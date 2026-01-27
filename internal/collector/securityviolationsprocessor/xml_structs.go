// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import "encoding/xml"

// XML struct definitions for parsing violation_details

// ParameterData represents parameter data in violation XML
type ParameterData struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"name"`
	Value           string `xml:"value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

// ParamData represents alternate parameter data format in violation XML
type ParamData struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"name"`
	Value           string `xml:"value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

// ContextDataWrapper wraps context-specific data in XML
//
//nolint:govet // fieldalignment: XML struct field order matters for unmarshaling
type ContextDataWrapper struct {
	Text      string     `xml:",chardata"`
	ParamData *ParamData `xml:"param_data"`
}

// Header represents HTTP header data in violation XML
type Header struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"header_name"`
	Value           string `xml:"header_value"`
	ActualValue     string `xml:"header_actual_value"`
	MatchedValue    string `xml:"header_matched_value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

// Cookie represents cookie data in violation XML
type Cookie struct {
	Text            string `xml:",chardata"`
	Name            string `xml:"cookie_name"`
	Value           string `xml:"cookie_value"`
	IsBase64Decoded bool   `xml:"is_base64_decoded"`
}

// UriObjectData represents URI object data in violation XML
type UriObjectData struct {
	Text   string `xml:",chardata"`
	Object string `xml:"object"`
}

// SigData represents signature data in violation XML
//
//nolint:revive // nested struct for XML unmarshaling
type SigData struct {
	Text         string `xml:",chardata"`
	SigID        string `xml:"sig_id"`
	BlockingMask string `xml:"blocking_mask"`
	KwData       struct {
		Text   string `xml:",chardata"`
		Buffer string `xml:"buffer"`
		Offset string `xml:"offset"`
		Length string `xml:"length"`
	} `xml:"kw_data"`
}

// Violation represents an individual violation in the XML structure
//
//nolint:govet // fieldalignment: XML struct field order matters for unmarshaling
type Violation struct {
	Text                   string              `xml:",chardata"`
	ViolIndex              string              `xml:"viol_index"`
	ViolName               string              `xml:"viol_name"`
	Context                string              `xml:"context"`
	ContextDataWrap        *ContextDataWrapper `xml:"context_data"`
	ParameterData          *ParameterData      `xml:"parameter_data"`
	ParamData              *ParamData          `xml:"param_data"`
	ParamName              string              `xml:"param_name"`
	IsBase64Decoded        bool                `xml:"is_base64_decoded"`
	Header                 *Header             `xml:"header"`
	HeaderData             *Header             `xml:"header_data"`
	HeaderName             string              `xml:"header_name"`
	HeaderLength           string              `xml:"header_len"`
	HeaderLengthLimit      string              `xml:"header_len_limit"`
	Cookie                 *Cookie             `xml:"cookie"`
	CookieName             string              `xml:"cookie_name"`
	CookieLength           string              `xml:"cookie_len"`
	CookieLengthLimit      string              `xml:"cookie_len_limit"`
	Buffer                 string              `xml:"buffer"`
	SpecificDesc           string              `xml:"specific_desc"`
	Uri                    string              `xml:"uri"`
	UriObjectData          *UriObjectData      `xml:"object_data"`
	UriLength              string              `xml:"uri_len"`
	UriLengthLimit         string              `xml:"uri_len_limit"`
	DefinedLength          string              `xml:"defined_length"`
	DetectedLength         string              `xml:"detected_length"`
	TotalLen               string              `xml:"total_len"`
	TotalLenLimit          string              `xml:"total_len_limit"`
	Staging                string              `xml:"staging"`
	HTTPSanityChecksStatus string              `xml:"http_sanity_checks_status"`
	HTTPSubViolationStatus string              `xml:"http_sub_violation_status"`
	HTTPSubViolation       string              `xml:"http_sub_violation"`
	WildcardEntity         string              `xml:"wildcard_entity"`
	LanguageType           string              `xml:"language_type"`
	MetacharIndex          []string            `xml:"metachar_index"`
	SigData                []*SigData          `xml:"sig_data"`
}

// BADMSG represents the root structure of NGINX App Protect violation XML
//
//nolint:revive // nested structs for XML unmarshaling
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
		Text       string       `xml:",chardata"`
		Violations []*Violation `xml:"violation"`
	} `xml:"request-violations"`
}
