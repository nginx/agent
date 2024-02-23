# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [command.proto](#command-proto)
    - [ActionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ActionRequest)
    - [AgentConfig](#f5-nginx-agent-api-grpc-mpi-v1-AgentConfig)
    - [Auth](#f5-nginx-agent-api-grpc-mpi-v1-Auth)
    - [Command](#f5-nginx-agent-api-grpc-mpi-v1-Command)
    - [CommandStatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-CommandStatusRequest)
    - [ConfigApplyRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigApplyRequest)
    - [ConfigSyncRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigSyncRequest)
    - [ConfigUploadRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigUploadRequest)
    - [ConnectionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionRequest)
    - [ConnectionResponse](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse)
    - [ConnectionSettings](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionSettings)
    - [DataPlaneHealth](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth)
    - [DataPlaneRequest](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneRequest)
    - [DataPlaneStatus](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus)
    - [DefaultAction](#f5-nginx-agent-api-grpc-mpi-v1-DefaultAction)
    - [Exporter](#f5-nginx-agent-api-grpc-mpi-v1-Exporter)
    - [HealthRequest](#f5-nginx-agent-api-grpc-mpi-v1-HealthRequest)
    - [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance)
    - [InstanceAction](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction)
    - [InstanceConfig](#f5-nginx-agent-api-grpc-mpi-v1-InstanceConfig)
    - [InstanceHealth](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth)
    - [InstanceMeta](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta)
    - [KeepAlive](#f5-nginx-agent-api-grpc-mpi-v1-KeepAlive)
    - [ManagementPlaneRequest](#f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneRequest)
    - [Metrics](#f5-nginx-agent-api-grpc-mpi-v1-Metrics)
    - [NGINXConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXConfig)
    - [NGINXPlusConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXPlusConfig)
    - [Server](#f5-nginx-agent-api-grpc-mpi-v1-Server)
    - [Source](#f5-nginx-agent-api-grpc-mpi-v1-Source)
    - [StatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-StatusRequest)
    - [TLSSetting](#f5-nginx-agent-api-grpc-mpi-v1-TLSSetting)
  
    - [InstanceAction.InstanceActionType](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActionType)
    - [InstanceHealth.InstancHealthStatus](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth-InstancHealthStatus)
    - [InstanceMeta.InstanceType](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta-InstanceType)
    - [Server.ConnectionType](#f5-nginx-agent-api-grpc-mpi-v1-Server-ConnectionType)
  
    - [CommandService](#f5-nginx-agent-api-grpc-mpi-v1-CommandService)
  
- [common.proto](#common-proto)
    - [CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse)
    - [MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta)
  
    - [CommandResponse.CommandStatus](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse-CommandStatus)
  
- [file.proto](#file-proto)
    - [ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion)
    - [File](#f5-nginx-agent-api-grpc-mpi-v1-file-File)
    - [FileContents](#f5-nginx-agent-api-grpc-mpi-v1-file-FileContents)
    - [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta)
    - [FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview)
    - [FileRequest](#f5-nginx-agent-api-grpc-mpi-v1-file-FileRequest)
  
    - [File.FileAction](#f5-nginx-agent-api-grpc-mpi-v1-file-File-FileAction)
  
    - [FileService](#f5-nginx-agent-api-grpc-mpi-v1-file-FileService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="command-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## command.proto
These proto definitions follow https://protobuf.dev/programming-guides/style/
and recommendations outlined in https://static.sched.com/hosted_files/kccncna17/ad/2017%20CloudNativeCon%20-%20Mod%20gRPC%20Services.pdf


<a name="f5-nginx-agent-api-grpc-mpi-v1-ActionRequest"></a>

### ActionRequest
Perform an associated action on an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the instance identifier |
| action | [InstanceAction](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction) |  | the action to be performed on the instance |
| default_action | [DefaultAction](#f5-nginx-agent-api-grpc-mpi-v1-DefaultAction) |  | A default action placeholder |






<a name="f5-nginx-agent-api-grpc-mpi-v1-AgentConfig"></a>

### AgentConfig
This contains a series of NGINX Agent configurations


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| command | [Command](#f5-nginx-agent-api-grpc-mpi-v1-Command) |  | Command server settings |
| metrics | [Metrics](#f5-nginx-agent-api-grpc-mpi-v1-Metrics) |  | Metrics server settings |
| labels | [google.protobuf.Struct](#google-protobuf-Struct) | repeated | A series of key/value pairs to add more data to the NGINX Agent instance |
| features | [string](#string) | repeated | A list of features that the NGINX Agent has |
| message_buffer_size | [string](#string) |  | Message buffer size, maximum not acknowledged messages from the subscribe perspective |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Auth"></a>

### Auth
Authentication settings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  | A token |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Command"></a>

### Command
The command settings, associated with messaging from an external source


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| connection_settings | [ConnectionSettings](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionSettings) |  | The connection and security settingss for the command server |






<a name="f5-nginx-agent-api-grpc-mpi-v1-CommandStatusRequest"></a>

### CommandStatusRequest
Request an update on a particular command


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [common.MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta) |  | Meta-information associated with a message |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConfigApplyRequest"></a>

### ConfigApplyRequest
Additional information associated with a ConfigApplyRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_version | [file.ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) |  | the config version |
| overview | [file.FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) |  | an optional set of files related to the request

optional |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConfigSyncRequest"></a>

### ConfigSyncRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_version | [file.ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) |  | the config version |
| overview | [file.FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) |  | an optional set of files related to the request |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConfigUploadRequest"></a>

### ConfigUploadRequest
Additional information associated with a ConfigUploadRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the instance identifier |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConnectionRequest"></a>

### ConnectionRequest
The connection request is an intial handshake to establish a connection, sending NGINX Agent instance information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [common.MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta) |  | Meta-information associated with a message |
| agent | [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance) |  | instance information associated with the NGINX Agent |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse"></a>

### ConnectionResponse
A response to a ConnectionRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| response | [common.CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse) |  | the success or failure of the ConnectionRequest |
| agent_config | [AgentConfig](#f5-nginx-agent-api-grpc-mpi-v1-AgentConfig) |  | the recommendation NGINX Agent configurations provided by the ManagementPlane |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConnectionSettings"></a>

### ConnectionSettings
A set of connection information and it&#39;s associated auth, tls and backoff configurations


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [Server](#f5-nginx-agent-api-grpc-mpi-v1-Server) |  | Server settings that include connection information |
| auth | [Auth](#f5-nginx-agent-api-grpc-mpi-v1-Auth) |  | Authentication settings |
| tls | [TLSSetting](#f5-nginx-agent-api-grpc-mpi-v1-TLSSetting) |  | Optional TLS settings |
| keep_alive | [KeepAlive](#f5-nginx-agent-api-grpc-mpi-v1-KeepAlive) |  | keepalive settings for this connection |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth"></a>

### DataPlaneHealth
Health report of a set of instances


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [common.MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta) |  | Meta-information associated with a message |
| instance_health | [InstanceHealth](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth) | repeated | Health report of a set of instances |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneRequest"></a>

### DataPlaneRequest
Reports the status of an associated command. This may be in response to a ManagementPlaneRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [common.MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta) |  | Meta-information associated with a message |
| command_response | [common.CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse) |  | The command response with the associated request |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus"></a>

### DataPlaneStatus
Report on the status of the Data Plane


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [common.MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta) |  | Meta-information associated with a message |
| instances | [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance) | repeated | Report on instances on the Data Plane |
| latest_message_index | [string](#string) |  | The AckIndex latest message index that has been processed |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DefaultAction"></a>

### DefaultAction
A default action placeholder


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | [google.protobuf.Struct](#google-protobuf-Struct) | repeated |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Exporter"></a>

### Exporter
A destination configuration


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| report_interval | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | how often to report in google.protobuf.Timestamp format |
| connection_settings | [ConnectionSettings](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionSettings) |  | connection information to send data to a particular destination |






<a name="f5-nginx-agent-api-grpc-mpi-v1-HealthRequest"></a>

### HealthRequest
Additional information associated with a HealthRequest






<a name="f5-nginx-agent-api-grpc-mpi-v1-Instance"></a>

### Instance
This represents an instance being reported on


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_meta | [InstanceMeta](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta) |  | Meta-information associated with an instance |
| instance_config | [InstanceConfig](#f5-nginx-agent-api-grpc-mpi-v1-InstanceConfig) |  | Runtime configuration associated with an instance |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceAction"></a>

### InstanceAction
A set of actions that can be performed on an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action_type | [InstanceAction.InstanceActionType](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActionType) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceConfig"></a>

### InstanceConfig
Instance Configuration options


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| actions | [InstanceAction](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction) | repeated | provided actions associated with a particular instance. These are runtime based and provided by a particular version of the NGINX Agent |
| agent_config | [AgentConfig](#f5-nginx-agent-api-grpc-mpi-v1-AgentConfig) |  | NGINX Agent runtime configuration settings |
| nginx_config | [NGINXConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXConfig) |  | NGINX runtime configuration settings like stub_status, usually read from the NGINX config or NGINX process |
| nginx_plus_config | [NGINXPlusConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXPlusConfig) |  | NGINX Plus runtime configuration settings like api value, usually read from the NGINX config, NGINX process or NGINX Plus API |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth"></a>

### InstanceHealth
Report on the health of a particular instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |
| instance_health_status | [InstanceHealth.InstancHealthStatus](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth-InstancHealthStatus) |  | Health status |
| description | [string](#string) |  | Provides a human readable context around why a health status is a particular state |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta"></a>

### InstanceMeta
Metainformation relating to the reported instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the identifier associated with the instance |
| instance_type | [InstanceMeta.InstanceType](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta-InstanceType) |  | the types of instances possible |
| version | [string](#string) |  | the version of the instance |






<a name="f5-nginx-agent-api-grpc-mpi-v1-KeepAlive"></a>

### KeepAlive
keepalive settings based on https://pkg.go.dev/google.golang.org/grpc/keepalive#ClientParameters


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | After a duration of this time if the client doesn&#39;t see any activity it pings the server to see if the transport is still alive. If set below 10s, a minimum value of 10s will be used instead. |
| timeout | [double](#double) |  | After having pinged for keepalive check, the client waits for a duration of Timeout and if no activity is seen even after that the connection is closed. |
| permit_without_stream | [bool](#bool) |  | If true, client sends keepalive pings even with no active RPCs. If false, when there are no active RPCs, Time and Timeout will be ignored and no keepalive pings will be sent. |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneRequest"></a>

### ManagementPlaneRequest
A Management Plane request for information, triggers an associated rpc on the DataPlane


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [common.MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta) |  | Meta-information associated with a message |
| status_request | [StatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-StatusRequest) |  | triggers a DataPlaneStatus rpc |
| health_request | [HealthRequest](#f5-nginx-agent-api-grpc-mpi-v1-HealthRequest) |  | triggers a DataPlaneHealth rpc |
| config_apply_request | [ConfigApplyRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigApplyRequest) |  | triggers a rpc GetFile(FileRequest) for overview list, if overview is missing, triggers a rpc Overview(ConfigVersion) first |
| config_upload_request | [ConfigUploadRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigUploadRequest) |  | triggers a series of rpc SendFile(File) for that instances |
| config_sync_request | [ConfigSyncRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigSyncRequest) |  | triggers a reconciliation of with a command_response for a particular action |
| action_request | [ActionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ActionRequest) |  | triggers a DataPlaneRequest with a command_response for a particular action |
| command_status_request | [CommandStatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-CommandStatusRequest) |  | triggers a DataPlaneRequest with a command_response for a particular correlation_id |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Metrics"></a>

### Metrics
The metrics settings associated with orgins (sources) of the metrics and destinations (exporter)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sources | [Source](#f5-nginx-agent-api-grpc-mpi-v1-Source) | repeated | The connection and security settingss for the sources |
| exporters | [Exporter](#f5-nginx-agent-api-grpc-mpi-v1-Exporter) | repeated | The connection and security settingss for the exporters server |
| bulk_size | [string](#string) |  | the local buffer size that we will cache if connectivity issues exist |






<a name="f5-nginx-agent-api-grpc-mpi-v1-NGINXConfig"></a>

### NGINXConfig
A set of runtime NGINX configuration that gets populated


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| binary_path | [string](#string) |  | where the binary location is, if empty, this is a remote instance |
| hostname | [string](#string) |  | the hostname associated with NGINX |
| ip_address | [string](#string) |  | the ip address associated with NGINX |
| stub_status | [string](#string) |  | the stub status API location |
| access_logs | [string](#string) | repeated | a list of access_logs |
| error_logs | [string](#string) | repeated | a list of error_logs |






<a name="f5-nginx-agent-api-grpc-mpi-v1-NGINXPlusConfig"></a>

### NGINXPlusConfig
A set of runtime NGINX configuration that gets populated


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| binary_path | [string](#string) |  | where the binary location is, if empty, this is a remote instance |
| hostname | [string](#string) |  | the hostname associated with NGINX Plus |
| ip_address | [string](#string) |  | the ip address associated with NGINX Plus |
| api | [string](#string) |  | the API information for NGINX Plus API |
| access_logs | [string](#string) | repeated | is this correct for plus? |
| error_logs | [string](#string) | repeated | is this correct for plus? |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Server"></a>

### Server
Server settings like hostname


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  | the host information |
| port | [int32](#int32) |  | the port information |
| connection_type | [Server.ConnectionType](#f5-nginx-agent-api-grpc-mpi-v1-Server-ConnectionType) |  | Specific connection type |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Source"></a>

### Source
A source configuration


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| collection_interval | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | how often to collect data from a particular source. In google.protobuf.Timestamp format |
| connection_settings | [ConnectionSettings](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionSettings) |  | connection information to connect to a particular source |






<a name="f5-nginx-agent-api-grpc-mpi-v1-StatusRequest"></a>

### StatusRequest
Additional information associated with a StatusRequest






<a name="f5-nginx-agent-api-grpc-mpi-v1-TLSSetting"></a>

### TLSSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable | [bool](#bool) |  | enable tls |
| cert | [string](#string) |  | tls cert |
| key | [string](#string) |  | tls key |
| ca | [string](#string) |  | certificate authoirty cert |
| skip_verify | [bool](#bool) |  | enable verification of a server&#39;s certificate chain and host name |





 


<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActionType"></a>

### InstanceAction.InstanceActionType


| Name | Number | Description |
| ---- | ------ | ----------- |
| INSTANCE_ACTION_TYPE_UNSPECIFIED | 0 | Default action |



<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth-InstancHealthStatus"></a>

### InstanceHealth.InstancHealthStatus
Health status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| INSTANCE_HEALTH_STATUS_UNSPECIFIED | 0 | Unspecified status |
| INSTANCE_HEALTH_STATUS_HEALTHY | 1 | Healthy status |
| INSTANCE_HEALTH_STATUS_UNHEALTHY | 2 | Unhealthy status |
| INSTANCE_HEALTH_STATUS_DEGRADED | 3 | Degraded status |



<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta-InstanceType"></a>

### InstanceMeta.InstanceType
the types of instances possible

| Name | Number | Description |
| ---- | ------ | ----------- |
| INSTANCE_TYPE_UNSPECIFIED | 0 | Unspecified instance type |
| INSTANCE_TYPE_AGENT | 1 | NGINX Agent |
| INSTANCE_TYPE_NGINX | 2 | NGINX |
| INSTANCE_TYPE_NGINX_PLUS | 3 | NGINX Plus |
| INSTANCE_TYPE_UNIT | 4 | NGINX Unit |



<a name="f5-nginx-agent-api-grpc-mpi-v1-Server-ConnectionType"></a>

### Server.ConnectionType
Connection Type enumeration

| Name | Number | Description |
| ---- | ------ | ----------- |
| CONNECTION_TYPE_UNSPECIFIED | 0 | Default connection type |
| CONNECTION_TYPE_HTTP | 1 | HTTP connection type |
| CONNECTION_TYPE_GRPC | 2 | gRPC connection type |
| CONNECTION_TYPE_UNIX | 3 | unix socket connection type |


 

 


<a name="f5-nginx-agent-api-grpc-mpi-v1-CommandService"></a>

### CommandService
A service outlining the command and control options for a DataPlane Client
All operations are written from a client perspective
The RPC calls generally flow Client -&gt; Server, except for Subscribe which contains a bidirectional stream
The ManagementPlaneRequest sent in the Subscribe stream triggers one or more client actions.
Messages provided by the Management Plane must be a FIFO ordered queue. Messages in the queue must have a monotonically-increasing integer index. 
The indexes do not need to be sequential. The index must be a 64-bit signed integer.
The index must not reset for the entire lifetime of a unique Agent (i.e. the index does not reset to 0 only because of a temporary disconnection or new session). 
Messages must not be removed from the Mangement Plane queue until Ack’d by the Agent. 
Messages sent but not yet Ack’d must be kept in an “in-flight” buffer as they may need to be retried.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateConnection | [ConnectionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionRequest) | [ConnectionResponse](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse) | Connects NGINX Agent to the Management Plane agnostic of instance data |
| UpdateStatus | [DataPlaneStatus](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus) | [.google.protobuf.Empty](#google-protobuf-Empty) | Reports on instances and their configurations |
| UpdateHealth | [DataPlaneHealth](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth) | [.google.protobuf.Empty](#google-protobuf-Empty) | Reports on instance health |
| Subscribe | [DataPlaneRequest](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneRequest) stream | [ManagementPlaneRequest](#f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneRequest) stream | A decoupled communication mechanism between the data plane and management plane. |

 



<a name="common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## common.proto



<a name="f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse"></a>

### CommandResponse
Represents a the status response of an command


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CommandResponse.CommandStatus](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse-CommandStatus) |  | Command status |
| message | [string](#string) |  | Provides a user friendly message to describe the response |
| error | [string](#string) |  | Provides an error message of why the command failed |






<a name="f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta"></a>

### MessageMeta
Meta-information associated with a message


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_id | [string](#string) |  | monotonically increasing integer |
| correlation_id | [string](#string) |  | if 2 or more messages associated with the same workflow, use this field as an association |





 


<a name="f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse-CommandStatus"></a>

### CommandResponse.CommandStatus
Command status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| COMMAND_STATUS_UNSPECIFIED | 0 | Unspecified status of command |
| COMMAND_STATUS_OK | 1 | Command was successful |
| COMMAND_STATUS_ERROR | 2 | Command failed |
| COMMAND_STATUS_IN_PROGRESS | 3 | Command in-progress |


 

 

 



<a name="file-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## file.proto



<a name="f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion"></a>

### ConfigVersion
Represents a specific configuration version associated with an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the instance identifier |
| version | [string](#string) |  | the version of the configuration |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-File"></a>

### File
Represents meta data about a file


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_meta | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) |  | Meta information about the file, the name (including path) and hash |
| modified_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last modified time of the file (created time if never modified) |
| permissions | [string](#string) |  | the permission set associated with a particular file |
| size | [int64](#int64) |  | Size of the file in bytes |
| action | [File.FileAction](#f5-nginx-agent-api-grpc-mpi-v1-file-File-FileAction) |  | optional action |
| contents | [FileContents](#f5-nginx-agent-api-grpc-mpi-v1-file-FileContents) |  | optional contents |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileContents"></a>

### FileContents
Represents the bytes contents of the file https://protobuf.dev/programming-guides/api/#dont-encode-data-in-a-string


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [bytes](#bytes) |  | byte representation of a file without encoding |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta"></a>

### FileMeta
Meta information about the file, the name (including path) and hash


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | the name of the file |
| hash | [string](#string) |  | the hash of the file contents |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview"></a>

### FileOverview
Represents a collection of files


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [File](#f5-nginx-agent-api-grpc-mpi-v1-file-File) | repeated | A list of files |
| version | [ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) |  | the configuration version of the current set of files |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileRequest"></a>

### FileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [f5.nginx.agent.api.grpc.mpi.v1.common.MessageMeta](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageMeta) |  | Meta-information associated with a message |
| file_meta | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) |  | Meta-information associated with the file |





 


<a name="f5-nginx-agent-api-grpc-mpi-v1-file-File-FileAction"></a>

### File.FileAction
Action enumeration

| Name | Number | Description |
| ---- | ------ | ----------- |
| FILE_ACTION_UNSPECIFIED | 0 | Default value, no action |
| FILE_ACTION_UNCHANGED | 1 | No changes to the file |
| FILE_ACTION_ADD | 2 | New file |
| FILE_ACTION_UPDATE | 3 | Updated file |
| FILE_ACTION_DELETE | 4 | File deleted |


 

 


<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileService"></a>

### FileService
This specifies the FileService operations for transferring file data between a client and server.
All operations are written from a client perspective and flow Client -&gt; Server
The server must set a max file size (in bytes), and that size must be used to configure 
the gRPC server and client for the FileService such that the FileContents object can be sent with bytes of the configured size. 
The actual configured max size for gRPC objects must be maxFileSize &#43; sizeOfSha256HashString since a FileContents object contains both. 
A SHA256 hash string is 64 bytes, therefore the configured max message size should be maxFileSize &#43; 64.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetOverview | [ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) | [FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) | Get the overview of files for a particular configuration version of an instance |
| UpdateOverview | [FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) | [.google.protobuf.Empty](#google-protobuf-Empty) | Update the overview of files for a particular set of file changes on the data plane |
| GetFile | [FileRequest](#f5-nginx-agent-api-grpc-mpi-v1-file-FileRequest) | [FileContents](#f5-nginx-agent-api-grpc-mpi-v1-file-FileContents) | Get the file contents for a particular file |
| UpdateFile | [File](#f5-nginx-agent-api-grpc-mpi-v1-file-File) | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) | Update a file from the Agent to the Server |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

