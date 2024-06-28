// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package bus

const (
	AddInstancesTopic                 = "add-instances"
	UpdatedInstancesTopic             = "updated-instances"
	DeletedInstancesTopic             = "deleted-instances"
	ResourceUpdateTopic               = "resource-update"
	NginxConfigUpdateTopic            = "nginx-config-update"
	InstanceHealthTopic               = "instance-health"
	ConfigUploadRequestTopic          = "config-upload-request"
	DataPlaneResponseTopic            = "data-plane-response"
	ConnectionCreatedTopic            = "connection-created"
	ConfigApplyRequestTopic           = "config-apply-topic"
	WriteConfigSuccessfulTopic        = "write-config-successful-topic"
	ConfigApplySuccessfulRequestTopic = "config-apply-successful-topic"
	ConfigApplyFailedRequestTopic     = "config-apply-failed-topic"
)
