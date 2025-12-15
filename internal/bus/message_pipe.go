// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package bus

import (
	"context"
	"log/slog"
	"reflect"
	"sync"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/pkg/id"
	messagebus "github.com/vardius/message-bus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	Payload interface{}

	Message struct {
		Data  Payload
		Topic string
	}

	MessageWithContext struct {
		ctx     context.Context
		message *Message
	}

	Info struct {
		Name string
	}

	MessagePipeInterface interface {
		Register(size int, plugins []Plugin) error
		DeRegister(ctx context.Context, plugins []string) error
		Process(ctx context.Context, messages ...*Message)
		Run(ctx context.Context)
		Plugins() []Plugin
		IsPluginRegistered(pluginName string) bool
	}

	Plugin interface {
		Init(ctx context.Context, messagePipe MessagePipeInterface) error
		Close(ctx context.Context) error
		Info() *Info
		Process(ctx context.Context, msg *Message)
		Subscriptions() []string
		Reconfigure(ctx context.Context, agentConfig *config.Config) error
	}

	MessagePipe struct {
		agentConfig    *config.Config
		bus            messagebus.MessageBus
		messageChannel chan *MessageWithContext
		plugins        []Plugin
		pluginsMutex   sync.Mutex
		configMutex    sync.Mutex
	}
)

func NewMessagePipe(size int, agentConfig *config.Config) *MessagePipe {
	return &MessagePipe{
		messageChannel: make(chan *MessageWithContext, size),
		pluginsMutex:   sync.Mutex{},
		agentConfig:    agentConfig,
	}
}

func (p *MessagePipe) Register(size int, plugins []Plugin) error {
	p.pluginsMutex.Lock()
	defer p.pluginsMutex.Unlock()

	p.plugins = append(p.plugins, plugins...)
	p.bus = messagebus.New(size)

	pluginsRegistered := []string{}

	for _, plugin := range p.plugins {
		for _, subscription := range plugin.Subscriptions() {
			err := p.bus.Subscribe(subscription, plugin.Process)
			if err != nil {
				return err
			}
		}
		pluginsRegistered = append(pluginsRegistered, plugin.Info().Name)
	}

	slog.Info("Finished registering plugins", "plugins", pluginsRegistered)

	return nil
}

func (p *MessagePipe) DeRegister(ctx context.Context, pluginNames []string) error {
	p.pluginsMutex.Lock()
	defer p.pluginsMutex.Unlock()

	plugins := p.findPlugins(pluginNames)

	for _, plugin := range plugins {
		index := p.Index(plugin.Info().Name, p.plugins)

		err := p.unsubscribePlugin(ctx, index, plugin)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *MessagePipe) Process(ctx context.Context, messages ...*Message) {
	for _, message := range messages {
		p.messageChannel <- &MessageWithContext{ctx, message}
	}
}

//nolint:contextcheck,revive // need to use context from the message for the correlationID not use the parent context
func (p *MessagePipe) Run(ctx context.Context) {
	p.pluginsMutex.Lock()
	p.initPlugins(ctx)
	p.pluginsMutex.Unlock()

	for {
		select {
		case <-ctx.Done():
			p.pluginsMutex.Lock()
			for _, r := range p.plugins {
				r.Close(ctx)
			}
			p.pluginsMutex.Unlock()

			return
		case m := <-p.messageChannel:
			if m.message != nil {
				switch m.message.Topic {
				case AgentConfigUpdateTopic:
					p.handleAgentConfigUpdateTopic(m.ctx, m.message)
				case ConnectionAgentConfigUpdateTopic:
					p.handleConnectionAgentConfigUpdateTopic(m.ctx, m.message)
				default:
					p.bus.Publish(m.message.Topic, m.ctx, m.message)
				}
			}
		}
	}
}

func (p *MessagePipe) Plugins() []Plugin {
	return p.plugins
}

func (p *MessagePipe) IsPluginRegistered(pluginName string) bool {
	isPluginRegistered := false

	for _, plugin := range p.Plugins() {
		if plugin.Info().Name == pluginName {
			isPluginRegistered = true
		}
	}

	return isPluginRegistered
}

func (p *MessagePipe) Index(pluginName string, plugins []Plugin) int {
	for index, plugin := range plugins {
		if pluginName == plugin.Info().Name {
			return index
		}
	}

	return -1
}

func (p *MessagePipe) Reconfigure(ctx context.Context, agentConfig *mpi.AgentConfig, topic, correlationID string) {
	var reconfigureError error
	p.configMutex.Lock()
	defer p.configMutex.Unlock()
	currentConfig := p.agentConfig

	// convert agent config from *mpi.AgentConfig to *config.Config
	updateAgentConfig := config.FromAgentRemoteConfigProto(agentConfig)

	// The check for updates to the config needs to be done here as the command plugin needs the latest agent config
	// to be sent in response to create connection requests.
	p.updateConfig(ctx, updateAgentConfig)

	// Reconfigure each plugin with the new agent config
	for _, plugin := range p.plugins {
		slog.DebugContext(ctx, "Reconfigure plugin", "plugin", plugin.Info().Name)
		reconfigureError = plugin.Reconfigure(ctx, p.agentConfig)
		if reconfigureError != nil {
			slog.ErrorContext(ctx, "Reconfigure plugin failed", "plugin", plugin.Info().Name)
			break
		}
	}

	if reconfigureError != nil {
		slog.ErrorContext(ctx, "Error updating plugin with updated agent config, reverting",
			"error", reconfigureError.Error())

		// If the agent update was received from a create connection request no data plane response needs to be sent
		if topic == AgentConfigUpdateTopic {
			response := p.createDataPlaneResponse(
				correlationID,
				mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				mpi.DataPlaneResponse_UPDATE_AGENT_CONFIG_REQUEST,
				"Failed to update agent config",
				reconfigureError.Error(),
			)
			p.bus.Publish(DataPlaneResponseTopic, ctx, &Message{Topic: DataPlaneResponseTopic, Data: response})
		}

		p.agentConfig = currentConfig
		for _, plugin := range p.plugins {
			err := plugin.Reconfigure(ctx, currentConfig)
			if err != nil {
				slog.ErrorContext(ctx, "Error reverting agent config", "error", err.Error())
			}
		}
	}

	slog.InfoContext(ctx, "Finished reconfiguring plugins", "plugins", p.plugins)
	if topic == AgentConfigUpdateTopic {
		response := p.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_OK,
			mpi.DataPlaneResponse_UPDATE_AGENT_CONFIG_REQUEST,
			"Successfully updated agent config",
			"",
		)
		p.bus.Publish(DataPlaneResponseTopic, ctx, &Message{Topic: DataPlaneResponseTopic, Data: response})
	}
}

