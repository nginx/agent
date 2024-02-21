# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [command.proto](#command-proto)
    - [ActionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ActionRequest)
    - [AgentConfig](#f5-nginx-agent-api-grpc-mpi-v1-AgentConfig)
    - [CommandStatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-CommandStatusRequest)
    - [ConfigApplyRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigApplyRequest)
    - [ConfigUploadRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigUploadRequest)
    - [ConnectionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionRequest)
    - [ConnectionResponse](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse)
    - [DataPlaneHealth](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth)
    - [DataPlaneMessage](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneMessage)
    - [DataPlaneStatus](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus)
    - [DefaultAction](#f5-nginx-agent-api-grpc-mpi-v1-DefaultAction)
    - [HealthRequest](#f5-nginx-agent-api-grpc-mpi-v1-HealthRequest)
    - [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance)
    - [InstanceAction](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction)
    - [InstanceConfig](#f5-nginx-agent-api-grpc-mpi-v1-InstanceConfig)
    - [InstanceHealth](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth)
    - [InstanceMeta](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta)
    - [ManagementPlaneMessage](#f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneMessage)
    - [NGINXConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXConfig)
    - [NGINXPlusConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXPlusConfig)
    - [Server](#f5-nginx-agent-api-grpc-mpi-v1-Server)
    - [StatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-StatusRequest)
  
    - [InstanceAction.InstanceActions](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActions)
    - [InstanceHealth.InstancHealthStatus](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth-InstancHealthStatus)
    - [InstanceMeta.InstanceType](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta-InstanceType)
  
    - [CommandService](#f5-nginx-agent-api-grpc-mpi-v1-CommandService)
  
- [common.proto](#common-proto)
    - [CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse)
    - [MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest)
  
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
| server | [Server](#f5-nginx-agent-api-grpc-mpi-v1-Server) |  | server information to connect to management plane |
| labels | [google.protobuf.Struct](#google-protobuf-Struct) | repeated | A series of key/value pairs to add more data to the NGINX Agent instance |
| features | [string](#string) | repeated | A list of features that the NGINX Agent has

Max NAck setting? |






<a name="f5-nginx-agent-api-grpc-mpi-v1-CommandStatusRequest"></a>

### CommandStatusRequest
Request an update on a particular command


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConfigApplyRequest"></a>

### ConfigApplyRequest
Additional information associated with a ConfigApplyRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_version | [file.ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) |  | the config version |
| overview | [file.FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) |  | an optional set of files related to the request

optional |






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
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| agent | [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance) |  | instance information associated with the NGINX Agent |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse"></a>

### ConnectionResponse
A response to a ConnectionRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| response | [common.CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse) |  | the success or failure of the ConnectionRequest |
| agent_config | [AgentConfig](#f5-nginx-agent-api-grpc-mpi-v1-AgentConfig) |  | the recommendation NGINX Agent configurations provided by the ManagementPlane |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth"></a>

### DataPlaneHealth
Health report of a set of instances


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| instance_health | [InstanceHealth](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth) | repeated | Health report of a set of instances |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneMessage"></a>

### DataPlaneMessage
Reports the status of an associated command. This may be in response to a ManagementPlaneMessage request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| command_response | [common.CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse) |  | The command response with the associated request |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus"></a>

### DataPlaneStatus
Report on the status of the Data Plane


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| instances | [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance) | repeated | Report on instances on the Data Plane |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DefaultAction"></a>

### DefaultAction
A default action placeholder


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | [google.protobuf.Struct](#google-protobuf-Struct) | repeated |  |






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
| action | [InstanceAction.InstanceActions](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActions) |  |  |






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






<a name="f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneMessage"></a>

### ManagementPlaneMessage
A Management Plane request for information, triggers an associated rpc on the DataPlane


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| status_request | [StatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-StatusRequest) |  | triggers a DataPlaneStatus rpc |
| health_request | [HealthRequest](#f5-nginx-agent-api-grpc-mpi-v1-HealthRequest) |  | triggers a DataPlaneHealth rpc |
| config_apply_request | [ConfigApplyRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigApplyRequest) |  | triggers a rpc GetFile(FileRequest) for overview list, if overview is missing, triggers a rpc Overview(ConfigVersion) first |
| config_upload_request | [ConfigUploadRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConfigUploadRequest) |  | triggers a series of rpc SendFile(File) for that instances |
| action_request | [ActionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ActionRequest) |  | triggers a DataPlaneMessage with a command_response for a particular action |
| command_status_request | [CommandStatusRequest](#f5-nginx-agent-api-grpc-mpi-v1-CommandStatusRequest) |  | triggers a DataPlaneMessage with a command_response for a particular correlation_id |






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
| api | [string](#string) |  | the API inforation for NGINX Plus API |
| access_logs | [string](#string) | repeated | is this correct for plus? |
| error_logs | [string](#string) | repeated | is this correct for plus? |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Server"></a>

### Server



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  |  |
| port | [string](#string) |  | Auth auth = 3; TlsCert tls_cert = 4; |






<a name="f5-nginx-agent-api-grpc-mpi-v1-StatusRequest"></a>

### StatusRequest
Additional information associated with a StatusRequest





 


<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActions"></a>

### InstanceAction.InstanceActions


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | Default action |



<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth-InstancHealthStatus"></a>

### InstanceHealth.InstancHealthStatus
Health status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | Unknown status |
| HEALTHY | 1 | Healthy status |
| UNHEALTHY | 2 | Unhealthy status |
| DEGRADED | 3 | Degraded status |



<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta-InstanceType"></a>

### InstanceMeta.InstanceType
the types of instances possible

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| AGENT | 1 | NGINX Agent |
| NGINX | 2 | NGINX |
| NGINX_PLUS | 3 | NGINX Plus |
| UNIT | 4 | NGINX Unit |


 

 


<a name="f5-nginx-agent-api-grpc-mpi-v1-CommandService"></a>

### CommandService
A service outlining the command and control options for a DataPlane

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Connect | [ConnectionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionRequest) | [ConnectionResponse](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse) | Connects NGINX Agents to the Management Plane agnostic of instance data |
| Status | [DataPlaneStatus](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus) | [.google.protobuf.Empty](#google-protobuf-Empty) | Reports on instances and their configurations |
| Health | [DataPlaneHealth](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth) | [.google.protobuf.Empty](#google-protobuf-Empty) | Reports on instance health |
| Subscribe | [DataPlaneMessage](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneMessage) stream | [ManagementPlaneMessage](#f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneMessage) stream | A decoupled communication mechanism between the data plane and management plane. |

 



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






<a name="f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest"></a>

### MessageRequest
Meta-information associated with a request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_id | [string](#string) |  | monotonically increasing integer |
| correlation_id | [string](#string) |  | if 2 or more messages associated with the same workflow, use this field as an association |





 


<a name="f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse-CommandStatus"></a>

### CommandResponse.CommandStatus
Command status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| CMD_UNKNOWN | 0 | Unknown status of command |
| CMD_OK | 1 | Command was successful |
| CMD_ERROR | 2 | Command failed |
| CMD_IN_PROGRESS | 3 | Command in-progress |


 

 

 



<a name="file-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## file.proto



<a name="f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion"></a>

### ConfigVersion
Represents a specific configuration version associated with an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |
| version | [string](#string) |  |  |






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
| contents | [FileContents](#f5-nginx-agent-api-grpc-mpi-v1-file-FileContents) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileContents"></a>

### FileContents
Represents the bytes contents of the file https://protobuf.dev/programming-guides/api/#dont-encode-data-in-a-string


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [bytes](#bytes) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta"></a>

### FileMeta
Meta information about the file, the name (including path) and hash


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| hash | [string](#string) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview"></a>

### FileOverview
Represents a collection of files


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [File](#f5-nginx-agent-api-grpc-mpi-v1-file-File) | repeated |  |
| previous_version | [string](#string) |  |  |
| current_version | [string](#string) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileRequest"></a>

### FileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [f5.nginx.agent.api.grpc.mpi.v1.common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| meta | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) |  |  |





 


<a name="f5-nginx-agent-api-grpc-mpi-v1-file-File-FileAction"></a>

### File.FileAction
Action enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNSET | 0 | Default value, no action |
| UNCHANGED | 1 | No changes to the file |
| ADD | 2 | New file |
| UPDATE | 3 | Updated file |
| DELETE | 4 | File deleted |


 

 


<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileService"></a>

### FileService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Overview | [ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) | [FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) | Returns the collection of files for a particular configuration version of an instance |
| GetFile | [FileRequest](#f5-nginx-agent-api-grpc-mpi-v1-file-FileRequest) | [FileContents](#f5-nginx-agent-api-grpc-mpi-v1-file-FileContents) | Get the file contents for a particular file |
| SendFile | [File](#f5-nginx-agent-api-grpc-mpi-v1-file-File) | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) | Send a file from the |

 



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

