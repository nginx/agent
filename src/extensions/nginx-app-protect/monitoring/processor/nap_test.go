/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package processor

import (
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestParseNAPExpanded(t *testing.T) {
	testCases := []struct {
		testName     string
		logEntry     string
		expNAPConfig *NAPConfig
		expError     error
	}{
		{
			testName: "ValidEntry",
			logEntry: fmt.Sprintf(`%s`, func() string {
				input, _ := os.ReadFile("./testdata/expanded_nap_waf.log.txt")
				return string(input)
			}()),
			expNAPConfig: &NAPConfig{},
			expError:     nil,
		},
	}

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			_, err := parseNAP(tc.logEntry, log.WithFields(logrus.Fields{
				"extension": "test",
			}))
			assert.Equal(t, tc.expError, err)
		})
	}
}

func TestParseNAPCommaEncoding(t *testing.T) {
	testCases := []struct {
		testName     string
		logEntry     string
		expNAPConfig *NAPConfig
		expError     error
	}{
		{
			testName: "Valid Entry with comma in both URI and Request",
			logEntry: fmt.Sprintf(`%s`, func() string {
				input, _ := os.ReadFile("./testdata/uri_request_contain_escaped_comma.log.txt")
				return string(input)
			}()),
			expNAPConfig: &NAPConfig{
				HTTPURI: "/with,comma",
				Request: "GET /with,comma HTTP/1.1\\r\\nHost: 10.146.183.68\\r\\nConnection: keep-alive\\r\\nCache-Control: max-age=0\\r\\nUpgrade-Insecure-Requests: 1\\r\\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36\\r\\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9\\r\\nAccept-Encoding: gzip, deflate\\r\\nAccept-Language: en-US,en;q=0.9\\r\\n\\r\\n",
			},
			expError: nil,
		},
	}

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			result, err := parseNAP(tc.logEntry, log.WithFields(logrus.Fields{
				"extension": "test",
			}))
			assert.Equal(t, tc.expNAPConfig.HTTPURI, result.HTTPURI)
			assert.Equal(t, tc.expNAPConfig.Request, result.Request)
			assert.Equal(t, tc.expError, err)
		})
	}
}
