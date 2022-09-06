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
			logEntry: fmt.Sprintf(`"%s"`, func() string {
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
