# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [mpi/v1/common.proto](#mpi_v1_common-proto)
    - [CommandResponse](#mpi-v1-CommandResponse)
    - [MessageMeta](#mpi-v1-MessageMeta)
  
    - [CommandResponse.CommandStatus](#mpi-v1-CommandResponse-CommandStatus)
  
- [mpi/v1/files.proto](#mpi_v1_files-proto)
    - [ConfigVersion](#mpi-v1-ConfigVersion)
    - [File](#mpi-v1-File)
    - [FileContents](#mpi-v1-FileContents)
    - [FileMeta](#mpi-v1-FileMeta)
    - [FileOverview](#mpi-v1-FileOverview)
    - [GetFileRequest](#mpi-v1-GetFileRequest)
    - [GetFileResponse](#mpi-v1-GetFileResponse)
    - [GetOverviewRequest](#mpi-v1-GetOverviewRequest)
    - [GetOverviewResponse](#mpi-v1-GetOverviewResponse)
    - [UpdateFileRequest](#mpi-v1-UpdateFileRequest)
    - [UpdateFileResponse](#mpi-v1-UpdateFileResponse)
    - [UpdateOverviewRequest](#mpi-v1-UpdateOverviewRequest)
    - [UpdateOverviewResponse](#mpi-v1-UpdateOverviewResponse)
  
    - [File.FileAction](#mpi-v1-File-FileAction)
  
    - [FileService](#mpi-v1-FileService)
  
- [mpi/v1/command.proto](#mpi_v1_command-proto)
    - [APIActionRequest](#mpi-v1-APIActionRequest)
    - [AgentConfig](#mpi-v1-AgentConfig)
    - [CommandServer](#mpi-v1-CommandServer)
    - [CommandStatusRequest](#mpi-v1-CommandStatusRequest)
    - [ConfigApplyRequest](#mpi-v1-ConfigApplyRequest)
    - [ConfigUploadRequest](#mpi-v1-ConfigUploadRequest)
    - [ContainerInfo](#mpi-v1-ContainerInfo)
    - [CreateConnectionRequest](#mpi-v1-CreateConnectionRequest)
    - [CreateConnectionResponse](#mpi-v1-CreateConnectionResponse)
    - [DataPlaneResponse](#mpi-v1-DataPlaneResponse)
    - [FileServer](#mpi-v1-FileServer)
    - [HealthRequest](#mpi-v1-HealthRequest)
    - [HostInfo](#mpi-v1-HostInfo)
    - [Instance](#mpi-v1-Instance)
    - [InstanceAction](#mpi-v1-InstanceAction)
    - [InstanceChild](#mpi-v1-InstanceChild)
    - [InstanceConfig](#mpi-v1-InstanceConfig)
    - [InstanceHealth](#mpi-v1-InstanceHealth)
    - [InstanceMeta](#mpi-v1-InstanceMeta)
    - [InstanceRuntime](#mpi-v1-InstanceRuntime)
    - [ManagementPlaneRequest](#mpi-v1-ManagementPlaneRequest)
    - [MetricsServer](#mpi-v1-MetricsServer)
    - [NGINXPlusRuntimeInfo](#mpi-v1-NGINXPlusRuntimeInfo)
    - [NGINXRuntimeInfo](#mpi-v1-NGINXRuntimeInfo)
    - [ReleaseInfo](#mpi-v1-ReleaseInfo)
    - [Resource](#mpi-v1-Resource)
    - [StatusRequest](#mpi-v1-StatusRequest)
    - [UpdateDataPlaneHealthRequest](#mpi-v1-UpdateDataPlaneHealthRequest)
    - [UpdateDataPlaneHealthResponse](#mpi-v1-UpdateDataPlaneHealthResponse)
    - [UpdateDataPlaneStatusRequest](#mpi-v1-UpdateDataPlaneStatusRequest)
    - [UpdateDataPlaneStatusResponse](#mpi-v1-UpdateDataPlaneStatusResponse)
  
    - [InstanceHealth.InstanceHealthStatus](#mpi-v1-InstanceHealth-InstanceHealthStatus)
    - [InstanceMeta.InstanceType](#mpi-v1-InstanceMeta-InstanceType)
  
    - [CommandService](#mpi-v1-CommandService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="mpi_v1_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mpi/v1/common.proto
Copyright (c) F5, Inc.

This source code is licensed under the Apache License, Version 2.0 license found in the
LICENSE file in the root directory of this source tree.


<a name="mpi-v1-CommandResponse"></a>

### CommandResponse
Represents a the status response of an command


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CommandResponse.CommandStatus](#mpi-v1-CommandResponse-CommandStatus) |  | Command status |
| message | [string](#string) |  | Provides a user friendly message to describe the response |
| error | [string](#string) |  | Provides an error message of why the command failed, only populated when CommandStatus is COMMAND_STATUS_ERROR |






<a name="mpi-v1-MessageMeta"></a>

### MessageMeta
Meta-information associated with a message


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_id | [string](#string) |  | uuid v7 monotonically increasing string |
| correlation_id | [string](#string) |  | if 2 or more messages associated with the same workflow, use this field as an association |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | timestamp for human readable timestamp in UTC format |





 


<a name="mpi-v1-CommandResponse-CommandStatus"></a>

### CommandResponse.CommandStatus
Command status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| COMMAND_STATUS_UNSPECIFIED | 0 | Unspecified status of command |
| COMMAND_STATUS_OK | 1 | Command was successful |
| COMMAND_STATUS_ERROR | 2 | Command error |
| COMMAND_STATUS_IN_PROGRESS | 3 | Command in-progress |
| COMMAND_STATUS_FAILURE | 4 | Command failure |


 

 

 



<a name="mpi_v1_files-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mpi/v1/files.proto



<a name="mpi-v1-ConfigVersion"></a>

### ConfigVersion
Represents a specific configuration version associated with an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | The instance identifier |
| version | [string](#string) |  | The version of the configuration |






<a name="mpi-v1-File"></a>

### File
Represents meta data about a file


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_meta | [FileMeta](#mpi-v1-FileMeta) |  | Meta information about the file, the name (including path) and hash |
| action | [File.FileAction](#mpi-v1-File-FileAction) | optional | Optional action |






<a name="mpi-v1-FileContents"></a>

### FileContents
Represents the bytes contents of the file https://protobuf.dev/programming-guides/api/#dont-encode-data-in-a-string


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [bytes](#bytes) |  | Byte representation of a file without encoding |






<a name="mpi-v1-FileMeta"></a>

### FileMeta
Meta information about the file, the name (including path) and hash


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The full path of the file |
| hash | [string](#string) |  | The hash of the file contents sha256, hex encoded |
| modified_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Last modified time of the file (created time if never modified) |
| permissions | [string](#string) |  | The permission set associated with a particular file |
| size | [int64](#int64) |  | The size of the file in bytes |






<a name="mpi-v1-FileOverview"></a>

### FileOverview
Represents a collection of files


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| files | [File](#mpi-v1-File) | repeated | A list of files |
| config_version | [ConfigVersion](#mpi-v1-ConfigVersion) |  | The configuration version of the current set of files |






<a name="mpi-v1-GetFileRequest"></a>

### GetFileRequest
Represents the get file request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| file_meta | [FileMeta](#mpi-v1-FileMeta) |  | Meta-information associated with the file |






<a name="mpi-v1-GetFileResponse"></a>

### GetFileResponse
Represents the response to a get file request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [FileContents](#mpi-v1-FileContents) |  | The contents of a file |






<a name="mpi-v1-GetOverviewRequest"></a>

### GetOverviewRequest
Represents a request payload for a file overview


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| config_version | [ConfigVersion](#mpi-v1-ConfigVersion) |  | The config version of the overview you are requesting |






<a name="mpi-v1-GetOverviewResponse"></a>

### GetOverviewResponse
Represents a response payload for an overview of files for a particular configuration version of an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| overview | [FileOverview](#mpi-v1-FileOverview) |  | The file overview of an instance |






<a name="mpi-v1-UpdateFileRequest"></a>

### UpdateFileRequest
Represents the update file request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [File](#mpi-v1-File) |  | The file requested to be updated |
| contents | [FileContents](#mpi-v1-FileContents) |  | The contents of a file |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |






<a name="mpi-v1-UpdateFileResponse"></a>

### UpdateFileResponse
Represents the response to an update file request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_meta | [FileMeta](#mpi-v1-FileMeta) |  | Meta-information associated with the updated file |






<a name="mpi-v1-UpdateOverviewRequest"></a>

### UpdateOverviewRequest
Represents a the payload for an overview an update of  files for a particular configuration version of an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| overview | [FileOverview](#mpi-v1-FileOverview) |  | The file overview of an instance |






<a name="mpi-v1-UpdateOverviewResponse"></a>

### UpdateOverviewResponse
Represents a the response from an UpdateOverviewRequest - intentionally left empty





 


<a name="mpi-v1-File-FileAction"></a>

### File.FileAction
Action enumeration

| Name | Number | Description |
| ---- | ------ | ----------- |
| FILE_ACTION_UNSPECIFIED | 0 | Default value, no action |
| FILE_ACTION_UNCHANGED | 1 | No changes to the file |
| FILE_ACTION_ADD | 2 | New file |
| FILE_ACTION_UPDATE | 3 | Updated file |
| FILE_ACTION_DELETE | 4 | File deleted |


 

 


<a name="mpi-v1-FileService"></a>

### FileService
This specifies the FileService operations for transferring file data between a client and server.
All operations are written from a client perspective and flow Client -&gt; Server
The server must set a max file size (in bytes), and that size must be used to configure 
the gRPC server and client for the FileService such that the FileContents object can be sent with bytes of the configured size. 
The actual configured max size for gRPC objects must be maxFileSize &#43; sizeOfSha256HashString since a FileContents object contains both. 
A SHA256 hash string is 64 bytes, therefore the configured max message size should be maxFileSize &#43; 64.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetOverview | [GetOverviewRequest](#mpi-v1-GetOverviewRequest) | [GetOverviewResponse](#mpi-v1-GetOverviewResponse) | Get the overview of files for a particular configuration version of an instance |
| UpdateOverview | [UpdateOverviewRequest](#mpi-v1-UpdateOverviewRequest) | [UpdateOverviewResponse](#mpi-v1-UpdateOverviewResponse) | Update the overview of files for a particular set of file changes on the data plane |
| GetFile | [GetFileRequest](#mpi-v1-GetFileRequest) | [GetFileResponse](#mpi-v1-GetFileResponse) | Get the file contents for a particular file |
| UpdateFile | [UpdateFileRequest](#mpi-v1-UpdateFileRequest) | [UpdateFileResponse](#mpi-v1-UpdateFileResponse) | Update a file from the Agent to the Server |

 



<a name="mpi_v1_command-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mpi/v1/command.proto
These proto definitions follow https://protobuf.dev/programming-guides/style/
and recommendations outlined in https://static.sched.com/hosted_files/kccncna17/ad/2017%20CloudNativeCon%20-%20Mod%20gRPC%20Services.pdf


<a name="mpi-v1-APIActionRequest"></a>

### APIActionRequest
Perform an associated API action on an instance






<a name="mpi-v1-AgentConfig"></a>

### AgentConfig
This contains a series of NGINX Agent configurations


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| command | [CommandServer](#mpi-v1-CommandServer) |  | Command server settings |
| metrics | [MetricsServer](#mpi-v1-MetricsServer) |  | Metrics server settings |
| file | [FileServer](#mpi-v1-FileServer) |  | File server settings |
| labels | [google.protobuf.Struct](#google-protobuf-Struct) | repeated | A series of key/value pairs to add more data to the NGINX Agent instance |
| features | [string](#string) | repeated | A list of features that the NGINX Agent has |
| message_buffer_size | [string](#string) |  | Message buffer size, maximum not acknowledged messages from the subscribe perspective |






<a name="mpi-v1-CommandServer"></a>

### CommandServer
The command settings, associated with messaging from an external source






<a name="mpi-v1-CommandStatusRequest"></a>

### CommandStatusRequest
Request an update on a particular command






<a name="mpi-v1-ConfigApplyRequest"></a>

### ConfigApplyRequest
Additional information associated with a ConfigApplyRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_version | [ConfigVersion](#mpi-v1-ConfigVersion) |  | the config version |
| overview | [FileOverview](#mpi-v1-FileOverview) | optional | an optional set of files related to the request |






<a name="mpi-v1-ConfigUploadRequest"></a>

### ConfigUploadRequest
Additional information associated with a ConfigUploadRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the instance identifier |
| overview | [FileOverview](#mpi-v1-FileOverview) | optional | an optional set of files related to the request |






<a name="mpi-v1-ContainerInfo"></a>

### ContainerInfo
Container information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| container_id | [string](#string) |  | The identifier of the container |






<a name="mpi-v1-CreateConnectionRequest"></a>

### CreateConnectionRequest
The connection request is an initial handshake to establish a connection, sending NGINX Agent instance information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| resource | [Resource](#mpi-v1-Resource) |  | Instance and infrastructure information associated with the NGINX Agent |






<a name="mpi-v1-CreateConnectionResponse"></a>

### CreateConnectionResponse
A response to a CreateConnectionRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| response | [CommandResponse](#mpi-v1-CommandResponse) |  | The success or failure of the CreateConnectionRequest |
| agent_config | [AgentConfig](#mpi-v1-AgentConfig) |  | The recommendation NGINX Agent configurations provided by the ManagementPlane |






<a name="mpi-v1-DataPlaneResponse"></a>

### DataPlaneResponse
Reports the status of an associated command. This may be in response to a ManagementPlaneRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| command_response | [CommandResponse](#mpi-v1-CommandResponse) |  | The command response with the associated request |
| instance_id | [string](#string) |  | The instance identifier, if applicable, for this response |






<a name="mpi-v1-FileServer"></a>

### FileServer
The file settings associated with file server for configurations






<a name="mpi-v1-HealthRequest"></a>

### HealthRequest
Additional information associated with a HealthRequest






<a name="mpi-v1-HostInfo"></a>

### HostInfo
Represents the host system information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host_id | [string](#string) |  | The host identifier |
| hostname | [string](#string) |  | The name of the host |
| release_info | [ReleaseInfo](#mpi-v1-ReleaseInfo) |  | Release information of the host |






<a name="mpi-v1-Instance"></a>

### Instance
This represents an instance being reported on


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_meta | [InstanceMeta](#mpi-v1-InstanceMeta) |  | Meta-information associated with an instance |
| instance_config | [InstanceConfig](#mpi-v1-InstanceConfig) |  | Read and write configuration associated with an instance that can be modified via this definition |
| instance_runtime | [InstanceRuntime](#mpi-v1-InstanceRuntime) |  | Read-only meta data associated with the instance running in it&#39;s environment |






<a name="mpi-v1-InstanceAction"></a>

### InstanceAction
A set of actions that can be performed on an instance






<a name="mpi-v1-InstanceChild"></a>

### InstanceChild



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| process_id | [int32](#int32) |  | the process identifier |






<a name="mpi-v1-InstanceConfig"></a>

### InstanceConfig
Instance Configuration options


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| actions | [InstanceAction](#mpi-v1-InstanceAction) | repeated | provided actions associated with a particular instance. These are runtime based and provided by a particular version of the NGINX Agent |
| agent_config | [AgentConfig](#mpi-v1-AgentConfig) |  | NGINX Agent runtime configuration settings |






<a name="mpi-v1-InstanceHealth"></a>

### InstanceHealth



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |
| instance_health_status | [InstanceHealth.InstanceHealthStatus](#mpi-v1-InstanceHealth-InstanceHealthStatus) |  | Health status |
| description | [string](#string) |  | Provides a human readable context around why a health status is a particular state |






<a name="mpi-v1-InstanceMeta"></a>

### InstanceMeta
Meta-information relating to the reported instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the identifier associated with the instance |
| instance_type | [InstanceMeta.InstanceType](#mpi-v1-InstanceMeta-InstanceType) |  | the types of instances possible |
| version | [string](#string) |  | the version of the instance |






<a name="mpi-v1-InstanceRuntime"></a>

### InstanceRuntime



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| process_id | [int32](#int32) |  | the process identifier |
| binary_path | [string](#string) |  | the binary path location |
| config_path | [string](#string) |  | the config path location |
| nginx_runtime_info | [NGINXRuntimeInfo](#mpi-v1-NGINXRuntimeInfo) |  | NGINX runtime configuration settings like stub_status, usually read from the NGINX config or NGINX process |
| nginx_plus_runtime_info | [NGINXPlusRuntimeInfo](#mpi-v1-NGINXPlusRuntimeInfo) |  | NGINX Plus runtime configuration settings like api value, usually read from the NGINX config, NGINX process or NGINX Plus API |
| instance_children | [InstanceChild](#mpi-v1-InstanceChild) | repeated | List of worker processes |






<a name="mpi-v1-ManagementPlaneRequest"></a>

### ManagementPlaneRequest
A Management Plane request for information, triggers an associated rpc on the Data Plane


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| status_request | [StatusRequest](#mpi-v1-StatusRequest) |  | triggers a DataPlaneStatus rpc |
| health_request | [HealthRequest](#mpi-v1-HealthRequest) |  | triggers a DataPlaneHealth rpc |
| config_apply_request | [ConfigApplyRequest](#mpi-v1-ConfigApplyRequest) |  | triggers a rpc GetFile(FileRequest) for overview list, if overview is missing, triggers a rpc GetOverview(ConfigVersion) first |
| config_upload_request | [ConfigUploadRequest](#mpi-v1-ConfigUploadRequest) |  | triggers a series of rpc UpdateFile(File) for that instances |
| action_request | [APIActionRequest](#mpi-v1-APIActionRequest) |  | triggers a DataPlaneResponse with a command_response for a particular action |
| command_status_request | [CommandStatusRequest](#mpi-v1-CommandStatusRequest) |  | triggers a DataPlaneResponse with a command_response for a particular correlation_id |






<a name="mpi-v1-MetricsServer"></a>

### MetricsServer
The metrics settings associated with origins (sources) of the metrics and destinations (exporter)






<a name="mpi-v1-NGINXPlusRuntimeInfo"></a>

### NGINXPlusRuntimeInfo
A set of runtime NGINX Plus settings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stub_status | [string](#string) |  | the stub status API URL |
| access_logs | [string](#string) | repeated | a list of access_logs |
| error_logs | [string](#string) | repeated | a list of error_logs |
| loadable_modules | [string](#string) | repeated | List of NGINX potentially loadable modules (installed but not loaded). |
| dynamic_modules | [string](#string) | repeated | List of NGINX dynamic modules. |
| plus_api | [string](#string) |  | the plus API location |






<a name="mpi-v1-NGINXRuntimeInfo"></a>

### NGINXRuntimeInfo
A set of runtime NGINX OSS settings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stub_status | [string](#string) |  | the stub status API URL |
| access_logs | [string](#string) | repeated | a list of access_logs |
| error_logs | [string](#string) | repeated | a list of error_logs |
| loadable_modules | [string](#string) | repeated | List of NGINX potentially loadable modules (installed but not loaded). |
| dynamic_modules | [string](#string) | repeated | List of NGINX dynamic modules. |






<a name="mpi-v1-ReleaseInfo"></a>

### ReleaseInfo
Release information of the host


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| codename | [string](#string) |  | OS type (e.g. freebsd, linux, etc) |
| id | [string](#string) |  | OS name (e.g. ubuntu, linuxmint, etc) |
| name | [string](#string) |  | OS family (e.g. debian, rhel) |
| version_id | [string](#string) |  | Version of the OS kernel |
| version | [string](#string) |  | Version of the OS |






<a name="mpi-v1-Resource"></a>

### Resource
A representation of instances and runtime resource information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [string](#string) |  | A resource identifier |
| instances | [Instance](#mpi-v1-Instance) | repeated | A list of instances associated with this resource |
| host_info | [HostInfo](#mpi-v1-HostInfo) |  | If running on bare-metal, provides additional information |
| container_info | [ContainerInfo](#mpi-v1-ContainerInfo) |  | If running in a containerized environment, provides additional information |






<a name="mpi-v1-StatusRequest"></a>

### StatusRequest
Additional information associated with a StatusRequest






<a name="mpi-v1-UpdateDataPlaneHealthRequest"></a>

### UpdateDataPlaneHealthRequest
Health report of a set of instances


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| instance_healths | [InstanceHealth](#mpi-v1-InstanceHealth) | repeated | Health report of a set of instances |






<a name="mpi-v1-UpdateDataPlaneHealthResponse"></a>

### UpdateDataPlaneHealthResponse
Response to a UpdateDataPlaneHealthRequest - intentionally empty






<a name="mpi-v1-UpdateDataPlaneStatusRequest"></a>

### UpdateDataPlaneStatusRequest
Report on the status of the Data Plane


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| resource | [Resource](#mpi-v1-Resource) |  | the representation of a data plane |






<a name="mpi-v1-UpdateDataPlaneStatusResponse"></a>

### UpdateDataPlaneStatusResponse
Respond to a UpdateDataPlaneStatusRequest - intentionally empty





 


<a name="mpi-v1-InstanceHealth-InstanceHealthStatus"></a>

### InstanceHealth.InstanceHealthStatus
Health status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| INSTANCE_HEALTH_STATUS_UNSPECIFIED | 0 | Unspecified status |
| INSTANCE_HEALTH_STATUS_HEALTHY | 1 | Healthy status |
| INSTANCE_HEALTH_STATUS_UNHEALTHY | 2 | Unhealthy status |
| INSTANCE_HEALTH_STATUS_DEGRADED | 3 | Degraded status |



<a name="mpi-v1-InstanceMeta-InstanceType"></a>

### InstanceMeta.InstanceType
the types of instances possible

| Name | Number | Description |
| ---- | ------ | ----------- |
| INSTANCE_TYPE_UNSPECIFIED | 0 | Unspecified instance type |
| INSTANCE_TYPE_AGENT | 1 | NGINX Agent |
| INSTANCE_TYPE_NGINX | 2 | NGINX |
| INSTANCE_TYPE_NGINX_PLUS | 3 | NGINX Plus |
| INSTANCE_TYPE_UNIT | 4 | NGINX Unit |


 

 


<a name="mpi-v1-CommandService"></a>

### CommandService
A service outlining the command and control options for a Data Plane Client
All operations are written from a client perspective
The RPC calls generally flow Client -&gt; Server, except for Subscribe which contains a bidirectional stream
The ManagementPlaneRequest sent in the Subscribe stream triggers one or more client actions.
Messages provided by the Management Plane must be a FIFO ordered queue. Messages in the queue must have a monotonically-increasing integer index. 
The indexes do not need to be sequential. The index must be a 64-bit signed integer.
The index must not reset for the entire lifetime of a unique Agent (i.e. the index does not reset to 0 only because of a temporary disconnection or new session). 
Messages must not be removed from the Management Plane queue until Ack’d by the Agent. 
Messages sent but not yet Ack’d must be kept in an “in-flight” buffer as they may need to be retried.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateConnection | [CreateConnectionRequest](#mpi-v1-CreateConnectionRequest) | [CreateConnectionResponse](#mpi-v1-CreateConnectionResponse) | Connects NGINX Agent to the Management Plane agnostic of instance data |
| UpdateDataPlaneStatus | [UpdateDataPlaneStatusRequest](#mpi-v1-UpdateDataPlaneStatusRequest) | [UpdateDataPlaneStatusResponse](#mpi-v1-UpdateDataPlaneStatusResponse) | Reports on instances and their configurations |
| UpdateDataPlaneHealth | [UpdateDataPlaneHealthRequest](#mpi-v1-UpdateDataPlaneHealthRequest) | [UpdateDataPlaneHealthResponse](#mpi-v1-UpdateDataPlaneHealthResponse) | Reports on instance health |
| Subscribe | [DataPlaneResponse](#mpi-v1-DataPlaneResponse) stream | [ManagementPlaneRequest](#mpi-v1-ManagementPlaneRequest) stream | A decoupled communication mechanism between the data plane and management plane. buf:lint:ignore RPC_RESPONSE_STANDARD_NAME buf:lint:ignore RPC_REQUEST_STANDARD_NAME |

 



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

