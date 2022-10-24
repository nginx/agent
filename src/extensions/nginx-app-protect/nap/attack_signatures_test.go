package nap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAttackSigVersionFile         = "/tmp/test-attack-sigs-version.yaml"
	testAttackSigVersionFileContents = `---
checksum: t+N7AHGIKPhdDwb8zMZh2w
filename: signatures.bin.tgz
revisionDatetime: 2022-02-24T20:32:01Z`
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
				RevisionDatetime: "2022-02-24T20:32:01Z",
			},
			expVersion: "2022.02.24",
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
				err := os.WriteFile(tc.versionFile, []byte(testAttackSigVersionFileContents), 0644)
				require.NoError(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					require.NoError(t, err)
				}()
			}

			version, err := getAttackSignaturesVersion(tc.versionFile)
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expVersion, version)
		})
	}
}
