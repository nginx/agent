// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package files_test

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/pkg/files"
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
	tests := []struct {
		name              string
		clientFunc        func(cl *v1fakes.FakeFileService_GetFileStreamServer)
		header            v1.FileDataChunk_Header
		expectedErrString string
		content           []byte
	}{
		{
			name:              "Test 1: zero header",
			expectedErrString: "file size in header is zero",
		},
		{
			name: "Test 2: under chunk size",
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
		{
			name: "Test 3: exact chunk size",
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
		{
			name: "Test 4: over chunk size",
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
		{
			name: "Test 5: read under size (meta file size < actual content size)",
			header: v1.FileDataChunk_Header{
				Header: &v1.FileDataChunkHeader{
					FileMeta: &v1.FileMeta{
						Size: 2500,
					},
					Chunks:    2,
					ChunkSize: 1500,
				},
			},
			content:           randBytes(2300),
			expectedErrString: "unable to read chunk id 1",
		},
		{
			name: "Test 6: read over size (meta file size > actual content size)",
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
		{
			name: "Test 7: send error (header)",
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
				cl.SendReturns(errors.New("error"))
			},
			expectedErrString: "unable to send header chunk",
		},
		{
			name: "Test 8: send error (content)",
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
						return errors.New("foo")
					}

					return nil
				})
			},
			expectedErrString: "unable to send chunk id 0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cl := &v1fakes.FakeFileService_GetFileStreamServer{}
			buf := &bytes.Buffer{}
			if test.clientFunc != nil {
				test.clientFunc(cl)
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

			err := files.SendChunkedFile(
				&v1.MessageMeta{},
				test.header,
				bytes.NewReader(test.content),
				cl,
			)
			if test.expectedErrString != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErrString)

				return
			}
			require.NoError(t, err)

			chunks := test.header.Header.GetChunks()
			assert.EqualValues(t, chunks+1, cl.SendCallCount())
			for i := range int(chunks + 1) {
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
			if len(test.content) > len(sentBytes) {
				assert.Equal(t, test.content[0:len(sentBytes)], sentBytes)
			} else {
				assert.Equal(t, test.content, sentBytes)
			}
		})
	}
}

type badWriter struct{}

func (b badWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("error")
}

//nolint:revive,govet,maintidx
func TestRecvChunkedFile(t *testing.T) {
	recvErr := errors.New("recv error")
	type recvReturn struct {
		chunk *v1.FileDataChunk
		err   error
	}
	tests := []struct {
		name              string
		recvReturn        []recvReturn
		expectedErrString string
		content           []byte
		writer            io.Writer
	}{
		{
			name:              "Test 1: empty send",
			expectedErrString: "no header chunk",
		},
		{
			name: "Test 2: header with zeros",
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
			expectedErrString: "file size in header is zero",
		},
		{
			name: "Test 3: data unmatched",
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
			// last chunk can be undersized
			expectedErrString: "expected additional 1500 bytes",
		},
		{
			name: "Test 4: data unmatched - chunks size",
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
			expectedErrString: "content chunk size of 0 does not match expected size of 1500",
		},
		{
			name: "Test 5: data unmatched - extra",
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
			expectedErrString: "unexpected content: 1500 bytes more data than expected",
		},
		{
			name: "Test 6: data unmatched - chunk id",
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
			expectedErrString: "content chunk id of 5 does not match expected id of 0",
		},
		{
			name: "Test 7: content recv error",
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
			expectedErrString: "unable to receive chunk id 0",
		},
		{
			name: "Test 8: header recv error",
			recvReturn: []recvReturn{
				{
					err: recvErr,
				},
			},
			expectedErrString: "unable to receive header chunk",
		},
		{
			name: "Test 9: data unmatched - nil content",
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
			expectedErrString: "no content",
		},
		{
			name: "Test 10: write error",
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
			writer:            &badWriter{},
			expectedErrString: "unable to write chunk id 0",
		},
		{
			name: "Test 11: no error",
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cl := &v1fakes.FakeFileService_GetFileStreamClient{}
			cl.RecvCalls(func() (*v1.FileDataChunk, error) {
				index := cl.RecvCallCount() - 1
				if index < len(test.recvReturn) {
					return test.recvReturn[index].chunk, test.recvReturn[index].err
				}

				return (*v1.FileDataChunk)(nil), nil
			})
			if test.writer == nil {
				test.writer = &bytes.Buffer{}
			}
			_, err := files.RecvChunkedFile(cl, test.writer)
			if test.expectedErrString != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErrString)

				return
			}
			require.NoError(t, err)
		})
	}
}
