// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	filePermission = 0o700
)

func CreateDirWithErrorCheck(t testing.TB, dirName string) {
	t.Helper()

	err := os.MkdirAll(dirName, filePermission)

	require.NoError(t, err)
}

func CreateFileWithErrorCheck(t testing.TB, dir, fileName string) *os.File {
	t.Helper()

	testConf, err := os.CreateTemp(dir, fileName)
	require.NoError(t, err)

	return testConf
}

func RemoveFileWithErrorCheck(t testing.TB, fileName string) {
	t.Helper()

	err := os.Remove(fileName)

	require.NoError(t, err)
}

// RemoveASCIIControlSignals removes all non-printable ASCII control characters from a string.
func RemoveASCIIControlSignals(t testing.TB, input string) string {
	t.Helper()

	// Use a regex to match and remove ASCII control characters (0x00 to 0x1F and 0x7F).
	// by matching all control characters (ASCII 0–31 and 127).
	re := regexp.MustCompile(`[[:cntrl:]]`)

	return re.ReplaceAllString(input, "")
}
