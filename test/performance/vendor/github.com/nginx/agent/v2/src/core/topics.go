/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

const (
	UNKNOWN                         = "unknown"
	RegistrationPrefix              = "registration."
	RegistrationCompletedTopic      = RegistrationPrefix + "completed"
	CommNginxConfig                 = "nginx.config"
	NginxConfigUpload               = "nginx.config.upload"
	NginxReload                     = "nginx.reload"
	NginxReloadComplete             = "nginx.reload.complete"
	NginxStart                      = "nginx.start"
	NginxStop                       = "nginx.stop"
	NginxPluginConfigured           = "nginx.plugin.config"
	NginxStatusAPIUpdate            = "nginx.status.api.update"
	NginxInstancesFound             = "nginx.instances.found"
	NginxMasterProcCreated          = "nginx.master.created"
	NginxMasterProcKilled           = "nginx.master.killed"
	NginxWorkerProcCreated          = "nginx.worker.created"
	NginxWorkerProcKilled           = "nginx.worker.killed"
	NginxDetailProcUpdate           = "nginx.proc.update"
	NginxConfigValidationPending    = "nginx.config.validation.pending"
	NginxConfigValidationFailed     = "nginx.config.validation.failed"
	NginxConfigValidationSucceeded  = "nginx.config.validation.succeeded"
	NginxConfigApplyFailed          = "nginx.config.apply.failed"
	NginxConfigApplySucceeded       = "nginx.config.apply.succeeded"
	CommPrefix                      = "comms."
	CommStatus                      = CommPrefix + "status"
	CommMetrics                     = CommPrefix + "metrics"
	CommRegister                    = CommPrefix + "register"
	CommResponse                    = CommPrefix + "response"
	AgentStarted                    = "agent.started"
	AgentConnected                  = "agent.connected"
	AgentConfig                     = "agent.config"
	AgentConfigChanged              = "agent.config.changed"
	AgentCollectorsUpdate           = "agent.collectors.update"
	MetricReport                    = "metrics.report"
	LoggerPrefix                    = "logger."
	LoggerLevel                     = LoggerPrefix + "level"
	LoggerPath                      = LoggerPrefix + "path"
	DataplaneChanged                = "dataplane.changed"
	DataplaneFilesChanged           = "dataplane.fileschanged"
	Events                          = "events"
	FileWatcherEnabled              = "file.watcher.enabled"
	ConfigRollbackResponse          = "config.rollback.response"
	DataplaneSoftwareDetailsUpdated = "dataplane.software.details.updated"
	EnableExtension                 = "enable.extension"
	AgentAPIConfigApplyResponse     = "agent.api.config.apply.response"
)
