# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [mpi/v1/command.proto](#mpi_v1_command-proto)
    - [AgentConfig](#mpi-v1-AgentConfig)
    - [Command](#mpi-v1-Command)
    - [CreateConnectionRequest](#mpi-v1-CreateConnectionRequest)
    - [CreateConnectionResponse](#mpi-v1-CreateConnectionResponse)
    - [DataPlaneResponse](#mpi-v1-DataPlaneResponse)
    - [Instance](#mpi-v1-Instance)
    - [InstanceConfig](#mpi-v1-InstanceConfig)
    - [InstanceMeta](#mpi-v1-InstanceMeta)
    - [ManagementPlaneRequest](#mpi-v1-ManagementPlaneRequest)
    - [Metrics](#mpi-v1-Metrics)
    - [UpdateDataPlaneHealthRequest](#mpi-v1-UpdateDataPlaneHealthRequest)
    - [UpdateDataPlaneHealthResponse](#mpi-v1-UpdateDataPlaneHealthResponse)
    - [UpdateDataPlaneStatusRequest](#mpi-v1-UpdateDataPlaneStatusRequest)
    - [UpdateDataPlaneStatusResponse](#mpi-v1-UpdateDataPlaneStatusResponse)
  
    - [InstanceMeta.InstanceType](#mpi-v1-InstanceMeta-InstanceType)
  
    - [CommandService](#mpi-v1-CommandService)
  
- [mpi/v1/common.proto](#mpi_v1_common-proto)
    - [CommandResponse](#mpi-v1-CommandResponse)
    - [MessageMeta](#mpi-v1-MessageMeta)
  
    - [CommandResponse.CommandStatus](#mpi-v1-CommandResponse-CommandStatus)
  
- [Scalar Value Types](#scalar-value-types)



<a name="mpi_v1_command-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mpi/v1/command.proto
These proto definitions follow https://protobuf.dev/programming-guides/style/
and recommendations outlined in https://static.sched.com/hosted_files/kccncna17/ad/2017%20CloudNativeCon%20-%20Mod%20gRPC%20Services.pdf


<a name="mpi-v1-AgentConfig"></a>

### AgentConfig
This contains a series of NGINX Agent configurations


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| command | [Command](#mpi-v1-Command) |  | Command server settings |
| metrics | [Metrics](#mpi-v1-Metrics) |  | Metrics server settings |
| labels | [google.protobuf.Struct](#google-protobuf-Struct) | repeated | A series of key/value pairs to add more data to the NGINX Agent instance |
| features | [string](#string) | repeated | A list of features that the NGINX Agent has |
| message_buffer_size | [string](#string) |  | Message buffer size, maximum not acknowledged messages from the subscribe perspective |






<a name="mpi-v1-Command"></a>

### Command
The command settings, associated with messaging from an external source






<a name="mpi-v1-CreateConnectionRequest"></a>

### CreateConnectionRequest
The connection request is an intial handshake to establish a connection, sending NGINX Agent instance information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| agent | [Instance](#mpi-v1-Instance) |  | instance information associated with the NGINX Agent |






<a name="mpi-v1-CreateConnectionResponse"></a>

### CreateConnectionResponse
A response to a CreateConnectionRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| response | [CommandResponse](#mpi-v1-CommandResponse) |  | the success or failure of the CreateConnectionRequest |
| agent_config | [AgentConfig](#mpi-v1-AgentConfig) |  | the recommendation NGINX Agent configurations provided by the ManagementPlane |






<a name="mpi-v1-DataPlaneResponse"></a>

### DataPlaneResponse
Reports the status of an associated command. This may be in response to a ManagementPlaneRequest






<a name="mpi-v1-Instance"></a>

### Instance
This represents an instance being reported on


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_meta | [InstanceMeta](#mpi-v1-InstanceMeta) |  | Meta-information associated with an instance |
| instance_config | [InstanceConfig](#mpi-v1-InstanceConfig) |  | Runtime configuration associated with an instance |






<a name="mpi-v1-InstanceConfig"></a>

### InstanceConfig
Instance Configuration options


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent_config | [AgentConfig](#mpi-v1-AgentConfig) |  | NGINX Agent runtime configuration settings |






<a name="mpi-v1-InstanceMeta"></a>

### InstanceMeta
Metainformation relating to the reported instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the identifier associated with the instance |
| instance_type | [InstanceMeta.InstanceType](#mpi-v1-InstanceMeta-InstanceType) |  | the types of instances possible |
| version | [string](#string) |  | the version of the instance |






<a name="mpi-v1-ManagementPlaneRequest"></a>

### ManagementPlaneRequest
A Management Plane request for information, triggers an associated rpc on the DataPlane






<a name="mpi-v1-Metrics"></a>

### Metrics
The metrics settings associated with orgins (sources) of the metrics and destinations (exporter)






<a name="mpi-v1-UpdateDataPlaneHealthRequest"></a>

### UpdateDataPlaneHealthRequest
Health report of a set of instances






<a name="mpi-v1-UpdateDataPlaneHealthResponse"></a>

### UpdateDataPlaneHealthResponse
Response to a UpdateDataPlaneHealthRequest - intentionally empty






<a name="mpi-v1-UpdateDataPlaneStatusRequest"></a>

### UpdateDataPlaneStatusRequest
Report on the status of the Data Plane






<a name="mpi-v1-UpdateDataPlaneStatusResponse"></a>

### UpdateDataPlaneStatusResponse
Respond to a UpdateDataPlaneStatusRequest - intentionally empty





 


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
| CreateConnection | [CreateConnectionRequest](#mpi-v1-CreateConnectionRequest) | [CreateConnectionResponse](#mpi-v1-CreateConnectionResponse) | Connects NGINX Agent to the Management Plane agnostic of instance data |
| UpdateDataPlaneStatus | [UpdateDataPlaneStatusRequest](#mpi-v1-UpdateDataPlaneStatusRequest) | [UpdateDataPlaneStatusResponse](#mpi-v1-UpdateDataPlaneStatusResponse) | Reports on instances and their configurations |
| UpdateDataPlaneHealth | [UpdateDataPlaneHealthRequest](#mpi-v1-UpdateDataPlaneHealthRequest) | [UpdateDataPlaneHealthResponse](#mpi-v1-UpdateDataPlaneHealthResponse) | Reports on instance health |
| Subscribe | [DataPlaneResponse](#mpi-v1-DataPlaneResponse) stream | [ManagementPlaneRequest](#mpi-v1-ManagementPlaneRequest) stream | A decoupled communication mechanism between the data plane and management plane. buf:lint:ignore RPC_RESPONSE_STANDARD_NAME buf:lint:ignore RPC_REQUEST_STANDARD_NAME |

 



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
| COMMAND_STATUS_ERROR | 2 | Command failed |
| COMMAND_STATUS_IN_PROGRESS | 3 | Command in-progress |


 

 

 



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

