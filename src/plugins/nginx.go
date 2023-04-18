/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	re "regexp"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/zip"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/payloads"
	"github.com/nginx/agent/v2/src/core/tailer"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/nap"
)

const (
	configAppliedProcessedResponse = "config apply request successfully processed"
	configAppliedResponse          = "config applied successfully"
)

var (
	validationTimeout = 60 * time.Second
	reloadErrorList   = []*re.Regexp{
		re.MustCompile(`.*bind\(\) to .* failed \(98: Address already in use\).*`),
		re.MustCompile(`.*bind\(\) to .* failed \(98: Unknown error\).*`),
	}
)

// Nginx is the metadata of our nginx binary
type Nginx struct {
	messagePipeline                core.MessagePipeInterface
	nginxBinary                    core.NginxBinary
	processes                      []core.Process
	processesMutex                 sync.RWMutex
	monitorMutex                   sync.Mutex
	env                            core.Environment
	cmdr                           client.Commander
	config                         *config.Config
	isNginxAppProtectEnabled       bool
	isFeatureNginxConfigEnabled    bool
	configApplyStatusChannel       chan *proto.Command_NginxConfigResponse
	nginxAppProtectSoftwareDetails *proto.AppProtectWAFDetails
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
	elapsedTime   time.Duration
}

func NewNginx(cmdr client.Commander, nginxBinary core.NginxBinary, env core.Environment, loadedConfig *config.Config) *Nginx {
	isFeatureNginxConfigEnabled := loadedConfig.IsFeatureEnabled(agent_config.FeatureNginxConfig)

	isNginxAppProtectEnabled := loadedConfig.IsExtensionEnabled(agent_config.NginxAppProtectExtensionPlugin)

	return &Nginx{
		nginxBinary:                    nginxBinary,
		processes:                      env.Processes(),
		env:                            env,
		cmdr:                           cmdr,
		config:                         loadedConfig,
		isNginxAppProtectEnabled:       isNginxAppProtectEnabled,
		isFeatureNginxConfigEnabled:    isFeatureNginxConfigEnabled,
		configApplyStatusChannel:       make(chan *proto.Command_NginxConfigResponse, 1),
		nginxAppProtectSoftwareDetails: &proto.AppProtectWAFDetails{},
	}
}

