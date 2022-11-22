package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"

	log "github.com/sirupsen/logrus"
)

type AgentAPI struct {
	config       *config.Config
	env          core.Environment
	pipeline     core.MessagePipeInterface
	server       http.Server
	nginxBinary  core.NginxBinary
	nginxHandler *NginxHandler
}

type NginxHandler struct {
	config          *config.Config
	env             core.Environment
	pipeline        core.MessagePipeInterface
	nginxBinary     core.NginxBinary
	responseChannel chan *proto.Command_NginxConfigResponse
}

func NewAgentAPI(config *config.Config, env core.Environment, nginxBinary core.NginxBinary) *AgentAPI {
	return &AgentAPI{
		config:      config,
		env:         env,
		nginxBinary: nginxBinary,
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
	case core.RestAPIConfigApplyResponse:
		switch response := message.Data().(type) {
		case *proto.Command_NginxConfigResponse:
			a.nginxHandler.responseChannel <- response
		default:
			log.Warnf("Unknown Command_NginxConfigResponse type: %T(%v)", message.Data(), message.Data())
		}
	}
}

func (a *AgentAPI) Info() *core.Info {
	return core.NewInfo("Agent API Plugin", "v0.0.1")
}

func (a *AgentAPI) Subscriptions() []string {
	return []string{core.RestAPIConfigApplyResponse}
}

func (a *AgentAPI) createHttpServer() {
	a.nginxHandler = &NginxHandler{
		config:          a.config,
		pipeline:        a.pipeline,
		env:             a.env,
		nginxBinary:     a.nginxBinary,
		responseChannel: make(chan *proto.Command_NginxConfigResponse),
	}

	mux := http.NewServeMux()
	mux.Handle("/nginx/", a.nginxHandler)

	a.server = http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.AgentAPI.Port),
		Handler: mux,
	}

	log.Debug("Starting Agent API HTTP server")

	if err := a.server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("error listening to port: %v", err)
	}
}

var (
	instancesRegex = regexp.MustCompile(`^\/nginx[\/]*$`)
	configRegex    = regexp.MustCompile(`^\/nginx/config[\/]*$`)
)

func (h *NginxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	switch {
	case instancesRegex.MatchString(r.URL.Path):
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := sendInstanceDetailsPayload(h.getNginxDetails(), w, r)
		if err != nil {
			log.Warnf("Failed to send instance details payload: %v", err)
		}
	case configRegex.MatchString(r.URL.Path):
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := h.updateConfig(w, r)
		if err != nil {
			log.Warnf("Failed to update config: %v", err)
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

func (h *NginxHandler) updateConfig(w http.ResponseWriter, r *http.Request) error {
	log.Info("Updating config")

	r.ParseMultipartForm(32 << 20)
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return fmt.Errorf("can't read form file: %v", err)
	}
	defer file.Close()

	nginxDetails := h.getNginxDetails()

	log.Debug("Updating instances")

	for _, nginxDetail := range nginxDetails {
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			return fmt.Errorf("can't read file, %v", err)
		}

		fullFilePath := nginxDetail.ConfPath

		protoFile := &proto.File{
			Name:        fullFilePath,
			Permissions: "0755",
			Contents:    buf.Bytes(),
		}

		log.Tracef("protoFile: %v", protoFile)

		configApply, err := sdk.NewConfigApply(protoFile.GetName(), h.config.AllowedDirectoriesMap)
		if err != nil {
			return fmt.Errorf("unable to write config: %v", err)
		}

		log.Debug("Writing config")

		err = h.env.WriteFiles(configApply, []*proto.File{protoFile}, "", h.config.AllowedDirectoriesMap)
		if err != nil {
			rollbackErr := configApply.Rollback(err)
			return fmt.Errorf("config rollback failed: %v", rollbackErr)
		}

		err = configApply.Complete()
		if err != nil {
			return fmt.Errorf("unable to write config: %v", err)
		}

		log.Debug("File written")

		conf, err := h.nginxBinary.ReadConfig(fullFilePath, nginxDetail.NginxId, h.env.GetSystemUUID())
		if err != nil {
			return fmt.Errorf("unable to read config: %v", err)
		}

		log.Debug("ReadConfig")

		h.pipeline.Process(core.NewMessage(core.CommNginxConfig, conf))

		log.Debug("Waiting for response or timeout")
		select {
		case response := <-h.responseChannel:
			fmt.Fprintf(w, "%v", fileHeader.Header)
			reqBodyBytes := new(bytes.Buffer)
			if err := json.NewEncoder(reqBodyBytes).Encode(response); err != nil {
				return fmt.Errorf("failed to encode response: %v", err)

			}
			if _, err := w.Write(reqBodyBytes.Bytes()); err != nil {
				return fmt.Errorf("failed to write response: %v", err)
			}
			return nil
		case <-time.After(30 * time.Second):
			log.Warn("Config update failed: timeout")
			w.WriteHeader(http.StatusRequestTimeout)
			return nil
		}
	}
	return nil
}
