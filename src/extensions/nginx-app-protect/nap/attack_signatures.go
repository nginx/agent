/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/nginx/agent/v2/src/core"

	"gopkg.in/yaml.v2"
)

// getAttackSignaturesVersion gets the version of the attack signatures package that is
// installed on the system, the version format is YYYY.MM.DD.
func getAttackSignaturesVersion(versionFile string) (string, error) {
	// Check if attack signatures version file exists
	logger.Debugf("Checking for the required NAP attack signatures version file - %v\n", versionFile)
	installed, err := core.FileExists(versionFile)
	if !installed && err == nil {
		return "", nil
	} else if err != nil {
		return "", err
	}

	// Get the version bytes
	versionBytes, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return "", err
	}

	// Read bytes into object
	attackSigVersionDateTime := napRevisionDateTime{}
	err = yaml.UnmarshalStrict([]byte(versionBytes), &attackSigVersionDateTime)
	if err != nil {
		return "", err
	}

	// Convert revision date into the proper version format
	attackSigTime, err := time.Parse(time.RFC3339, attackSigVersionDateTime.RevisionDatetime)
	if err != nil {
		return "", err
	}
	attackSignatureReleaseVersion := fmt.Sprintf("%d.%02d.%02d", attackSigTime.Year(), attackSigTime.Month(), attackSigTime.Day())
	logger.Debugf("Converted attack signature version (%s) found in %s to - %s\n", attackSigVersionDateTime.RevisionDatetime, ATTACK_SIGNATURES_UPDATE_FILE, attackSignatureReleaseVersion)

	return attackSignatureReleaseVersion, nil
}
