package nap

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	testutils "github.com/nginx/agent/v2/test/utils"
)

const (
	testNAPFile = "/tmp/test-nap"
)

func TestNewNginxAppProtect(t *testing.T) {
	// TODO: Add a test case where NAP is installed
	testCases := []struct {
		testName string
		expNAP   *NginxAppProtect
		expError error
	}{
		{
			testName: "NAPNotInstalled",
			expNAP: &NginxAppProtect{
				Status:                  MISSING.String(),
				Release:                 NAPRelease{},
				AttackSignaturesVersion: "",
				ThreatCampaignsVersion:  "",
				optDirPath:              "",
				symLinkDir:              "",
			},
			expError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// get installation status
			nap, err := NewNginxAppProtect(tc.expNAP.optDirPath, tc.expNAP.symLinkDir)

			// Validate returned info
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expNAP, nap)
		})
	}
}

func TestGenerateNAPReport(t *testing.T) {
	testCases := []struct {
		testName     string
		nap          NginxAppProtect
		expNAPReport NAPReport
	}{
		{
			testName: "NAPInstalled",
			nap: NginxAppProtect{
				Status:                  INSTALLED.String(),
				Release:                 testUnmappedBuildRelease,
				AttackSignaturesVersion: "2022.02.24",
				ThreatCampaignsVersion:  "2022.03.01",
			},
			expNAPReport: NAPReport{
				Status:                  INSTALLED.String(),
				NAPVersion:              testUnmappedBuildRelease.VersioningDetails.NAPRelease,
				AttackSignaturesVersion: "2022.02.24",
				ThreatCampaignsVersion:  "2022.03.01",
			},
		},
		{
			testName: "NAPMissing",
			nap: NginxAppProtect{
				Status:                  MISSING.String(),
				Release:                 NAPRelease{},
				AttackSignaturesVersion: "",
				ThreatCampaignsVersion:  "",
			},
			expNAPReport: NAPReport{
				Status:                  MISSING.String(),
				NAPVersion:              "",
				AttackSignaturesVersion: "",
				ThreatCampaignsVersion:  "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// get NAP report message status
			napReport := tc.nap.GenerateNAPReport()

			// Validate returned info
			assert.Equal(t, tc.expNAPReport, napReport)
		})
	}
}

func TestNAPInstalled(t *testing.T) {
	testCases := []struct {
		testName        string
		requiredFiles   []string
		createFiles     bool
		expInstallation bool
		expError        error
	}{
		{
			testName:        "NAPMissing",
			requiredFiles:   requiredNAPFiles,
			createFiles:     false,
			expInstallation: false,
			expError:        nil,
		},
		{
			testName:        "NAPInstalled",
			requiredFiles:   []string{fmt.Sprintf("%s-%s", testNAPFile, uuid.New())},
			createFiles:     true,
			expInstallation: true,
			expError:        nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			// Create the fake files if required by test
			if tc.createFiles {
				for _, file := range tc.requiredFiles {
					err := os.WriteFile(file, []byte("fake file for testing nginx-security"), 0644)
					assert.Nil(t, err)

					defer func(f string) {
						err := os.Remove(f)
						assert.Nil(t, err)
					}(file)
				}
			}

			// get installation status
			installed, err := napInstalled(tc.requiredFiles)

			// Validate returned info
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expInstallation, installed)
		})
	}
}

func TestNAPRunning(t *testing.T) {
	testCases := []struct {
		testName      string
		procsToCreate []string
		expRunning    bool
		expError      error
	}{
		{
			testName:      "NAPNotRunning",
			procsToCreate: []string{},
			expRunning:    false,
			expError:      nil,
		},
		{
			testName:      "NAPRunning",
			procsToCreate: requireNAPProcesses,
			expRunning:    true,
			expError:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create fake process(es)
			killFakeProcesses := testutils.StartFakeProcesses(tc.procsToCreate, "5")
			t.Cleanup(killFakeProcesses)

			// Get running status
			running, err := napRunning()

			// Validate returned info
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expRunning, running)
		})
	}
}

func TestNAPStatus(t *testing.T) {
	testCases := []struct {
		testName      string
		procsToCreate []string
		requiredFiles []string
		createFiles   bool
		expStatus     Status
		expError      error
	}{
		{
			testName:      "NAPMissing",
			procsToCreate: nil,
			requiredFiles: requiredNAPFiles,
			createFiles:   false,
			expStatus:     MISSING,
			expError:      nil,
		},
		{
			testName:      "NAPInstalled",
			procsToCreate: nil,
			requiredFiles: []string{fmt.Sprintf("%s-%s", testNAPFile, uuid.New())},
			createFiles:   true,
			expStatus:     INSTALLED,
			expError:      nil,
		},
		{
			testName:      "NAPRunning",
			procsToCreate: requireNAPProcesses,
			requiredFiles: []string{fmt.Sprintf("%s-%s", testNAPFile, uuid.New())},
			createFiles:   true,
			expStatus:     RUNNING,
			expError:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			// Create the fake files if required by test
			if tc.createFiles {
				for _, file := range tc.requiredFiles {
					err := os.WriteFile(file, []byte("fake file for testing nginx-security"), 0644)
					assert.Nil(t, err)

					defer func(f string) {
						err := os.Remove(f)
						assert.Nil(t, err)
					}(file)
				}

			}

			// Create fake process(es)
			if tc.procsToCreate != nil {
				processCheckFunc = func(processesToCheck []string) ([]string, error) {
					return []string{}, nil
				}
			} else {
				processCheckFunc = func(processesToCheck []string) ([]string, error) {
					return []string{"fakeProc"}, nil
				}
			}

			// Get running status
			status, err := napStatus(tc.requiredFiles)

			// Validate returned info
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expStatus, status)
		})
	}
}
