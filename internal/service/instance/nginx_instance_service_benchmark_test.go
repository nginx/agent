// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"bytes"
	"context"
	"fmt"
	"testing"
)

func BenchmarkNginxService_parseNginxVersionCommandOutput(b *testing.B) {
	ctx := context.Background()

	output := fmt.Sprintf(`nginx version: nginx/1.23.3
	built by clang 14.0.0 (clang-1400.0.29.202)
	built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
	TLS SNI support enabled
	configure arguments: %s`, ossConfigArgs)

	for i := 0; i < b.N; i++ {
		parseNginxVersionCommandOutput(ctx, bytes.NewBufferString(output))
	}
}
