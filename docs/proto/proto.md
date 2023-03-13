# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [agent.proto](#agent-proto)
    - [AgentConfig](#f5-nginx-agent-sdk-AgentConfig)
    - [AgentConfigRequest](#f5-nginx-agent-sdk-AgentConfigRequest)
    - [AgentConnectRequest](#f5-nginx-agent-sdk-AgentConnectRequest)
    - [AgentConnectResponse](#f5-nginx-agent-sdk-AgentConnectResponse)
    - [AgentConnectStatus](#f5-nginx-agent-sdk-AgentConnectStatus)
    - [AgentDetails](#f5-nginx-agent-sdk-AgentDetails)
    - [AgentLogging](#f5-nginx-agent-sdk-AgentLogging)
    - [AgentMeta](#f5-nginx-agent-sdk-AgentMeta)
  
    - [AgentConnectStatus.StatusCode](#f5-nginx-agent-sdk-AgentConnectStatus-StatusCode)
    - [AgentLogging.Level](#f5-nginx-agent-sdk-AgentLogging-Level)
  
- [command.proto](#command-proto)
    - [AgentActivityStatus](#f5-nginx-agent-sdk-AgentActivityStatus)
    - [ChunkedResourceChunk](#f5-nginx-agent-sdk-ChunkedResourceChunk)
    - [ChunkedResourceHeader](#f5-nginx-agent-sdk-ChunkedResourceHeader)
    - [Command](#f5-nginx-agent-sdk-Command)
    - [CommandStatusResponse](#f5-nginx-agent-sdk-CommandStatusResponse)
    - [DataChunk](#f5-nginx-agent-sdk-DataChunk)
    - [DataplaneSoftwareHealth](#f5-nginx-agent-sdk-DataplaneSoftwareHealth)
    - [DataplaneStatus](#f5-nginx-agent-sdk-DataplaneStatus)
    - [DataplaneUpdate](#f5-nginx-agent-sdk-DataplaneUpdate)
    - [DownloadRequest](#f5-nginx-agent-sdk-DownloadRequest)
    - [NginxConfigResponse](#f5-nginx-agent-sdk-NginxConfigResponse)
    - [NginxConfigStatus](#f5-nginx-agent-sdk-NginxConfigStatus)
    - [UploadStatus](#f5-nginx-agent-sdk-UploadStatus)
  
    - [Command.CommandType](#f5-nginx-agent-sdk-Command-CommandType)
    - [CommandStatusResponse.CommandErrorCode](#f5-nginx-agent-sdk-CommandStatusResponse-CommandErrorCode)
    - [CommandStatusResponse.CommandStatus](#f5-nginx-agent-sdk-CommandStatusResponse-CommandStatus)
    - [NginxConfigStatus.Status](#f5-nginx-agent-sdk-NginxConfigStatus-Status)
    - [UploadStatus.TransferStatus](#f5-nginx-agent-sdk-UploadStatus-TransferStatus)
  
- [command_svc.proto](#command_svc-proto)
    - [Commander](#f5-nginx-agent-sdk-Commander)
  
- [common.proto](#common-proto)
    - [CertificateDates](#f5-nginx-agent-sdk-CertificateDates)
    - [CertificateName](#f5-nginx-agent-sdk-CertificateName)
    - [Directory](#f5-nginx-agent-sdk-Directory)
    - [DirectoryMap](#f5-nginx-agent-sdk-DirectoryMap)
    - [File](#f5-nginx-agent-sdk-File)
    - [Metadata](#f5-nginx-agent-sdk-Metadata)
    - [SslCertificate](#f5-nginx-agent-sdk-SslCertificate)
    - [SslCertificates](#f5-nginx-agent-sdk-SslCertificates)
    - [ZippedFile](#f5-nginx-agent-sdk-ZippedFile)
  
    - [File.Action](#f5-nginx-agent-sdk-File-Action)
  
- [config.proto](#config-proto)
    - [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor)
    - [ConfigReport](#f5-nginx-agent-sdk-ConfigReport)
  
- [dp_software_details.proto](#dp_software_details-proto)
    - [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails)
  
- [host.proto](#host-proto)
    - [Address](#f5-nginx-agent-sdk-Address)
    - [CpuInfo](#f5-nginx-agent-sdk-CpuInfo)
    - [CpuInfo.CacheEntry](#f5-nginx-agent-sdk-CpuInfo-CacheEntry)
    - [DiskPartition](#f5-nginx-agent-sdk-DiskPartition)
    - [HostInfo](#f5-nginx-agent-sdk-HostInfo)
    - [Network](#f5-nginx-agent-sdk-Network)
    - [NetworkInterface](#f5-nginx-agent-sdk-NetworkInterface)
    - [ReleaseInfo](#f5-nginx-agent-sdk-ReleaseInfo)
  
- [metrics.proto](#metrics-proto)
    - [Dimension](#f5-nginx-agent-sdk-Dimension)
    - [MetricsReport](#f5-nginx-agent-sdk-MetricsReport)
    - [SimpleMetric](#f5-nginx-agent-sdk-SimpleMetric)
    - [StatsEntity](#f5-nginx-agent-sdk-StatsEntity)
  
    - [MetricsReport.Type](#f5-nginx-agent-sdk-MetricsReport-Type)
  
- [metrics.svc.proto](#metrics-svc-proto)
    - [MetricsService](#f5-nginx-agent-sdk-MetricsService)
  
- [nap.proto](#nap-proto)
    - [AppProtectWAFDetails](#f5-nginx-agent-sdk-AppProtectWAFDetails)
    - [AppProtectWAFHealth](#f5-nginx-agent-sdk-AppProtectWAFHealth)
  
    - [AppProtectWAFHealth.AppProtectWAFStatus](#f5-nginx-agent-sdk-AppProtectWAFHealth-AppProtectWAFStatus)
  
- [nginx.proto](#nginx-proto)
    - [AccessLog](#f5-nginx-agent-sdk-AccessLog)
    - [AccessLogs](#f5-nginx-agent-sdk-AccessLogs)
    - [ErrorLog](#f5-nginx-agent-sdk-ErrorLog)
    - [ErrorLogs](#f5-nginx-agent-sdk-ErrorLogs)
    - [NginxConfig](#f5-nginx-agent-sdk-NginxConfig)
    - [NginxDetails](#f5-nginx-agent-sdk-NginxDetails)
    - [NginxHealth](#f5-nginx-agent-sdk-NginxHealth)
    - [NginxPlusMetaData](#f5-nginx-agent-sdk-NginxPlusMetaData)
    - [NginxSslMetaData](#f5-nginx-agent-sdk-NginxSslMetaData)
  
    - [NginxConfigAction](#f5-nginx-agent-sdk-NginxConfigAction)
    - [NginxHealth.NginxStatus](#f5-nginx-agent-sdk-NginxHealth-NginxStatus)
    - [NginxSslMetaData.NginxSslType](#f5-nginx-agent-sdk-NginxSslMetaData-NginxSslType)
  
- [Scalar Value Types](#scalar-value-types)



<a name="agent-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## agent.proto



<a name="f5-nginx-agent-sdk-AgentConfig"></a>

### AgentConfig
Represents an agent&#39;s configuration. The message is sent from the management server to the agent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| details | [AgentDetails](#f5-nginx-agent-sdk-AgentDetails) |  | Provides information about the agent |
| loggers | [AgentLogging](#f5-nginx-agent-sdk-AgentLogging) |  | Provides information about the agent logging. This is will be implemented in a future release. |
| configs | [ConfigReport](#f5-nginx-agent-sdk-ConfigReport) |  | Provides meta information about the nginx configurations |






<a name="f5-nginx-agent-sdk-AgentConfigRequest"></a>

### AgentConfigRequest
Represents an agent config request that is sent from the agent to the management server.
This is used by the agent to request the agent configuration from the management server.






<a name="f5-nginx-agent-sdk-AgentConnectRequest"></a>

### AgentConnectRequest
Represents an agent connect request that is sent from the agent to the management server


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [AgentMeta](#f5-nginx-agent-sdk-AgentMeta) |  | Provides meta information about the agent |
| details | [NginxDetails](#f5-nginx-agent-sdk-NginxDetails) | repeated | Provides information about the NGINX instances that are present. This data will be moving to dataplane_software_details in a future release |
| host | [HostInfo](#f5-nginx-agent-sdk-HostInfo) |  | Provides information about the host system |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) | repeated | Provides information about software installed in the system (e.g. App Protect WAF, NGINX, etc.) |






<a name="f5-nginx-agent-sdk-AgentConnectResponse"></a>

### AgentConnectResponse
Represents an agent connect response that is sent from the management server to the agent


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent_config | [AgentConfig](#f5-nginx-agent-sdk-AgentConfig) |  | Agent configuration |
| status | [AgentConnectStatus](#f5-nginx-agent-sdk-AgentConnectStatus) |  | Agent connect request status |






<a name="f5-nginx-agent-sdk-AgentConnectStatus"></a>

### AgentConnectStatus
Represents an agent connect status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statusCode | [AgentConnectStatus.StatusCode](#f5-nginx-agent-sdk-AgentConnectStatus-StatusCode) |  | Provides a status of the agent connect response |
| message | [string](#string) |  | Provides a user friendly message to describe the response |
| error | [string](#string) |  | Provides an error message of why the agent connect request was rejected |






<a name="f5-nginx-agent-sdk-AgentDetails"></a>

### AgentDetails
Represents agent details. This message is sent from the management server to the agent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| features | [string](#string) | repeated | List of agent feature that are enabled |
| extensions | [string](#string) | repeated | List of agent extensions that are enabled |
| tags | [string](#string) | repeated | List of tags |
| alias | [string](#string) |  | Alias name for the agent |






<a name="f5-nginx-agent-sdk-AgentLogging"></a>

### AgentLogging
Represents agent logging details


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [AgentLogging.Level](#f5-nginx-agent-sdk-AgentLogging-Level) |  | Log level |
| dir | [string](#string) |  | Directory where the logs are located |
| file | [string](#string) |  | Name of the log file |
| max_size | [uint32](#uint32) |  | Max size of the log file in MB |
| max_backups | [uint32](#uint32) |  | Max number of backups |
| max_age | [uint32](#uint32) |  | Max age of a log file in days |
| compress | [bool](#bool) |  | Is the log file compressed |






<a name="f5-nginx-agent-sdk-AgentMeta"></a>

### AgentMeta
Represents agent metadata


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | Version of the agent |
| display_name | [string](#string) |  | User friendly name for the agent |
| tag | [string](#string) | repeated | List of tags |
| instance_group | [string](#string) |  | Instance group name used to group NGINX instances |
| updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Last time agent was updated |
| system_uid | [string](#string) |  | ID of the system where the agent is installed |
| agent_details | [AgentDetails](#f5-nginx-agent-sdk-AgentDetails) |  | Provides other agent information |





 


<a name="f5-nginx-agent-sdk-AgentConnectStatus-StatusCode"></a>

### AgentConnectStatus.StatusCode
Different status codes for agent connect response

| Name | Number | Description |
| ---- | ------ | ----------- |
| CONNECT_UNKNOWN | 0 | Unknown status of the agent connect request |
| CONNECT_OK | 1 | Agent connect request was successful |
| CONNECT_REJECTED_OTHER | 2 | Agent connect request was rejected |
| CONNECT_REJECTED_DUP_ID | 3 | Agent connect request was rejected because an agent with the same ID is already registered |



<a name="f5-nginx-agent-sdk-AgentLogging-Level"></a>

### AgentLogging.Level
Log level enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| INFO | 0 | info level |
| DEBUG | 1 | debug level |
| WARN | 2 | warn level |
| ERROR | 3 | error level |
| FATAL | 4 | fatal level |


 

 

 



<a name="command-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## command.proto



<a name="f5-nginx-agent-sdk-AgentActivityStatus"></a>

### AgentActivityStatus
Represent an agent activity status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_config_status | [NginxConfigStatus](#f5-nginx-agent-sdk-NginxConfigStatus) |  | NGINX configuration status |






<a name="f5-nginx-agent-sdk-ChunkedResourceChunk"></a>

### ChunkedResourceChunk
Represents a chunked resource chunk


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | Metadata information |
| chunk_id | [int32](#int32) |  | Chunk ID |
| data | [bytes](#bytes) |  | Chunk data |






<a name="f5-nginx-agent-sdk-ChunkedResourceHeader"></a>

### ChunkedResourceHeader
Represents a chunked resource Header


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | Metadata information |
| chunks | [int32](#int32) |  | Number of chunks expected in the transfer |
| checksum | [string](#string) |  | Chunk checksum |
| chunk_size | [int32](#int32) |  | Chunk size |






<a name="f5-nginx-agent-sdk-Command"></a>

### Command
Represents a command message, which is used for communication between the management server and the agent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | Provides metadata information associated with the command |
| type | [Command.CommandType](#f5-nginx-agent-sdk-Command-CommandType) |  | Used to determine the type of command |
| cmd_status | [CommandStatusResponse](#f5-nginx-agent-sdk-CommandStatusResponse) |  | Common command status response |
| nginx_config | [NginxConfig](#f5-nginx-agent-sdk-NginxConfig) |  | Used by the management server to notify the agent to download or upload NGINX configuration. |
| nginx_config_response | [NginxConfigResponse](#f5-nginx-agent-sdk-NginxConfigResponse) |  | Response sent to indicate if a NGINX config apply was successful or not |
| agent_connect_request | [AgentConnectRequest](#f5-nginx-agent-sdk-AgentConnectRequest) |  | Agent connect request that is sent from the agent to the management server to initialize registration |
| agent_connect_response | [AgentConnectResponse](#f5-nginx-agent-sdk-AgentConnectResponse) |  | Agent connect response that is sent from the management server to the agent to finalize registration |
| agent_config_request | [AgentConfigRequest](#f5-nginx-agent-sdk-AgentConfigRequest) |  | Agent config request that is sent by the agent to the management server to request agent configuration |
| agent_config | [AgentConfig](#f5-nginx-agent-sdk-AgentConfig) |  | Agent Config is sent by the management server to the agent when is receives an AgentConfigRequest from the agent |
| dataplane_status | [DataplaneStatus](#f5-nginx-agent-sdk-DataplaneStatus) |  | Dataplane status is sent by the agent to the management server to report the information like the health of the system |
| event_report | [events.EventReport](#f5-nginx-agent-sdk-events-EventReport) |  | Reports events the agent is aware of like the start/stop of the agent, NGINX config applies, etc. |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) |  | Provides details of additional software running on the dataplane |
| dataplane_update | [DataplaneUpdate](#f5-nginx-agent-sdk-DataplaneUpdate) |  | Provides details of any changes on the dataplane |






<a name="f5-nginx-agent-sdk-CommandStatusResponse"></a>

### CommandStatusResponse
Represents a command status response


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CommandStatusResponse.CommandStatus](#f5-nginx-agent-sdk-CommandStatusResponse-CommandStatus) |  | Command status |
| error_code | [CommandStatusResponse.CommandErrorCode](#f5-nginx-agent-sdk-CommandStatusResponse-CommandErrorCode) |  | Error code |
| message | [string](#string) |  | Provides a user friendly message to describe the response |
| error | [string](#string) |  | Provides an error message of why the command failed |






<a name="f5-nginx-agent-sdk-DataChunk"></a>

### DataChunk
Represents a data chunck


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| header | [ChunkedResourceHeader](#f5-nginx-agent-sdk-ChunkedResourceHeader) |  | Chunk header |
| data | [ChunkedResourceChunk](#f5-nginx-agent-sdk-ChunkedResourceChunk) |  | Chunk data |






<a name="f5-nginx-agent-sdk-DataplaneSoftwareHealth"></a>

### DataplaneSoftwareHealth
Represents a dataplane software health


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_health | [NginxHealth](#f5-nginx-agent-sdk-NginxHealth) |  | Health of NGINX instance |
| app_protect_waf_health | [AppProtectWAFHealth](#f5-nginx-agent-sdk-AppProtectWAFHealth) |  | Health of App Protect WAF |






<a name="f5-nginx-agent-sdk-DataplaneStatus"></a>

### DataplaneStatus
Represents a dataplane status, which is used by the agent to periodically report the status of NGINX, agent activities and other dataplane software activities.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| system_id | [string](#string) |  | System ID |
| details | [NginxDetails](#f5-nginx-agent-sdk-NginxDetails) | repeated | List of NGINX details. This field will be moving to DataplaneSoftwareDetails in a future release. |
| host | [HostInfo](#f5-nginx-agent-sdk-HostInfo) |  | Host information |
| healths | [NginxHealth](#f5-nginx-agent-sdk-NginxHealth) | repeated | List of NGINX health information. This field will be moving to DataplaneSoftwareHealth in a future release. |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) | repeated | List of software details. This includes details about NGINX and any other software installed in the system that the agent is interested in. |
| dataplane_software_healths | [DataplaneSoftwareHealth](#f5-nginx-agent-sdk-DataplaneSoftwareHealth) | repeated | List of software health statues. This includes the health of NGINX and any other software installed in the system that the agent is interested in. |
| agent_activity_status | [AgentActivityStatus](#f5-nginx-agent-sdk-AgentActivityStatus) | repeated | List of activity statuses. Reports on the status of activities that the agent is currently executing. |






<a name="f5-nginx-agent-sdk-DataplaneUpdate"></a>

### DataplaneUpdate
Represents a dataplane update


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [HostInfo](#f5-nginx-agent-sdk-HostInfo) |  | Host information |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) | repeated | List of software details. This includes details about NGINX and any other software installed in the system that the agent is interested in. |






<a name="f5-nginx-agent-sdk-DownloadRequest"></a>

### DownloadRequest
Represents a download request


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | Metadata information |






<a name="f5-nginx-agent-sdk-NginxConfigResponse"></a>

### NginxConfigResponse
Represents a NGINX config response


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CommandStatusResponse](#f5-nginx-agent-sdk-CommandStatusResponse) |  | Command status |
| action | [NginxConfigAction](#f5-nginx-agent-sdk-NginxConfigAction) |  | NGINX config action |
| config_data | [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor) |  | NGINX config description |






<a name="f5-nginx-agent-sdk-NginxConfigStatus"></a>

### NginxConfigStatus
Represents a NGINX configuration status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| correlation_id | [string](#string) |  | CorrelationID is an ID used by the producer of the message to track the flow of events |
| status | [NginxConfigStatus.Status](#f5-nginx-agent-sdk-NginxConfigStatus-Status) |  | Provides a status for the NGINX configuration |
| message | [string](#string) |  | Provides a user friendly message to describe the current state of the NGINX configuration. |
| nginx_id | [string](#string) |  | NGINX ID |






<a name="f5-nginx-agent-sdk-UploadStatus"></a>

### UploadStatus
Represents an upload status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | Metadata information |
| status | [UploadStatus.TransferStatus](#f5-nginx-agent-sdk-UploadStatus-TransferStatus) |  | Transfer status |
| reason | [string](#string) |  | Provides an error message of why the upload failed |





 


<a name="f5-nginx-agent-sdk-Command-CommandType"></a>

### Command.CommandType
Command type enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| NORMAL | 0 | All commands default to normal |
| DOWNLOAD | 1 | The download type is used when sending NginxConfig from the management server to the agent. It is used to instruct the agent to download the NGINX config from the management server. |
| UPLOAD | 2 | The upload type is used when sending NginxConfig from the agent to the management server. It is used to instruct the agent to upload the NGINX config from the agent. This will be implemented in a future release. |



<a name="f5-nginx-agent-sdk-CommandStatusResponse-CommandErrorCode"></a>

### CommandStatusResponse.CommandErrorCode
Command error code enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| ERR_OK | 0 | No Error (This is the default value) |
| ERR_UNKNOWN | 1 | Unknown error |



<a name="f5-nginx-agent-sdk-CommandStatusResponse-CommandStatus"></a>

### CommandStatusResponse.CommandStatus
Command status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| CMD_UNKNOWN | 0 | Unknown status of command |
| CMD_OK | 1 | Command was successful |
| CMD_ERROR | 2 | Command failed |



<a name="f5-nginx-agent-sdk-NginxConfigStatus-Status"></a>

### NginxConfigStatus.Status
NGINX configuration status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| PENDING | 0 | The configuration is still in the process of being applied. |
| OK | 1 | The configuration has being successfully applied. |
| ERROR | 2 | The configuration has failed to be applied |



<a name="f5-nginx-agent-sdk-UploadStatus-TransferStatus"></a>

### UploadStatus.TransferStatus
Transfer status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | Unknown status |
| OK | 1 | Upload was successful |
| FAILED | 2 | Upload failed |


 

 

 



<a name="command_svc-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## command_svc.proto


 

 

 


<a name="f5-nginx-agent-sdk-Commander"></a>

### Commander
Represents a service used to sent command messages between the management server and the agent.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CommandChannel | [Command](#f5-nginx-agent-sdk-Command) stream | [Command](#f5-nginx-agent-sdk-Command) stream | A Bidirectional streaming RPC established by the agent and is kept open |
| Download | [DownloadRequest](#f5-nginx-agent-sdk-DownloadRequest) | [DataChunk](#f5-nginx-agent-sdk-DataChunk) stream | A streaming RPC established by the agent and is used to download resources associated with commands The download stream will be kept open for the duration of the data transfer and will be closed when its done. The transfer is a stream of chunks as follows: header -&gt; data chunk 1 -&gt; data chunk N. Each data chunk is of a size smaller than the maximum gRPC payload |
| Upload | [DataChunk](#f5-nginx-agent-sdk-DataChunk) stream | [UploadStatus](#f5-nginx-agent-sdk-UploadStatus) | A streaming RPC established by the agent and is used to upload resources associated with commands |

 



<a name="common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## common.proto



<a name="f5-nginx-agent-sdk-CertificateDates"></a>

### CertificateDates
Represents the dates for which a certificate is valid


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| not_before | [int64](#int64) |  | The start date that for when the certificate is valid |
| not_after | [int64](#int64) |  | The end date that for when the certificate is valid |






<a name="f5-nginx-agent-sdk-CertificateName"></a>

### CertificateName
Represents a Distinguished Name (DN)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| common_name | [string](#string) |  | The fully qualified domain name (e.g. www.example.com) |
| country | [string](#string) | repeated | Country |
| state | [string](#string) | repeated | State |
| locality | [string](#string) | repeated | Locality |
| organization | [string](#string) | repeated | Organization |
| organizational_unit | [string](#string) | repeated | Organizational Unit |






<a name="f5-nginx-agent-sdk-Directory"></a>

### Directory
Represents a directory


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the directory |
| mtime | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the directory was last modified |
| permissions | [string](#string) |  | Directory permissions (e.g. 0644) |
| size | [int64](#int64) |  | Size of the directory in bytes |
| files | [File](#f5-nginx-agent-sdk-File) | repeated | List of files in the directory |






<a name="f5-nginx-agent-sdk-DirectoryMap"></a>

### DirectoryMap
Represents a map of directories &amp; files on the system


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| directories | [Directory](#f5-nginx-agent-sdk-Directory) | repeated | List of directories |






<a name="f5-nginx-agent-sdk-File"></a>

### File
Represents a file


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the file |
| lines | [int32](#int32) |  | Number of lines in the file |
| mtime | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the file was last modified |
| permissions | [string](#string) |  | File permissions (e.g. 0644) |
| size | [int64](#int64) |  | Size of the file in bytes |
| contents | [bytes](#bytes) |  | The contents of the file in bytes |
| action | [File.Action](#f5-nginx-agent-sdk-File-Action) |  | Action to take on the file (e.g. update, delete, etc) |






<a name="f5-nginx-agent-sdk-Metadata"></a>

### Metadata
Represents the metadata for a message


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | timestamp defines the time of message creation |
| client_id | [string](#string) |  | Client ID |
| message_id | [string](#string) |  | Message ID |
| cloud_account_id | [string](#string) |  | Cloud Account ID (e.g. AWS/Azure/GCP account ID) |






<a name="f5-nginx-agent-sdk-SslCertificate"></a>

### SslCertificate
Represents a SSL certificate file


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_name | [string](#string) |  | Name of the file |
| size | [int64](#int64) |  | Size of the file in bytes |
| mtime | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the file was last modified |
| validity | [CertificateDates](#f5-nginx-agent-sdk-CertificateDates) |  | A time when the certificate is valid |
| issuer | [CertificateName](#f5-nginx-agent-sdk-CertificateName) |  | This field contains the distinguished name (DN) of the certificate issuer |
| subject | [CertificateName](#f5-nginx-agent-sdk-CertificateName) |  | This dedicated object name associated with the public key, for which the certificate is issued |
| subj_alt_names | [string](#string) | repeated | Subject alternative names that allows users to specify additional host names for the SSL certificate |
| ocsp_url | [string](#string) | repeated | Online Certificate Status Protocol URL |
| public_key_algorithm | [string](#string) |  | Public key encryption algorithm (e.g. RSA) |
| signature_algorithm | [string](#string) |  | The signature algorithm contain a hashing algorithm and an encryption algorithm (e.g. sha256RSA where sha256 is the hashing algorithm and RSA is the encryption algorithm) |
| serial_number | [string](#string) |  | Used to uniquely identify the certificate within a CA&#39;s systems |
| subject_key_identifier | [string](#string) |  | The subject key identifier extension provides a means of identifying certificates that contain a particular public key |
| fingerprint | [string](#string) |  | SSL certificate fingerprint |
| fingerprint_algorithm | [string](#string) |  | SSL certificate fingerprint algorithm |
| version | [int64](#int64) |  | There are three versions of certificates: 1, 2 and 3, numbered as 0, 1 and 2. Version 1 supports only the basic fields; Version 2 adds unique identifiers, which represent two additional fields; Version 3 adds extensions. |
| authority_key_identifier | [string](#string) |  | The authority key identifier extension provides a means of identifying the Public Key corresponding to the Private Key used to sign a certificate |






<a name="f5-nginx-agent-sdk-SslCertificates"></a>

### SslCertificates
Represents a list of SSL certificates files


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ssl_certs | [SslCertificate](#f5-nginx-agent-sdk-SslCertificate) | repeated | List of SSL certificates |






<a name="f5-nginx-agent-sdk-ZippedFile"></a>

### ZippedFile
Represents a zipped file


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [bytes](#bytes) |  | The contents of the file in bytes |
| checksum | [string](#string) |  | File checksum |
| root_directory | [string](#string) |  | The directory where the file is located |





 


<a name="f5-nginx-agent-sdk-File-Action"></a>

### File.Action
Action enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| unset | 0 | Default value |
| unchanged | 1 | No changes to the file |
| add | 2 | New file |
| update | 3 | Updated file |
| delete | 4 | File deleted |


 

 

 



<a name="config-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config.proto



<a name="f5-nginx-agent-sdk-ConfigDescriptor"></a>

### ConfigDescriptor
Represents a config descriptor


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| system_id | [string](#string) |  | System ID |
| nginx_id | [string](#string) |  | NGINX ID |
| checksum | [string](#string) |  | Config file checksum |






<a name="f5-nginx-agent-sdk-ConfigReport"></a>

### ConfigReport
Represents a config report


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | Provides metadata information associated with the message |
| configs | [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor) | repeated | List of NGINX config descriptors |





 

 

 

 



<a name="dp_software_details-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## dp_software_details.proto



<a name="f5-nginx-agent-sdk-DataplaneSoftwareDetails"></a>

### DataplaneSoftwareDetails
Represents dataplane software details which contains details for additional software running on the dataplane that pertains to NGINX Agent


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_protect_waf_details | [AppProtectWAFDetails](#f5-nginx-agent-sdk-AppProtectWAFDetails) |  | App Protect WAF software details |
| nginx_details | [NginxDetails](#f5-nginx-agent-sdk-NginxDetails) |  | NGINX software details |





 

 

 

 



<a name="host-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## host.proto



<a name="f5-nginx-agent-sdk-Address"></a>

### Address
Represents an IP address


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefixlen | [int64](#int64) |  | Prefix length |
| netmask | [string](#string) |  | Netmask |
| address | [string](#string) |  | IP Address |






<a name="f5-nginx-agent-sdk-CpuInfo"></a>

### CpuInfo
Represents CPU information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| model | [string](#string) |  | Model of CPU |
| cores | [int32](#int32) |  | Number of cores |
| architecture | [string](#string) |  | CPU architecture |
| mhz | [double](#double) |  | CPU clock speed in MHz |
| hypervisor | [string](#string) |  | Hypervisor (e.g. VMWare, KVM, etc.) |
| cpus | [int32](#int32) |  | Total number of CPUs |
| virtualization | [string](#string) |  | Type of hypervisor (e.g guest or host) |
| cache | [CpuInfo.CacheEntry](#f5-nginx-agent-sdk-CpuInfo-CacheEntry) | repeated | Map of caches with names as the keys and size in bytes as the values |






<a name="f5-nginx-agent-sdk-CpuInfo-CacheEntry"></a>

### CpuInfo.CacheEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-DiskPartition"></a>

### DiskPartition
Represents a disk partition


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mount_point | [string](#string) |  | Mount point location |
| device | [string](#string) |  | Device file path |
| fs_type | [string](#string) |  | File system type (e.g. hfs, swap, etc) |






<a name="f5-nginx-agent-sdk-HostInfo"></a>

### HostInfo
Represents the host system information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent | [string](#string) |  | NGINX Agent version |
| boot | [uint64](#uint64) |  | Host boot time |
| hostname | [string](#string) |  | Hostname |
| display_name | [string](#string) |  | Display Name |
| os_type | [string](#string) |  | OS type (e.g. freebsd, linux, etc) |
| uuid | [string](#string) |  | Host UUID |
| uname | [string](#string) |  | The native cpu architecture queried at runtime, as returned by `uname -m` or empty string in case of error |
| partitons | [DiskPartition](#f5-nginx-agent-sdk-DiskPartition) | repeated | List of disk partitions |
| network | [Network](#f5-nginx-agent-sdk-Network) |  | Network information |
| processor | [CpuInfo](#f5-nginx-agent-sdk-CpuInfo) | repeated | List of CPU processor information |
| release | [ReleaseInfo](#f5-nginx-agent-sdk-ReleaseInfo) |  | Release Information |
| tags | [string](#string) | repeated | List of tags |
| agent_accessible_dirs | [string](#string) |  | List of directories that the NGINX Agent is allowed to access on the host |






<a name="f5-nginx-agent-sdk-Network"></a>

### Network
Represents a network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| interfaces | [NetworkInterface](#f5-nginx-agent-sdk-NetworkInterface) | repeated | List of network interfaces |
| default | [string](#string) |  | Default network name |






<a name="f5-nginx-agent-sdk-NetworkInterface"></a>

### NetworkInterface
Represents a network interface


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mac | [string](#string) |  | MAC address |
| ipv6 | [Address](#f5-nginx-agent-sdk-Address) | repeated | List of IPV6 addresses |
| ipv4 | [Address](#f5-nginx-agent-sdk-Address) | repeated | List of IPV4 addresses |
| name | [string](#string) |  | Name of network interface |






<a name="f5-nginx-agent-sdk-ReleaseInfo"></a>

### ReleaseInfo
Represents release information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| codename | [string](#string) |  | OS type (e.g. freebsd, linux, etc) |
| id | [string](#string) |  | OS name (e.g. ubuntu, linuxmint, etc) |
| name | [string](#string) |  | OS family (e.g. debian, rhel) |
| version_id | [string](#string) |  | Version of the OS kernel |
| version | [string](#string) |  | Version of the OS |





 

 

 

 



<a name="metrics-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## metrics.proto



<a name="f5-nginx-agent-sdk-Dimension"></a>

### Dimension
Represents a dimension which is a dimensional attribute used when classifying and categorizing data


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Dimension name |
| value | [string](#string) |  | Dimension value |






<a name="f5-nginx-agent-sdk-MetricsReport"></a>

### MetricsReport
Represents a metric report


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | Provides meta information about the metrics |
| type | [MetricsReport.Type](#f5-nginx-agent-sdk-MetricsReport-Type) |  | Type of metrics |
| data | [StatsEntity](#f5-nginx-agent-sdk-StatsEntity) | repeated | List of stats entities |






<a name="f5-nginx-agent-sdk-SimpleMetric"></a>

### SimpleMetric
Represents a simple metric


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Metric name |
| value | [double](#double) |  | Metric value |






<a name="f5-nginx-agent-sdk-StatsEntity"></a>

### StatsEntity
Represents a stats entity which is a timestamped entry for dimensions and metrics


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Timestamp defines the time of stats entity creation |
| dimensions | [Dimension](#f5-nginx-agent-sdk-Dimension) | repeated | List of dimensions |
| simplemetrics | [SimpleMetric](#f5-nginx-agent-sdk-SimpleMetric) | repeated | List of metrics |





 


<a name="f5-nginx-agent-sdk-MetricsReport-Type"></a>

### MetricsReport.Type
Metric type enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| SYSTEM | 0 | System metric type |
| INSTANCE | 1 | NGINX instance metric type |
| AGENT | 2 | Agent metric type |


 

 

 



<a name="metrics-svc-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## metrics.svc.proto


 

 

 


<a name="f5-nginx-agent-sdk-MetricsService"></a>

### MetricsService
Represents a metrics service which is responsible for ingesting high volume metrics and events

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Stream | [MetricsReport](#f5-nginx-agent-sdk-MetricsReport) stream | [.google.protobuf.Empty](#google-protobuf-Empty) | A client-to-server streaming RPC to deliver high volume metrics reports. |
| StreamEvents | [events.EventReport](#f5-nginx-agent-sdk-events-EventReport) stream | [.google.protobuf.Empty](#google-protobuf-Empty) | A client-to-server streaming RPC to deliver high volume event reports. |

 



<a name="nap-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## nap.proto



<a name="f5-nginx-agent-sdk-AppProtectWAFDetails"></a>

### AppProtectWAFDetails
Represents App Protect WAF details


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| waf_version | [string](#string) |  | WAF version |
| attack_signatures_version | [string](#string) |  | Attack signatures version (This is being deprecated and will be removed in a future release) |
| threat_campaigns_version | [string](#string) |  | Threat signatures version (This is being deprecated and will be removed in a future release) |
| health | [AppProtectWAFHealth](#f5-nginx-agent-sdk-AppProtectWAFHealth) |  | App Protect Health details (This is being deprecated and will be removed in a future release) |
| waf_location | [string](#string) |  | Location of WAF metadata file |
| precompiled_publication | [bool](#bool) |  | Determines whether the publication of NGINX App Protect pre-compiled content from an external source is supported |






<a name="f5-nginx-agent-sdk-AppProtectWAFHealth"></a>

### AppProtectWAFHealth
Represents the health of App Protect WAF


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| system_id | [string](#string) |  | System ID |
| app_protect_waf_status | [AppProtectWAFHealth.AppProtectWAFStatus](#f5-nginx-agent-sdk-AppProtectWAFHealth-AppProtectWAFStatus) |  | App Protect WAF status |
| degraded_reason | [string](#string) |  | Provides an error message of why App Protect WAF is degraded |





 


<a name="f5-nginx-agent-sdk-AppProtectWAFHealth-AppProtectWAFStatus"></a>

### AppProtectWAFHealth.AppProtectWAFStatus
Status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | Unknown status |
| ACTIVE | 1 | Active status |
| DEGRADED | 2 | Degraded status |


 

 

 



<a name="nginx-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## nginx.proto



<a name="f5-nginx-agent-sdk-AccessLog"></a>

### AccessLog
Represents an access log file


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of file |
| format | [string](#string) |  | Format of the file |
| permissions | [string](#string) |  | File Permissions |
| readable | [bool](#bool) |  | Determines if the file is readable or not |






<a name="f5-nginx-agent-sdk-AccessLogs"></a>

### AccessLogs
Represents access log files


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_log | [AccessLog](#f5-nginx-agent-sdk-AccessLog) | repeated | List of access log files |






<a name="f5-nginx-agent-sdk-ErrorLog"></a>

### ErrorLog
Represents an error log file


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of file |
| log_level | [string](#string) |  | Log level |
| permissions | [string](#string) |  | File Permissions |
| readable | [bool](#bool) |  | Determines if the file is readable or not |






<a name="f5-nginx-agent-sdk-ErrorLogs"></a>

### ErrorLogs
Represents error log files


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error_log | [ErrorLog](#f5-nginx-agent-sdk-ErrorLog) | repeated | List of error log files |






<a name="f5-nginx-agent-sdk-NginxConfig"></a>

### NginxConfig
Represents a NGINX config


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [NginxConfigAction](#f5-nginx-agent-sdk-NginxConfigAction) |  | NGINX config action |
| config_data | [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor) |  | Metadata information about the configuration |
| zconfig | [ZippedFile](#f5-nginx-agent-sdk-ZippedFile) |  | Zipped file of all NGINX config files |
| zaux | [ZippedFile](#f5-nginx-agent-sdk-ZippedFile) |  | Zipped file of all auxiliary files |
| access_logs | [AccessLogs](#f5-nginx-agent-sdk-AccessLogs) |  | Information about all access log files |
| error_logs | [ErrorLogs](#f5-nginx-agent-sdk-ErrorLogs) |  | Information about all error log files |
| ssl | [SslCertificates](#f5-nginx-agent-sdk-SslCertificates) |  | Information about all SSL certificates files |
| directory_map | [DirectoryMap](#f5-nginx-agent-sdk-DirectoryMap) |  | Directory map of all config and aux files |






<a name="f5-nginx-agent-sdk-NginxDetails"></a>

### NginxDetails
Represents NGINX details about a single NGINX instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_id | [string](#string) |  | NGINX ID. Example: b636d4376dea15405589692d3c5d3869ff3a9b26b0e7bb4bb1aa7e658ace1437 |
| version | [string](#string) |  | NGINX version. Example: 1.23.2 |
| conf_path | [string](#string) |  | Path to NGINX configuration. Example: /usr/local/nginx/conf/nginx.conf |
| process_id | [string](#string) |  | Process ID of NGINX instance. Example: 8 |
| process_path | [string](#string) |  | The path to the NGINX executable. Example: /usr/local/nginx/sbin/nginx |
| start_time | [int64](#int64) |  | The start time of the NGINX instance. Example: 1670429190000 |
| built_from_source | [bool](#bool) |  | Determines if the NGINX instance was built from the source code in github or not. Example: false |
| loadable_modules | [string](#string) | repeated | List of NGINX loadable modules. Example: [] |
| runtime_modules | [string](#string) | repeated | List of NGINX runtime modules. Example: [ &#34;http_stub_status_module&#34; ] |
| plus | [NginxPlusMetaData](#f5-nginx-agent-sdk-NginxPlusMetaData) |  | NGINX Plus metadata. |
| ssl | [NginxSslMetaData](#f5-nginx-agent-sdk-NginxSslMetaData) |  | NGINX SSL metadata. |
| status_url | [string](#string) |  | Status URL. Example: http://localhost:8080/api |
| configure_args | [string](#string) | repeated | Command line arguments that were used when the NGINX instance was started. Example: [ &#34;&#34;, &#34;with-http_stub_status_module&#34; ] |






<a name="f5-nginx-agent-sdk-NginxHealth"></a>

### NginxHealth
Represents the health of a NGINX instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_id | [string](#string) |  | NGINX ID |
| nginx_status | [NginxHealth.NginxStatus](#f5-nginx-agent-sdk-NginxHealth-NginxStatus) |  | NGINX status |
| degraded_reason | [string](#string) |  | Provides an error message of why a NGINX instance is degraded |






<a name="f5-nginx-agent-sdk-NginxPlusMetaData"></a>

### NginxPlusMetaData
Represents NGINX Plus metadata


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  | Determines if its a plus instance or not. Example: true |
| release | [string](#string) |  | NGINX Plus version. Example: R27 |






<a name="f5-nginx-agent-sdk-NginxSslMetaData"></a>

### NginxSslMetaData
Represents NGINX SSL metadata


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ssl_type | [NginxSslMetaData.NginxSslType](#f5-nginx-agent-sdk-NginxSslMetaData-NginxSslType) |  | SSL Type. Example: 0 |
| details | [string](#string) | repeated | List of SSL information (e.g. version, type, etc). Example: null |





 


<a name="f5-nginx-agent-sdk-NginxConfigAction"></a>

### NginxConfigAction
NGINX config action enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | Unknown action |
| APPLY | 1 | Apply config action |
| TEST | 2 | Test config action (This will be implemented in a future release) |
| ROLLBACK | 3 | Rollback config action (This will be implemented in a future release) |
| RETURN | 4 | Return config action (This will be implemented in a future release) |
| FORCE | 5 | Force config apply action |



<a name="f5-nginx-agent-sdk-NginxHealth-NginxStatus"></a>

### NginxHealth.NginxStatus
NGINX status enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | Unknown status |
| ACTIVE | 1 | Active status |
| DEGRADED | 2 | Degraded status |



<a name="f5-nginx-agent-sdk-NginxSslMetaData-NginxSslType"></a>

### NginxSslMetaData.NginxSslType
SSL type enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| BUILT | 0 | SSL complied with NGINX |
| RUN | 1 | SSL not complied with NGINX |


 

 

 



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

