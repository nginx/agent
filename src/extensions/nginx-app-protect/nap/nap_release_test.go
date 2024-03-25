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
	testNAPVersionFile = "/tmp/test-nap-version"
	testNAPReleaseFile = "/tmp/test-nap-release"
	testNoFile         = "/tmp/no-file"
	testNAPVersion     = "4.815.0" // This is the actual build number for NAP 4.8.1
	testNAPRelease     = "4.8.1"
	testUnsupported    = "0.1.2"
)

var testUnmappedBuildRelease = ReleaseUnmappedBuild(testNAPVersion, testUnsupported)

func TestInstalledNAPBuildVersion(t *testing.T) {
	testCases := []struct {
		testName        string
		versionFile     string
		version         string
		releaseFile     string
		release         string
		expBuildVersion string
		expRelease      string
		expError        error
	}{
		{
			testName:        "NAPVersionFileMissing",
			versionFile:     testNoFile,
			version:         "",
			expBuildVersion: "",
			expError:        fmt.Errorf(FILE_NOT_FOUND, testNoFile),
		},
		{
			testName:        "SuccessfullyGetNAPVersion",
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
				err := os.WriteFile(tc.versionFile, []byte(tc.version), 0o644)
				assert.Nil(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					assert.Nil(t, err)
				}()
			}

			// Get build version
			buildVersion, err := installedNAP(tc.versionFile)

			// Validate returned build version
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expBuildVersion, buildVersion)
		})
	}
}

func buildFromPTR(v, r string) *NAPRelease {
	version := ReleaseUnmappedBuild(v, r)
	return &version
}

func TestInstalledNAPRelease(t *testing.T) {
	testCases := []struct {
		testName    string
		releaseFile string
		release     string
		expRelease  *NAPRelease
		expError    error
	}{
		{
			testName:    "NAPReleaseFileMissing",
			releaseFile: testNoFile,
			release:     "",
			expRelease:  nil,
			expError:    fmt.Errorf(FILE_NOT_FOUND, testNAPVersionFile),
		},
		{
			testName:    "SuccessfullyGetNAPReleaseVersion",
			releaseFile: testNAPReleaseFile,
			release:     testNAPRelease,
			expRelease:  buildFromPTR(testNAPVersion, testNAPRelease),
			expError:    nil,
		},
		{
			testName:    "UnmappedBuildForSupportedReleases",
			releaseFile: testNAPReleaseFile,
			release:     testUnsupported,
			expRelease:  &testUnmappedBuildRelease,
			expError:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create a fake version file if required by test
			if tc.release != "" {
				err := os.WriteFile(tc.releaseFile, []byte(tc.release), 0o644)
				assert.Nil(t, err)
				err = os.WriteFile(testNAPVersionFile, []byte(testNAPVersion), 0o644)
				assert.Nil(t, err)

				defer func() {
					err := os.Remove(tc.releaseFile)
					assert.Nil(t, err)
					err = os.Remove(testNAPVersionFile)
					assert.Nil(t, err)
				}()
			}

			// Get release
			release, err := installedNAPRelease(testNAPVersionFile, tc.releaseFile)

			// Validate returned release
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expRelease, release)
		})
	}
}
