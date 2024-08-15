/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Host: localhost:8081
// swagger:meta
package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2"
	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	prometheus_metrics "github.com/nginx/agent/v2/src/extensions/prometheus-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

// swagger:response MetricsResponse
// in: body
type _ string

const (
	okStatus      = "OK"
	pendingStatus = "PENDING"
	errorStatus   = "ERROR"
	unknownStatus = "UNKNOWN"

	registration      string = "registration"
	commandConnection string = "commandConnection"
	metricsConnection string = "metricsConnection"
)

var (
	healthRegex       = regexp.MustCompile(`^\/health[\/]*$`)
	instancesRegex    = regexp.MustCompile(`^\/nginx[\/]*$`)
	configRegex       = regexp.MustCompile(`^\/nginx/config[\/]*$`)
	configStatusRegex = regexp.MustCompile(`^\/nginx/config/status[\/]*$`)
)

type AgentAPI struct {
	config       *config.Config
	env          core.Environment
	pipeline     core.MessagePipeInterface
	server       http.Server
	nginxBinary  core.NginxBinary
	nginxHandler *NginxHandler
	rootHandler  *RootHandler
	exporter     *prometheus_metrics.Exporter
	processes    []*core.Process
}

type RootHandler struct {
	config               *config.Config
	isGrpcRegistered     bool
	lastCommandSent      time.Time
	lastMetricReportSent time.Time
	startTime            time.Time
}

type NginxHandler struct {
	config                 *config.Config
	env                    core.Environment
	pipeline               core.MessagePipeInterface
	nginxBinary            core.NginxBinary
	responseChannel        chan *proto.Command_NginxConfigResponse
	configResponseStatuses map[string]*proto.NginxConfigStatus
	processesMutex         sync.RWMutex
	processes              []*core.Process
}

// swagger:parameters apply-nginx-config
type ParameterRequest struct {
	// in: formData
	// swagger:file
	File interface{} `json:"file"`
}

type AgentAPIConfigApplyRequest struct {
	correlationId string
	config        *proto.NginxConfig
}

// swagger:model NginxInstanceResponse
type NginxInstanceResponse struct {
	// NGINX ID
	// example: b636d4376dea15405589692d3c5d3869ff3a9b26b0e7bb4bb1aa7e658ace1437
	NginxId string `json:"nginx_id"`
	// Message
	// example: config applied successfully
	Message string `json:"message"`
	// Status
	// example: OK
	Status string `json:"status"`
}

// swagger:model AgentAPIConfigApplyResponse
type AgentAPIConfigApplyResponse struct {
	// Correlation ID
	// example: 6204037c-30e6-408b-8aaa-dd8219860b4b
	CorrelationId string `json:"correlation_id"`
	// NGINX Instances
	NginxInstances []NginxInstanceResponse `json:"nginx_instances"`
}

// swagger:model AgentAPICommonResponse
type AgentAPICommonResponse struct {
	// Correlation ID
	// example: 6204037c-30e6-408b-8aaa-dd8219860b4b
	CorrelationId string `json:"correlation_id"`
	// Message
	// example: No NGINX instances found
	Message string `json:"message"`
}

// swagger:model AgentAPIConfigApplyStatusResponse
type AgentAPIConfigApplyStatusResponse struct {
	// Correlation ID
	// example: 6204037c-30e6-408b-8aaa-dd8219860b4b
	CorrelationId string `json:"correlation_id"`
	// Message
	// example: pending config apply
	Message string `json:"message"`
	// Status
	// example: PENDING
	Status string `json:"status"`
}

// swagger:model HealthStatusCheck
type HealthStatusCheck struct {
	// Health check name
	// example: commandConnection
	Name string `json:"name"`
	// Health check status
	// example: OK
	Status string `json:"status"`
}

// swagger:model HealthResponse
type HealthResponse struct {
	// Overall health status
	// example: OK
	Status string `json:"status"`
	// Array of health checks
	Checks []HealthStatusCheck `json:"checks"`
}

const (
	contentTypeHeader = "Content-Type"
	jsonMimeType      = "application/json"
)

func NewAgentAPI(config *config.Config, env core.Environment, nginxBinary core.NginxBinary, processes []*core.Process) *AgentAPI {
	return &AgentAPI{
		config:      config,
		env:         env,
		nginxBinary: nginxBinary,
		exporter:    prometheus_metrics.NewExporter(&proto.MetricsReport{}),
		processes:   processes,
	}
}

