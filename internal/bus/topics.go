// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package bus

const (
	AddInstancesTopic            = "add-instances"
	UpdatedInstancesTopic        = "updated-instances"
	DeletedInstancesTopic        = "deleted-instances"
	ResourceUpdateTopic          = "resource-update"
	TokenUpdateTopic             = "token-update"
	NginxConfigUpdateTopic       = "nginx-config-update"
	InstanceHealthTopic          = "instance-health"
	ConfigUploadRequestTopic     = "config-upload-request"
	DataPlaneResponseTopic       = "data-plane-response"
	ConnectionCreatedTopic       = "connection-created"
	CredentialUpdateTopic        = "credential-updated"
	ConfigApplyRequestTopic      = "config-apply-request"
	WriteConfigSuccessfulTopic   = "write-config-successful"
	ConfigApplySuccessfulTopic   = "config-apply-successful"
	ConfigApplyFailedTopic       = "config-apply-failed"
	ConfigApplyCompleteTopic     = "config-apply-complete"
	RollbackWriteTopic           = "rollback-write"
	DataPlaneHealthRequestTopic  = "data-plane-health-request"
	DataPlaneHealthResponseTopic = "data-plane-health-response"
	APIActionRequestTopic        = "api-action-request"
)
