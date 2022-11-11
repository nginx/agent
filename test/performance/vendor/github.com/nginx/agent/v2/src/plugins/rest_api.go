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
	config       *config.Config
	env          core.Environment
	nginxBinary  core.NginxBinary
	nginxHandler *NginxHandler
}

type NginxHandler struct {
	env         core.Environment
	nginxBinary core.NginxBinary
}

func NewRestApi(config *config.Config, env core.Environment, nginxBinary core.NginxBinary) *RestApi {
	return &RestApi{config: config, env: env, nginxBinary: nginxBinary}
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
	r.nginxHandler = &NginxHandler{r.env, r.nginxBinary}
	mux.Handle("/nginx/", r.nginxHandler)

	log.Debug("Starting REST API HTTP server")

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", r.config.Server.RestPort),
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
