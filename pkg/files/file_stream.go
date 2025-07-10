// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package files

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

// SendChunkedFile reads the src into [FileDataChunkContent]s, and sends a valid sequence of
// [FileDataChunk]s down the stream.
func SendChunkedFile(
	meta *v1.MessageMeta,
	header v1.FileDataChunk_Header,
	src io.Reader,
	dst grpc.ServerStreamingServer[v1.FileDataChunk],
) error {
	chunkCount := int(header.Header.GetChunks())
	chunkSize := int(header.Header.GetChunkSize())
	total := int(header.Header.GetFileMeta().GetSize())
	if chunkSize == 0 || chunkCount == 0 || total == 0 {
		return fmt.Errorf("file size in header is zero: %+v", header.Header)
	}
	if err := dst.Send(&v1.FileDataChunk{
		Meta:  meta,
		Chunk: &header,
	}); err != nil {
		return fmt.Errorf("unable to send header chunk: %w", err)
	}
	// allocate the buffer we need for reading from io.Reader
	// this is set to the size of the chunks we need to send.
	// since "send" is synchronous and data in buffer re-marshaled, we shouldn't
	// need to reallocate for each chunk.
	buf := make([]byte, chunkSize)
	for i := range chunkCount {
		// using ReadFull here, since the Read may get partial fills depends on impl, due to Read being
		// at most once op per call. But we want to fill our buffer every loop, for the defined chunk size.
		// Set size of the buffer, so we don't get the error from io.ReadFull, if the data fits just right
		n, err := io.ReadFull(src, buf[0:min(chunkSize, total)])
		total -= n
		if err != nil && total != 0 {
			// partial read
			return fmt.Errorf("unable to read chunk id %d: %w", i, err)
		}
		if err = dst.Send(&v1.FileDataChunk{
			Meta: meta,
			Chunk: &v1.FileDataChunk_Content{
				Content: &v1.FileDataChunkContent{
					ChunkId: uint32(i),
					Data:    buf[0:n],
				},
			},
		}); err != nil {
			return fmt.Errorf("unable to send chunk id %d: %w", i, err)
		}
	}

	return nil
}

// RecvChunkedFile receives [FileDataChunkContent]s from the stream and writes the file contents
// to the dst.
func RecvChunkedFile(
	src grpc.ServerStreamingClient[v1.FileDataChunk],
	dst io.Writer,
) (header *v1.FileMeta, err error) {
	// receive the header first
	chunk, err := src.Recv()
	if err != nil {
		return header, fmt.Errorf("unable to receive header chunk: %w", err)
	}

	// validate and extract header info
	headerChunk := chunk.GetHeader()
	if headerChunk == nil {
		return header, errors.New("no header chunk")
	}

	header = headerChunk.GetFileMeta()
	chunkCount := int(headerChunk.GetChunks())
	chunkSize := int(headerChunk.GetChunkSize())
	total := int(header.GetSize())

	if chunkSize == 0 || chunkCount == 0 || total == 0 {
		return header, fmt.Errorf("file size in header is zero: %+v", headerChunk)
	}

	return header, recvContents(src, dst, chunkCount, chunkSize, total)
}

func recvContents(
	src grpc.ServerStreamingClient[v1.FileDataChunk],
	dst io.Writer,
	chunkCount int,
	chunkSize int,
	totalSize int,
) error {
	// receive and write all content chunks
	for i := range chunkCount {
		chunk, err := src.Recv()
		if err != nil {
			return fmt.Errorf("unable to receive chunk id %d: %w", i, err)
		}

		if err = validateRecvChunk(chunk, chunkSize, chunkCount-1, i); err != nil {
			return err
		}
		data := chunk.GetContent().GetData()
		if _, err = dst.Write(data); err != nil {
			return fmt.Errorf("unable to write chunk id %d: %w", i, err)
		}
		totalSize -= len(data)
		if 0 > totalSize {
			return fmt.Errorf("unexpected content: %d bytes more data than expected", 0-totalSize)
		}
	}
	if totalSize > 0 {
		return fmt.Errorf("unexpected content: unexpected end of content, expected additional %d bytes", totalSize)
	}

	return nil
}

func validateRecvChunk(chunk *v1.FileDataChunk, chunkSize, lastChunkIndex, chunkID int) error {
	content := chunk.GetContent()
	if content == nil {
		return fmt.Errorf("no content in chunk id %d", chunkID)
	}
	if content.GetChunkId() != uint32(chunkID) {
		return fmt.Errorf("content chunk id of %d does not match expected id of %d",
			content.GetChunkId(), chunkID)
	}
	data := content.GetData()
	if len(data) != chunkSize && chunkID != lastChunkIndex {
		return fmt.Errorf("content chunk size of %d does not match expected size of %d",
			len(data), chunkSize)
	}

	return nil
}
