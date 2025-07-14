// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package files implements utility routines for gathering information about files and their contents.
package files

import (
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

func BenchmarkGenerateConfigVersion(b *testing.B) {
	file1 := &mpi.File{
		FileMeta: &mpi.FileMeta{
			Name: "file1",
			Hash: "3151431543",
		},
	}
	file2 := &mpi.File{
		FileMeta: &mpi.FileMeta{
			Name: "file2",
			Hash: "4234235325",
		},
	}

	for range b.N {
		files := []*mpi.File{
			file1,
			file2,
		}
		GenerateConfigVersion(files)
	}
}