func (p *MessagePipe) handleConnectionAgentConfigUpdateTopic(ctx context.Context, msg *Message) {
	slog.DebugContext(ctx, "Handling connection agent config update topic", "topic", msg.Topic)
	agentConfig, ok := msg.Data.(*mpi.AgentConfig)
	if !ok {
		slog.ErrorContext(ctx, "Failed to parse agent config update message")
		return
	}

	p.Reconfigure(ctx, agentConfig, msg.Topic, "")
}

func (p *MessagePipe) handleAgentConfigUpdateTopic(ctx context.Context, msg *Message) {
	slog.DebugContext(ctx, "Received agent config update topic", "topic", msg.Topic)
	mpRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)
	if !ok {
		slog.ErrorContext(ctx, "Failed to parse agent config update message")
		return
	}

	reconfigureRequest, ok := mpRequest.GetRequest().(*mpi.ManagementPlaneRequest_UpdateAgentConfigRequest)
	if !ok {
		slog.ErrorContext(ctx, "Failed to parse agent config update message")
		return
	}

	correlationID := reconfigureRequest.UpdateAgentConfigRequest.GetMessageMeta().GetCorrelationId()
	p.Reconfigure(ctx, reconfigureRequest.UpdateAgentConfigRequest.GetAgentConfig(), msg.Topic, correlationID)
}

func (p *MessagePipe) updateConfig(ctx context.Context, updateAgentConfig *config.Config) {
	slog.InfoContext(ctx, "Updating agent config")
	if updateAgentConfig.Log != nil && !reflect.DeepEqual(p.agentConfig.Log, updateAgentConfig.Log) {
		slog.InfoContext(ctx, "Agent log level has been updated", "previous", p.agentConfig.Log,
			"update", updateAgentConfig.Log)
		p.agentConfig.Log = updateAgentConfig.Log

		slogger := logger.New(
			p.agentConfig.Log.Path,
			p.agentConfig.Log.Level,
		)
		slog.SetDefault(slogger)
	}

	if updateAgentConfig.Labels != nil && validateLabels(updateAgentConfig.Labels) &&
		!reflect.DeepEqual(p.agentConfig.Labels, updateAgentConfig.Labels) {
		slog.InfoContext(ctx, "Agent labels have been updated", "previous", p.agentConfig.Labels,
			"update", updateAgentConfig.Labels)
		p.agentConfig.Labels = updateAgentConfig.Labels

		// OTel Headers also need to be updated when labels have been updated
		if p.agentConfig.Collector != nil {
			slog.DebugContext(ctx, "Agent OTel headers have been updated")
			config.AddLabelsAsOTelHeaders(p.agentConfig.Collector, updateAgentConfig.Labels)
		}
	}
	slog.DebugContext(ctx, "Updated agent config")
}

func (p *MessagePipe) unsubscribePlugin(ctx context.Context, index int, plugin Plugin) error {
	if index != -1 {
		p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)

		err := plugin.Close(ctx)
		if err != nil {
			return err
		}

		for _, subscription := range plugin.Subscriptions() {
			unsubErr := p.bus.Unsubscribe(subscription, plugin.Process)
			if unsubErr != nil {
				return unsubErr
			}
		}
	}

	return nil
}

func (p *MessagePipe) findPlugins(pluginNames []string) []Plugin {
	var plugins []Plugin

	for _, name := range pluginNames {
		for _, plugin := range p.plugins {
			if plugin.Info().Name == name {
				plugins = append(plugins, plugin)
			}
		}
	}

	return plugins
}

func (p *MessagePipe) initPlugins(ctx context.Context) {
	for index, plugin := range p.plugins {
		err := plugin.Init(ctx, p)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to initialize plugin", "plugin", plugin.Info().Name, "error", err)

			unsubscribeError := p.unsubscribePlugin(ctx, index, plugin)
			if unsubscribeError != nil {
				slog.ErrorContext(
					ctx,
					"Failed to unsubscribe plugin",
					"plugin", plugin.Info().Name,
					"error", unsubscribeError,
				)
			}
		}
	}
}

func (p *MessagePipe) createDataPlaneResponse(
	correlationID string,
	status mpi.CommandResponse_CommandStatus,
	requestType mpi.DataPlaneResponse_RequestType,
	message, err string,
) *mpi.DataPlaneResponse {
	return &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  status,
			Message: message,
			Error:   err,
		},
		RequestType: requestType,
	}
}

func validateLabels(labels map[string]any) bool {
	for _, value := range labels {
		if val, ok := value.(string); ok {
			if !config.ValidateLabel(val) {
				return false
			}
		}
	}

	return true
}
