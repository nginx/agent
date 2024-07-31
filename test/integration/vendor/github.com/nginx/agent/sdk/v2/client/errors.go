/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrDownloadHeaderUnexpectedNumber = &backoff.PermanentError{Err: status.Error(codes.DataLoss, "unexpected number of headers")}
	ErrDownloadChecksumMismatch       = &backoff.PermanentError{Err: status.Error(codes.DataLoss, "download checksum mismatch")}
	ErrDownloadDataChunkNoData        = &backoff.PermanentError{Err: status.Error(codes.DataLoss, "download DataChunk without data")}
	ErrUnmarshallingData              = &backoff.PermanentError{Err: status.Error(codes.DataLoss, "unable to unmarshal data")}
)
