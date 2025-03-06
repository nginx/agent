// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package v1

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
)

var (
	ErrInvalidHeader     = errors.New("invalid header")
	ErrUnexpectedContent = errors.New("unexpected content")
	ErrSend              = errors.New("send error")
	ErrWrite             = errors.New("write error")
	ErrFailedRead        = errors.New("failed to read")
)

// SendChunkedFile reads the src into [FileDataChunkContent]s, and sends a valid sequence of
// [FileDataChunk]s down the stream.
func SendChunkedFile(
	meta *MessageMeta,
	header FileDataChunk_Header,
	src io.Reader,
	dst grpc.ServerStreamingServer[FileDataChunk],
) error {
	chunkCount := int(header.Header.GetChunks())
	chunkSize := int(header.Header.GetChunkSize())
	total := int(header.Header.GetFileMeta().GetSize())
	if chunkSize == 0 || chunkCount == 0 || total == 0 {
		return fmt.Errorf("%w:  %v", ErrInvalidHeader, header.Header)
	}
	if err := dst.Send(&FileDataChunk{
		Meta:  meta,
		Chunk: &header,
	}); err != nil {
		return fmt.Errorf("%w: %s (header)", ErrSend, err)
	}
	// allocate the buffer we need for reading from io.Reader
	// this is set to the size of the chunks we need to send.
	// since "send" is synchronous and data in buffer re-marshaled, we shouldn't
	// need to reallocate for each chunk.
	buf := make([]byte, chunkSize)
	for i := 0; i < chunkCount; i++ {
		// using ReadFull here, since the Read may get partial fills depends on impl, due to Read being
		// at most once op per call. But we want to fill our buffer every loop, for the defined chunk size.
		// Set size of the buffer, so we don't get the error from io.ReadFull, if the data fits just right
		n, err := io.ReadFull(src, buf[0:min(chunkSize, total)])
		total -= n
		if err != nil && total != 0 {
			// partial read
			return fmt.Errorf("%w: %s", ErrFailedRead, err)
		}
		if err = dst.Send(&FileDataChunk{
			Meta: meta,
			Chunk: &FileDataChunk_Content{
				Content: &FileDataChunkContent{
					ChunkId: uint32(i),
					Data:    buf[0:n],
				},
			},
		}); err != nil {
			return fmt.Errorf("%w: %s (content)", ErrSend, err)
		}
	}

	return nil
}

// RecvChunkedFile receives [FileDataChunkContent]s from the stream and writes the file contents
// to the dst.
func RecvChunkedFile(
	src grpc.ServerStreamingClient[FileDataChunk],
	dst io.Writer,
) (header *FileMeta, err error) {
	// receive the header first
	chunk, err := src.Recv()
	if err != nil {
		return header, fmt.Errorf("%w: header error %s", ErrFailedRead, err)
	}

	// validate and extract header info
	headerChunk := chunk.GetHeader()
	if headerChunk == nil {
		return header, fmt.Errorf("%w: invalid header chunk", ErrInvalidHeader)
	}

	header = headerChunk.GetFileMeta()
	chunkCount := int(headerChunk.GetChunks())
	chunkSize := int(headerChunk.GetChunkSize())
	total := int(header.GetSize())

	if chunkSize == 0 || chunkCount == 0 || total == 0 {
		return header, fmt.Errorf("%w: %v", ErrInvalidHeader, headerChunk)
	}

	return header, recvContents(src, dst, chunkCount, chunkSize, total)
}

func recvContents(
	src grpc.ServerStreamingClient[FileDataChunk],
	dst io.Writer,
	chunkCount int,
	chunkSize int,
	totalSize int,
) error {
	// receive and write all content chunks
	for i := 0; i < chunkCount; i++ {
		chunk, err := src.Recv()
		if err != nil {
			return fmt.Errorf("%w: content error %s", ErrFailedRead, err)
		}

		if err = validateRecvChunk(chunk, chunkSize, chunkCount-1, i); err != nil {
			return err
		}
		data := chunk.GetContent().GetData()
		if _, err = dst.Write(data); err != nil {
			return fmt.Errorf("%w: %s", ErrWrite, err)
		}
		totalSize -= len(data)
		if 0 > totalSize {
			return fmt.Errorf("%w: %d more data than expected",
				ErrUnexpectedContent, 0-totalSize)
		}
	}
	if totalSize > 0 {
		return fmt.Errorf("%w: unexpected end of content, %d left",
			ErrUnexpectedContent, totalSize)
	}

	return nil
}

func validateRecvChunk(chunk *FileDataChunk, chunkSize, lastChunkIndex, i int) error {
	content := chunk.GetContent()
	if content == nil {
		return fmt.Errorf("%w: no content", ErrUnexpectedContent)
	}
	if content.GetChunkId() != uint32(i) {
		return fmt.Errorf("%w: unexpected chunk id %d, expected %d",
			ErrUnexpectedContent, content.GetChunkId(), i)
	}
	data := content.GetData()
	if len(data) != chunkSize && i != lastChunkIndex {
		return fmt.Errorf("%w: content chunk size %d, expected %d",
			ErrUnexpectedContent, len(data), chunkSize)
	}

	return nil
}
