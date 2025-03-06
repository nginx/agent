// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package v1_test

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
)

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

//nolint:gosec
func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return b
}

//nolint:gocognit,revive
func TestSendChunkedFile(t *testing.T) {
	for desc, td := range map[string]struct {
		clientFunc  func(cl *v1fakes.FakeFileService_GetFileStreamServer)
		header      v1.FileDataChunk_Header
		expectedErr error
		content     []byte
	}{
		"empty": {
			expectedErr: v1.ErrInvalidHeader,
		},
		"under chunk size": {
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 3,
					},
					Chunks:    1,
					ChunkSize: 1500,
				},
			},
			content: randBytes(3),
		},
		"exact chunk size": {
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 1500,
					},
					Chunks:    1,
					ChunkSize: 1500,
				},
			},
			content: randBytes(1500),
		},
		"over chunk size": {
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 2300,
					},
					Chunks:    2,
					ChunkSize: 1500,
				},
			},
			content: randBytes(2300),
		},
		"read under size (meta file size < actual content size)": {
			expectedErr: v1.ErrFailedRead,
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 2500,
					},
					Chunks:    2,
					ChunkSize: 1500,
				},
			},
			content: randBytes(2300),
		},
		"read over size (meta file size > actual content size)": {
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 2200,
					},
					Chunks:    2,
					ChunkSize: 1500,
				},
			},
			// we send up to the Size in the file meta
			content: randBytes(2300),
		},
		"send error (header)": {
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 2300,
					},
					Chunks:    2,
					ChunkSize: 1500,
				},
			},
			// we send up to the Size in the file meta
			content: randBytes(2300),
			clientFunc: func(cl *v1fakes.FakeFileService_GetFileStreamServer) {
				cl.SendReturns(fmt.Errorf("error"))
			},
			expectedErr: v1.ErrSend,
		},
		"send error (content)": {
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 2300,
					},
					Chunks:    2,
					ChunkSize: 1500,
				},
			},
			// we send up to the Size in the file meta
			content: randBytes(2300),
			clientFunc: func(cl *v1fakes.FakeFileService_GetFileStreamServer) {
				cl.SendCalls(func(chunk *v1.FileDataChunk) error {
					if cl.SendCallCount() > 1 {
						return fmt.Errorf("foo")
					}

					return nil
				})
			},
			expectedErr: v1.ErrSend,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			cl := &v1fakes.FakeFileService_GetFileStreamServer{}
			buf := &bytes.Buffer{}
			if td.clientFunc != nil {
				td.clientFunc(cl)
			} else {
				cl.SendCalls(func(chunk *v1.FileDataChunk) error {
					d, ok := chunk.GetChunk().(*v1.FileDataChunk_Content)
					if !ok {
						return nil
					}
					_, err := buf.Write(d.Content.GetData())
					require.NoError(t, err)

					return nil
				})
			}

			err := v1.SendChunkedFile(
				&v1.MessageMeta{},
				td.header,
				bytes.NewReader(td.content),
				cl,
			)
			if td.expectedErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, td.expectedErr)

				return
			}
			require.NoError(t, err)

			chunks := td.header.Header.GetChunks()
			assert.EqualValues(t, chunks+1, cl.SendCallCount())
			for i := 0; i < int(chunks+1); i++ {
				arg := cl.SendArgsForCall(i)
				switch i {
				case 0:
					assert.IsType(t, &v1.FileDataChunk_Header{}, arg.GetChunk())
					continue
				default:
					assert.IsType(t, &v1.FileDataChunk_Content{}, arg.GetChunk())
				}
			}
			sentBytes := buf.Bytes()
			if len(td.content) > len(sentBytes) {
				assert.Equal(t, td.content[0:len(sentBytes)], sentBytes)
			} else {
				assert.Equal(t, td.content, sentBytes)
			}
		})
	}
}

type badWriter struct{}

func (b badWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("error")
}

// nolint: revive,govet,maintidx
func TestRecvChunkedFile(t *testing.T) {
	recvErr := fmt.Errorf("recv error")

	type recvReturn struct {
		chunk *v1.FileDataChunk
		err   error
	}
	for desc, td := range map[string]struct {
		recvReturn        []recvReturn
		expectedErr       error
		expectedErrString string
		content           []byte
		writer            io.Writer
	}{
		"empty": {
			expectedErr: v1.ErrInvalidHeader,
		},
		"header with zero": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{},
							},
						},
					},
				},
			},
			expectedErr: v1.ErrInvalidHeader,
		},
		"data unmatched": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    1,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 0,
							},
						},
					},
				},
			},
			expectedErr: v1.ErrUnexpectedContent,
			// last chunk can be undersized
			expectedErrString: "1500 left",
		},
		"data unmatched - chunks size": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    2,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 0,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 1,
							},
						},
					},
				},
			},
			expectedErr:       v1.ErrUnexpectedContent,
			expectedErrString: "content chunk size 0, expected 1500",
		},
		"data unmatched - extra": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    2,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 0,
								Data:    randBytes(1500),
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 1,
								Data:    randBytes(1500),
							},
						},
					},
				},
			},
			expectedErr:       v1.ErrUnexpectedContent,
			expectedErrString: "1500 more data than expected",
		},
		"data unmatched - chunk id": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    2,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 5,
								Data:    randBytes(1500),
							},
						},
					},
				},
			},
			expectedErr:       v1.ErrUnexpectedContent,
			expectedErrString: "unexpected chunk id 5, expected 0",
		},
		"content recv error": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    2,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					err: recvErr,
				},
			},
			expectedErr:       v1.ErrFailedRead,
			expectedErrString: "content error " + recvErr.Error(),
		},
		"header recv error": {
			recvReturn: []recvReturn{
				{
					err: recvErr,
				},
			},
			expectedErr:       v1.ErrFailedRead,
			expectedErrString: "header error " + recvErr.Error(),
		},
		"data unmatched - nil content": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    1,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{},
					},
				},
			},
			expectedErr:       v1.ErrUnexpectedContent,
			expectedErrString: "no content",
		},
		"write error": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    1,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 0,
								Data:    randBytes(1500),
							},
						},
					},
				},
			},
			writer:      &badWriter{},
			expectedErr: v1.ErrWrite,
		},
		"good data": {
			recvReturn: []recvReturn{
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Header{
							Header: &v1.FileDataChunkHeader{
								FileMeta: &v1.FileMeta{
									Size: 1500,
								},
								Chunks:    1,
								ChunkSize: 1500,
							},
						},
					},
				},
				{
					chunk: &v1.FileDataChunk{
						Chunk: &v1.FileDataChunk_Content{
							Content: &v1.FileDataChunkContent{
								ChunkId: 0,
								Data:    randBytes(1500),
							},
						},
					},
				},
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			cl := &v1fakes.FakeFileService_GetFileStreamClient{}
			cl.RecvCalls(func() (*v1.FileDataChunk, error) {
				index := cl.RecvCallCount() - 1
				if index < len(td.recvReturn) {
					return td.recvReturn[index].chunk, td.recvReturn[index].err
				}

				return (*v1.FileDataChunk)(nil), nil
			})
			if td.writer == nil {
				td.writer = &bytes.Buffer{}
			}
			_, err := v1.RecvChunkedFile(cl, td.writer)
			if td.expectedErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, td.expectedErr)
				if td.expectedErrString != "" {
					assert.Contains(t, err.Error(), td.expectedErrString)
				}

				return
			}
			require.NoError(t, err)
		})
	}
}
