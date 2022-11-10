package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
)

type RestApi struct {
	env         core.Environment
	nginxBinary core.NginxBinary
	handler     *NginxHandler
}

type NginxHandler struct {
	env         core.Environment
	nginxBinary core.NginxBinary
}

func NewRestApi(config *config.Config, env core.Environment, nginxBinary core.NginxBinary) *RestApi {
	return &RestApi{env: env, nginxBinary: nginxBinary}
}

func (r *RestApi) Init(core.MessagePipeInterface) {
	log.Info("REST API initializing")
	go r.createHttpServer()
}

func (r *RestApi) Close() {
	log.Info("REST API is wrapping up")
}

func (r *RestApi) Process(message *core.Message) {
	log.Tracef("Process function in the rest_api.go, %s %v", message.Topic(), message.Data())
}

func (r *RestApi) Info() *core.Info {
	return core.NewInfo("REST API Plugin", "v0.0.1")
}

func (r *RestApi) Subscriptions() []string {
	return []string{}
}

func (r *RestApi) createHttpServer() {
	mux := http.NewServeMux()
	r.handler = &NginxHandler{r.env, r.nginxBinary}
	mux.Handle("/nginx/", r.handler)

	log.Debug("Starting REST API HTTP server")

	server := http.Server{
		Addr:    ":9090",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
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
		h.sendInstanceDetailsPayload(h.getNginxDetails(), w, r)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
		return
	}
}

func (h *NginxHandler) sendInstanceDetailsPayload(nginxDetails []*proto.NginxDetails, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	if len(nginxDetails) == 0 {
		log.Debug("No nginx instances found")
		_, err := fmt.Fprint(w, "[]")
		if err != nil {
			log.Warnf("Failed to send instance details payload: %v", err)
		}
		return
	}

	responseBodyBytes := new(bytes.Buffer)
	json.NewEncoder(responseBodyBytes).Encode(nginxDetails)

	_, err := fmt.Fprint(w, responseBodyBytes.Bytes())
	if err != nil {
		log.Warnf("Failed to send instance details payload: %v", err)
	}
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
