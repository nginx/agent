/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	prometheus_metrics "github.com/nginx/agent/v2/src/extensions/prometheus-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type AgentAPI struct {
	config       *config.Config
	env          core.Environment
	server       http.Server
	nginxBinary  core.NginxBinary
	nginxHandler *NginxHandler
	exporter     *prometheus_metrics.Exporter
}

type NginxHandler struct {
	env         core.Environment
	nginxBinary core.NginxBinary
}

const (
	contentTypeHeader = "Content-Type"
	jsonMimeType      = "application/json"
)

var (
	instancesRegex = regexp.MustCompile(`^\/nginx[\/]*$`)
)

func NewAgentAPI(config *config.Config, env core.Environment, nginxBinary core.NginxBinary) *AgentAPI {
	return &AgentAPI{config: config, env: env, nginxBinary: nginxBinary, exporter: prometheus_metrics.NewExporter(&proto.MetricsReport{})}
}

func (a *AgentAPI) Init(core.MessagePipeInterface) {
	log.Info("Agent API initializing")
	go a.createHttpServer()
}

func (a *AgentAPI) Close() {
	log.Info("Agent API is wrapping up")
	if err := a.server.Shutdown(context.Background()); err != nil {
		log.Errorf("Agent API HTTP Server Shutdown Error: %v", err)
	}
}

func (a *AgentAPI) Process(message *core.Message) {
	log.Tracef("Process function in the agent_api.go, %s %v", message.Topic(), message.Data())
	switch {
	case message.Exact(core.MetricReport):
		metricReport, ok := message.Data().(*proto.MetricsReport)
		if !ok {
			log.Warnf("Invalid message received, %T, for topic, %s", message.Data(), message.Topic())
			return
		}
		a.exporter.SetLatestMetricReport(metricReport)
		return
	}
}
func (a *AgentAPI) Info() *core.Info {
	return core.NewInfo("Agent API Plugin", "v0.0.1")
}

func (a *AgentAPI) Subscriptions() []string {
	return []string{core.MetricReport}
}

func (a *AgentAPI) createHttpServer() {
	mux := http.NewServeMux()
	a.nginxHandler = &NginxHandler{a.env, a.nginxBinary}
	registerer := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer

	registerer.MustRegister(a.exporter)
	mux.Handle("/metrics/", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))

	mux.Handle("/nginx/", a.nginxHandler)
    
	a.server = http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.AgentAPI.Port),
		Handler: mux,
	}

	if a.config.AgentAPI.Cert != "" && a.config.AgentAPI.Key != "" && a.config.AgentAPI.Port != 0 {
		log.Info("Starting Agent API HTTP server with cert and key and port from config")
		if err := a.server.ListenAndServeTLS(a.config.AgentAPI.Cert, a.config.AgentAPI.Key); err != http.ErrServerClosed {
			log.Fatalf("error listening to port: %v", err)
		}
	} else if a.config.AgentAPI.Port != 0 {
		log.Info("Starting Agent API HTTP server with port from config and TLS disabled")
		if err := a.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("error listening to port: %v", err)
		}
	} else {
		log.Info("Agent API not started")
	}
}

func (h *NginxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeHeader, jsonMimeType)
	switch {
	case r.Method == http.MethodGet && instancesRegex.MatchString(r.URL.Path):
		err := sendInstanceDetailsPayload(h.getNginxDetails(), w, r)
		if err != nil {
			log.Warnf("Failed to send instance details payload: %v", err)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		_, err := fmt.Fprint(w, []byte("not found"))
		if err != nil {
			log.Warnf("Failed to send api response: %v", err)
		}
	}
}

func sendInstanceDetailsPayload(nginxDetails []*proto.NginxDetails, w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)

	if len(nginxDetails) == 0 {
		log.Debug("No nginx instances found")
		_, err := fmt.Fprint(w, "[]")
		if err != nil {
			return fmt.Errorf("failed to send payload: %v", err)
		}

		return nil
	}

	respBody := new(bytes.Buffer)
	err := json.NewEncoder(respBody).Encode(nginxDetails)
	if err != nil {
		return fmt.Errorf("failed to encode payload: %v", err)
	}

	_, err = fmt.Fprint(w, respBody)
	if err != nil {
		return fmt.Errorf("failed to send payload: %v", err)
	}

	return nil
}

func (h *NginxHandler) getNginxDetails() []*proto.NginxDetails {
	var nginxDetails []*proto.NginxDetails

	for _, proc := range h.env.Processes() {
		if proc.IsMaster {
			nginxDetails = append(nginxDetails, h.nginxBinary.GetNginxDetailsFromProcess(proc))
		}
	}
	return nginxDetails
}
