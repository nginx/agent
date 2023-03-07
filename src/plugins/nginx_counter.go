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
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"

	"go.uber.org/atomic"
)

type NginxCounter struct {
	ctx            context.Context
	conf           *config.Config
	signalChannel  chan os.Signal
	nginxBinary    core.NginxBinary
	env            core.Environment
	lastUpdated    int64
	serverAddress  []string
	connected      *atomic.Bool
	socketListener net.Listener
	processMutex   sync.RWMutex
	nginxes        map[string]*proto.NginxDetails
}
type Payload struct {
	LastUpdated int64 `json:",string"`
}

func NewNginxCounter(conf *config.Config, nginxBinary core.NginxBinary, env core.Environment) *NginxCounter {
	return &NginxCounter{
		conf:          conf,
		signalChannel: make(chan os.Signal, 1),
		nginxBinary:   nginxBinary,
		env:           env,
		lastUpdated:   time.Now().Unix(),
		connected:     atomic.NewBool(false),
	}
}

func (nc *NginxCounter) Init(pipeline core.MessagePipeInterface) {
	log.Infof("NGINX Counter initializing %v", nc.conf.Nginx)
	nc.serverAddress = strings.Split(nc.conf.Nginx.NginxCountingSocket, ":")
	nc.ctx = pipeline.Context()
	if nc.serverAddress[0] == "unix" {
		if err := os.RemoveAll(nc.serverAddress[1]); err != nil {
			log.Warn("Can not parse server socket address, is the configuration correct?")
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		nc.agentServer(nc.serverAddress)
	}()
	wg.Wait()
}

func (nc *NginxCounter) agentServer(serverAddress []string) {
	var err error
	signal.Notify(nc.signalChannel, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)

	nc.socketListener, err = net.Listen(serverAddress[0], serverAddress[1])

	if err != nil {
		log.Warn("failed to start NGINX counter listener")
	}

	err = core.EnableWritePermissionForSocket(serverAddress[1])

	if err != nil {
		log.Warn("unable to set correct write permissions for NGINX counter socket")
	}

	if nc.socketListener != nil {
		go nc.recieveSocketMessage(nc.socketListener)
	}
}

func (nc *NginxCounter) recieveSocketMessage(socketListener net.Listener) {
	for {
		connectionResponse, err := socketListener.Accept()
		if err != nil {
			log.Warnf("Unable to accept from NGINX counter socket")
			break
		}

		go nc.handleResponse(connectionResponse)
	}
}

func (nc *NginxCounter) Close() {
	if nc.socketListener != nil {
		nc.socketListener.Close()
	}

	log.Info("NGINX Counter is wrapping up")
	if err := os.RemoveAll(nc.serverAddress[1]); err != nil {
		log.Warn("Error removing socket")
	}
}

func (nc *NginxCounter) Info() *core.Info {
	return core.NewInfo("NGINX Counter", "v0.0.1")
}

func (nc *NginxCounter) Process(msg *core.Message) {
	switch {
	case msg.Exact(core.AgentConnected):
		nc.connected.Toggle()
	case msg.Exact(core.NginxDetailProcUpdate):
		// get the processes from this payload
		nc.processMutex.Lock()
		processes := msg.Data().([]core.Process)
		nc.nginxBinary.UpdateNginxDetailsFromProcesses(processes)
		nc.nginxes = nc.nginxBinary.GetNginxDetailsMapFromProcesses(processes)
		defer nc.processMutex.Unlock()
	}
}

func (nc *NginxCounter) Subscriptions() []string {
	return []string{core.NginxDetailProcUpdate, core.AgentConnected}
}

func (nc *NginxCounter) handleResponse(connection net.Conn) {
	log.Trace("Receiving Server")
	for {
		buf := make([]byte, 512)
		responseBytes, err := connection.Read(buf)
		if err != nil {
			return
		}

		bufferedResponse := buf[0:responseBytes]
		log.Tracef("Socket Server handleResponse: %s", string(bufferedResponse))

		if nc.connected.Load() {

			nc.processMutex.RLock()
			nginxes := nc.nginxes
			nc.processMutex.RUnlock()

			log.Tracef("Reporting the following nginx instances: %v", nginxes)

			for _, nginx := range nginxes {
				if nginx.Plus.Enabled {
					nc.lastUpdated = time.Now().Unix()
					log.Debugf("lastUpdated %d nginxId: %s", nc.lastUpdated, nginx.NginxId)

					payload := Payload{LastUpdated: nc.lastUpdated}
					data, err := json.Marshal(payload)
					if err != nil {
						log.Warnf("err trying to unmarshal payload %v", err)
					}

					_, err = connection.Write(data)
					if err != nil {
						log.Warn("Could not write to NGINX counter socket")
					}
				}
			}
		}
	}
}