func (a *AgentAPI) Init(pipeline core.MessagePipeInterface) {
	log.Info("Agent API initializing")
	a.pipeline = pipeline
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

	switch message.Topic() {
	case core.AgentAPIConfigApplyResponse:
		switch response := message.Data().(type) {
		case *proto.Command_NginxConfigResponse:
			a.nginxHandler.responseChannel <- response
		default:
			log.Warnf("Unknown Command_NginxConfigResponse type: %T(%v)", message.Data(), message.Data())
		}
	case core.CommMetrics:
		switch response := message.Data().(type) {
		case *metrics.MetricsReportBundle:
			a.exporter.SetLatestMetricReport(response)
		default:
			log.Warnf("Unknown MetricReportBundle type: %T(%v)", message.Data(), message.Data())
		}
	case core.NginxConfigValidationPending, core.NginxConfigApplyFailed, core.NginxConfigApplySucceeded:
		switch response := message.Data().(type) {
		case *proto.AgentActivityStatus:
			nginxConfigStatus := response.GetNginxConfigStatus()
			a.nginxHandler.configResponseStatuses[nginxConfigStatus.GetNginxId()] = nginxConfigStatus
		default:
			log.Errorf("Expected the type %T but got %T", &proto.AgentActivityStatus{}, response)
		}
	case core.NginxDetailProcUpdate:
		a.processes = message.Data().([]*core.Process)
		if a.nginxHandler != nil {
			a.nginxHandler.syncProcessInfo(a.processes)
		}
	case core.AgentConnected:
		a.rootHandler.isGrpcRegistered = true
	case core.CommandSent:
		a.rootHandler.lastCommandSent = time.Now()
	case core.MetricReportSent:
		a.rootHandler.lastMetricReportSent = time.Now()
	}
}

func (a *AgentAPI) Info() *core.Info {
	return core.NewInfo(agent_config.FeatureAgentAPI, "v0.0.1")
}

func (a *AgentAPI) Subscriptions() []string {
	return []string{
		core.AgentAPIConfigApplyResponse,
		core.NginxConfigValidationPending,
		core.NginxConfigApplyFailed,
		core.NginxConfigApplySucceeded,
		core.NginxDetailProcUpdate,
		core.AgentConnected,
		core.CommandSent,
		core.MetricReportSent,
		core.CommMetrics,
	}
}

func (a *AgentAPI) createHttpServer() {
	a.rootHandler = &RootHandler{
		config:           a.config,
		isGrpcRegistered: false,
		startTime:        time.Now(),
	}

	a.nginxHandler = &NginxHandler{
		config:                 a.config,
		pipeline:               a.pipeline,
		env:                    a.env,
		nginxBinary:            a.nginxBinary,
		responseChannel:        make(chan *proto.Command_NginxConfigResponse),
		configResponseStatuses: make(map[string]*proto.NginxConfigStatus),
		processes:              a.processes,
	}

	mux := http.NewServeMux()

	mux.Handle("/metrics/", a.getPrometheusHandler())
	mux.Handle("/nginx/", a.nginxHandler)
	mux.Handle("/", a.rootHandler)

	handler := cors.New(cors.Options{AllowedMethods: []string{"OPTIONS", "GET", "PUT"}}).Handler(mux)
	a.server = http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.AgentAPI.Host, a.config.AgentAPI.Port),
		Handler: handler,
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

// swagger:route GET /metrics/ nginx-agent get-prometheus-metrics
//
// # Get Prometheus Metrics
//
// # Returns prometheus metrics
//
// Produces:
//   - text/plain
//
// responses:
//
//	200: MetricsResponse
func (a *AgentAPI) getPrometheusHandler() http.Handler {
	registerer := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer

	registerer.MustRegister(a.exporter)
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
}

