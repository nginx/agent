/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"errors"

	"github.com/cenkalti/backoff/v4"
)

var (
	ErrDownloadHeaderUnexpectedNumber = &backoff.PermanentError{Err: errors.New("unexpected number of headers")}
	ErrDownloadChecksumMismatch       = &backoff.PermanentError{Err: errors.New("download checksum mismatch")}
	ErrDownloadDataChunkNoData        = &backoff.PermanentError{Err: errors.New("download DataChunk without data")}
	ErrNotConnected                   = &backoff.PermanentError{Err: errors.New("not connected")}
	ErrUnmarshallingData              = &backoff.PermanentError{Err: errors.New("unable to unmarshal data")}
)
