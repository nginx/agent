// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"encoding/json"
	"log/slog"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	response "github.com/nginx/agent/v3/internal/datasource/proto"
	"github.com/nginx/agent/v3/internal/logger"
)

const emptyResponse = "{}"

type APIAction struct {
	NginxService nginxServiceInterface
}

func (a *APIAction) HandleUpdateStreamServersRequest(ctx context.Context, action *mpi.NGINXPlusAction,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()

	add, update, del, err := a.NginxService.UpdateStreamServers(ctx, instance,
		action.GetUpdateStreamServers().GetUpstreamStreamName(), action.GetUpdateStreamServers().GetServers())
	if err != nil {
		slog.ErrorContext(ctx, "Unable to update stream servers of upstream", "request",
			action.GetUpdateHttpUpstreamServers(), "error", err)

		return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, err.Error())
	}

	slog.DebugContext(ctx, "Successfully updated stream upstream servers", "http_upstream_name",
		action.GetUpdateHttpUpstreamServers().GetHttpUpstreamName(), "add", len(add), "update", len(update),
		"delete", len(del))

	return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		"Successfully updated stream upstream servers", instanceID, "")
}

func (a *APIAction) HandleGetStreamUpstreamsRequest(ctx context.Context,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()
	streamUpstreamsResponse := emptyResponse

	streamUpstreams, err := a.NginxService.GetStreamUpstreams(ctx, instance)
	if err != nil {
		slog.ErrorContext(ctx, "Unable to get stream upstreams", "error", err)
		return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, err.Error())
	}

	if streamUpstreams != nil {
		streamUpstreamsJSON, jsonErr := json.Marshal(streamUpstreams)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Unable to marshal stream upstreams", "err", err)
		}
		streamUpstreamsResponse = string(streamUpstreamsJSON)
	}

	return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		streamUpstreamsResponse, instanceID, "")
}

func (a *APIAction) HandleGetUpstreamsRequest(ctx context.Context, instance *mpi.Instance) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()
	upstreamsResponse := emptyResponse

	upstreams, err := a.NginxService.GetUpstreams(ctx, instance)
	if err != nil {
		slog.InfoContext(ctx, "Unable to get upstreams", "error", err)

		return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, err.Error())
	}

	if upstreams != nil {
		upstreamsJSON, jsonErr := json.Marshal(upstreams)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Unable to marshal upstreams", "err", err)
		}
		upstreamsResponse = string(upstreamsJSON)
	}

	return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		upstreamsResponse, instanceID, "")
}

func (a *APIAction) HandleUpdateHTTPUpstreamsRequest(ctx context.Context, action *mpi.NGINXPlusAction,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()

	add, update, del, err := a.NginxService.UpdateHTTPUpstreamServers(ctx, instance,
		action.GetUpdateHttpUpstreamServers().GetHttpUpstreamName(),
		action.GetUpdateHttpUpstreamServers().GetServers())
	if err != nil {
		slog.ErrorContext(ctx, "Unable to update HTTP servers of upstream", "request",
			action.GetUpdateHttpUpstreamServers(), "error", err)

		return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, err.Error())
	}

	slog.DebugContext(ctx, "Successfully updated http upstream servers", "http_upstream_name",
		action.GetUpdateHttpUpstreamServers().GetHttpUpstreamName(), "add", len(add), "update", len(update),
		"delete", len(del))

	return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		"Successfully updated HTTP Upstreams", instanceID, "")
}

func (a *APIAction) HandleGetHTTPUpstreamsServersRequest(ctx context.Context, action *mpi.NGINXPlusAction,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()
	upstreamsResponse := emptyResponse

	upstreams, err := a.NginxService.GetHTTPUpstreamServers(ctx, instance,
		action.GetGetHttpUpstreamServers().GetHttpUpstreamName())
	if err != nil {
		slog.ErrorContext(ctx, "Unable to get HTTP servers of upstream", "error", err)
		return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, err.Error())
	}

	if upstreams != nil {
		upstreamsJSON, jsonErr := json.Marshal(upstreams)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Unable to marshal http upstreams", "err", err)
		}
		upstreamsResponse = string(upstreamsJSON)
	}

	return response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		upstreamsResponse, instanceID, "")
}
