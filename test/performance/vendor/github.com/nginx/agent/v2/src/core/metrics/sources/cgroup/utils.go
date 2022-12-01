/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package cgroup

import (
	"bufio"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

var pageSize = int64(os.Getpagesize())

type CgroupSource interface {
	Stats() float64
}

func ReadLines(filename string) ([]string, error) {
	return ReadLinesOffsetN(filename, 0, -1)
}

func ReadLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := 0; i < n+int(offset) || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
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

func GetV1DefaultMaxValue() string {
	max := int64(math.MaxInt64)
	return strconv.FormatInt(int64((max/pageSize)*pageSize), 10)
}
