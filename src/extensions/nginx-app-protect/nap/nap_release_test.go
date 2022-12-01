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
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testNAPVersionFile     = "/tmp/test-nap-version"
	testNAPVersion         = "3.780.1" // This is the actual build number for NAP 3.9
	testUnsupportedVersion = "0.1.2"
)

var (
	testUnmappedBuildRelease = ReleaseUnmappedBuild(testUnsupportedVersion)
)

func TestInstalledNAPBuildVersion(t *testing.T) {
	testCases := []struct {
		testName        string
		versionFile     string
		version         string
		expBuildVersion string
		expError        error
	}{
		{
			testName:        "NAPVersionFileMissing",
			versionFile:     NAP_VERSION_FILE,
			version:         "",
			expBuildVersion: "",
			expError:        fmt.Errorf(FILE_NOT_FOUND, NAP_VERSION_FILE),
		},
		{
			testName:        "SuccessfullyGetNAPBuildVersion",
			versionFile:     testNAPVersionFile,
			version:         testNAPVersion,
			expBuildVersion: testNAPVersion,
			expError:        nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			// Create a fake version file if required by test
			if tc.version != "" {
				err := os.WriteFile(tc.versionFile, []byte(tc.version), 0644)
				assert.Nil(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					assert.Nil(t, err)
				}()
			}

			// Get build version
			buildVersion, err := installedNAPBuildVersion(tc.versionFile)

			// Validate returned build version
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expBuildVersion, buildVersion)
		})
	}
}

func buildFromPTR(v string) *NAPRelease {
	version := ReleaseUnmappedBuild(v)
	return &version
}
func TestInstalledNAPRelease(t *testing.T) {
	testCases := []struct {
		testName          string
		versionFile       string
		version           string
		expReleaseVersion *NAPRelease
		expError          error
	}{
		{
			testName:          "NAPVersionFileMissing",
			versionFile:       NAP_VERSION_FILE,
			version:           "",
			expReleaseVersion: nil,
			expError:          fmt.Errorf(FILE_NOT_FOUND, NAP_VERSION_FILE),
		},
		{
			testName:          "SuccessfullyGetNAPReleaseVersion",
			versionFile:       testNAPVersionFile,
			version:           testNAPVersion,
			expReleaseVersion: buildFromPTR(testNAPVersion),
			expError:          nil,
		},
		{
			testName:          "UnmappedBuildForSupportedReleases",
			versionFile:       testNAPVersionFile,
			version:           testUnsupportedVersion,
			expReleaseVersion: &testUnmappedBuildRelease,
			expError:          nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			// Create a fake version file if required by test
			if tc.version != "" {
				err := os.WriteFile(tc.versionFile, []byte(tc.version), 0644)
				assert.Nil(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					assert.Nil(t, err)
				}()
			}

			// Get build version
			releaseVersion, err := installedNAPRelease(tc.versionFile)

			// Validate returned build version
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expReleaseVersion, releaseVersion)
		})
	}
}
