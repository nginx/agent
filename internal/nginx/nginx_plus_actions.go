// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

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

func (a *APIAction) HandleGetStreamUpstreamsRequest(ctx context.Context,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	return a.handleUpstreamGetRequest(
		ctx,
		instance,
		func(ctx context.Context, instance *mpi.Instance) (interface{}, error) {
			return a.NginxService.GetStreamUpstreams(ctx, instance)
		},
		"Unable to get stream upstreams",
	)
}

func (a *APIAction) HandleGetUpstreamsRequest(ctx context.Context, instance *mpi.Instance) *mpi.DataPlaneResponse {
	return a.handleUpstreamGetRequest(
		ctx,
		instance,
		func(ctx context.Context, instance *mpi.Instance) (interface{}, error) {
			return a.NginxService.GetUpstreams(ctx, instance)
		},
		"Unable to get upstreams",
	)
}

func (a *APIAction) HandleGetHTTPUpstreamsServersRequest(
	ctx context.Context,
	action *mpi.NGINXPlusAction,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()
	upstreamsResponse := emptyResponse

	upstreams, err := a.NginxService.GetHTTPUpstreamServers(
		ctx,
		instance,
		action.GetGetHttpUpstreamServers().GetHttpUpstreamName(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "Unable to get HTTP servers of upstream", "error", err)
		return response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_API_ACTION_REQUEST,
			instanceID,
		)
	}

	if upstreams != nil {
		upstreamsJSON, jsonErr := json.Marshal(upstreams)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Unable to marshal http upstreams", "error", jsonErr)
		}
		upstreamsResponse = string(upstreamsJSON)
	}

	return response.CreateDataPlaneResponse(
		correlationID,
		&mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: upstreamsResponse,
			Error:   "",
		},
		mpi.DataPlaneResponse_API_ACTION_REQUEST,
		instanceID,
	)
}

//nolint:dupl // Having common code duplicated for clarity and ease of maintenance
func (a *APIAction) HandleUpdateStreamServersRequest(
	ctx context.Context,
	action *mpi.NGINXPlusAction,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()

	add, update, del, err := a.NginxService.UpdateStreamServers(
		ctx,
		instance,
		action.GetUpdateStreamServers().GetUpstreamStreamName(),
		action.GetUpdateStreamServers().GetServers(),
	)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Unable to update stream servers of upstream",
			"request", action.GetUpdateStreamServers(),
			"error", err,
		)

		return response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_API_ACTION_REQUEST,
			instanceID,
		)
	}

	slog.DebugContext(
		ctx,
		"Successfully updated stream upstream servers",
		"http_upstream_name", action.GetUpdateHttpUpstreamServers().GetHttpUpstreamName(),
		"add", len(add),
		"update", len(update),
		"delete", len(del),
	)

	return response.CreateDataPlaneResponse(
		correlationID,
		&mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Successfully updated stream upstream servers",
			Error:   "",
		},
		mpi.DataPlaneResponse_API_ACTION_REQUEST,
		instanceID,
	)
}

//nolint:dupl // Having common code duplicated for clarity and ease of maintenance
func (a *APIAction) HandleUpdateHTTPUpstreamsRequest(
	ctx context.Context,
	action *mpi.NGINXPlusAction,
	instance *mpi.Instance,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()

	add, update, del, err := a.NginxService.UpdateHTTPUpstreamServers(ctx, instance,
		action.GetUpdateHttpUpstreamServers().GetHttpUpstreamName(),
		action.GetUpdateHttpUpstreamServers().GetServers())
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Unable to update HTTP servers of upstream",
			"request", action.GetUpdateHttpUpstreamServers(),
			"error", err,
		)

		return response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_API_ACTION_REQUEST,
			instanceID,
		)
	}

	slog.DebugContext(
		ctx,
		"Successfully updated http upstream servers",
		"http_upstream_name", action.GetUpdateHttpUpstreamServers().GetHttpUpstreamName(),
		"add", len(add),
		"update", len(update),
		"delete", len(del),
	)

	return response.CreateDataPlaneResponse(
		correlationID,
		&mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Successfully updated HTTP Upstreams",
			Error:   "",
		},
		mpi.DataPlaneResponse_API_ACTION_REQUEST,
		instanceID,
	)
}

// handleUpstreamGetRequest is a generic helper function to handle GET requests for API actions
func (a *APIAction) handleUpstreamGetRequest(
	ctx context.Context,
	instance *mpi.Instance,
	getData func(context.Context, *mpi.Instance) (interface{}, error),
	errorMsg string,
) *mpi.DataPlaneResponse {
	correlationID := logger.CorrelationID(ctx)
	instanceID := instance.GetInstanceMeta().GetInstanceId()
	jsonResponse := emptyResponse

	data, err := getData(ctx, instance)
	if err != nil {
		slog.ErrorContext(ctx, errorMsg, "error", err)
		return response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_API_ACTION_REQUEST,
			instanceID,
		)
	}

	if data != nil {
		dataJSON, jsonErr := json.Marshal(data)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Unable to marshal data", "error", jsonErr)
		}
		jsonResponse = string(dataJSON)
	}

	return response.CreateDataPlaneResponse(
		correlationID,
		&mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: jsonResponse,
			Error:   "",
		},
		mpi.DataPlaneResponse_API_ACTION_REQUEST,
		instanceID,
	)
}
