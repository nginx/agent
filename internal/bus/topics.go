// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package bus

const (
	InstanceConfigUpdateRequestTopic = "instance-config-update-request"
	InstanceConfigUpdateStatusTopic  = "instance-config-update-status"
	InstanceConfigContextTopic       = "instance-config-context"
	MetricsTopic                     = "metrics"
	ConfigClientTopic                = "config-client"
	AddInstancesTopic                = "add-instances"
	UpdatedInstancesTopic            = "updated-instances"
	DeletedInstancesTopic            = "deleted-instances"
	ResourceUpdateTopic              = "resource-update"
	NginxConfigUpdateTopic           = "nginx-config-update"
	InstanceHealthTopic              = "instance-health"
	ConfigUploadRequestTopic         = "config-upload-request"
	DataPlaneResponseTopic           = "data-plane-response"
	ConfigApplyRequestTopic          = "config-apply-topic"
)
