/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package files

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gogo/protobuf/types"
)

func GetFileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(0644)
	}

	return os.FileMode(result)
}

func GetPermissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
}

func TimeConvert(t time.Time) *types.Timestamp {
	ts, err := types.TimestampProto(t)
	if err != nil {
		return types.TimestampNow()
	}
	return ts
}
