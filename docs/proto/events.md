# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [event.proto](#event-proto)
    - [ActivityEvent](#f5-nginx-agent-sdk-events-ActivityEvent)
    - [ContextData](#f5-nginx-agent-sdk-events-ContextData)
    - [Event](#f5-nginx-agent-sdk-events-Event)
    - [EventReport](#f5-nginx-agent-sdk-events-EventReport)
    - [Metadata](#f5-nginx-agent-sdk-events-Metadata)
    - [SecurityViolationEvent](#f5-nginx-agent-sdk-events-SecurityViolationEvent)
    - [SignatureData](#f5-nginx-agent-sdk-events-SignatureData)
    - [ViolationData](#f5-nginx-agent-sdk-events-ViolationData)
  
- [Scalar Value Types](#scalar-value-types)



<a name="event-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## event.proto



<a name="f5-nginx-agent-sdk-events-ActivityEvent"></a>

### ActivityEvent
Represents an activity event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Message | [string](#string) |  | Activtiy event message |
| Dimensions | [f5.nginx.agent.sdk.common.Dimension](#f5-nginx-agent-sdk-common-Dimension) | repeated | Array of dimensions |






<a name="f5-nginx-agent-sdk-events-ContextData"></a>

### ContextData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Name | [string](#string) |  |  |
| Value | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-events-Event"></a>

### Event
Represents an event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Metadata | [Metadata](#f5-nginx-agent-sdk-events-Metadata) |  | Event metadata |
| ActivityEvent | [ActivityEvent](#f5-nginx-agent-sdk-events-ActivityEvent) |  | Activity event |
| SecurityViolationEvent | [SecurityViolationEvent](#f5-nginx-agent-sdk-events-SecurityViolationEvent) |  | Security violation event |






<a name="f5-nginx-agent-sdk-events-EventReport"></a>

### EventReport
Represents an event report


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Events | [Event](#f5-nginx-agent-sdk-events-Event) | repeated | Array of events |






<a name="f5-nginx-agent-sdk-events-Metadata"></a>

### Metadata
Represents the metadata for an event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Module | [string](#string) |  | Module is the process that generate the event |
| UUID | [string](#string) |  | UUID is a unique identifier for each event |
| CorrelationID | [string](#string) |  | CorrelationID is an ID used by the producer of the message to track the flow of events |
| Timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Timestamp defines the time of event generation |
| EventLevel | [string](#string) |  | EventLevel defines the criticality of event |
| Type | [string](#string) |  | Type is used to identify the event type for further processing |
| Category | [string](#string) |  | Category is used for classifying the event type into a higher level entity |






<a name="f5-nginx-agent-sdk-events-SecurityViolationEvent"></a>

### SecurityViolationEvent
Represents a security violation that is emitted by the Agent


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| PolicyName | [string](#string) |  |  |
| SupportID | [string](#string) |  |  |
| Outcome | [string](#string) |  |  |
| OutcomeReason | [string](#string) |  |  |
| BlockingExceptionReason | [string](#string) |  |  |
| Method | [string](#string) |  |  |
| Protocol | [string](#string) |  |  |
| XForwardedForHeaderValue | [string](#string) |  |  |
| URI | [string](#string) |  |  |
| Request | [string](#string) |  |  |
| IsTruncated | [string](#string) |  |  |
| RequestStatus | [string](#string) |  |  |
| ResponseCode | [string](#string) |  |  |
| ServerAddr | [string](#string) |  |  |
| VSName | [string](#string) |  |  |
| RemoteAddr | [string](#string) |  |  |
| RemotePort | [string](#string) |  |  |
| ServerPort | [string](#string) |  |  |
| Violations | [string](#string) |  |  |
| SubViolations | [string](#string) |  |  |
| ViolationRating | [string](#string) |  |  |
| SigSetNames | [string](#string) |  |  |
| SigCVEs | [string](#string) |  |  |
| ClientClass | [string](#string) |  |  |
| ClientApplication | [string](#string) |  |  |
| ClientApplicationVersion | [string](#string) |  |  |
| Severity | [string](#string) |  |  |
| ThreatCampaignNames | [string](#string) |  |  |
| BotAnomalies | [string](#string) |  |  |
| BotCategory | [string](#string) |  |  |
| EnforcedBotAnomalies | [string](#string) |  |  |
| BotSignatureName | [string](#string) |  |  |
| ViolationContexts | [string](#string) |  |  |
| ViolationsData | [ViolationData](#f5-nginx-agent-sdk-events-ViolationData) | repeated |  |
| SystemID | [string](#string) |  |  |
| InstanceTags | [string](#string) |  |  |
| InstanceGroup | [string](#string) |  |  |
| DisplayName | [string](#string) |  |  |
| NginxID | [string](#string) |  |  |
| ParentHostname | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-events-SignatureData"></a>

### SignatureData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  |  |
| BlockingMask | [string](#string) |  |  |
| Buffer | [string](#string) |  |  |
| Offset | [string](#string) |  |  |
| Length | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-events-ViolationData"></a>

### ViolationData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Name | [string](#string) |  |  |
| Context | [string](#string) |  |  |
| ContextData | [ContextData](#f5-nginx-agent-sdk-events-ContextData) |  |  |
| Signatures | [SignatureData](#f5-nginx-agent-sdk-events-SignatureData) | repeated |  |





 

 

 

 



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

