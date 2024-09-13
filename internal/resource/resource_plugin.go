// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/nginx/agent/v3/internal/bus"
)

// The resource plugin listens for a writeConfigSuccessfulTopic from the file plugin after the config apply
// files have been written. The resource plugin then, validates the config, reloads the instance and monitors the logs.
// This is done in the resource plugin to make the file plugin usable for every type of instance.

type Resource struct {
	messagePipe     bus.MessagePipeInterface
	resourceService resourceServiceInterface
	agentConfig     *config.Config
}

var _ bus.Plugin = (*Resource)(nil)

func NewResource(agentConfig *config.Config) *Resource {
	return &Resource{
		agentConfig: agentConfig,
	}
}

func (r *Resource) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting resource plugin")

	r.messagePipe = messagePipe
	r.resourceService = NewResourceService(ctx, r.agentConfig)

	return nil
}

func (*Resource) Close(ctx context.Context) error {
	slog.DebugContext(ctx, "Closing resource plugin")
	return nil
}

func (*Resource) Info() *bus.Info {
	return &bus.Info{
		Name: "resource",
	}
}

// cyclomatic complexity 11 max is 10
// nolint: revive, cyclop
func (r *Resource) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.AddInstancesTopic:
		instanceList, ok := msg.Data.([]*mpi.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

			return
		}

		resource := r.resourceService.AddInstances(instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})

		return
	case bus.UpdatedInstancesTopic:
		instanceList, ok := msg.Data.([]*mpi.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

			return
		}
		resource := r.resourceService.UpdateInstances(instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})

		return

	case bus.DeletedInstancesTopic:
		instanceList, ok := msg.Data.([]*mpi.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

			return
		}
		resource := r.resourceService.DeleteInstances(instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})

		return
	case bus.WriteConfigSuccessfulTopic:
		r.handleWriteConfigSuccessful(ctx, msg)
	case bus.RollbackWriteTopic:
		r.handleRollbackWrite(ctx, msg)
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (*Resource) Subscriptions() []string {
	return []string{
		bus.AddInstancesTopic,
		bus.UpdatedInstancesTopic,
		bus.DeletedInstancesTopic,
		bus.WriteConfigSuccessfulTopic,
		bus.RollbackWriteTopic,
	}
}

func (r *Resource) handleWriteConfigSuccessful(ctx context.Context, msg *bus.Message) {
	data, ok := msg.Data.(*model.ConfigApplyMessage)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplyMessage", "payload", msg.Data)

		return
	}
	err := r.resourceService.ApplyConfig(ctx, data.InstanceID)
	if err != nil {
		data.Error = err
		slog.Error("errors found during config apply, sending error status, rolling back config", "err", err)
		response := r.createDataPlaneResponse(data.CorrelationID, mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Config apply failed, rolling back config", data.InstanceID, err.Error())
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyFailedTopic, Data: data})

		return
	}
	response := r.createDataPlaneResponse(data.CorrelationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		"Config apply successful", data.InstanceID, "")

	instance := r.resourceService.Instance(data.InstanceID)

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
	r.messagePipe.Process(
		ctx,
		&bus.Message{
			Topic: bus.ConfigApplySuccessfulTopic,
			Data:  instance.GetInstanceMeta().GetInstanceId(),
		},
	)
}

func (r *Resource) handleRollbackWrite(ctx context.Context, msg *bus.Message) {
	data, ok := msg.Data.(*model.ConfigApplyMessage)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplyMessage", "payload", msg.Data)

		return
	}
	err := r.resourceService.ApplyConfig(ctx, data.InstanceID)
	if err != nil {
		slog.Error("errors found during rollback, sending failure status", "err", err)

		rollbackResponse := r.createDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR, "Rollback failed", data.InstanceID, err.Error())

		applyResponse := r.createDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE, "Config apply failed, rollback failed",
			data.InstanceID, data.Error.Error())

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: applyResponse})
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.RollbackCompleteTopic, Data: data.InstanceID})

		return
	}

	applyResponse := r.createDataPlaneResponse(data.CorrelationID,
		mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		"Config apply failed, rollback successful", data.InstanceID, data.Error.Error())

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: applyResponse})
	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.RollbackCompleteTopic, Data: data.InstanceID})
}

func (*Resource) createDataPlaneResponse(correlationID string, status mpi.CommandResponse_CommandStatus,
	message, instanceID, err string,
) *mpi.DataPlaneResponse {
	return &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  status,
			Message: message,
			Error:   err,
		},
		InstanceId: instanceID,
	}
}
