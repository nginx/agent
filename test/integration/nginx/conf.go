package conf

import (
	"strings"
	"text/template"
)

const (
	MarkerEnvironment  = "environment"
	MarkerApp          = "app"
	MarkerComponent    = "component"
	MarkerGateway      = "gateway"
	MarkerPublishedApi = "published_api"
)

type NginxConf struct {
	HttpBlock   *HttpBlock
	StreamBlock *StreamBlock
}

type HttpBlock struct {
	F5MetricsServer  string
	F5MetricsMarkers map[string]string
	Upstreams        map[string]Upstream
	Servers          []Server
}

type StreamBlock struct {
	F5MetricsServer  string
	F5MetricsMarkers map[string]string
	Upstreams        map[string]Upstream
	Servers          []StreamServer
}

type Upstream struct {
	Server string
}

type StreamServer struct {
	Listen           string
	F5MetricsMarkers map[string]string
	UpstreamName     string
	Directives       []string
}

type Server struct {
	Listen           string
	F5MetricsMarkers map[string]string
	Locations        map[string]Location
}

type Location struct {
	F5MetricsMarkers map[string]string
	UpstreamName     string
	Directives       []string
}

func NewNginxConf() *NginxConf {
	return &NginxConf{}
}

func (c NginxConf) Build() (string, error) {
	return execTemplate("conf", NginxConfTemplate, c)
}

func (c HttpBlock) Build() (string, error) {
	return execTemplate("http_block", HttpBlockTemplate, c)
}

func (c StreamBlock) Build() (string, error) {
	return execTemplate("stream_block", StreamBlockTemplate, c)
}

func (c Server) Build() (string, error) {
	return execTemplate("server_block", ServerBlockTemplate, c)
}

func (c StreamServer) Build() (string, error) {
	return execTemplate("stream_server_block", StreamServerBlockTemplate, c)
}

func (c Location) Build() (string, error) {
	return execTemplate("location_block", LocationBlockTemplate, c)
}

func execTemplate(name string, content string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return "", err
	}
	builder := strings.Builder{}
	if err = tmpl.Execute(&builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}
