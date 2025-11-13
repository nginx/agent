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
	NginxConfigUpdateTopic       = "nginx-config-update"
	InstanceHealthTopic          = "instance-health"
	ConfigUploadRequestTopic     = "config-upload-request"
	DataPlaneResponseTopic       = "data-plane-response"
	ConnectionCreatedTopic       = "connection-created"
	CredentialUpdatedTopic       = "credential-updated"
	ConnectionResetTopic         = "connection-reset"
	ConfigApplyRequestTopic      = "config-apply-request"
	WriteConfigSuccessfulTopic   = "write-config-successful"
	ReloadSuccessfulTopic        = "reload-successful"
	EnableWatchersTopic          = "enable-watchers"
	ConfigApplyFailedTopic       = "config-apply-failed"
	ConfigApplyCompleteTopic     = "config-apply-complete"
	RollbackWriteTopic           = "rollback-write"
	DataPlaneHealthRequestTopic  = "data-plane-health-request"
	DataPlaneHealthResponseTopic = "data-plane-health-response"
	APIActionRequestTopic        = "api-action-request"
	AgentConfigUpdateTopic       = "agent-config-update"
)
