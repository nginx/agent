/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"fmt"
	"os"
	"time"

	"github.com/nginx/agent/v2/src/core"

	"gopkg.in/yaml.v2"
)

// getThreatCampaignsVersion gets the version of the Threat campaigns package that is
// installed on the system, the version format is YYYY.MM.DD.
func getThreatCampaignsVersion(versionFile string) (string, error) {
	// Check if attack signatures version file exists
	logger.Debugf("Checking for the required NAP threat campaigns version file - %v\n", versionFile)
	installed, err := core.FileExists(versionFile)
	if !installed && err == nil {
		return "", nil
	} else if err != nil {
		return "", err
	}

	// Get the version bytes
	versionBytes, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}

	// Read bytes into object
	threatCampVersionDateTime := napRevisionDateTime{}
	err = yaml.UnmarshalStrict([]byte(versionBytes), &threatCampVersionDateTime)
	if err != nil {
		return "", err
	}

	// Convert revision date into the proper version format
	threatCampTime, err := time.Parse(time.RFC3339, threatCampVersionDateTime.RevisionDatetime)
	if err != nil {
		return "", err
	}
	threatCampaignsReleaseVersion := fmt.Sprintf("%d.%02d.%02d", threatCampTime.Year(), threatCampTime.Month(), threatCampTime.Day())
	logger.Debugf("Converted threat campaigns version (%s) found in %s to - %s\n", threatCampVersionDateTime.RevisionDatetime, THREAT_CAMPAIGNS_UPDATE_FILE, threatCampaignsReleaseVersion)

	return threatCampaignsReleaseVersion, nil
}
