// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"sync/atomic"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FakeClientStreamingClient struct {
	sendCount atomic.Int32
}

func (f *FakeClientStreamingClient) Send(req *mpi.FileDataChunk) error {
	f.sendCount.Add(1)
	return nil
}

func (f *FakeClientStreamingClient) CloseAndRecv() (*mpi.UpdateFileResponse, error) {
	return &mpi.UpdateFileResponse{}, nil
}

func (f *FakeClientStreamingClient) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (f *FakeClientStreamingClient) Trailer() metadata.MD {
	return nil
}

func (f *FakeClientStreamingClient) CloseSend() error {
	return nil
}

func (f *FakeClientStreamingClient) Context() context.Context {
	return context.Background()
}

func (f *FakeClientStreamingClient) SendMsg(m any) error {
	return nil
}

func (f *FakeClientStreamingClient) RecvMsg(m any) error {
	return nil
}

type FakeServerStreamingClient struct {
	chunks         map[uint32][]byte
	fileName       string
	currentChunkID uint32
}

func (f *FakeServerStreamingClient) Recv() (*mpi.FileDataChunk, error) {
	fileDataChunk := &mpi.FileDataChunk{
		Meta: &mpi.MessageMeta{
			MessageId:     "123",
			CorrelationId: "1234",
			Timestamp:     timestamppb.Now(),
		},
	}

	if f.currentChunkID == 0 {
		fileDataChunk.Chunk = &mpi.FileDataChunk_Header{
			Header: &mpi.FileDataChunkHeader{
				FileMeta: &mpi.FileMeta{
					Name:        f.fileName,
					Permissions: "666",
				},
				Chunks:    52,
				ChunkSize: 1,
			},
		}
	} else {
		fileDataChunk.Chunk = &mpi.FileDataChunk_Content{
			Content: &mpi.FileDataChunkContent{
				ChunkId: f.currentChunkID,
				Data:    f.chunks[f.currentChunkID-1],
			},
		}
	}

	f.currentChunkID++

	return fileDataChunk, nil
}

func (f *FakeServerStreamingClient) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (f *FakeServerStreamingClient) Trailer() metadata.MD {
	return metadata.MD{}
}

func (f *FakeServerStreamingClient) CloseSend() error {
	return nil
}

func (f *FakeServerStreamingClient) Context() context.Context {
	return context.Background()
}

func (f *FakeServerStreamingClient) SendMsg(m any) error {
	return nil
}

func (f *FakeServerStreamingClient) RecvMsg(m any) error {
	return nil
}
