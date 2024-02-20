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
  
    - [File.Action](#f5-nginx-agent-api-grpc-mpi-v1-file-File-Action)
  
    - [FileService](#f5-nginx-agent-api-grpc-mpi-v1-file-FileService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="command-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## command.proto



<a name="f5-nginx-agent-api-grpc-mpi-v1-ActionRequest"></a>

### ActionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |
| action | [InstanceAction](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction) |  |  |
| default_action | [DefaultAction](#f5-nginx-agent-api-grpc-mpi-v1-DefaultAction) |  | add actions as we support new capabilities |






<a name="f5-nginx-agent-api-grpc-mpi-v1-AgentConfig"></a>

### AgentConfig
need to build this out


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [Server](#f5-nginx-agent-api-grpc-mpi-v1-Server) |  | can we connect to more than one management plane? |
| labels | [google.protobuf.Struct](#google-protobuf-Struct) | repeated | Max NAck setting |
| features | [string](#string) | repeated |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-CommandStatusRequest"></a>

### CommandStatusRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConfigApplyRequest"></a>

### ConfigApplyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_version | [file.ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) |  |  |
| overview | [file.FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) |  | optional |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConfigUploadRequest"></a>

### ConfigUploadRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConnectionRequest"></a>

### ConnectionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| agent | [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse"></a>

### ConnectionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| response | [common.CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse) |  |  |
| agent_config | [AgentConfig](#f5-nginx-agent-api-grpc-mpi-v1-AgentConfig) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth"></a>

### DataPlaneHealth



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| instance_health | [InstanceHealth](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth) | repeated |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneMessage"></a>

### DataPlaneMessage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| command_response | [common.CommandResponse](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse) |  | triggers a RPC, acks message has been acted on |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus"></a>

### DataPlaneStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| instances | [Instance](#f5-nginx-agent-api-grpc-mpi-v1-Instance) | repeated |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-DefaultAction"></a>

### DefaultAction



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | [google.protobuf.Struct](#google-protobuf-Struct) | repeated |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-HealthRequest"></a>

### HealthRequest







<a name="f5-nginx-agent-api-grpc-mpi-v1-Instance"></a>

### Instance
only send changed values


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_meta | [InstanceMeta](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta) |  |  |
| instance_config | [InstanceConfig](#f5-nginx-agent-api-grpc-mpi-v1-InstanceConfig) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceAction"></a>

### InstanceAction



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [InstanceAction.InstanceActions](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActions) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceConfig"></a>

### InstanceConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| actions | [InstanceAction](#f5-nginx-agent-api-grpc-mpi-v1-InstanceAction) | repeated | repeated enum of actions?!?

can put into a common object and reuse |
| agent_config | [AgentConfig](#f5-nginx-agent-api-grpc-mpi-v1-AgentConfig) |  |  |
| nginx_config | [NGINXConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXConfig) |  |  |
| nginx_plus_config | [NGINXPlusConfig](#f5-nginx-agent-api-grpc-mpi-v1-NGINXPlusConfig) |  | ... others NIC, NGF, Unit |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth"></a>

### InstanceHealth



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |
| instance_health_status | [InstanceHealth.InstancHealthStatus](#f5-nginx-agent-api-grpc-mpi-v1-InstanceHealth-InstancHealthStatus) |  | Health status |
| description | [string](#string) |  | Provides a human readable context around why a health status is a particular state |






<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta"></a>

### InstanceMeta



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |
| instance_type | [InstanceMeta.InstanceType](#f5-nginx-agent-api-grpc-mpi-v1-InstanceMeta-InstanceType) |  |  |
| version | [string](#string) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneMessage"></a>

### ManagementPlaneMessage



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
this to be built out


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| binary_path | [string](#string) |  |  |
| hostname | [string](#string) |  |  |
| ip_address | [string](#string) |  |  |
| stub_status | [string](#string) |  |  |
| access_logs | [string](#string) | repeated |  |
| error_logs | [string](#string) | repeated |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-NGINXPlusConfig"></a>

### NGINXPlusConfig
this to be built out


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| binary_path | [string](#string) |  |  |
| hostname | [string](#string) |  |  |
| ip_address | [string](#string) |  |  |
| api | [string](#string) |  |  |
| access_logs | [string](#string) | repeated | is this correct for plus? |
| error_logs | [string](#string) | repeated | is this correct for plus? |






<a name="f5-nginx-agent-api-grpc-mpi-v1-Server"></a>

### Server



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  |  |
| port | [string](#string) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-StatusRequest"></a>

### StatusRequest






 


<a name="f5-nginx-agent-api-grpc-mpi-v1-InstanceAction-InstanceActions"></a>

### InstanceAction.InstanceActions


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | TBD |



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


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| AGENT | 1 |  |
| NGINX_OSS | 2 |  |
| NGINX_PLUS | 3 |  |
| UNIT | 4 |  |


 

 


<a name="f5-nginx-agent-api-grpc-mpi-v1-CommandService"></a>

### CommandService
following https://protobuf.dev/programming-guides/style/
and https://static.sched.com/hosted_files/kccncna17/ad/2017%20CloudNativeCon%20-%20Mod%20gRPC%20Services.pdf

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Connect | [ConnectionRequest](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionRequest) | [ConnectionResponse](#f5-nginx-agent-api-grpc-mpi-v1-ConnectionResponse) | connects Agents to the Management Plane |
| Status | [DataPlaneStatus](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneStatus) | [.google.protobuf.Empty](#google-protobuf-Empty) | reports on instances and their configurations |
| Health | [DataPlaneHealth](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneHealth) | [.google.protobuf.Empty](#google-protobuf-Empty) | reports on instance health |
| Subscribe | [DataPlaneMessage](#f5-nginx-agent-api-grpc-mpi-v1-DataPlaneMessage) stream | [ManagementPlaneMessage](#f5-nginx-agent-api-grpc-mpi-v1-ManagementPlaneMessage) stream | other messages |

 



<a name="common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## common.proto



<a name="f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse"></a>

### CommandResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CommandResponse.CommandStatus](#f5-nginx-agent-api-grpc-mpi-v1-common-CommandResponse-CommandStatus) |  | Command status |
| message | [string](#string) |  | Provides a user friendly message to describe the response |
| error | [string](#string) |  | Provides an error message of why the command failed |






<a name="f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest"></a>

### MessageRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_id | [string](#string) |  |  |
| correlation_id | [string](#string) |  |  |





 


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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  |  |
| version | [string](#string) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-File"></a>

### File



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_meta | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) |  | Name of the file |
| modified_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| permissions | [string](#string) |  |  |
| size | [int64](#int64) |  | Size of the file in bytes |
| action | [File.Action](#f5-nginx-agent-api-grpc-mpi-v1-file-File-Action) |  | optional action |
| contents | [FileContents](#f5-nginx-agent-api-grpc-mpi-v1-file-FileContents) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileContents"></a>

### FileContents



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [bytes](#bytes) |  | https://protobuf.dev/programming-guides/api/#dont-encode-data-in-a-string |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta"></a>

### FileMeta



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| hash | [string](#string) |  |  |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview"></a>

### FileOverview



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [File](#f5-nginx-agent-api-grpc-mpi-v1-file-File) | repeated |  |
| previous_version | [string](#string) |  | optional |
| current_version | [string](#string) |  | optional |






<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileRequest"></a>

### FileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_metadata | [f5.nginx.agent.api.grpc.mpi.v1.common.MessageRequest](#f5-nginx-agent-api-grpc-mpi-v1-common-MessageRequest) |  |  |
| meta | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) |  |  |





 


<a name="f5-nginx-agent-api-grpc-mpi-v1-file-File-Action"></a>

### File.Action
Action enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNSET | 0 | Default value |
| UNCHANGED | 1 | No changes to the file

required?!? |
| ADD | 2 | New file |
| UPDATE | 3 | Updated file |
| DELETE | 4 | File deleted |


 

 


<a name="f5-nginx-agent-api-grpc-mpi-v1-file-FileService"></a>

### FileService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Overview | [ConfigVersion](#f5-nginx-agent-api-grpc-mpi-v1-file-ConfigVersion) | [FileOverview](#f5-nginx-agent-api-grpc-mpi-v1-file-FileOverview) |  |
| GetFile | [FileRequest](#f5-nginx-agent-api-grpc-mpi-v1-file-FileRequest) | [FileContents](#f5-nginx-agent-api-grpc-mpi-v1-file-FileContents) |  |
| SendFile | [File](#f5-nginx-agent-api-grpc-mpi-v1-file-File) | [FileMeta](#f5-nginx-agent-api-grpc-mpi-v1-file-FileMeta) |  |

 



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

