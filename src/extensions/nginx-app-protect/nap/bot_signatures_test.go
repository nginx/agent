/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBotSigVersionFile         = "/tmp/test-bot-sigs-version.yaml"
	testBotSigVersionFileContents = `---
checksum: nmlUse0aQzwLvuAaqDW/Jw
filename: bot_signatures.bin.tgz
revisionDatetime: 2025-08-21T11:30:33Z
`
)

func TestGetBotSignaturesVersion(t *testing.T) {
	testCases := []struct {
		testName       string
		versionFile    string
		botSigDateTime *napRevisionDateTime
		expVersion     string
		expError       error
	}{
		{
			testName:    "BotSignaturesInstalled",
			versionFile: testBotSigVersionFile,
			botSigDateTime: &napRevisionDateTime{
				RevisionDatetime: "2025-08-21T11:30:33Z",
			},
			expVersion: "2025.08.21",
			expError:   nil,
		},
		{
			testName:       "BotSignaturesNotInstalled",
			versionFile:    BOT_SIGNATURES_UPDATE_FILE,
			botSigDateTime: nil,
			expVersion:     "",
			expError:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create a fake version file if required by test
			if tc.botSigDateTime != nil {
				err := os.WriteFile(tc.versionFile, []byte(testBotSigVersionFileContents), 0o644)
				require.NoError(t, err)

				defer func() {
					err := os.Remove(tc.versionFile)
					require.NoError(t, err)
				}()
			}

			version, err := getBotSignaturesVersion(tc.versionFile)
			assert.Equal(t, err, tc.expError)
			assert.Equal(t, tc.expVersion, version)
		})
	}
}
