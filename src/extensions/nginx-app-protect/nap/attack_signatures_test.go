package nap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const (
	testAttackSigVersionFile = "/tmp/test-attack-sigs-version.yaml"
	testAttackSigDateTime    = "2022-02-24T20:32:01Z"
	testAttackSigVersion     = "2022.02.24"
)

func TestGetAttackSignaturesVersion(t *testing.T) {
	testCases := []struct {
		testName          string
		versionFile       string
		attackSigDateTime *napRevisionDateTime
		expVersion        string
		expError          error
	}{
		{
			testName:    "AttackSignaturesInstalled",
			versionFile: testAttackSigVersionFile,
			attackSigDateTime: &napRevisionDateTime{
				RevisionDatetime: testAttackSigDateTime,
			},
			expVersion: testAttackSigVersion,
			expError:   nil,
		},
		{
			testName:          "AttackSignaturesNotInstalled",
			versionFile:       ATTACK_SIGNATURES_UPDATE_FILE,
			attackSigDateTime: nil,
			expVersion:        "",
			expError:          nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			// Create a fake version file if required by test
			if tc.attackSigDateTime != nil {
				yamlBytes, err := yaml.Marshal(tc.attackSigDateTime)
				assert.Nil(t, err)

				err = os.WriteFile(tc.versionFile, yamlBytes, 0644)
				assert.Nil(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					assert.Nil(t, err)
				}()
			}

			// Get attack signature version version
			version, err := getAttackSignaturesVersion(tc.versionFile)

			// Validate returned info
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expVersion, version)
		})
	}
}