func (h *NginxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeHeader, jsonMimeType)

	switch {
	case instancesRegex.MatchString(r.URL.Path):
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := h.sendInstanceDetailsPayload(w, r)
		if err != nil {
			log.Warnf("Failed to send instance details payload: %v", err)
		}
	case configRegex.MatchString(r.URL.Path):
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if h.config.IsFeatureEnabled(agent_config.FeatureNginxConfig) || h.config.IsFeatureEnabled(agent_config.FeatureNginxConfigAsync) {
			err := h.updateConfig(w, r)
			if err != nil {
				log.Warnf("Failed to update config: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			response := AgentAPIConfigApplyStatusResponse{
				CorrelationId: uuid.New().String(),
				Message:       "unable to process NGINX config apply request as the nginx-config-async feature is disabled",
				Status:        errorStatus,
			}
			err := writeObjectToResponseBody(w, response)
			if err != nil {
				log.Warn(err)
			}
			log.Warn("Config Apply Feature Disabled")
		}

	case configStatusRegex.MatchString(r.URL.Path):
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := h.getConfigStatus(w, r)
		if err != nil {
			log.Warnf("Failed to get config status: %v", err)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		_, err := fmt.Fprint(w, []byte("not found"))
		if err != nil {
			log.Warnf("Failed to send api response: %v", err)
		}
	}
}

// swagger:route GET /nginx/ nginx-agent get-nginx-instances
//
// # Get NGINX Instances
//
// # Returns a list of NGINX instances
//
// responses:
//
//	200: []NginxDetails
//	500
func (h *NginxHandler) sendInstanceDetailsPayload(w http.ResponseWriter, _ *http.Request) error {
	nginxDetails := h.getNginxDetails()
	w.WriteHeader(http.StatusOK)

	if len(nginxDetails) == 0 {
		log.Debug("No nginx instances found")
		_, err := fmt.Fprint(w, "[]")
		if err != nil {
			return fmt.Errorf("failed to send payload: %v", err)
		}

		return nil
	}

	return writeObjectToResponseBody(w, nginxDetails)
}

// swagger:route PUT /nginx/config/ nginx-agent apply-nginx-config
//
// # Apply NGINX configuration to all NGINX instances
//
// # Returns a config apply status
// Consumes:
//   - multipart/form-data
//
// Produces:
//   - application/json
//
// responses:
//
//	200: AgentAPIConfigApplyResponse
//	400: AgentAPICommonResponse
//	408: AgentAPIConfigApplyStatusResponse
//	500: AgentAPICommonResponse
func (h *NginxHandler) updateConfig(w http.ResponseWriter, r *http.Request) error {
	correlationId := uuid.New().String()

	buf, err := readFileFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := AgentAPICommonResponse{
			CorrelationId: correlationId,
			Message:       err.Error(),
		}
		return writeObjectToResponseBody(w, response)
	}

	nginxDetails := h.getNginxDetails()

	for _, nginxDetail := range nginxDetails {
		err := h.applyNginxConfig(nginxDetail, buf, correlationId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := AgentAPICommonResponse{
				CorrelationId: correlationId,
				Message:       err.Error(),
			}
			return writeObjectToResponseBody(w, response)
		}
	}

	if len(nginxDetails) > 0 {
		agentAPIConfigApplyResponse := &AgentAPIConfigApplyResponse{CorrelationId: correlationId, NginxInstances: make([]NginxInstanceResponse, 0)}

		select {
		case response := <-h.responseChannel:
			nginxResponse := NginxInstanceResponse{
				NginxId: response.NginxConfigResponse.GetConfigData().GetNginxId(),
				Message: response.NginxConfigResponse.GetStatus().GetMessage(),
				Status:  okStatus,
			}

			if response.NginxConfigResponse.GetStatus().GetStatus() != proto.CommandStatusResponse_CMD_OK {
				if response.NginxConfigResponse.Status.Error == nginxConfigAsyncFeatureDisabled {
					w.WriteHeader(http.StatusForbidden)
				} else {
					w.WriteHeader(http.StatusBadRequest)
				}
				nginxResponse.Status = errorStatus
			} else {
				if response.NginxConfigResponse.GetStatus().GetMessage() == configAppliedProcessedResponse {
					w.WriteHeader(http.StatusRequestTimeout)
					nginxResponse.Status = pendingStatus
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}

			agentAPIConfigApplyResponse.NginxInstances = append(agentAPIConfigApplyResponse.NginxInstances, nginxResponse)

			// If the number of responses match the number of NGINX instances then return a response.
			// Otherwise wait until all config apply requests are complete for all NGINX instances.
			if len(agentAPIConfigApplyResponse.NginxInstances) == len(nginxDetails) {
				return writeObjectToResponseBody(w, agentAPIConfigApplyResponse)
			}

		case <-time.After(validationTimeout):
			w.WriteHeader(http.StatusRequestTimeout)
			agentAPIConfigApplyStatusResponse := AgentAPIConfigApplyStatusResponse{
				CorrelationId: correlationId,
				Message:       "pending config apply",
				Status:        pendingStatus,
			}

			return writeObjectToResponseBody(w, agentAPIConfigApplyStatusResponse)
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		response := AgentAPICommonResponse{
			CorrelationId: correlationId,
			Message:       "No NGINX instances found",
		}
		return writeObjectToResponseBody(w, response)
	}

	w.WriteHeader(http.StatusInternalServerError)
	return nil
}

func readFileFromRequest(r *http.Request) (*bytes.Buffer, error) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Errorf("unable to parse config apply request, %v", err)
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("can't read form file: %v", err)
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("can't read file, %v", err)
	}
	return buf, nil
}

