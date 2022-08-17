package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/reader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const frameSeparator = ';'

func TestReaderIsAbleToHandleMultipleClients(t *testing.T) {
	mu := &sync.Mutex{}
	rand.Seed(time.Now().Unix())
	addr := fmt.Sprintf("/tmp/advanced_metrics_reader_test_%d.sr", rand.Int63())

	r := reader.NewReader(addr)
	outChannel := r.OutChannel()
	ctx, cancel := context.WithCancel(context.Background())

	dataToSend := [][]byte{
		[]byte("frame1"),
		[]byte("frame2"),
		[]byte("frame3"),
		[]byte("frame4"),
		[]byte("frame5"),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		err := r.Run(ctx)
		assert.NoError(t, err)
	}()

	workersCout := 10
	wg.Add(workersCout)
	for i := 0; i < workersCout; i++ {
		go func() {
			defer wg.Done()
			assertMessageSent(t, addr, dataToSend)
		}()
	}

	receivedMessages := make([][]byte, 0)
	assert.Eventually(t, func() bool {
	r_loop:
		for {
			select {
			case f := <-outChannel:
				receivedMessages = append(receivedMessages, f.Messages()...)
			default:
				break r_loop
			}
		}
		return len(dataToSend)*workersCout == len(receivedMessages)
	}, time.Second, time.Microsecond*10)

	expectedData := make([][]byte, 0)
	for i := 0; i < workersCout; i++ {
		expectedData = append(expectedData, dataToSend...)
	}

	assert.ElementsMatch(t, receivedMessages, expectedData)

	cancel()
	wg.Wait()
}

func TestReaderIsAbleToHandlePartiallySendFrame(t *testing.T) {
	mu := &sync.Mutex{}
	rand.Seed(time.Now().Unix())
	addr := fmt.Sprintf("/tmp/advanced_metrics_reader_test_%d.sr", rand.Int63())

	r := reader.NewReader(addr)
	outChannel := r.OutChannel()
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		err := r.Run(ctx)
		assert.NoError(t, err)
	}()

	var conn net.Conn
	require.Eventually(t, func() bool {
		c, err := net.Dial("unix", addr)
		conn = c
		return err == nil
	}, time.Second, time.Millisecond*100)

	expectedData := [][]byte{
		[]byte("frame1"),
		[]byte("frame2"),
	}
	data := fmt.Sprintf("%s%c%s%cframe3", expectedData[0], frameSeparator, expectedData[1], frameSeparator)
	n, err := conn.Write([]byte(data))
	assert.NoError(t, err)
	assert.Equal(t, n, len(data))
	err = conn.Close()
	assert.NoError(t, err)

	receivedMessages := make([][]byte, 0)
	f := <-outChannel
	receivedMessages = append(receivedMessages, f.Messages()...)
	f.Release()

	assert.ElementsMatch(t, receivedMessages, expectedData)

	cancel()
	wg.Wait()
}

func TestReaderIsAbleToCloseOngoingConnections(t *testing.T) {
	mu := &sync.Mutex{}
	rand.Seed(time.Now().Unix())
	addr := fmt.Sprintf("/tmp/advanced_metrics_reader_test_%d.sr", rand.Int63())

	r := reader.NewReader(addr)
	outChannel := r.OutChannel()
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := r.Run(ctx)
		mu.Lock()
		defer mu.Unlock()
		assert.NoError(t, err)
	}()

	var conn net.Conn
	require.Eventually(t, func() bool {
		c, err := net.Dial("unix", addr)
		conn = c
		return err == nil
	}, time.Second, time.Millisecond*100)

	expectedData := [][]byte{
		[]byte("frame1"),
		[]byte("frame2"),
	}
	data := fmt.Sprintf("%s%c%s%cframe3", expectedData[0], frameSeparator, expectedData[1], frameSeparator)
	n, err := conn.Write([]byte(data))
	assert.NoError(t, err)
	assert.Equal(t, n, len(data))

	receivedMessages := make([][]byte, 0)
	f := <-outChannel
	receivedMessages = append(receivedMessages, f.Messages()...)
	f.Release()

	assert.ElementsMatch(t, receivedMessages, expectedData)

	cancel()
	wg.Wait()

	_, err = conn.Write([]byte{})
	assert.Error(t, err)
}

func TestReaderWithGeneratedData(t *testing.T) {
	mu := &sync.Mutex{}
	rand.Seed(time.Now().Unix())
	addr := fmt.Sprintf("/tmp/advanced_metrics_reader_test_%d.sr", rand.Int63())

	r := reader.NewReader(addr)
	outChannel := r.OutChannel()
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := r.Run(ctx)
		mu.Lock()
		defer mu.Unlock()
		assert.NoError(t, err)
	}()

	var conn net.Conn
	require.Eventually(t, func() bool {
		c, err := net.Dial("unix", addr)
		conn = c
		return err == nil
	}, time.Second, time.Millisecond*100)

	go func() {
		mu.Lock()
		defer mu.Unlock()
		dataByte := []byte(generatedData)
		n, err := io.Copy(conn, bytes.NewBuffer(dataByte))
		assert.NoError(t, err)
		assert.Equal(t, int(n), len(generatedData))
		err = conn.Close()
		assert.NoError(t, err)
	}()

	messagesCount := strings.Count(generatedData, ";")
	receivedMessagesCount := 0

	for f := range outChannel {
		receivedMessagesCount += len(f.Messages())
		f.Release()
		if receivedMessagesCount >= messagesCount {
			break
		}
	}

	assert.Equal(t, receivedMessagesCount, messagesCount)

	cancel()
	wg.Wait()
}

func assertMessageSent(t *testing.T, addr string, dataToSend [][]byte) {
	var conn net.Conn
	require.Eventually(t, func() bool {
		c, err := net.Dial("unix", addr)
		conn = c
		return err == nil
	}, time.Second, time.Millisecond*100)

	for _, data := range dataToSend {
		n, err := conn.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, n, len(data))

		n, err = conn.Write([]byte{frameSeparator})
		assert.NoError(t, err)
		assert.Equal(t, n, 1)
	}

	conn.Close()
}
