package core

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	testFile = "/tmp/test-file"
)

func TestFileExists(t *testing.T) {
	testCases := []struct {
		testName     string
		fileToCheck  string
		createFile   bool
		expFileFound bool
		expError     error
	}{
		{
			testName:     "FileFound",
			fileToCheck:  fmt.Sprintf("%s-%s", testFile, uuid.New()),
			createFile:   true,
			expFileFound: true,
			expError:     nil,
		},
		{
			testName:     "FileNotFound",
			fileToCheck:  fmt.Sprintf("%s-%s", testFile, uuid.New()),
			createFile:   false,
			expFileFound: false,
			expError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create the fake file if required by test
			if tc.createFile {
				err := os.WriteFile(tc.fileToCheck, []byte("fake file for testing nginx-security"), 0644)
				assert.Nil(t, err)

				defer func(f string) {
					err := os.Remove(f)
					assert.Nil(t, err)
				}(tc.fileToCheck)
			}

			//Check if file exists
			exists, err := FileExists(tc.fileToCheck)
			assert.Equal(t, tc.expError, err)
			assert.Equal(t, tc.expFileFound, exists)
		})
	}
}

func TestFilesExists(t *testing.T) {
	testCases := []struct {
		testName     string
		filesToCheck []string
		createFiles  bool
		expFileFound bool
		expError     error
	}{
		{
			testName: "FilesFound",
			filesToCheck: []string{
				fmt.Sprintf("%s-%s", testFile, uuid.New()),
				fmt.Sprintf("%s-%s", testFile, uuid.New()),
				fmt.Sprintf("%s-%s", testFile, uuid.New()),
			},
			createFiles:  true,
			expFileFound: true,
			expError:     nil,
		},
		{
			testName: "FilesNotFound",
			filesToCheck: []string{
				fmt.Sprintf("%s-%s", testFile, uuid.New()),
				fmt.Sprintf("%s-%s", testFile, uuid.New()),
				fmt.Sprintf("%s-%s", testFile, uuid.New()),
			},
			createFiles:  false,
			expFileFound: false,
			expError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create the fake files if required by test
			if tc.createFiles {
				for _, file := range tc.filesToCheck {
					err := os.WriteFile(file, []byte("fake file for testing nginx-security"), 0644)
					assert.Nil(t, err)

					defer func(f string) {
						err := os.Remove(f)
						assert.Nil(t, err)
					}(file)
				}
			}

			//Check if file exists
			exists, err := FilesExists(tc.filesToCheck)
			assert.Equal(t, tc.expError, err)
			assert.Equal(t, tc.expFileFound, exists)
		})
	}
}
