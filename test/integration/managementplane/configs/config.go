// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package configs

import (
	_ "embed"
	"fmt"
)

//go:embed nginx.conf
var embedNginxConfWithMultipleInclude string

//go:embed nginx.conf
var embedNginxPlusConfWithMultipleInclude string

func NginxConfigWithMultipleInclude(includeFile1, includeFile2, includeFile3 string) string {
	return fmt.Sprintf(embedNginxConfWithMultipleInclude, includeFile1, includeFile2, includeFile3)
}

func NginxPlusConfigWithMultipleInclude(includeFile1, includeFile2, includeFile3 string) string {
	return fmt.Sprintf(embedNginxPlusConfWithMultipleInclude, includeFile1, includeFile2, includeFile3)
}
