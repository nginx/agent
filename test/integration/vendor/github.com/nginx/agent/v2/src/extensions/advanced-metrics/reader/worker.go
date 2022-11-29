package reader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// This value is derived from the hardcoded max buffer size in module metrics
// #define MAX_SEND_BUF_SIZE (1024 * 64)
const maxWorkerBufferSize = 1024 * 64

type worker struct {
	conn         net.Conn
	frameChannel chan Frame

	maxBufferSize int
}

func newWorker(conn net.Conn, frameChannel chan Frame) *worker {
	return &worker{
		conn:         conn,
		frameChannel: frameChannel,

		maxBufferSize: maxWorkerBufferSize,
	}
}

func (w *worker) Run(ctx context.Context) error {
	acceptLoopGroup, ctx := errgroup.WithContext(ctx)
	acceptLoopGroup.Go(func() error {
		return w.readLoop(ctx)
	})
	acceptLoopGroup.Go(func() error {
		return w.closeConnection(ctx)
	})

	return acceptLoopGroup.Wait()
}

func (w *worker) closeConnection(ctx context.Context) error {
	<-ctx.Done()
	err := w.conn.Close()
	if err != nil {
		return fmt.Errorf("fail to close connection: %w", err)
	}

	return nil
}

func (w *worker) readLoop(ctx context.Context) error {
	bufferPool := sync.Pool{}
	bufferPool.New = func() interface{} {
		return NewFixedSizeBuffer(w.maxBufferSize)
	}
	releaseFunction := func(buffer *fixedSizeBuffer) {
		bufferPool.Put(buffer)
	}

	buffer := bufferPool.Get().(*fixedSizeBuffer)
	for {
		if buffer.size >= w.maxBufferSize {
			return fmt.Errorf("fail to process frames, data exceeded buffer size: %d", w.maxBufferSize)
		}

		err := buffer.readFrom(w.conn)
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			if buffer.size > 0 {
				log.Warn("Connection was closed when unterminated frame is still in buffer. Part of the data will be dropped.")
			}
			log.Info("Connection was gracefully closed")
			return nil
		}
		if err != nil {
			if buffer.size > 0 {
				log.Warn("Connection error. Unterminated frame is still in buffer. Part of the data will be dropped.")
			}
			return fmt.Errorf("fail to read data: %w", err)
		}

		frameSize := frameSize(buffer)
		if frameSize == -1 {
			continue
		}

		tmpBuffer := bufferPool.Get().(*fixedSizeBuffer)
		partial := partialMessage(buffer, frameSize)
		appended := tmpBuffer.append(partial)

		if appended < len(partial) {
			return fmt.Errorf("fail to process frames, data exceeded buffer size: %d", w.maxBufferSize)
		}

		frame := &frame{
			buffer:    buffer,
			frameSize: frameSize,
			release:   releaseFunction,
		}
		select {
		case w.frameChannel <- frame:
		case <-ctx.Done():
		}
		buffer = tmpBuffer
	}
}

func frameSize(buffer *fixedSizeBuffer) int {
	index := bytes.LastIndexByte(buffer.get(), frameSeparatorByte)
	if index == -1 {
		return index
	}
	return index + 1
}

func partialMessage(buffer *fixedSizeBuffer, frameSize int) []byte {
	return buffer.get()[frameSize:]
}
