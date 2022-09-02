package plugins

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/client"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	appProtectMetadataFilePath = "/etc/nms/app_protect_metadata.json"
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
	}
}

func (n *Nginx) uploadConfig(config *proto.ConfigDescriptor, messageId string) error {
	log.Debugf("Uploading config for %v", config)

	if !n.isConfUploadEnabled {
		return errors.New("unable to upload config as nginx-config feature is disabled")
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

// The applyConfig does the following:
// - Download config
// - Stop file watcher
// - Write the config
// - Valid config
// - Upload the config
// - Start file watcher
// - Reload nginx
func (n *Nginx) applyConfig(cmd *proto.Command, cfg *proto.Command_NginxConfig) (status *proto.Command_NginxConfigResponse) {
	log.Debugf("Applying config for message id, %s", cmd.GetMeta().MessageId)

	status = &proto.Command_NginxConfigResponse{
		NginxConfigResponse: &proto.NginxConfigResponse{
			Status:     newOKStatus("config applied successfully").CmdStatus,
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
		return n.handleErrorStatus(status, message)
	}

	err = n.nginxBinary.ValidateConfig(nginx.NginxId, nginx.ProcessPath, nginx.ConfPath, config, configApply)

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
		return n.handleErrorStatus(status, message)
	} else if configApply != nil {
		if err = configApply.Complete(); err != nil {
			log.Errorf("Config complete failed: %v", err)
		}
	}

	uploadResponse := &proto.Command_NginxConfigResponse{
		NginxConfigResponse: &proto.NginxConfigResponse{
			Action:     proto.NginxConfigAction_UNKNOWN,
			Status:     newOKStatus("config uploaded status").CmdStatus,
			ConfigData: nil,
		},
	}

	err = n.uploadConfig(
		&proto.ConfigDescriptor{
			SystemId: n.env.GetSystemUUID(),
			NginxId:  config.GetConfigData().GetNginxId(),
		},
		cmd.Meta.GetMessageId(),
	)
	if err != nil {
		uploadResponse.NginxConfigResponse.Status = newErrStatus("config uploaded error: " + err.Error()).CmdStatus
	}

	uploadResponseCommand := newStatusCommand(cmd)
	uploadResponseCommand.Data = uploadResponse

	n.messagePipeline.Process(core.NewMessage(core.CommResponse, uploadResponseCommand))
	log.Debug("Enabling file watcher")
	n.messagePipeline.Process(core.NewMessage(core.FileWatcherEnabled, true))

	reloadErr := n.nginxBinary.Reload(nginx.ProcessId, nginx.ProcessPath)
	if reloadErr != nil {
		status.NginxConfigResponse.Status = newErrStatus("Config apply failed (write): " + reloadErr.Error()).CmdStatus
	}

	nginxReloadEventMeta := NginxReloadResponse{
		succeeded:     reloadErr == nil,
		correlationId: cmd.Meta.MessageId,
		timestamp:     types.TimestampNow(),
		nginxDetails:  nginx,
	}
	n.messagePipeline.Process(core.NewMessage(core.NginxReloadComplete, nginxReloadEventMeta))
	log.Debug("Config Apply Complete")
	return status
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
