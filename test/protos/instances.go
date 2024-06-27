// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"fmt"
	"os"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ossInstanceID        = "e1374cb1-462d-3b6c-9f3b-f28332b5f10c"
	plusInstanceID       = "40f9dda0-e45f-34cf-bba7-f173700f50a2"
	secondOssInstanceID  = "557cdf06-08fd-31eb-a8e7-daafd3a93db7"
	unsuportedInstanceID = "fcd99f8f-00fb-3097-8d75-32ae269b46c3"
	correlationID        = "dfsbhj6-bc92-30c1-a9c9-85591422068e"
	processID            = 1234
	processID2           = 5678
	childID              = 789
	childID2             = 567
	childID3             = 987
	childID4             = 321
)

func GetAgentInstance(processID int32, agentConfig *config.Config) *v1.Instance {
	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   agentConfig.UUID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
			Version:      agentConfig.Version,
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/run/nginx-agent",
			ConfigPath: agentConfig.Path,
		},
	}
}

func GetNginxOssInstance(expectedModules []string) *v1.Instance {
	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   ossInstanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
			Version:      "1.25.3",
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
			ConfigPath: "/usr/local/etc/nginx/nginx.conf",
			Details: &v1.InstanceRuntime_NginxRuntimeInfo{
				NginxRuntimeInfo: &v1.NGINXRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: expectedModules,
					DynamicModules: []string{
						"http_addition_module", "http_auth_request_module", "http_dav_module",
						"http_degradation_module", "http_flv_module", "http_gunzip_module", "http_gzip_static_module",
						"http_mp4_module", "http_random_index_module", "http_realip_module", "http_secure_link_module",
						"http_slice_module", "http_ssl_module", "http_stub_status_module", "http_sub_module",
						"http_v2_module", "mail_ssl_module", "stream_realip_module", "stream_ssl_module",
						"stream_ssl_preread_module",
					},
				},
			},
			InstanceChildren: []*v1.InstanceChild{{ProcessId: childID}, {ProcessId: childID2}},
		},
	}
}

func GetNginxPlusInstance(expectedModules []string) *v1.Instance {
	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   plusInstanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS,
			Version:      "1.25.3 (nginx-plus-r31-p1)",
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
			ConfigPath: "/etc/nginx/nginx.conf",
			Details: &v1.InstanceRuntime_NginxPlusRuntimeInfo{
				NginxPlusRuntimeInfo: &v1.NGINXPlusRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: expectedModules,
					DynamicModules: []string{
						"http_addition_module", "http_auth_jwt_module", "http_auth_request_module", "http_dav_module",
						"http_f4f_module", "http_flv_module", "http_gunzip_module", "http_gzip_static_module",
						"http_hls_module", "http_mp4_module", "http_proxy_protocol_vendor_module",
						"http_random_index_module", "http_realip_module", "http_secure_link_module",
						"http_session_log_module", "http_slice_module", "http_ssl_module", "http_stub_status_module",
						"http_sub_module", "http_v2_module", "http_v3_module", "mail_ssl_module",
						"stream_mqtt_filter_module", "stream_mqtt_preread_module",
						"stream_proxy_protocol_vendor_module", "stream_realip_module", "stream_ssl_module",
						"stream_ssl_preread_module",
					},
					PlusApi: "",
				},
			},
			InstanceChildren: []*v1.InstanceChild{{ProcessId: childID}, {ProcessId: childID2}},
		},
	}
}

func GetUnsupportedInstance() *v1.Instance {
	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   unsuportedInstanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_UNIT,
			Version:      "",
		},
		InstanceConfig:  nil,
		InstanceRuntime: nil,
	}
}

func GetMultipleInstances(expectedModules []string) []*v1.Instance {
	process1 := GetNginxOssInstance(expectedModules)
	process2 := getSecondNginxOssInstance(expectedModules)

	return []*v1.Instance{process1, process2}
}

func GetMultipleInstancesWithUnsupportedInstance() []*v1.Instance {
	process1 := GetNginxOssInstance([]string{})
	process2 := GetUnsupportedInstance()

	return []*v1.Instance{process1, process2}
}

func GetInstancesNoParentProcess(expectedModules []string) []*v1.Instance {
	process1 := GetNginxOssInstance(expectedModules)
	process1.GetInstanceRuntime().ProcessId = 0

	process2 := getSecondNginxOssInstance(expectedModules)
	process2.GetInstanceRuntime().ProcessId = 0

	return []*v1.Instance{process1, process2}
}

func CreateSuccessStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_SUCCESS,
		Message:       "Config applied successfully",
		Timestamp:     timestamppb.Now(),
	}
}

func CreateFailStatus(err string) *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_FAILED,
		Message:       err,
	}
}

func GetFileCache(files ...*os.File) (map[string]*v1.FileMeta, error) {
	cache := make(map[string]*v1.FileMeta)
	for _, file := range files {
		lastModified, err := CreateProtoTime("2024-01-09T13:22:21Z")
		if err != nil {
			return nil, err
		}

		cache[file.Name()] = &v1.FileMeta{
			ModifiedTime: lastModified,
			Name:         file.Name(),
			Hash:         "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		}
	}

	return cache, nil
}

func getSecondNginxOssInstance(expectedModules []string) *v1.Instance {
	process2 := GetNginxOssInstance(expectedModules)
	process2.GetInstanceRuntime().ProcessId = processID2
	process2.GetInstanceMeta().InstanceId = secondOssInstanceID
	process2.GetInstanceRuntime().BinaryPath = "/opt/homebrew/etc/nginx/1.25.3/bin/nginx"
	process2.GetInstanceRuntime().InstanceChildren = []*v1.InstanceChild{{ProcessId: childID3}, {ProcessId: childID4}}

	return process2
}

func GetHealthyInstanceHealth() *v1.InstanceHealth {
	return &v1.InstanceHealth{
		InstanceId:           GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
		InstanceHealthStatus: v1.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
	}
}

func GetUnhealthyInstanceHealth() *v1.InstanceHealth {
	return &v1.InstanceHealth{
		InstanceId:           GetNginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
		InstanceHealthStatus: v1.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
	}
}

func GetUnspecifiedInstanceHealth() *v1.InstanceHealth {
	return &v1.InstanceHealth{
		InstanceId:           unsuportedInstanceID,
		InstanceHealthStatus: v1.InstanceHealth_INSTANCE_HEALTH_STATUS_UNSPECIFIED,
		Description: fmt.Sprintf("failed to get health for instance %s, error: unable "+
			"to determine health", unsuportedInstanceID),
	}
}
