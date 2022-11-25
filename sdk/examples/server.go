/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package main

import (
	"embed"
	"encoding/json"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/examples/services"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	sdkProto "github.com/nginx/agent/sdk/v2/proto"

	"google.golang.org/grpc"
)

const (
	HTTP_ADDR = ":54790"
	GRPC_ADDR = ":54789"
	PROTOCOL  = "tcp"
)

//go:embed index.html
var content embed.FS

func createListener(address string) (listener net.Listener, close func() error) {
	listen, err := net.Listen(PROTOCOL, address)
	if err != nil {
		panic(err)
	}
	return listen, listen.Close
}

func main() {
	httpListener, httpClose := createListener(HTTP_ADDR)
	defer httpClose()

	grpcListener, grpcClose := createListener(GRPC_ADDR)
	defer grpcClose()

	srvOptions := sdkGRPC.DefaultServerDialOptions
	grpcServer := grpc.NewServer(srvOptions...)

	metricsService := services.NewMetricsService()
	sdkProto.RegisterMetricsServiceServer(grpcServer, metricsService)

	commandService := services.NewCommandService()
	sdkProto.RegisterCommanderServer(grpcServer, commandService)

	//Serve gRPC Server
	log.Println("http listening")
	log.Println("grpc listening")

	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatal("error starting server")
		}
	}()

	http.Handle("/", http.FileServer(http.FS(content)))

	http.Handle("/registered", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		payload, err := json.Marshal(commandService.GetRegistration())
		if err != nil {
			log.Warnf("%v", err)
			return
		}
		rw.Write(payload)
	}))

	http.Handle("/nginxes", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		payload, err := json.Marshal(commandService.GetNginxes())
		if err != nil {
			log.Warnf("%v", err)
			return
		}
		rw.Write(payload)
	}))

	http.Handle("/configs/chunked", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		payload, err := json.Marshal(commandService.GetChunks())
		if err != nil {
			log.Warnf("%v", err)
			return
		}
		rw.Write(payload)
	}))

	http.Handle("/configs/raw", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		confFiles, auxFiles := commandService.GetContents()
		response := map[string]interface{}{}
		for _, confFile := range confFiles {
			response[confFile.GetName()] = string(confFile.GetContents())
		}

		for _, aux := range auxFiles {
			response[aux.GetName()] = string(aux.GetContents())
		}

		payload, err := json.MarshalIndent(response, "", "\t")
		if err != nil {
			log.Warnf("%v", err)
			return
		}
		rw.Write(payload)
	}))

	http.Handle("/configs", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		payload, err := json.Marshal(commandService.GetConfigs())
		if err != nil {
			log.Warnf("%v", err)
			return
		}
		rw.Write(payload)
	}))

	http.Handle("/metrics", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		payload, err := json.Marshal(metricsService.GetMetrics())
		if err != nil {
			log.Warnf("%v", err)
			return
		}
		rw.Write(payload)
	}))

	http.Serve(httpListener, nil)
}
