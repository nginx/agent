/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testThreatCampaignsVersionFile         = "/tmp/test-threat-campaigns-version.yaml"
	testThreatCampaignsVersionFileContents = `---
checksum: ALCdgk8CQgQQLRJ1ydZA4g
filename: threat_campaigns.bin.tgz
revisionDatetime: 2022-03-01T20:32:01Z
distro: focal
osType: debian`
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
				RevisionDatetime: "2022-03-01T20:32:01Z",
			},
			expVersion: "2022.03.01",
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
				err := os.WriteFile(tc.versionFile, []byte(testThreatCampaignsVersionFileContents), 0o644)
				require.NoError(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					require.NoError(t, err)
				}()
			}

			version, err := getThreatCampaignsVersion(tc.versionFile)
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expVersion, version)
		})
	}
}
