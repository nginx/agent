/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package files

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/gogo/protobuf/types"
)

func GetFileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(0o644)
	}

	return os.FileMode(result)
}

func GetPermissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
}

func GetLineCount(path string) (int, error) {
	reader, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read file(%s) while trying to get lineCount: %v", path, err)
	}
	defer reader.Close()

	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := reader.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil
		case err != nil:
			return count, fmt.Errorf("failed to read file(%s): %v", path, err)
		}
	}
}

func TimeConvert(t time.Time) *types.Timestamp {
	ts, err := types.TimestampProto(t)
	if err != nil {
		return types.TimestampNow()
	}
	return ts
}
