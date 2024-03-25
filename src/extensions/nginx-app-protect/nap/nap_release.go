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
	"strings"

	"github.com/nginx/agent/v2/src/core"
)

// installedNAPRelease gets the NAP release version based off the Nginx App Protect installed
// on the system.
func installedNAPRelease(versionFile, releaseFile string) (*NAPRelease, error) {
	// Get build version of NAP, so we can determine the release details
	napBuildVersion, err := installedNAP(versionFile)
	if err != nil {
		return nil, err
	}
	napRelease, err := installedNAP(releaseFile)
	if err != nil {
		return nil, err
	}

	unmappedRelease := ReleaseUnmappedBuild(napBuildVersion, napRelease)
	return &unmappedRelease, nil
}

// installedNAP gets the NAP version or release based off the Nginx App Protect installed
// on the system.
func installedNAP(file string) (string, error) {
	// Check if nap version file exists
	exists, err := core.FileExists(file)
	if !exists && err == nil {
		return "", fmt.Errorf(FILE_NOT_FOUND, file)
	} else if err != nil {
		return "", err
	}

	bytes, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	// Remove the trailing '\n' from the string since it was read
	// from a file
	napNumber := strings.Split(string(bytes), "\n")[0]

	return napNumber, nil
}
