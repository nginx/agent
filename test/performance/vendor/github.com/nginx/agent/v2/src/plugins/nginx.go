package plugins

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/grpc"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	appProtectMetadataFilePath = "/etc/nms/app_protect_metadata.json"
)

var (
	validationTimeout = 15 * time.Second
)

// Nginx is the metadata of our nginx binary
type Nginx struct {
	messagePipeline     core.MessagePipeInterface
	nginxBinary         core.NginxBinary
	processes           []core.Process
	env                 core.Environment
	cmdr                client.Commander
	config              *config.Config
	isNAPEnabled        bool
	isConfUploadEnabled bool
}

type ConfigRollbackResponse struct {
	succeeded     bool
	correlationId string
	timestamp     *types.Timestamp
	nginxDetails  *proto.NginxDetails
}

type NginxReloadResponse struct {
	succeeded     bool
	correlationId string
	timestamp     *types.Timestamp
	nginxDetails  *proto.NginxDetails
}

type NginxConfigValidationResponse struct {
	err           error
	correlationId string
	nginxDetails  *proto.NginxDetails
	config        *proto.NginxConfig
	configApply   *sdk.ConfigApply
}

func NewNginx(cmdr client.Commander, nginxBinary core.NginxBinary, env core.Environment, loadedConfig *config.Config) *Nginx {
	var isNAPEnabled bool
	if loadedConfig.NginxAppProtect != (config.NginxAppProtect{}) {
		isNAPEnabled = true
	}

	isConfUploadEnabled := isConfUploadEnabled(loadedConfig)

	return &Nginx{nginxBinary: nginxBinary, processes: env.Processes(), env: env, cmdr: cmdr, config: loadedConfig, isNAPEnabled: isNAPEnabled, isConfUploadEnabled: isConfUploadEnabled}
}

// Init initializes the plugin
func (n *Nginx) Init(pipeline core.MessagePipeInterface) {
	log.Info("NginxBinary initializing")
	n.messagePipeline = pipeline
	n.nginxBinary.UpdateNginxDetailsFromProcesses(n.processes)
	nginxDetails := n.nginxBinary.GetNginxDetailsMapFromProcesses(n.processes)

	pipeline.Process(
		core.NewMessage(core.NginxPluginConfigured, n),
		core.NewMessage(core.NginxInstancesFound, nginxDetails),
	)
}

// Process processes the messages from the messaging pipe
func (n *Nginx) Process(message *core.Message) {
	switch message.Topic() {
	case core.CommNginxConfig:
		switch cmd := message.Data().(type) {
		case *proto.Command:
			n.processCmd(cmd)
		}
	case core.NginxConfigUpload:
		switch cfg := message.Data().(type) {
		case *proto.ConfigDescriptor:
			err := n.uploadConfig(cfg, uuid.New().String())
			if err != nil {
				log.Warnf("Error uploading config: %v", err)
			}
		}
	case core.NginxDetailProcUpdate:
		procs := message.Data().([]core.Process)
		n.nginxBinary.UpdateNginxDetailsFromProcesses(procs)
	case core.DataplaneChanged:
		n.uploadConfigs()
	case core.AgentConfigChanged:
		// If the agent config on disk changed update this with relevant config info
		n.syncAgentConfigChange()
	case core.NginxConfigValidationSucceeded:
		switch response := message.Data().(type) {
		case *NginxConfigValidationResponse:
			n.completeConfigApply(response)
		}
	case core.NginxConfigValidationFailed:
		switch response := message.Data().(type) {
		case *NginxConfigValidationResponse:
			n.rollbackConfigApply(response)
		}
	case core.EnableExtension:
		switch data := message.Data().(type) {
		case string:
			if data == config.NginxAppProtectKey {
				n.isNAPEnabled = true
			}
		}
	}
}

func (n *Nginx) Subscriptions() []string {
	return []string{
		core.CommNginxConfig,
		core.NginxConfigUpload,
		core.NginxDetailProcUpdate,
		core.DataplaneChanged,
		core.AgentConfigChanged,
		core.EnableExtension,
		core.NginxConfigValidationPending,
		core.NginxConfigValidationSucceeded,
		core.NginxConfigValidationFailed,
	}
}

