// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package internal

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

type Source interface {
	Stats() float64
}

func ReadLines(filename string) ([]string, error) {
	return ReadLinesOffsetN(filename, 0, -1)
}

// nolint: revive
func ReadLinesOffsetN(filename string, offset uint, n int) (lines []string, err error) {
	f, err := os.Open(filename)
	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}()

	if err != nil {
		return []string{}, err
	}

	var ret []string

	r := bufio.NewReader(f)
	for i := 0; i < n+int(offset) || n < 0; i++ {
		line, readErr := r.ReadString('\n')
		if readErr != nil {
			if readErr == io.EOF && len(line) > 0 {
				ret = append(ret, strings.Trim(line, "\n"))
			}

			break
		}
		if i < int(offset) {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}

func ReadSingleValueCgroupFile(filename string) (string, error) {
	lines, err := ReadLinesOffsetN(filename, 0, 1)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(lines[0]), nil
}

func ReadIntegerValueCgroupFile(filename string) (uint64, error) {
	value, err := ReadSingleValueCgroupFile(filename)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(value, 10, 64)
}

func IsCgroupV2(basePath string) bool {
	if _, err := os.Stat(basePath + "/cgroup.controllers"); err == nil {
		return true
	}

	return false
}