// Init initializes the plugin
func (n *Nginx) Init(pipeline core.MessagePipeInterface) {
	log.Info("NginxBinary initializing")
	n.messagePipeline = pipeline
	n.nginxBinary.UpdateNginxDetailsFromProcesses(n.getNginxProccessInfo())
	nginxDetails := n.nginxBinary.GetNginxDetailsMapFromProcesses(n.getNginxProccessInfo())

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
		case *AgentAPIConfigApplyRequest:
			status := n.writeConfigAndReloadNginx(cmd.correlationId, cmd.config, proto.NginxConfigAction_APPLY)
			if status.NginxConfigResponse.GetStatus().GetMessage() != configAppliedProcessedResponse {
				n.messagePipeline.Process(core.NewMessage(core.AgentAPIConfigApplyResponse, status))
			}
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
		n.syncProcessInfo(procs)
		n.nginxBinary.UpdateNginxDetailsFromProcesses(procs)
	case core.DataplaneChanged:
		n.uploadConfigs()
	case core.DataplaneSoftwareDetailsUpdated:
		switch details := message.Data().(type) {
		case *payloads.DataplaneSoftwareDetailsUpdate:
			n.processDataplaneSoftwareDetailsUpdate(details)
		}
	case core.AgentConfigChanged:
		// If the agent config on disk changed update this with relevant config info
		n.syncAgentConfigChange()
	case core.NginxConfigValidationSucceeded:
		switch response := message.Data().(type) {
		case *NginxConfigValidationResponse:
			status := n.completeConfigApply(response)
			if response.elapsedTime < validationTimeout {
				n.configApplyStatusChannel <- status
			}
		}
	case core.NginxConfigValidationFailed:
		switch response := message.Data().(type) {
		case *NginxConfigValidationResponse:
			n.rollbackConfigApply(response.correlationId, response.config.GetConfigData().GetNginxId(), response.nginxDetails, response.configApply, response.err)
			status := &proto.Command_NginxConfigResponse{
				NginxConfigResponse: &proto.NginxConfigResponse{
					Status:     newErrStatus(fmt.Sprintf("Config apply failed (write): " + response.err.Error())).CmdStatus,
					Action:     proto.NginxConfigAction_APPLY,
					ConfigData: response.config.ConfigData,
				},
			}
			if response.elapsedTime < validationTimeout {
				n.configApplyStatusChannel <- status
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
		core.DataplaneSoftwareDetailsUpdated,
		core.AgentConfigChanged,
		core.EnableExtension,
		core.NginxConfigValidationPending,
		core.NginxConfigValidationSucceeded,
		core.NginxConfigValidationFailed,
	}
}

func (n *Nginx) getNginxProccessInfo() []core.Process {
	n.processesMutex.RLock()
	defer n.processesMutex.RUnlock()
	return n.processes
}

func (n *Nginx) syncProcessInfo(processInfo []core.Process) {
	n.processesMutex.Lock()
	defer n.processesMutex.Unlock()
	n.processes = processInfo
}

func (n *Nginx) uploadConfig(config *proto.ConfigDescriptor, messageId string) error {
	log.Debugf("Uploading config for %v", config)

	if !n.isFeatureNginxConfigEnabled {
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

	if n.isNginxAppProtectEnabled {
		err = nap.UpdateMetadata(cfg, n.nginxAppProtectSoftwareDetails)
		if err != nil {
			log.Errorf("Unable to update NAP metadata: %v", err)
		}
		cfg, err = sdk.AddAuxfileToNginxConfig(nginx.GetConfPath(), cfg, n.nginxAppProtectSoftwareDetails.GetWafLocation(), n.config.AllowedDirectoriesMap, true)
		if err != nil {
			log.Errorf("Unable to add aux file %s to nginx config: %v", n.nginxAppProtectSoftwareDetails.GetWafLocation(), err)
			return err
		}
	}

	if err := n.cmdr.Upload(context.Background(), cfg, messageId); err != nil {
		log.Errorf("Unable to upload nginx config : %v", err)
		return err
	}

	return nil
}

func (n *Nginx) processDataplaneSoftwareDetailsUpdate(update *payloads.DataplaneSoftwareDetailsUpdate) {
	log.Tracef("software details updated software %+v", update)

	if update.GetPluginName() == agent_config.NginxAppProtectExtensionPlugin {
		n.nginxAppProtectSoftwareDetails = update.GetDataplaneSoftwareDetails().GetAppProtectWafDetails()
	}
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
		case proto.NginxConfigAction_APPLY, proto.NginxConfigAction_FORCE:
			if n.isFeatureNginxConfigEnabled {
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
	status = &proto.Command_NginxConfigResponse{
		NginxConfigResponse: &proto.NginxConfigResponse{
			Status:     newOKStatus(configAppliedProcessedResponse).CmdStatus,
			Action:     proto.NginxConfigAction_APPLY,
			ConfigData: cfg.NginxConfig.ConfigData,
		},
	}

	config, err := n.cmdr.Download(context.Background(), cmd.GetMeta())
	if err != nil {
		status.NginxConfigResponse.Status = newErrStatus("Config apply failed (download): " + err.Error()).CmdStatus
		return status
	}

	if err != nil {
		status.NginxConfigResponse.Status = newErrStatus("Config apply failed: " + err.Error()).CmdStatus
		return status
	}

	status = n.writeConfigAndReloadNginx(cmd.Meta.MessageId, config, cmd.GetNginxConfig().GetAction())

	log.Debug("Config Apply Complete")
	return status
}

func (n *Nginx) writeConfigAndReloadNginx(correlationId string, config *proto.NginxConfig, action proto.NginxConfigAction) *proto.Command_NginxConfigResponse {
	status := &proto.Command_NginxConfigResponse{
		NginxConfigResponse: &proto.NginxConfigResponse{
			Status:     newOKStatus(configAppliedProcessedResponse).CmdStatus,
			Action:     proto.NginxConfigAction_APPLY,
			ConfigData: config.ConfigData,
		},
	}

	n.messagePipeline.Process(core.NewMessage(core.NginxConfigValidationPending, &proto.AgentActivityStatus{
		Status: &proto.AgentActivityStatus_NginxConfigStatus{
			NginxConfigStatus: &proto.NginxConfigStatus{
				CorrelationId: correlationId,
				Status:        proto.NginxConfigStatus_PENDING,
				Message:       "config apply pending",
				NginxId:       config.GetConfigData().GetNginxId(),
			},
		},
	}))

	if config.GetConfigData().GetNginxId() == "" {
		status.NginxConfigResponse.Status = newErrStatus(fmt.Sprintf("Config apply failed (preflight): no Nginx Id in ConfigDescriptor %v", config.GetConfigData())).CmdStatus
		return status
	}

	if action == proto.NginxConfigAction_APPLY {
		configValid, err := n.ValidateNginxAppProtectVersion(config)
		if !configValid {
			status.NginxConfigResponse.Status = newErrStatus(err.Error()).CmdStatus
			return status
		}
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
				correlationId: correlationId,
				timestamp:     types.TimestampNow(),
				nginxDetails:  nginx,
			}
			n.messagePipeline.Process(core.NewMessage(core.ConfigRollbackResponse, configRollbackResponse))
		}

		message := fmt.Sprintf("Config apply failed (write): " + err.Error())
		return n.handleErrorStatus(status, message)
	}

	go n.validateConfig(nginx, correlationId, config, configApply)

	// If the NGINX config can be validated with the validationTimeout the result will be returned straight away.
	// This is timeout is temporary to ensure we support backwards compatibility. In a future release this timeout
	// will be removed.
	select {
	case result := <-n.configApplyStatusChannel:
		return result
	case <-time.After(validationTimeout):
		log.Warnf("Validation of the NGINX config in taking longer than the validationTimeout %s", validationTimeout)
		return status
	}
}

// This function will run a nginx config validation in a separate go routine. If the validation takes less than 15 seconds then the result is returned straight away,
// otherwise nil is returned and the validation continues on in the background until it is complete. The result is always added to the message pipeline for other plugins
// to use.
func (n *Nginx) validateConfig(nginx *proto.NginxDetails, correlationId string, config *proto.NginxConfig, configApply *sdk.ConfigApply) {
	start := time.Now()

	err := n.nginxBinary.ValidateConfig(nginx.NginxId, nginx.ProcessPath, nginx.ConfPath, config, configApply)
	if err == nil {
		_, err = n.nginxBinary.ReadConfig(nginx.GetConfPath(), config.GetConfigData().GetNginxId(), n.env.GetSystemUUID())
	}

	elapsedTime := time.Since(start)
	log.Tracef("nginx config validation took %s to complete", elapsedTime)

	if err != nil {
		response := &NginxConfigValidationResponse{
			err:           fmt.Errorf("error running nginx -t -c %s:\n %v", nginx.ConfPath, err),
			correlationId: correlationId,
			nginxDetails:  nginx,
			config:        config,
			configApply:   configApply,
			elapsedTime:   elapsedTime,
		}
		n.messagePipeline.Process(core.NewMessage(core.NginxConfigValidationFailed, response))
	} else {
		response := &NginxConfigValidationResponse{
			err:           nil,
			correlationId: correlationId,
			nginxDetails:  nginx,
			config:        config,
			configApply:   configApply,
			elapsedTime:   elapsedTime,
		}
		n.messagePipeline.Process(core.NewMessage(core.NginxConfigValidationSucceeded, response))
	}
}

func (n *Nginx) completeConfigApply(response *NginxConfigValidationResponse) (status *proto.Command_NginxConfigResponse) {
	nginxConfigStatusMessage := configAppliedResponse
	rollback := false

	var rollbackError error
	// get the process info before reload
	// so we can compare against after reload
	processInfo := n.getNginxProccessInfo()

	reloadErr := n.nginxBinary.Reload(response.nginxDetails.ProcessId, response.nginxDetails.ProcessPath)
	if reloadErr != nil {
		nginxConfigStatusMessage = fmt.Sprintf("Config apply failed (write): %v", reloadErr)
		log.Errorf(nginxConfigStatusMessage)
		rollbackError = reloadErr
		rollback = true
	} else {
		rollbackError = n.monitor(processInfo)
		if rollbackError != nil {
			nginxConfigStatusMessage = fmt.Sprintf("Config apply failed. Errors found during monitoring period after applying a new configuration: %v", rollbackError)
			rollback = true
		} else {
			log.Info("No errors found in NGINX errors logs after NGINX reload")
		}
	}

	nginxReloadEventMeta := NginxReloadResponse{
		succeeded:     reloadErr == nil,
		correlationId: response.correlationId,
		timestamp:     types.TimestampNow(),
		nginxDetails:  response.nginxDetails,
	}

	n.messagePipeline.Process(core.NewMessage(core.NginxReloadComplete, nginxReloadEventMeta))

	if rollback {
		go n.rollbackConfigApply(response.correlationId, response.config.GetConfigData().GetNginxId(), response.nginxDetails, response.configApply, rollbackError)

		status = &proto.Command_NginxConfigResponse{
			NginxConfigResponse: &proto.NginxConfigResponse{
				Status:     newErrStatus(nginxConfigStatusMessage).CmdStatus,
				Action:     proto.NginxConfigAction_APPLY,
				ConfigData: response.config.ConfigData,
			},
		}
	} else {
		if response.configApply != nil {
			if err := response.configApply.Complete(); err != nil {
				nginxConfigStatusMessage = fmt.Sprintf("Config complete failed: %v", err)
				log.Errorf(nginxConfigStatusMessage)
			}
		}

		// Upload NGINX config only if GPRC server is configured
		if n.config.IsGrpcServerConfigured() {
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
		}

		log.Debug("Enabling file watcher")
		n.messagePipeline.Process(core.NewMessage(core.FileWatcherEnabled, true))

		agentActivityStatus := &proto.AgentActivityStatus{
			Status: &proto.AgentActivityStatus_NginxConfigStatus{
				NginxConfigStatus: &proto.NginxConfigStatus{
					CorrelationId: response.correlationId,
					Status:        proto.NginxConfigStatus_OK,
					Message:       nginxConfigStatusMessage,
					NginxId:       response.config.GetConfigData().GetNginxId(),
				},
			},
		}

		n.messagePipeline.Process(core.NewMessage(core.NginxConfigApplySucceeded, agentActivityStatus))

		status = &proto.Command_NginxConfigResponse{
			NginxConfigResponse: &proto.NginxConfigResponse{
				Status:     newOKStatus(nginxConfigStatusMessage).CmdStatus,
				Action:     proto.NginxConfigAction_APPLY,
				ConfigData: response.config.ConfigData,
			},
		}
	}

	log.Debug("Config Apply Complete")

	return status
}

func (n *Nginx) monitor(processInfo []core.Process) error {
	log.Info("Monitoring post reload for changes")
	n.monitorMutex.Lock()
	defer n.monitorMutex.Unlock()
	var errorsFound []string
	errorLogs := n.nginxBinary.GetErrorLogs()

	logErrorChannel := make(chan string, len(errorLogs))
	defer close(logErrorChannel)

	go n.monitorLogs(errorLogs, logErrorChannel)

	// Expect to receive one message from a message for each NGINX error log file in the logErrorChannel
	numberOfExpectedMessages := len(errorLogs)

	for i := 0; i < numberOfExpectedMessages; i++ {
		err := <-logErrorChannel
		log.Tracef("message received in logErrorChannel: %s", err)
		if err != "" {
			errorsFound = append(errorsFound, err)
		}
	}

	log.Info("Finished monitoring post reload")

	if len(errorsFound) > 0 {
		return errors.New(errorsFound[0])
	}

	return nil
}

func (n *Nginx) monitorLogs(errorLogs map[string]string, errorChannel chan string) {
	if len(errorLogs) == 0 {
		log.Trace("No logs so returning monitoring of logs")
		return
	}

	for logFile := range errorLogs {
		go n.tailLog(logFile, errorChannel)
	}
}

func (n *Nginx) tailLog(logFile string, errorChannel chan string) {
	t, err := tailer.NewTailer(logFile)
	if err != nil {
		log.Errorf("Unable to tail error log %q after NGINX reload: %v", logFile, err)
		// this is not an error in the logs, ignoring tailing
		errorChannel <- ""
		return
	}

	ctx, cncl := context.WithTimeout(context.Background(), n.config.Nginx.ConfigReloadMonitoringPeriod)
	defer cncl()

	data := make(chan string, 1024)
	go t.Tail(ctx, data)

	tick := time.NewTicker(n.config.Nginx.ConfigReloadMonitoringPeriod)
	defer tick.Stop()

	for {
		select {
		case d := <-data:
			for _, errorRegex := range reloadErrorList {
				if errorRegex.MatchString(d) {
					errorChannel <- d
					return
				}
			}
		case <-tick.C:
			errorChannel <- ""
			return
		}
	}
}

func (n *Nginx) rollbackConfigApply(correlationId string, nginxId string, nginxDetails *proto.NginxDetails, configApply *sdk.ConfigApply, err error) {
	nginxConfigStatusMessage := fmt.Sprintf("Config apply failed (write): %v", err.Error())
	log.Error(nginxConfigStatusMessage)

	if configApply != nil {
		succeeded := true

		if rollbackErr := configApply.Rollback(err); rollbackErr != nil {
			nginxConfigStatusMessage := fmt.Sprintf("Config rollback failed: %v", rollbackErr)
			log.Error(nginxConfigStatusMessage)
			succeeded = false
		}

		configRollbackResponse := ConfigRollbackResponse{
			succeeded:     succeeded,
			correlationId: correlationId,
			timestamp:     types.TimestampNow(),
			nginxDetails:  nginxDetails,
		}

		n.messagePipeline.Process(core.NewMessage(core.ConfigRollbackResponse, configRollbackResponse))

		agentActivityStatus := &proto.AgentActivityStatus{
			Status: &proto.AgentActivityStatus_NginxConfigStatus{
				NginxConfigStatus: &proto.NginxConfigStatus{
					CorrelationId: correlationId,
					Status:        proto.NginxConfigStatus_ERROR,
					Message:       nginxConfigStatusMessage,
					NginxId:       nginxId,
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
	n.syncProcessInfo(n.env.Processes())
	for nginxID := range n.nginxBinary.GetNginxDetailsMapFromProcesses(n.getNginxProccessInfo()) {
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

	n.isFeatureNginxConfigEnabled = conf.IsFeatureEnabled(agent_config.FeatureNginxConfig)

	n.config = conf
}

func (n *Nginx) ValidateNginxAppProtectVersion(nginxConfig *proto.NginxConfig) (bool, error) {
	if isFileInDirectoryMap(nginxConfig.GetDirectoryMap(), n.nginxAppProtectSoftwareDetails.GetWafLocation()) {
		if aux := nginxConfig.GetZaux(); aux != nil && len(aux.Contents) > 0 {
			auxFiles, err := zip.UnPack(aux)
			if err != nil {
				return false, fmt.Errorf("config apply failed (preflight): not able to read unpack aux files %v", nginxConfig.GetZaux())
			}
			for _, file := range auxFiles {
				if filepath.Base(file.GetName()) == filepath.Base(n.nginxAppProtectSoftwareDetails.GetWafLocation()) {
					var napMetdata nap.Metadata
					err := json.Unmarshal(file.GetContents(), &napMetdata)
					if err != nil {
						return false, fmt.Errorf("config apply failed (preflight): not able to read WAF file in metadata %v", nginxConfig.GetConfigData())
					}
					if napMetdata.NapVersion != "" && n.nginxAppProtectSoftwareDetails.GetWafVersion() != napMetdata.NapVersion {
						return false, fmt.Errorf("config apply failed (preflight): config metadata mismatch %v", nginxConfig.GetConfigData())
					}
				}
			}
		}
	}

	return true, nil
}

func isFileInDirectoryMap(directoryMap *proto.DirectoryMap, path string) bool {
	if directoryMap == nil {
		return false
	}
	if (directoryMap != &proto.DirectoryMap{}) {
		for _, directory := range directoryMap.Directories {
			for _, file := range directory.GetFiles() {
				if filepath.Base(file.GetName()) == filepath.Base(path) {
					return true
				}
			}
		}
	}
	return false
}