func (n *Nginx) uploadConfig(config *proto.ConfigDescriptor, messageId string) error {
	log.Debugf("Uploading config for %v", config)

	if !n.isConfUploadEnabled {
		log.Info("unable to upload config as nginx-config feature is disabled")
		return nil
	}

	if config.GetNginxId() == "" {
		return nil
	}
	nginx := n.nginxBinary.GetNginxDetailsByID(config.GetNginxId())
	if nginx == nil {
		message := fmt.Sprintf("Unable to find nginx instance %s. Instance could be offline or uninstalled.", config.GetNginxId())
		log.Warn(message)
		return errors.New(message)
	}

	log.Tracef("Reading config in directory %v for nginx instance %v", nginx.GetConfPath(), config.GetNginxId())
	cfg, err := n.nginxBinary.ReadConfig(nginx.GetConfPath(), config.GetNginxId(), config.GetSystemId())
	if err != nil {
		log.Errorf("Unable to read nginx config %s: %v", nginx.GetConfPath(), err)
		return err
	}

	if n.isNAPEnabled {
		cfg, err = sdk.AddAuxfileToNginxConfig(nginx.GetConfPath(), cfg, appProtectMetadataFilePath, n.config.AllowedDirectoriesMap, true)
		if err != nil {
			log.Errorf("Unable to add aux file %s to nginx config: %v", appProtectMetadataFilePath, err)
			return err
		}
	}

	if err := n.cmdr.Upload(context.Background(), cfg, messageId); err != nil {
		log.Errorf("Unable to upload nginx config : %v", err)
		return err
	}

	return nil
}

func (n *Nginx) processCmd(cmd *proto.Command) {
	switch commandData := cmd.Data.(type) {
	case *proto.Command_NginxConfig:
		log.Infof("nginx config %s command %+v", commandData.NginxConfig.Action, commandData)
		status := &proto.Command_NginxConfigResponse{
			NginxConfigResponse: &proto.NginxConfigResponse{
				Status:     nil,
				Action:     proto.NginxConfigAction_UNKNOWN,
				ConfigData: nil,
			},
		}

		switch commandData.NginxConfig.Action {
		case proto.NginxConfigAction_APPLY:
			if n.isConfUploadEnabled {
				status = n.applyConfig(cmd, commandData)
			} else {
				log.Warnf("unable to upload config as nginx-config feature is disabled")
			}
		case proto.NginxConfigAction_TEST:
			// TODO: Test agent config?
			status.NginxConfigResponse.Status = newErrStatus("Config test not implemented").CmdStatus
			status.NginxConfigResponse.Action = proto.NginxConfigAction_TEST
		case proto.NginxConfigAction_ROLLBACK:
			// TODO: Rollback config?
			status.NginxConfigResponse.Status = newErrStatus("Config rollback not implemented").CmdStatus
			status.NginxConfigResponse.Action = proto.NginxConfigAction_ROLLBACK
		case proto.NginxConfigAction_RETURN:
			// TODO: Upload config
			status.NginxConfigResponse.Status = newErrStatus("Config return not implemented").CmdStatus
			status.NginxConfigResponse.Action = proto.NginxConfigAction_RETURN
		default:
			log.Infof("unknown nginx config action")
			status.NginxConfigResponse.Status = newErrStatus("unknown Config action not implemented").CmdStatus
		}

		resp := newStatusCommand(cmd)
		resp.Data = status
		if resp.GetNginxConfigResponse().GetStatus().GetError() != "" {
			log.Errorf("config action failed: %s", resp.GetNginxConfigResponse().GetStatus().GetError())
		}

		n.messagePipeline.Process(core.NewMessage(core.CommResponse, resp))
	}
}

