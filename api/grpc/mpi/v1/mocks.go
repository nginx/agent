// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package v1

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . CommandServiceClient
//counterfeiter:generate . FileServiceClient
//counterfeiter:generate . FileService_GetFileStreamServer
//counterfeiter:generate . FileService_GetFileStreamClient
