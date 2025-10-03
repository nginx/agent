// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	"reflect"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
)

func NginxPlusRuntimeInfoEqual(nginxPlusRuntimeInfo *mpi.NGINXPlusRuntimeInfo,
	nginxConfigContext *model.NginxConfigContext, accessLogs, errorLogs []string,
) bool {
	convertedPlusAPIs := convertToMpiAPIDetailsSlice(nginxConfigContext.PlusAPIs)

	if !reflect.DeepEqual(nginxPlusRuntimeInfo.GetPlusApis(), convertedPlusAPIs) {
		return false
	}

	if nginxPlusRuntimeInfo.GetPlusApi().GetWriteEnabled() != nginxConfigContext.PlusAPI.WriteEnabled ||
		nginxPlusRuntimeInfo.GetPlusApi().GetCa() != nginxConfigContext.PlusAPI.Ca {
		return false
	}

	if !reflect.DeepEqual(nginxPlusRuntimeInfo.GetAccessLogs(), accessLogs) ||
		!reflect.DeepEqual(nginxPlusRuntimeInfo.GetErrorLogs(), errorLogs) ||
		nginxPlusRuntimeInfo.GetStubStatus().GetListen() != nginxConfigContext.StubStatus.Listen ||
		nginxPlusRuntimeInfo.GetPlusApi().GetListen() != nginxConfigContext.PlusAPI.Listen ||
		nginxPlusRuntimeInfo.GetStubStatus().GetLocation() != nginxConfigContext.StubStatus.Location ||
		nginxPlusRuntimeInfo.GetPlusApi().GetLocation() != nginxConfigContext.PlusAPI.Location {
		return false
	}

	return true
}

func NginxRuntimeInfoEqual(nginxRuntimeInfo *mpi.NGINXRuntimeInfo, nginxConfigContext *model.NginxConfigContext,
	accessLogs, errorLogs []string,
) bool {
	if !reflect.DeepEqual(nginxRuntimeInfo.GetAccessLogs(), accessLogs) ||
		!reflect.DeepEqual(nginxRuntimeInfo.GetErrorLogs(), errorLogs) ||
		nginxRuntimeInfo.GetStubStatus().GetListen() != nginxConfigContext.StubStatus.Listen ||
		nginxRuntimeInfo.GetStubStatus().GetLocation() != nginxConfigContext.StubStatus.Location {
		return false
	}

	return true
}

func UpdateNginxInstanceRuntime(
	instance *mpi.Instance,
	nginxConfigContext *model.NginxConfigContext,
) (updatesRequired bool) {
	instanceType := instance.GetInstanceMeta().GetInstanceType()

	accessLogs := model.ConvertAccessLogs(nginxConfigContext.AccessLogs)
	errorLogs := model.ConvertErrorLogs(nginxConfigContext.ErrorLogs)

	if instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		nginxPlusRuntimeInfo := instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo()

		if !NginxPlusRuntimeInfoEqual(nginxPlusRuntimeInfo, nginxConfigContext, accessLogs, errorLogs) {
			nginxPlusRuntimeInfo.AccessLogs = accessLogs
			nginxPlusRuntimeInfo.ErrorLogs = errorLogs
			nginxPlusRuntimeInfo.StubStatus.Listen = nginxConfigContext.StubStatus.Listen
			nginxPlusRuntimeInfo.PlusApi.Listen = nginxConfigContext.PlusAPI.Listen
			nginxPlusRuntimeInfo.StubStatus.Location = nginxConfigContext.StubStatus.Location
			nginxPlusRuntimeInfo.PlusApi.Location = nginxConfigContext.PlusAPI.Location

			nginxPlusRuntimeInfo.PlusApi.WriteEnabled = nginxConfigContext.PlusAPI.WriteEnabled
			nginxPlusRuntimeInfo.PlusApi.Ca = nginxConfigContext.PlusAPI.Ca

			nginxPlusRuntimeInfo.PlusApis = convertToMpiAPIDetailsSlice(nginxConfigContext.PlusAPIs)
			updatesRequired = true
		}
	} else {
		nginxRuntimeInfo := instance.GetInstanceRuntime().GetNginxRuntimeInfo()

		if !NginxRuntimeInfoEqual(nginxRuntimeInfo, nginxConfigContext, accessLogs, errorLogs) {
			nginxRuntimeInfo.AccessLogs = accessLogs
			nginxRuntimeInfo.ErrorLogs = errorLogs
			nginxRuntimeInfo.StubStatus.Location = nginxConfigContext.StubStatus.Location
			nginxRuntimeInfo.StubStatus.Listen = nginxConfigContext.StubStatus.Listen
			updatesRequired = true
		}
	}

	return updatesRequired
}

func convertToMpiAPIDetails(modelAPI *model.APIDetails) *mpi.APIDetails {
	if modelAPI == nil {
		return nil
	}

	return &mpi.APIDetails{
		Listen:       modelAPI.Listen,
		Location:     modelAPI.Location,
		Ca:           modelAPI.Ca,
		WriteEnabled: modelAPI.WriteEnabled,
	}
}

func convertToMpiAPIDetailsSlice(modelAPIs []*model.APIDetails) []*mpi.APIDetails {
	if modelAPIs == nil {
		return nil
	}
	mpiAPIs := make([]*mpi.APIDetails, 0, len(modelAPIs))
	for _, api := range modelAPIs {
		mpiAPIs = append(mpiAPIs, convertToMpiAPIDetails(api))
	}

	return mpiAPIs
}