func (n *Nginx) applyConfig(cmd *proto.Command, cfg *proto.Command_NginxConfig) (status *proto.Command_NginxConfigResponse) {
	log.Debugf("Applying config for message id, %s", cmd.GetMeta().MessageId)

	n.messagePipeline.Process(core.NewMessage(core.NginxConfigValidationPending, &proto.AgentActivityStatus{
		Status: &proto.AgentActivityStatus_NginxConfigStatus{
			NginxConfigStatus: &proto.NginxConfigStatus{
				CorrelationId: cmd.Meta.MessageId,
				Status:        proto.NginxConfigStatus_PENDING,
				Message:       "config apply pending",
			},
		},
	}))

	status = &proto.Command_NginxConfigResponse{
		NginxConfigResponse: &proto.NginxConfigResponse{
			Status:     newOKStatus("config apply request successfully processed").CmdStatus,
			Action:     proto.NginxConfigAction_APPLY,
			ConfigData: cfg.NginxConfig.ConfigData,
		},
	}

	config, err := n.cmdr.Download(context.Background(), cmd.GetMeta())
	if err != nil {
		status.NginxConfigResponse.Status = newErrStatus("Config apply failed (download): " + err.Error()).CmdStatus
		return status
	}

	if config.GetConfigData().GetNginxId() == "" {
		status.NginxConfigResponse.Status = newErrStatus(fmt.Sprintf("Config apply failed (preflight): no Nginx Id in ConfigDescriptor %v", config.GetConfigData())).CmdStatus
		return status
	}

	log.Debugf("Disabling file watcher")
	n.messagePipeline.Process(core.NewMessage(core.FileWatcherEnabled, false))

	nginx := n.nginxBinary.GetNginxDetailsByID(config.GetConfigData().GetNginxId())

	if nginx == nil || nginx == (&proto.NginxDetails{}) {
		message := fmt.Sprintf("Config apply failed (preflight): no Nginx instance found for %v", config.GetConfigData().GetNginxId())
		return n.handleErrorStatus(status, message)
	}

	configApply, err := n.nginxBinary.WriteConfig(config)
	if err != nil {
		if configApply != nil {
			succeeded := true

			if rollbackErr := configApply.Rollback(err); rollbackErr != nil {
				log.Errorf("Config rollback failed: %v", rollbackErr)
				succeeded = false
			}

			configRollbackResponse := ConfigRollbackResponse{
				succeeded:     succeeded,
				correlationId: cmd.Meta.MessageId,
				timestamp:     types.TimestampNow(),
				nginxDetails:  nginx,
			}
			n.messagePipeline.Process(core.NewMessage(core.ConfigRollbackResponse, configRollbackResponse))
		}

		message := fmt.Sprintf("Config apply failed (write): " + err.Error())

		n.messagePipeline.Process(core.NewMessage(core.NginxConfigValidationPending, &proto.AgentActivityStatus{
			Status: &proto.AgentActivityStatus_NginxConfigStatus{
				NginxConfigStatus: &proto.NginxConfigStatus{
					CorrelationId: cmd.Meta.MessageId,
					Status:        proto.NginxConfigStatus_ERROR,
					Message:       message,
				},
			},
		}))

		return n.handleErrorStatus(status, message)
	}

	validationResponse := n.validateConfig(nginx, cmd.Meta.MessageId, config, configApply)
	if validationResponse != nil {
		if validationResponse.err == nil {
			agentActivityStatus := n.completeConfigApply(validationResponse)
			if agentActivityStatus.GetNginxConfigStatus().GetStatus() == proto.NginxConfigStatus_ERROR {
				status.NginxConfigResponse.Status = newErrStatus(agentActivityStatus.GetNginxConfigStatus().GetMessage()).CmdStatus
			} else {
				status.NginxConfigResponse.Status = newOKStatus(agentActivityStatus.GetNginxConfigStatus().GetMessage()).CmdStatus
			}
			return status
		} else {
			n.rollbackConfigApply(validationResponse)
			message := fmt.Sprintf("Config apply failed (write): " + validationResponse.err.Error())
			return n.handleErrorStatus(status, message)
		}
	} else {
		log.Debug("Validation of nginx config in progress")
		return status
	}
}

