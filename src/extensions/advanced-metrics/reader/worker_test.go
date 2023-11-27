/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package reader

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/reader/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestWorkerStopAndCloseConnectionOnContexCancelation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	connMock := mocks.NewMockConn(ctrl)
	connMock.EXPECT().Close().Return(nil)
	connMock.EXPECT().Read(gomock.Any()).Do(func(interface{}) {
		cancel()
	}).Return(0, net.ErrClosed)

	outChannel := make(chan Frame)
	worker := newWorker(connMock, outChannel)

	err := worker.Run(ctx)
	assert.NoError(t, err)
}

func TestWorkerStopAndCloseConnectionOnReadError(t *testing.T) {
	const error = "error"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	connMock := mocks.NewMockConn(ctrl)
	connMock.EXPECT().Close().Return(nil)
	connMock.EXPECT().Read(gomock.Any()).Return(0, errors.New(error))

	outChannel := make(chan Frame)
	worker := newWorker(connMock, outChannel)

	err := worker.Run(ctx)
	assert.EqualError(t, err, fmt.Sprintf("fail to read data: %s", error))
}

func TestWorkerFrameProcessing(t *testing.T) {
	tests := []struct {
		name               string
		data               [][]byte
		expectedMessages   [][]byte
		receiverBufferSize int
	}{
		{
			name: "full frame",
			data: [][]byte{
				[]byte("data;"),
			},
			expectedMessages: [][]byte{
				[]byte("data"),
			},
		},
		{
			name: "multiple full frames",
			data: [][]byte{
				[]byte("data;data2;"),
				[]byte("data3;data4;"),
				[]byte("data5;data6;data7;"),
			},
			expectedMessages: [][]byte{
				[]byte("data"),
				[]byte("data2"),
				[]byte("data3"),
				[]byte("data4"),
				[]byte("data5"),
				[]byte("data6"),
				[]byte("data7"),
			},
		},
		{
			name: "partial single frame",
			data: [][]byte{
				[]byte("da"),
				[]byte("t"),
				[]byte("a;"),
			},
			expectedMessages: [][]byte{
				[]byte("data"),
			},
		},
		{
			name: "partial multiple frames",
			data: [][]byte{
				[]byte("da"),
				[]byte("t"),
				[]byte("a;"),
				[]byte("da"),
				[]byte("t"),
				[]byte("a2;"),
				[]byte("da"),
				[]byte("t"),
				[]byte(""),
				[]byte("a3;"),
				[]byte("da"),
				[]byte("t"),
				[]byte("a4"),
				[]byte(";data5;data6;d"),
				[]byte(";data7;data8;datadatdatadatdatadatdatadata"),
				[]byte("datadatdatadatdatadatdatadata;"),
			},
			expectedMessages: [][]byte{
				[]byte("data"),
				[]byte("data2"),
				[]byte("data3"),
				[]byte("data4"),
				[]byte("data5"),
				[]byte("data6"),
				[]byte("d"),
				[]byte("data7"),
				[]byte("data8"),
				[]byte("datadatdatadatdatadatdatadatadatadatdatadatdatadatdatadata"),
			},
		},
		{
			name: "partial multiple frames, use small receiver buffer",
			data: [][]byte{
				[]byte(";"),
				[]byte("da"),
				[]byte("t"),
				[]byte("a;"),
				[]byte("da"),
				[]byte("t"),
				[]byte("a2;"),
				[]byte("da"),
				[]byte("t"),
				[]byte(""),
				[]byte("a3;"),
				[]byte("da"),
				[]byte("t"),
				[]byte("a4"),
				[]byte(";"),
			},
			expectedMessages: [][]byte{
				[]byte(""),
				[]byte("data"),
				[]byte("data2"),
				[]byte("data3"),
				[]byte("data4"),
			},
			receiverBufferSize: 6,
		},
		{
			name: "partial multiple frames, start with partial",
			data: [][]byte{
				[]byte(";"),
				[]byte("data"),
				[]byte(";data2"),
				[]byte(";data3"),
				[]byte(";;d;"),
			},
			expectedMessages: [][]byte{
				[]byte(""),
				[]byte("data"),
				[]byte("data2"),
				[]byte("data3"),
				[]byte(""),
				[]byte("d"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx, cancel := context.WithCancel(context.Background())

			connMock := mocks.NewMockConn(ctrl)
			connMock.EXPECT().Close().Return(nil)

			for _, data := range test.data {
				data := data
				connMock.EXPECT().Read(gomock.Any()).Return(len(data), nil).Do(func(buf []byte) {
					assert.LessOrEqual(t, len(data), len(buf))
					assert.Equal(t, len(data), copy(buf, data))
				})
			}

			connMock.EXPECT().Read(gomock.Any()).Return(0, net.ErrClosed).Do(func(buf []byte) {
				<-ctx.Done()
			})

			outChannel := make(chan Frame)
			worker := newWorker(connMock, outChannel)
			if test.receiverBufferSize != 0 {
				worker.maxBufferSize = test.receiverBufferSize
			}

			done := make(chan struct{})
			go func() {
				assert.NoError(t, worker.Run(ctx))
				done <- struct{}{}
			}()

			receivedData := receiveMessages(t, outChannel, len(test.expectedMessages))
			assert.Equal(t, test.expectedMessages, receivedData)

			cancel()
			<-done
		})
	}
}

func TestWorkerFrameProcessingErrorWhenMessageExceededBufferSize(t *testing.T) {
	const maxBufferSize = 12

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	connMock := mocks.NewMockConn(ctrl)
	connMock.EXPECT().Close().Return(nil)

	connMock.EXPECT().Read(gomock.Any()).Return(maxBufferSize/2, nil)
	connMock.EXPECT().Read(gomock.Any()).Return(maxBufferSize/2, nil)

	outChannel := make(chan Frame)
	worker := newWorker(connMock, outChannel)
	worker.maxBufferSize = maxBufferSize

	assert.EqualError(t, worker.Run(ctx), fmt.Sprintf("fail to process frames, data exceeded buffer size: %d", maxBufferSize))
}

func receiveMessages(t *testing.T, outChannel chan Frame, epxpectedSize int) [][]byte {
	receivedMessages := 0
	receivedData := make([][]byte, 0)

	assert.Eventually(t, func() bool {
		frame, ok := <-outChannel
		assert.True(t, ok)
		for _, message := range frame.Messages() {
			m := make([]byte, len(message))
			copy(m, message)
			receivedData = append(receivedData, m)
		}
		receivedMessages += len(frame.Messages())
		frame.Release()

		return receivedMessages == epxpectedSize
	}, time.Second, time.Millisecond*10)
	return receivedData
}

func BenchmarkWorkerFrameProcessing(b *testing.B) {
	logrus.SetLevel(logrus.FatalLevel)
	tests := map[string]struct {
		framePosition int
		readSize      int
	}{
		"single frame, no partial, small buff": {
			framePosition: 8,
			readSize:      9,
		},
		"single frame, no partial": {
			framePosition: maxWorkerBufferSize/4 - 1,
			readSize:      maxWorkerBufferSize / 4,
		},
		"single frame, with partial": {
			framePosition: maxWorkerBufferSize/4 - 100,
			readSize:      maxWorkerBufferSize / 4,
		},
		"single frame, with big partial": {
			framePosition: 0,
			readSize:      maxWorkerBufferSize / 3,
		},
		"single frame, full buffer no partial": {
			framePosition: maxWorkerBufferSize - 1,
			readSize:      maxWorkerBufferSize,
		},
		"single frame, full buffer partial": {
			framePosition: 0,
			readSize:      maxWorkerBufferSize / 2,
		},
	}

	for name, test := range tests {
		b.Run(name, func(b *testing.B) {
			outChannel := make(chan Frame)
			stub := &mocks.ConnStub{
				Repeats:       b.N,
				ReadBytes:     test.readSize,
				FramePosition: test.framePosition,
				Separator:     frameSeparatorByte,
			}
			worker := newWorker(stub, outChannel)

			ctx := context.Background()
			go func() {
				err := worker.Run(ctx)
				if err != nil {
					b.Fail()
				}
			}()

			for i := 0; i < b.N; i++ {
				f := <-outChannel
				f.Release()
			}
		})
	}
}
