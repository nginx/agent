package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
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
	exporter     *Exporter
}

type NginxHandler struct {
	env         core.Environment
	nginxBinary core.NginxBinary
}
type Exporter struct {
	latestMetricReport *proto.MetricsReport
}

func NewAgentAPI(config *config.Config, env core.Environment, nginxBinary core.NginxBinary) *AgentAPI {
	return &AgentAPI{config: config, env: env, nginxBinary: nginxBinary, exporter: NewExporter(&proto.MetricsReport{})}
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
	log.Error("------------------------- PROCESS ----------------------------")
	switch {
	case message.Exact(core.MetricReport):
		metricReport, ok := message.Data().(*proto.MetricsReport)
		if !ok {
			log.Warnf("Invalid message received, %T, for topic, %s", message.Data(), message.Topic())
			return
		}
		log.Error(metricReport)
		a.exporter.latestMetricReport = metricReport
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
	metricsEndpoint := flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
	registerer := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer

	registerer.MustRegister(a.exporter)
	mux.Handle(*metricsEndpoint, promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))

	mux.Handle("/nginx/", a.nginxHandler)

	log.Debug("Starting Agent API HTTP server")

	a.server = http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.AgentAPI.Port),
		Handler: mux,
	}

	if err := a.server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("error listening to port: %v", err)
	}
}

var (
	instancesRegex = regexp.MustCompile(`^\/nginx[\/]*$`)
)

func (h *NginxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
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

func NewExporter(report *proto.MetricsReport) *Exporter {
	return &Exporter{latestMetricReport: report}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})
	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()
	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}
func metricName(in string) string {
	return strings.Replace(in, ".", "_", -1)
}
func metricLabels(Dimensions []*proto.Dimension) map[string]string {
	m := make(map[string]string)
	for _, dimension := range Dimensions {
		name := metricName(dimension.Name)
		m[name] = dimension.Value
	}
	return m
}

func getValueType(metricName string) prometheus.ValueType {
	calMap := metrics.CalculationMap()

	if value, ok := calMap[metricName]; ok {
		if value == "sum" {
			return prometheus.CounterValue
		} else {
			return prometheus.GaugeValue
		}

	}

	return prometheus.GaugeValue
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, statsEntity := range e.latestMetricReport.Data {
		for _, metric := range statsEntity.Simplemetrics {
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					metricName(metric.Name),
					"Metric Report",
					nil,
					metricLabels(statsEntity.Dimensions),
				),
				getValueType(metric.Name), metric.Value,
			)
		}
	}
}
