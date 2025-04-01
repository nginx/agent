// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"fmt"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
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

func GetAgentInstance(processID int32, agentConfig *config.Config) *mpi.Instance {
	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   agentConfig.UUID,
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_AGENT,
			Version:      agentConfig.Version,
		},
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/run/nginx-agent",
			ConfigPath: agentConfig.Path,
		},
	}
}

func GetNginxOssInstance(expectedModules []string) *mpi.Instance {
	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   ossInstanceID,
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX,
			Version:      "1.25.3",
		},
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
			ConfigPath: "/usr/local/etc/nginx/nginx.conf",
			Details: &mpi.InstanceRuntime_NginxRuntimeInfo{
				NginxRuntimeInfo: &mpi.NGINXRuntimeInfo{
					StubStatus: &mpi.APIDetails{
						Location: "",
						Listen:   "",
					},
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
			InstanceChildren: []*mpi.InstanceChild{{ProcessId: childID}, {ProcessId: childID2}},
		},
	}
}

func GetNginxPlusInstance(expectedModules []string) *mpi.Instance {
	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   plusInstanceID,
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS,
			Version:      "1.25.3 (nginx-plus-r31-p1)",
		},
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
			ConfigPath: "/etc/nginx/nginx.conf",
			Details: &mpi.InstanceRuntime_NginxPlusRuntimeInfo{
				NginxPlusRuntimeInfo: &mpi.NGINXPlusRuntimeInfo{
					StubStatus: &mpi.APIDetails{
						Location: "",
						Listen:   "",
					},
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
					PlusApi: &mpi.APIDetails{
						Location: "",
						Listen:   "",
					},
				},
			},
			InstanceChildren: []*mpi.InstanceChild{{ProcessId: childID}, {ProcessId: childID2}},
		},
	}
}

func GetUnsupportedInstance() *mpi.Instance {
	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   unsuportedInstanceID,
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_UNIT,
			Version:      "",
		},
		InstanceConfig:  nil,
		InstanceRuntime: nil,
	}
}

func GetMultipleInstances(expectedModules []string) []*mpi.Instance {
	process1 := GetNginxOssInstance(expectedModules)
	process2 := getSecondNginxOssInstance(expectedModules)

	return []*mpi.Instance{process1, process2}
}

func GetInstancesNoParentProcess(expectedModules []string) []*mpi.Instance {
	process1 := GetNginxOssInstance(expectedModules)
	process1.GetInstanceRuntime().ProcessId = 0

	process2 := getSecondNginxOssInstance(expectedModules)
	process2.GetInstanceRuntime().ProcessId = 0

	return []*mpi.Instance{process1, process2}
}

func getSecondNginxOssInstance(expectedModules []string) *mpi.Instance {
	process2 := GetNginxOssInstance(expectedModules)
	process2.GetInstanceRuntime().ProcessId = processID2
	process2.GetInstanceMeta().InstanceId = secondOssInstanceID
	process2.GetInstanceRuntime().BinaryPath = "/opt/homebrew/etc/nginx/1.25.3/bin/nginx"
	process2.GetInstanceRuntime().InstanceChildren = []*mpi.InstanceChild{{ProcessId: childID3}, {ProcessId: childID4}}

	return process2
}

func GetHealthyInstanceHealth() *mpi.InstanceHealth {
	return &mpi.InstanceHealth{
		InstanceId:           GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
		InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
	}
}

func GetUnhealthyInstanceHealth() *mpi.InstanceHealth {
	return &mpi.InstanceHealth{
		InstanceId:           GetNginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
		InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
	}
}

func GetUnspecifiedInstanceHealth() *mpi.InstanceHealth {
	return &mpi.InstanceHealth{
		InstanceId:           unsuportedInstanceID,
		InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNSPECIFIED,
		Description: fmt.Sprintf("failed to get health for instance %s, error: unable "+
			"to determine health", unsuportedInstanceID),
	}
}
