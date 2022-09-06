package processor

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestParseNAPWAFExpanded(t *testing.T) {
	testCases := []struct {
		testName        string
		logEntry        string
		expNAPWAFConfig *NAPWAFConfig
		expError        error
	}{
		{
			testName: "ValidEntry",
			logEntry: fmt.Sprintf(`"%s"`, func() string {
				input, _ := ioutil.ReadFile("./testdata/expanded_nap_waf.log.txt")
				return string(input)
			}()),
			expNAPWAFConfig: &NAPWAFConfig{},
			expError:        nil,
		},
	}

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			_, err := parseNAPWAF(tc.logEntry, log.WithFields(logrus.Fields{
				"extension": "test",
			}))
			assert.Equal(t, tc.expError, err)
		})
	}
}
