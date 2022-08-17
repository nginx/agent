package nap

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/nginx/agent/v2/src/core"
)

// NewNAPReleaseMap is responsible for creating a NAPReleaseMap object that is contains
// info about each support NAP release.
func NewNAPReleaseMap() *NAPReleaseMap {
	return &NAPReleaseMap{
		ReleaseMap: map[string]NAPRelease{
			"3.11":  NAPRelease3_11(),
			"3.10":  NAPRelease3_10(),
			"3.9.1": NAPRelease3_9_1(),
			"3.9":   NAPRelease3_9(),
			"3.8":   NAPRelease3_8(),
			"3.7":   NAPRelease3_7(),
			"3.6":   NAPRelease3_6(),
			"3.5":   NAPRelease3_5(),
			"3.4":   NAPRelease3_4(),
			"3.3":   NAPRelease3_3(),
			"3.2":   NAPRelease3_2(),
			"3.1":   NAPRelease3_1(),
			"3.0":   NAPRelease3_0(),
		},
	}
}

// NAPReleaseInfo get the NAP release information for a specified NAP release version.
func NAPReleaseInfo(napReleaseVersion string) (*NAPRelease, error) {
	napRelease, exists := NewNAPReleaseMap().ReleaseMap[napReleaseVersion]
	if !exists {
		// Couldn't find details for supplied version
		msg := fmt.Sprintf(UNABLE_TO_FIND_RELEASE_VERION_INFO, napReleaseVersion)
		logger.Error(msg)
		return nil, errors.New(msg)
	}

	return &napRelease, nil
}

// installedNAPRelease gets the NAP release version based off the Nginx App Protect installed
// on the system.
func installedNAPRelease(versionFile string) (*NAPRelease, error) {
	// Get build version of NAP so we can determine the release  details
	napBuildVersion, err := installedNAPBuildVersion(versionFile)
	if err != nil {
		return nil, err
	}

	// Try to match NAP system build version to a build version in the NAP version mapping obj
	for releaseVersion, napRelease := range NewNAPReleaseMap().ReleaseMap {
		if napBuildVersion == napRelease.VersioningDetails.NAPBuild {
			logger.Debugf("Matched the NAP build version (%s) to the NAP release version (%s)\n", napBuildVersion, releaseVersion)
			return &napRelease, nil
		}
	}

	// No match found but we'll return a release with a build version
	logger.Errorf(UNABLE_TO_MATCH_NAP_BUILD_VERSION, napBuildVersion)
	logger.Warnf("Returning NAP release with only build number - %s", napBuildVersion)

	unmappedRelease := NAPReleaseUnmappedBuild(napBuildVersion)

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

	versionBytes, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return "", err
	}

	// Remove the trailing '\n' from the version string since it was read
	// from a file
	version := strings.Split(string(versionBytes), "\n")[0]

	return version, nil
}
