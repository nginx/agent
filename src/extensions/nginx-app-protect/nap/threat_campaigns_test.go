package nap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const (
	testThreatCampaignsVersionFile = "/tmp/test-threat-campaigns-version.yaml"
	testThreatCampaignsDateTime    = "2022-03-01T20:32:01Z"
	testThreatCampaignsVersion     = "2022.03.01"
)

func TestGetThreatCampaignsVersion(t *testing.T) {
	testCases := []struct {
		testName               string
		versionFile            string
		threatCampaignDateTime *napRevisionDateTime
		expVersion             string
		expError               error
	}{
		{
			testName:    "ThreatCampaignsInstalled",
			versionFile: testThreatCampaignsVersionFile,
			threatCampaignDateTime: &napRevisionDateTime{
				RevisionDatetime: testThreatCampaignsDateTime,
			},
			expVersion: testThreatCampaignsVersion,
			expError:   nil,
		},
		{
			testName:               "ThreatCampaignsNotInstalled",
			versionFile:            THREAT_CAMPAIGNS_UPDATE_FILE,
			threatCampaignDateTime: nil,
			expVersion:             "",
			expError:               nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create a fake version file if required by test
			if tc.threatCampaignDateTime != nil {
				yamlBytes, err := yaml.Marshal(tc.threatCampaignDateTime)
				assert.Nil(t, err)

				err = os.WriteFile(tc.versionFile, yamlBytes, 0644)
				assert.Nil(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					assert.Nil(t, err)
				}()
			}

			// Get threat campaign version
			version, err := getThreatCampaignsVersion(tc.versionFile)

			// Validate returned info
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expVersion, version)
		})
	}
}
