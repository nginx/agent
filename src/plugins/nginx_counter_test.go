/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestNginxCounter(t *testing.T) {
	tests := []struct {
		name            string
		detailsMap      map[string]*proto.NginxDetails
		processes       []core.Process
		socket          string
		expectedPayload Payload
	}{
		{
			name: "positive test case",
			detailsMap: map[string]*proto.NginxDetails{
				"12345": {
					NginxId:     "12345",
					ProcessPath: "/path/to/nginx",
					Plus: &proto.NginxPlusMetaData{
						Enabled: true,
					},
				},
			},
			processes: []core.Process{
				{
					Name:     "12345",
					IsMaster: true,
				},
			},
			socket:          fmt.Sprintf("unix:%s/nginx.sock", t.TempDir()),
			expectedPayload: Payload{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running %s", tt.name)
			var wg sync.WaitGroup
			wg.Add(1)

			testChannel := make(chan Payload)

			binary := tutils.NewMockNginxBinary()
			binary.On("UpdateNginxDetailsFromProcesses", mock.Anything)
			binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return((tt.detailsMap))

			config := &config.Config{
				Nginx: config.Nginx{
					NginxCountingSocket: tt.socket,
				},
			}

			nginxCounter := NewNginxCounter(config, binary, tutils.NewMockEnvironment())

			messagePipe := core.NewMockMessagePipe(context.Background())

			err := messagePipe.Register(1, []core.Plugin{nginxCounter}, []core.ExtensionPlugin{})
			assert.NoError(t, err)
			messagePipe.Run()

			nginxCounter.Process(core.NewMessage(core.NginxDetailProcUpdate, tt.processes))
			nginxCounter.Process(core.NewMessage(core.AgentConnected, nil))

			go func() {
				defer wg.Done()
				socketClient(t, testChannel, tt.socket)
			}()

			result := <-testChannel
			assert.NotEqual(t, result, tt.expectedPayload)

			wg.Wait()
		})
	}
}

func socketClient(t *testing.T, testChannel chan<- Payload, address string) {
	serverAddress := strings.Split(address, ":")
	connection, err := net.Dial(serverAddress[0], serverAddress[1])
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	go socketClientReader(t, testChannel, connection)

	code, err := connection.Write([]byte("hi"))
	if err != nil {
		t.Errorf("write failed: code: %d error: %v", code, err)
	}
	time.Sleep(1e9)
}

func socketClientReader(t *testing.T, testChannel chan<- Payload, r io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf[:])
		if err != nil {
			t.Errorf("error reading buffer %v", err)
		}
		var payload Payload
		err = json.Unmarshal(buf[0:n], &payload)
		if err != nil {
			t.Errorf("error unmarshalling json %v", err)
		}

		if payload.LastUpdated != 0 {
			testChannel <- payload
			break
		}
	}
}