// This function will run a nginx config validation in a separate go routine. If the validation takes less than 15 seconds then the result is returned straight away,
// otherwise nil is returned and the validation continues on in the background until it is complete. The result is always added to the message pipeline for other plugins
// to use.
func (n *Nginx) validateConfig(nginx *proto.NginxDetails, correlationId string, config *proto.NginxConfig, configApply *sdk.ConfigApply) *NginxConfigValidationResponse {
	validationChannel := make(chan *NginxConfigValidationResponse, 1)

	go func() {
		err := n.nginxBinary.ValidateConfig(nginx.NginxId, nginx.ProcessPath, nginx.ConfPath, config, configApply)
		if err != nil {
			response := &NginxConfigValidationResponse{
				err:           fmt.Errorf("error running nginx -t -c %s:\n %v", nginx.ConfPath, err),
				correlationId: correlationId,
				nginxDetails:  nginx,
				config:        config,
				configApply:   configApply,
			}
			n.messagePipeline.Process(core.NewMessage(core.NginxConfigValidationFailed, response))
			validationChannel <- response
		} else {
			response := &NginxConfigValidationResponse{
				err:           nil,
				correlationId: correlationId,
				nginxDetails:  nginx,
				config:        config,
				configApply:   configApply,
			}
			n.messagePipeline.Process(core.NewMessage(core.NginxConfigValidationSucceeded, response))
			validationChannel <- response
		}
	}()

	select {
	case result := <-validationChannel:
		return result
	case <-time.After(validationTimeout):
		return nil
	}
}

func (n *Nginx) completeConfigApply(response *NginxConfigValidationResponse) *proto.AgentActivityStatus {
	nginxConfigStatusMessage := "Config applied successfully"
	if response.configApply != nil {
		if err := response.configApply.Complete(); err != nil {
			nginxConfigStatusMessage = fmt.Sprintf("Config complete failed: %v", err)
			log.Errorf(nginxConfigStatusMessage)
		}
	}

	uploadResponse := &proto.Command_NginxConfigResponse{
		NginxConfigResponse: &proto.NginxConfigResponse{
			Action:     proto.NginxConfigAction_UNKNOWN,
			Status:     newOKStatus("config uploaded status").CmdStatus,
			ConfigData: nil,
		},
	}

	err := n.uploadConfig(
		&proto.ConfigDescriptor{
			SystemId: n.env.GetSystemUUID(),
			NginxId:  response.config.GetConfigData().GetNginxId(),
		},
		response.correlationId,
	)
	if err != nil {
		uploadResponse.NginxConfigResponse.Status = newErrStatus("Config uploaded error: " + err.Error()).CmdStatus
		nginxConfigStatusMessage = fmt.Sprintf("Config uploaded error: %v", err)
		log.Errorf(nginxConfigStatusMessage)
	}

	uploadResponseCommand := &proto.Command{Meta: grpc.NewMessageMeta(response.correlationId)}
	uploadResponseCommand.Data = uploadResponse

	n.messagePipeline.Process(core.NewMessage(core.CommResponse, uploadResponseCommand))
	log.Debug("Enabling file watcher")
	n.messagePipeline.Process(core.NewMessage(core.FileWatcherEnabled, true))

	reloadErr := n.nginxBinary.Reload(response.nginxDetails.ProcessId, response.nginxDetails.ProcessPath)
	if reloadErr != nil {
		nginxConfigStatusMessage = fmt.Sprintf("Config apply failed (write): %v", reloadErr)
		log.Errorf(nginxConfigStatusMessage)
	}

	nginxReloadEventMeta := NginxReloadResponse{
		succeeded:     reloadErr == nil,
		correlationId: response.correlationId,
		timestamp:     types.TimestampNow(),
		nginxDetails:  response.nginxDetails,
	}

	n.messagePipeline.Process(core.NewMessage(core.NginxReloadComplete, nginxReloadEventMeta))

	agentActivityStatus := &proto.AgentActivityStatus{
		Status: &proto.AgentActivityStatus_NginxConfigStatus{
			NginxConfigStatus: &proto.NginxConfigStatus{
				CorrelationId: response.correlationId,
				Status:        proto.NginxConfigStatus_OK,
				Message:       nginxConfigStatusMessage,
			},
		},
	}

	n.messagePipeline.Process(core.NewMessage(core.NginxConfigApplySucceeded, agentActivityStatus))

	log.Debug("Config Apply Complete")
	return agentActivityStatus
}

