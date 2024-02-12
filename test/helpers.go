// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateFileWithErrorCheck(t *testing.T, dir, fileName string) *os.File {
	t.Helper()
	testConf, err := os.CreateTemp(dir, fileName)
	require.NoError(t, err)

	return testConf
}

func RemoveFileWithErrorCheck(t *testing.T, fileName string) {
	t.Helper()
	err := os.Remove(fileName)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed on os.Remove of file %s", fileName))
	}
}

func CreateProtoTime(t *testing.T, timeString string) *timestamppb.Timestamp {
	t.Helper()
	newTime, err := time.Parse(time.RFC3339, timeString)
	require.NoError(t, err)

	protoTime := timestamppb.New(newTime)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed on creating timestamp %s", protoTime))
	}

	return protoTime
}
