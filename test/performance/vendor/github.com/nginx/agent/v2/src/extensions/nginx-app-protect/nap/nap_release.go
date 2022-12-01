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
func installedNAPRelease(versionFile string) (*NAPRelease, error) {
	// Get build version of NAP, so we can determine the release  details
	napBuildVersion, err := installedNAPBuildVersion(versionFile)
	if err != nil {
		return nil, err
	}

	unmappedRelease := ReleaseUnmappedBuild(napBuildVersion)
	return &unmappedRelease, nil
}

// installedNAPBuildVersion gets the NAP build version based off the Nginx App Protect installed
// on the system.
func installedNAPBuildVersion(versionFile string) (string, error) {
	// Check if nap version file exists
	exists, err := core.FileExists(versionFile)
	if !exists && err == nil {
		return "", fmt.Errorf(FILE_NOT_FOUND, versionFile)
	} else if err != nil {
		return "", err
	}

	versionBytes, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}

	// Remove the trailing '\n' from the version string since it was read
	// from a file
	version := strings.Split(string(versionBytes), "\n")[0]

	return version, nil
}
