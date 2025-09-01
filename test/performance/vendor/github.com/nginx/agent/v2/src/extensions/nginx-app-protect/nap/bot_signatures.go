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

// botSignaturesVersion gets the version of the bot signatures package that is
// installed on the system, the version format is YYYY.MM.DD.
func botSignaturesVersion(versionFile string) (string, error) {
	// Check if bot signatures version file exists
	logger.Debugf("Checking for the required NAP bot signatures version file - %v\n", versionFile)
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
	botSigVersionDateTime := napRevisionDateTime{}
	err = yaml.Unmarshal([]byte(versionBytes), &botSigVersionDateTime)
	if err != nil {
		return "", err
	}

	// Convert revision date into the proper version format
	botSigTime, err := time.Parse(time.RFC3339, botSigVersionDateTime.RevisionDatetime)
	if err != nil {
		return "", err
	}
	botSignatureReleaseVersion := fmt.Sprintf("%d.%02d.%02d", botSigTime.Year(), botSigTime.Month(), botSigTime.Day())
	logger.Debugf("Converted bot signature version (%s) found in %s to - %s\n", botSigVersionDateTime.RevisionDatetime, BOT_SIGNATURES_UPDATE_FILE, botSignatureReleaseVersion)

	return botSignatureReleaseVersion, nil
}