func (h *NginxHandler) getNginxDetails() []*proto.NginxDetails {
	var nginxDetails []*proto.NginxDetails

	for _, proc := range h.getNginxProccessInfo() {
		if proc.IsMaster {
			nginxDetails = append(nginxDetails, h.nginxBinary.GetNginxDetailsFromProcess(proc))
		}
	}
	return nginxDetails
}

func (h *NginxHandler) applyNginxConfig(nginxDetail *proto.NginxDetails, buf *bytes.Buffer, correlationId string) error {
	fullFilePath := nginxDetail.ConfPath

	// Create backup of nginx.conf file on host
	data, err := os.ReadFile(fullFilePath)
	if err != nil {
		return fmt.Errorf("unable to read file %s: %v", fullFilePath, err)
	}

	protoFile := &proto.File{
		Name:        fullFilePath,
		Permissions: "0755",
		Contents:    buf.Bytes(),
	}

	configApply, err := sdk.NewConfigApplyWithIgnoreDirectives(protoFile.GetName(), h.config.AllowedDirectoriesMap, h.config.IgnoreDirectives)
	if err != nil {
		return fmt.Errorf("unable to write config: %v", err)
	}

	// Temporarily write the new nginx.conf to disk
	err = h.env.WriteFiles(configApply, []*proto.File{protoFile}, "", h.config.AllowedDirectoriesMap)
	if err != nil {
		rollbackErr := configApply.Rollback(err)
		return fmt.Errorf("config rollback failed: %v", rollbackErr)
	}

	// Create NginxConfig object for new nginx.conf
	conf, err := h.nginxBinary.ReadConfig(fullFilePath, nginxDetail.NginxId, h.env.GetSystemUUID())
	if err != nil {
		rollbackErr := configApply.Rollback(err)
		return fmt.Errorf("unable to read config: %v", rollbackErr)
	}

	// Write back the original nginx.conf
	err = os.WriteFile(fullFilePath, data, 0o644)
	if err != nil {
		rollbackErr := configApply.Rollback(err)
		return fmt.Errorf("unable to write file %s: %v", fullFilePath, rollbackErr)
	}

	// Send a config apply request to the nginx.go plugin
	h.pipeline.Process(core.NewMessage(core.CommNginxConfig, &AgentAPIConfigApplyRequest{correlationId: correlationId, config: conf}))
	return nil
}

// swagger:route GET /nginx/config/status nginx-agent get-nginx-config-status
//
// # Get status NGINX config apply
//
// # Returns status NGINX config apply
//
//	Parameters:
//	     + name: correlation_id
//	       in: query
//	       description: Correlation ID of a NGINX config apply request
//	       required: true
//	       type: string
//
// responses:
//
//	200: AgentAPIConfigApplyResponse
//	400: AgentAPIConfigApplyStatusResponse
//	404: AgentAPIConfigApplyStatusResponse
//	500: AgentAPICommonResponse
func (h *NginxHandler) getConfigStatus(w http.ResponseWriter, r *http.Request) error {
	correlationId := r.URL.Query().Get("correlation_id")

	if correlationId == "" {
		w.WriteHeader(http.StatusBadRequest)

		agentAPIConfigApplyStatusResponse := AgentAPIConfigApplyStatusResponse{
			CorrelationId: correlationId,
			Message:       "Missing required query parameter correlation_id",
			Status:        unknownStatus,
		}

		return writeObjectToResponseBody(w, agentAPIConfigApplyStatusResponse)
	}

	agentAPIConfigApplyStatusResponse := AgentAPIConfigApplyResponse{
		CorrelationId:  correlationId,
		NginxInstances: []NginxInstanceResponse{},
	}

	for _, nginxConfigStatus := range h.configResponseStatuses {
		if nginxConfigStatus.GetCorrelationId() == correlationId {
			nginxInstanceResponse := NginxInstanceResponse{
				NginxId: nginxConfigStatus.GetNginxId(),
				Message: nginxConfigStatus.GetMessage(),
				Status:  nginxConfigStatus.GetStatus().String(),
			}
			agentAPIConfigApplyStatusResponse.NginxInstances = append(agentAPIConfigApplyStatusResponse.NginxInstances, nginxInstanceResponse)
		}
	}

	if len(agentAPIConfigApplyStatusResponse.NginxInstances) == 0 {
		w.WriteHeader(http.StatusNotFound)
		agentAPIConfigApplyStatusResponse := AgentAPIConfigApplyStatusResponse{
			CorrelationId: correlationId,
			Message:       fmt.Sprintf("Unable to find a config apply request with the correlation_id %s", correlationId),
			Status:        unknownStatus,
		}

		return writeObjectToResponseBody(w, agentAPIConfigApplyStatusResponse)
	}

	w.WriteHeader(http.StatusOK)
	return writeObjectToResponseBody(w, agentAPIConfigApplyStatusResponse)
}

