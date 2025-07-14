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

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

const goModuleFileName = "go.mod"

func GoVersion(t testing.TB, level int) (string, error) {
	t.Helper()

	fileName := goModuleFileName
	filePath, modBytes, err := modfileBytes(fileName, level)
	if err != nil || filePath == "" {
		return "", err
	}

	file, err := parseModfile(filePath, modBytes)
	if err != nil {
		return "", err
	}

	return file.Go.Version, nil
}

func RequiredModuleVersion(t testing.TB, moduleName string, level int) (string, error) {
	t.Helper()

	fileName := goModuleFileName
	filePath, modBytes, err := modfileBytes(fileName, level)
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
			return normalizeVersion(requiredModule.Mod.Version), nil
		}
	}

	return file.Go.Version, nil
}

func Env(tb testing.TB, envKey string) string {
	tb.Helper()

	envValue := os.Getenv(envKey)
	tb.Logf("Environment variable %s is set to %s", envKey, envValue)

	require.NotEmptyf(tb, envValue, "Environment variable %s should not be empty", envKey)

	return envValue
}

func normalizeVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		return version[1:]
	}

	return version
}

func generatePattern(n int) (string, error) {
	var pattern strings.Builder
	currDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	pattern.WriteString(currDir)
	pattern.WriteRune(os.PathSeparator)

	for range n {
		pattern.WriteString("..")
		pattern.WriteRune(os.PathSeparator)
	}

	return pattern.String(), nil
}

func modfileBytes(fileName string, level int) (string, []byte, error) {
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
	return modfile.Parse(modFilePath, modBytes, nil)
}