func (n *Nginx) rollbackConfigApply(response *NginxConfigValidationResponse) {
	nginxConfigStatusMessage := fmt.Sprintf("Config apply failed (write): %v", response.err.Error())
	log.Error(nginxConfigStatusMessage)

	if response.configApply != nil {
		succeeded := true

		if rollbackErr := response.configApply.Rollback(response.err); rollbackErr != nil {
			nginxConfigStatusMessage := fmt.Sprintf("Config rollback failed: %v", rollbackErr)
			log.Error(nginxConfigStatusMessage)
			succeeded = false
		}

		configRollbackResponse := ConfigRollbackResponse{
			succeeded:     succeeded,
			correlationId: response.correlationId,
			timestamp:     types.TimestampNow(),
			nginxDetails:  response.nginxDetails,
		}

		n.messagePipeline.Process(core.NewMessage(core.ConfigRollbackResponse, configRollbackResponse))

		agentActivityStatus := &proto.AgentActivityStatus{
			Status: &proto.AgentActivityStatus_NginxConfigStatus{
				NginxConfigStatus: &proto.NginxConfigStatus{
					CorrelationId: response.correlationId,
					Status:        proto.NginxConfigStatus_ERROR,
					Message:       nginxConfigStatusMessage,
				},
			},
		}

		n.messagePipeline.Process(core.NewMessage(core.NginxConfigApplyFailed, agentActivityStatus))
	}

	log.Debug("Enabling file watcher")
	n.messagePipeline.Process(core.NewMessage(core.FileWatcherEnabled, true))
}

func (n *Nginx) handleErrorStatus(status *proto.Command_NginxConfigResponse, message string) *proto.Command_NginxConfigResponse {
	status.NginxConfigResponse.Status = newErrStatus(message).CmdStatus

	log.Debug("Enabling file watcher")
	n.messagePipeline.Process(core.NewMessage(core.FileWatcherEnabled, true))

	return status
}

func (n *Nginx) uploadConfigs() {
	systemId := n.env.GetSystemUUID()

	for nginxID := range n.nginxBinary.GetNginxDetailsMapFromProcesses(n.env.Processes()) {
		err := n.uploadConfig(
			&proto.ConfigDescriptor{
				SystemId: systemId,
				NginxId:  nginxID,
			},
			uuid.NewString(),
		)
		if err != nil {
			log.Warnf("Unable to upload config for nginx instance %s, %v", nginxID, err)
		}
	}
}

// Info returns the version of this plugin
func (n *Nginx) Info() *core.Info {
	return core.NewInfo("NginxBinary", "v0.0.1")
}

// Close cleans up anything outstanding once the plugin ends
func (n *Nginx) Close() {
	log.Info("NginxBinary is wrapping up")
}

func (n *Nginx) syncAgentConfigChange() {
	conf, err := config.GetConfig(n.env.GetSystemUUID())
	if err != nil {
		log.Errorf("Failed to load config for updating: %v", err)
		return
	}
	log.Debugf("Nginx Plugins is updating to a new config - %v", conf)

	if conf.NginxAppProtect != (config.NginxAppProtect{}) {
		n.isNAPEnabled = true
	} else {
		n.isNAPEnabled = false
	}

	n.isConfUploadEnabled = isConfUploadEnabled(conf)

	n.config = conf
}

func isConfUploadEnabled(conf *config.Config) bool {
	for _, feature := range conf.Features {
		if feature == config.FeatureNginxConfig {
			return true
		}
	}
	return false
}