func (h *NginxHandler) getNginxProccessInfo() []*core.Process {
	h.processesMutex.RLock()
	defer h.processesMutex.RUnlock()
	return h.processes
}

func (h *NginxHandler) syncProcessInfo(processInfo []*core.Process) {
	h.processesMutex.Lock()
	defer h.processesMutex.Unlock()
	h.processes = processInfo
}

func (rh *RootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeHeader, jsonMimeType)

	switch {
	case healthRegex.MatchString(r.URL.Path):
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := rh.healthCheck(w)
		if err != nil {
			log.Warnf("Failed to get agent health: %v", err)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		_, err := fmt.Fprint(w, []byte("not found"))
		if err != nil {
			log.Warnf("Failed to send api response: %v", err)
		}
	}
}

// swagger:route GET /health nginx-agent health-check
//
// # Check the health of the NGINX Agent
//
// responses:
//
//	200: HealthResponse
func (rh *RootHandler) healthCheck(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)

	overallStatus := pendingStatus
	checks := []HealthStatusCheck{}

	registrationStatus := pendingStatus
	commandServiceStatus := pendingStatus
	metricsServiceStatus := pendingStatus

	if rh.config.IsGrpcServerConfigured() {
		if !rh.isGrpcRegistered {
			registrationStatus = errorStatus
			overallStatus = errorStatus
		} else {
			registrationStatus = okStatus
		}

		checks = append(checks, HealthStatusCheck{
			Name:   registration,
			Status: registrationStatus,
		})

		timeNow := time.Now()
		startTimeDiff := timeNow.Sub(rh.startTime)

		if !rh.lastCommandSent.IsZero() {
			lastCommandSentDiff := timeNow.Sub(rh.lastCommandSent)

			if lastCommandSentDiff > (2 * rh.config.Dataplane.Status.PollInterval) {
				commandServiceStatus = errorStatus
				overallStatus = errorStatus
			} else {
				commandServiceStatus = okStatus
			}
		} else if startTimeDiff > (2 * rh.config.Dataplane.Status.PollInterval) {
			commandServiceStatus = errorStatus
			overallStatus = errorStatus
		}

		checks = append(checks, HealthStatusCheck{
			Name:   commandConnection,
			Status: commandServiceStatus,
		})

		if rh.config.IsFeatureEnabled(agent_config.FeatureMetrics) || rh.config.IsFeatureEnabled(agent_config.FeatureMetricsSender) {
			if !rh.lastMetricReportSent.IsZero() {
				lastMetricReportSentDiff := timeNow.Sub(rh.lastMetricReportSent)

				if lastMetricReportSentDiff > (2 * rh.config.AgentMetrics.ReportInterval) {
					metricsServiceStatus = errorStatus
					overallStatus = errorStatus
				} else {
					metricsServiceStatus = okStatus
				}
			} else if startTimeDiff > (2 * rh.config.AgentMetrics.ReportInterval) {
				metricsServiceStatus = errorStatus
				overallStatus = errorStatus
			}

			checks = append(checks, HealthStatusCheck{
				Name:   metricsConnection,
				Status: metricsServiceStatus,
			})
		}
	}

	if registrationStatus == okStatus && commandServiceStatus == okStatus && metricsServiceStatus == okStatus {
		overallStatus = okStatus
	}

	healthResponse := &HealthResponse{
		Status: overallStatus,
		Checks: checks,
	}

	return writeObjectToResponseBody(w, healthResponse)
}

func writeObjectToResponseBody(w http.ResponseWriter, response any) error {
	respBody := new(bytes.Buffer)
	err := json.NewEncoder(respBody).Encode(response)
	if err != nil {
		return fmt.Errorf("failed to encode payload: %v", err)
	}

	_, err = fmt.Fprint(w, respBody)
	if err != nil {
		return fmt.Errorf("failed to send payload: %v", err)
	}
	return nil
}
