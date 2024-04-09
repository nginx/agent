// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

const goModuleFileName = "go.mod"

func GetGoVersion(t testing.TB, level int) (string, error) {
	t.Helper()

	fileName := goModuleFileName
	filePath, modBytes, err := getModfileBytes(fileName, level)
	if err != nil || filePath == "" {
		return "", err
	}

	file, err := parseModfile(filePath, modBytes)
	if err != nil {
		return "", err
	}

	return file.Go.Version, nil
}

func GetRequiredModuleVersion(t testing.TB, moduleName string, level int) (string, error) {
	t.Helper()

	fileName := goModuleFileName
	filePath, modBytes, err := getModfileBytes(fileName, level)
	if err != nil {
		return "", err
	}

	file, err := parseModfile(filePath, modBytes)
	if err != nil {
		return "", err
	}

	for _, requiredModule := range file.Require {
		t.Logf("%s : %s", requiredModule.Mod.Path, requiredModule.Mod.Version)
		if requiredModule.Mod.Path == moduleName {
			if strings.HasPrefix(requiredModule.Mod.Version, "v") {
				return requiredModule.Mod.Version[1:], nil
			}
			return requiredModule.Mod.Version, nil
		}
	}

	return file.Go.Version, nil
}

func generatePattern(n int) (string, error) {
	var pattern strings.Builder
	currDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	pattern.WriteString(currDir)
	pattern.WriteRune(os.PathSeparator)

	for i := 0; i < n; i++ {
		pattern.WriteString("..")
		pattern.WriteRune(os.PathSeparator)
	}

	return pattern.String(), nil
}

func getModfileBytes(fileName string, level int) (string, []byte, error) {
	prefix, err := generatePattern(level)
	if err != nil {
		return "", nil, err
	}

	filePath := fmt.Sprintf("%s%s", prefix, fileName)
	modBytes, err := os.ReadFile(filePath)
	if err != nil {
		filePath = fmt.Sprintf("..%s..%s%s", string(os.PathSeparator), string(os.PathSeparator), fileName)
		if modBytes, err = os.ReadFile(filePath); err != nil {
			return "", nil, err
		}
	}

	return filePath, modBytes, nil
}

func parseModfile(modFilePath string, modBytes []byte) (*modfile.File, error) {
	file, err := modfile.Parse(modFilePath, modBytes, nil)
	if err != nil {
		return nil, err
	}
	return file, nil
}
