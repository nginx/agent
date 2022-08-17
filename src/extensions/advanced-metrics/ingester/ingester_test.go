package ingester

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/ingester/mocks"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/reader"
	readerMock "github.com/nginx/agent/v2/src/extensions/advanced-metrics/reader/mocks"
)

func TestIngesterRunCanProcessIncomingData(t *testing.T) {
	ctrl := gomock.NewController(t)

	message1 := []byte("m1")
	message2 := []byte("m2")

	frameChannel := make(chan reader.Frame)
	stagingTableMock := mocks.NewMockStagingTable(ctrl)
	frameMock := readerMock.NewMockFrame(ctrl)

	frameMock.EXPECT().Messages().Return([][]byte{message1, message2})
	frameMock.EXPECT().Release()
	stagingTableMock.EXPECT().Add(newMessageFieldIterator(message1)).Return(nil)
	stagingTableMock.EXPECT().Add(newMessageFieldIterator(message2)).Return(nil)

	ingester := NewIngester(frameChannel, stagingTableMock)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ingester.Run(ctx)
	}()

	frameChannel <- frameMock
	cancel()
	wg.Wait()
}

func TestIngesterRunIsNotStopingOnTableAddError(t *testing.T) {
	ctrl := gomock.NewController(t)

	message1 := []byte("m1")
	message2 := []byte("m2")

	frameChannel := make(chan reader.Frame)
	stagingTableMock := mocks.NewMockStagingTable(ctrl)
	frameMock := readerMock.NewMockFrame(ctrl)

	frameMock.EXPECT().Messages().Return([][]byte{message1, message2})
	frameMock.EXPECT().Release()
	stagingTableMock.EXPECT().Add(newMessageFieldIterator(message1)).Return(errors.New("dummy error"))
	stagingTableMock.EXPECT().Add(newMessageFieldIterator(message2)).Return(nil)

	ingester := NewIngester(frameChannel, stagingTableMock)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ingester.Run(ctx)
	}()

	frameChannel <- frameMock
	cancel()
	wg.Wait()
}

func TestIngesterRunStopProcessingOnChannelClose(t *testing.T) {
	ctrl := gomock.NewController(t)

	frameChannel := make(chan reader.Frame)
	stagingTableMock := mocks.NewMockStagingTable(ctrl)

	ingester := NewIngester(frameChannel, stagingTableMock)

	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ingester.Run(ctx)
	}()

	close(frameChannel)
	wg.Wait()
}
