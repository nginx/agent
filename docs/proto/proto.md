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
  
- [config.proto](#config-proto)
    - [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor)
    - [ConfigReport](#f5-nginx-agent-sdk-ConfigReport)
  
- [dp_software_details.proto](#dp_software_details-proto)
    - [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails)
  
- [dpenv.proto](#dpenv-proto)
    - [EnvProperty](#f5-nginx-agent-sdk-EnvProperty)
    - [EnvPropertySet](#f5-nginx-agent-sdk-EnvPropertySet)
    - [EnvReport](#f5-nginx-agent-sdk-EnvReport)
  
    - [EnvReport.Type](#f5-nginx-agent-sdk-EnvReport-Type)
  
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
    - [PathInfo](#f5-nginx-agent-sdk-PathInfo)
    - [PlusInfo](#f5-nginx-agent-sdk-PlusInfo)
    - [SSLInfo](#f5-nginx-agent-sdk-SSLInfo)
  
    - [NginxConfigAction](#f5-nginx-agent-sdk-NginxConfigAction)
    - [NginxHealth.NginxStatus](#f5-nginx-agent-sdk-NginxHealth-NginxStatus)
    - [NginxSslMetaData.NginxSslType](#f5-nginx-agent-sdk-NginxSslMetaData-NginxSslType)
  
- [Scalar Value Types](#scalar-value-types)



<a name="agent-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## agent.proto



<a name="f5-nginx-agent-sdk-AgentConfig"></a>

### AgentConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| details | [AgentDetails](#f5-nginx-agent-sdk-AgentDetails) |  |  |
| loggers | [AgentLogging](#f5-nginx-agent-sdk-AgentLogging) |  |  |
| configs | [ConfigReport](#f5-nginx-agent-sdk-ConfigReport) |  |  |






<a name="f5-nginx-agent-sdk-AgentConfigRequest"></a>

### AgentConfigRequest







<a name="f5-nginx-agent-sdk-AgentConnectRequest"></a>

### AgentConnectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [AgentMeta](#f5-nginx-agent-sdk-AgentMeta) |  |  |
| details | [NginxDetails](#f5-nginx-agent-sdk-NginxDetails) | repeated | moving to dataplane_software_details |
| host | [HostInfo](#f5-nginx-agent-sdk-HostInfo) |  |  |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) | repeated |  |






<a name="f5-nginx-agent-sdk-AgentConnectResponse"></a>

### AgentConnectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent_config | [AgentConfig](#f5-nginx-agent-sdk-AgentConfig) |  |  |
| status | [AgentConnectStatus](#f5-nginx-agent-sdk-AgentConnectStatus) |  |  |






<a name="f5-nginx-agent-sdk-AgentConnectStatus"></a>

### AgentConnectStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statusCode | [AgentConnectStatus.StatusCode](#f5-nginx-agent-sdk-AgentConnectStatus-StatusCode) |  |  |
| message | [string](#string) |  |  |
| error | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-AgentDetails"></a>

### AgentDetails



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| features | [string](#string) | repeated |  |
| extensions | [string](#string) | repeated |  |
| tags | [string](#string) | repeated |  |
| alias | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-AgentLogging"></a>

### AgentLogging



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [AgentLogging.Level](#f5-nginx-agent-sdk-AgentLogging-Level) |  |  |
| dir | [string](#string) |  |  |
| file | [string](#string) |  |  |
| max_size | [uint32](#uint32) |  | max size in MB |
| max_backups | [uint32](#uint32) |  |  |
| max_age | [uint32](#uint32) |  | age in days |
| compress | [bool](#bool) |  |  |






<a name="f5-nginx-agent-sdk-AgentMeta"></a>

### AgentMeta



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| tag | [string](#string) | repeated |  |
| instance_group | [string](#string) |  |  |
| updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| system_uid | [string](#string) |  |  |
| agent_details | [AgentDetails](#f5-nginx-agent-sdk-AgentDetails) |  |  |





 


<a name="f5-nginx-agent-sdk-AgentConnectStatus-StatusCode"></a>

### AgentConnectStatus.StatusCode


| Name | Number | Description |
| ---- | ------ | ----------- |
| CONNECT_UNKNOWN | 0 |  |
| CONNECT_OK | 1 |  |
| CONNECT_REJECTED_OTHER | 2 |  |
| CONNECT_REJECTED_DUP_ID | 3 |  |



<a name="f5-nginx-agent-sdk-AgentLogging-Level"></a>

### AgentLogging.Level


| Name | Number | Description |
| ---- | ------ | ----------- |
| INFO | 0 |  |
| DEBUG | 1 |  |
| WARN | 2 |  |
| ERROR | 3 |  |
| FATAL | 4 |  |


 

 

 



<a name="command-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## command.proto



<a name="f5-nginx-agent-sdk-AgentActivityStatus"></a>

### AgentActivityStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_config_status | [NginxConfigStatus](#f5-nginx-agent-sdk-NginxConfigStatus) |  |  |






<a name="f5-nginx-agent-sdk-ChunkedResourceChunk"></a>

### ChunkedResourceChunk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  |  |
| chunk_id | [int32](#int32) |  |  |
| data | [bytes](#bytes) |  |  |






<a name="f5-nginx-agent-sdk-ChunkedResourceHeader"></a>

### ChunkedResourceHeader



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  |  |
| chunks | [int32](#int32) |  | number of chunks expected in this transfer |
| checksum | [string](#string) |  |  |
| chunk_size | [int32](#int32) |  |  |






<a name="f5-nginx-agent-sdk-Command"></a>

### Command
Command is the envelope sent between the management plane and the data plane, requesting some action or reporting a response


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  | add metadata later with fields like timestamp etc |
| type | [Command.CommandType](#f5-nginx-agent-sdk-Command-CommandType) |  | used as a dispatch flag to quickly send the command to the correct base processor that will then further sort based on the actual data type |
| cmd_status | [CommandStatusResponse](#f5-nginx-agent-sdk-CommandStatusResponse) |  | common command status response - used by most command responses |
| nginx_config | [NginxConfig](#f5-nginx-agent-sdk-NginxConfig) |  | request action on nginx config when sent C → A - all action values (see NgxConfig) notify config details when sent A → C - only RETURN action |
| nginx_config_response | [NginxConfigResponse](#f5-nginx-agent-sdk-NginxConfigResponse) |  | request action on nginx config when sent C → A - all action values (see NgxConfig) notify config details when sent A → C - only RETURN action |
| agent_connect_request | [AgentConnectRequest](#f5-nginx-agent-sdk-AgentConnectRequest) |  | request connection to a management plane, A → C |
| agent_connect_response | [AgentConnectResponse](#f5-nginx-agent-sdk-AgentConnectResponse) |  | connection response to the data plane, C → A |
| agent_config_request | [AgentConfigRequest](#f5-nginx-agent-sdk-AgentConfigRequest) |  | request Configuration parameters for agent, A → C |
| agent_config | [AgentConfig](#f5-nginx-agent-sdk-AgentConfig) |  | configuration parameters for Agent C → A. This message can be sent asynchronously as well |
| dataplane_status | [DataplaneStatus](#f5-nginx-agent-sdk-DataplaneStatus) |  | DataplaneStatus reports Dataplane metrics the Agent is aware of |
| event_report | [events.EventReport](#f5-nginx-agent-sdk-events-EventReport) |  | EventReport reports events the Agent is aware of, e.g. Start/Stop of Agent, Config Apply NGINX |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) |  | DataplaneSoftwareDetails contains details for additional software running on the dataplane that pertains to NGINX Agent |
| dataplane_update | [DataplaneUpdate](#f5-nginx-agent-sdk-DataplaneUpdate) |  | DataplaneUpdate contains details for dataplane resources that have changed |






<a name="f5-nginx-agent-sdk-CommandStatusResponse"></a>

### CommandStatusResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CommandStatusResponse.CommandStatus](#f5-nginx-agent-sdk-CommandStatusResponse-CommandStatus) |  |  |
| error_code | [CommandStatusResponse.CommandErrorCode](#f5-nginx-agent-sdk-CommandStatusResponse-CommandErrorCode) |  |  |
| message | [string](#string) |  |  |
| error | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-DataChunk"></a>

### DataChunk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| header | [ChunkedResourceHeader](#f5-nginx-agent-sdk-ChunkedResourceHeader) |  |  |
| data | [ChunkedResourceChunk](#f5-nginx-agent-sdk-ChunkedResourceChunk) |  |  |






<a name="f5-nginx-agent-sdk-DataplaneSoftwareHealth"></a>

### DataplaneSoftwareHealth



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_health | [NginxHealth](#f5-nginx-agent-sdk-NginxHealth) |  |  |
| app_protect_waf_health | [AppProtectWAFHealth](#f5-nginx-agent-sdk-AppProtectWAFHealth) |  |  |






<a name="f5-nginx-agent-sdk-DataplaneStatus"></a>

### DataplaneStatus
DataplaneStatus reports Dataplane metrics the Agent is aware of


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| system_id | [string](#string) |  |  |
| details | [NginxDetails](#f5-nginx-agent-sdk-NginxDetails) | repeated | moving to dataplane_software_details |
| host | [HostInfo](#f5-nginx-agent-sdk-HostInfo) |  |  |
| healths | [NginxHealth](#f5-nginx-agent-sdk-NginxHealth) | repeated | moving to DataplaneSoftwareHealth |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) | repeated |  |
| dataplane_software_healths | [DataplaneSoftwareHealth](#f5-nginx-agent-sdk-DataplaneSoftwareHealth) | repeated |  |
| agent_activity_status | [AgentActivityStatus](#f5-nginx-agent-sdk-AgentActivityStatus) | repeated |  |






<a name="f5-nginx-agent-sdk-DataplaneUpdate"></a>

### DataplaneUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [HostInfo](#f5-nginx-agent-sdk-HostInfo) |  |  |
| dataplane_software_details | [DataplaneSoftwareDetails](#f5-nginx-agent-sdk-DataplaneSoftwareDetails) | repeated |  |






<a name="f5-nginx-agent-sdk-DownloadRequest"></a>

### DownloadRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  |  |






<a name="f5-nginx-agent-sdk-NginxConfigResponse"></a>

### NginxConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CommandStatusResponse](#f5-nginx-agent-sdk-CommandStatusResponse) |  |  |
| action | [NginxConfigAction](#f5-nginx-agent-sdk-NginxConfigAction) |  |  |
| config_data | [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor) |  |  |






<a name="f5-nginx-agent-sdk-NginxConfigStatus"></a>

### NginxConfigStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| correlation_id | [string](#string) |  |  |
| status | [NginxConfigStatus.Status](#f5-nginx-agent-sdk-NginxConfigStatus-Status) |  |  |
| message | [string](#string) |  |  |
| nginx_id | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-UploadStatus"></a>

### UploadStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  |  |
| status | [UploadStatus.TransferStatus](#f5-nginx-agent-sdk-UploadStatus-TransferStatus) |  |  |
| reason | [string](#string) |  |  |





 


<a name="f5-nginx-agent-sdk-Command-CommandType"></a>

### Command.CommandType


| Name | Number | Description |
| ---- | ------ | ----------- |
| NORMAL | 0 |  |
| DOWNLOAD | 1 |  |
| UPLOAD | 2 |  |



<a name="f5-nginx-agent-sdk-CommandStatusResponse-CommandErrorCode"></a>

### CommandStatusResponse.CommandErrorCode


| Name | Number | Description |
| ---- | ------ | ----------- |
| ERR_OK | 0 | No Error |
| ERR_UNKNOWN | 1 | unknown error |



<a name="f5-nginx-agent-sdk-CommandStatusResponse-CommandStatus"></a>

### CommandStatusResponse.CommandStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| CMD_UNKNOWN | 0 |  |
| CMD_OK | 1 |  |
| CMD_ERROR | 2 |  |



<a name="f5-nginx-agent-sdk-NginxConfigStatus-Status"></a>

### NginxConfigStatus.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| PENDING | 0 |  |
| OK | 1 |  |
| ERROR | 2 |  |



<a name="f5-nginx-agent-sdk-UploadStatus-TransferStatus"></a>

### UploadStatus.TransferStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| OK | 1 |  |
| FAILED | 2 |  |


 

 

 



<a name="command_svc-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## command_svc.proto


 

 

 


<a name="f5-nginx-agent-sdk-Commander"></a>

### Commander
Interface exported by the server.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CommandChannel | [Command](#f5-nginx-agent-sdk-Command) stream | [Command](#f5-nginx-agent-sdk-Command) stream | A Bidirectional streaming RPC established by the data plane agent and is kept open |
| Download | [DownloadRequest](#f5-nginx-agent-sdk-DownloadRequest) | [DataChunk](#f5-nginx-agent-sdk-DataChunk) stream | A streaming RPC established by the data plane agent and is used to download resources associated with commands The download stream will be kept open for the duration of the data transfer and will be closed when its done/ The transfer is a stream of chunks as follows: - header - data chunk count to follow - resource identifier/metadata - data 1 ... - data

each data chunk is of a size smaller than the maximum gRPC payload |
| Upload | [DataChunk](#f5-nginx-agent-sdk-DataChunk) stream | [UploadStatus](#f5-nginx-agent-sdk-UploadStatus) | A streaming RPC established by the data plane agent and is used to upload resources associated with commands |

 



<a name="common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## common.proto



<a name="f5-nginx-agent-sdk-CertificateDates"></a>

### CertificateDates



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| not_before | [int64](#int64) |  |  |
| not_after | [int64](#int64) |  |  |






<a name="f5-nginx-agent-sdk-CertificateName"></a>

### CertificateName



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| common_name | [string](#string) |  |  |
| country | [string](#string) | repeated |  |
| state | [string](#string) | repeated |  |
| locality | [string](#string) | repeated |  |
| organization | [string](#string) | repeated |  |
| organizational_unit | [string](#string) | repeated |  |






<a name="f5-nginx-agent-sdk-Directory"></a>

### Directory



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| mtime | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| permissions | [string](#string) |  |  |
| size | [int64](#int64) |  |  |
| files | [File](#f5-nginx-agent-sdk-File) | repeated |  |






<a name="f5-nginx-agent-sdk-DirectoryMap"></a>

### DirectoryMap



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| directories | [Directory](#f5-nginx-agent-sdk-Directory) | repeated |  |






<a name="f5-nginx-agent-sdk-File"></a>

### File



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| lines | [int32](#int32) |  |  |
| mtime | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| permissions | [string](#string) |  |  |
| size | [int64](#int64) |  |  |
| contents | [bytes](#bytes) |  |  |






<a name="f5-nginx-agent-sdk-Metadata"></a>

### Metadata
Metadata timestamped info associating a client with a specific command message


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| client_id | [string](#string) |  |  |
| message_id | [string](#string) |  |  |
| cloud_account_id | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-SslCertificate"></a>

### SslCertificate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_name | [string](#string) |  |  |
| size | [int64](#int64) |  |  |
| mtime | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| validity | [CertificateDates](#f5-nginx-agent-sdk-CertificateDates) |  |  |
| issuer | [CertificateName](#f5-nginx-agent-sdk-CertificateName) |  |  |
| subject | [CertificateName](#f5-nginx-agent-sdk-CertificateName) |  |  |
| subj_alt_names | [string](#string) | repeated |  |
| ocsp_url | [string](#string) | repeated |  |
| public_key_algorithm | [string](#string) |  |  |
| signature_algorithm | [string](#string) |  |  |
| serial_number | [string](#string) |  |  |
| subject_key_identifier | [string](#string) |  |  |
| fingerprint | [string](#string) |  |  |
| fingerprint_algorithm | [string](#string) |  |  |
| version | [int64](#int64) |  |  |
| authority_key_identifier | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-SslCertificates"></a>

### SslCertificates



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ssl_certs | [SslCertificate](#f5-nginx-agent-sdk-SslCertificate) | repeated |  |






<a name="f5-nginx-agent-sdk-ZippedFile"></a>

### ZippedFile



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [bytes](#bytes) |  |  |
| checksum | [string](#string) |  |  |
| root_directory | [string](#string) |  |  |





 

 

 

 



<a name="config-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config.proto



<a name="f5-nginx-agent-sdk-ConfigDescriptor"></a>

### ConfigDescriptor



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| system_id | [string](#string) |  |  |
| nginx_id | [string](#string) |  |  |
| checksum | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-ConfigReport"></a>

### ConfigReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  |  |
| configs | [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor) | repeated |  |





 

 

 

 



<a name="dp_software_details-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## dp_software_details.proto



<a name="f5-nginx-agent-sdk-DataplaneSoftwareDetails"></a>

### DataplaneSoftwareDetails
DataplaneSoftwareDetails contains details for additional software running on the dataplane that pertains 
to NGINX Agent


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_protect_waf_details | [AppProtectWAFDetails](#f5-nginx-agent-sdk-AppProtectWAFDetails) |  |  |
| nginx_details | [NginxDetails](#f5-nginx-agent-sdk-NginxDetails) |  |  |





 

 

 

 



<a name="dpenv-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## dpenv.proto



<a name="f5-nginx-agent-sdk-EnvProperty"></a>

### EnvProperty
EnvPropety - a container for a Dataplane Environment property.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ival | [int64](#int64) |  | for example cpu count. |
| dval | [double](#double) |  | for example cpu utilization |
| sval | [string](#string) |  | for example os name, release name |






<a name="f5-nginx-agent-sdk-EnvPropertySet"></a>

### EnvPropertySet



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dimensions | [Dimension](#f5-nginx-agent-sdk-Dimension) | repeated |  |
| properties | [EnvProperty](#f5-nginx-agent-sdk-EnvProperty) | repeated |  |






<a name="f5-nginx-agent-sdk-EnvReport"></a>

### EnvReport
MetasReport a report containing status entities for a specific metric type


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  |  |
| type | [EnvReport.Type](#f5-nginx-agent-sdk-EnvReport-Type) |  |  |
| property_sets | [EnvPropertySet](#f5-nginx-agent-sdk-EnvPropertySet) | repeated |  |





 


<a name="f5-nginx-agent-sdk-EnvReport-Type"></a>

### EnvReport.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| SYSTEM | 0 |  |
| INSTANCE | 1 |  |
| AGENT | 2 |  |


 

 

 



<a name="host-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## host.proto



<a name="f5-nginx-agent-sdk-Address"></a>

### Address



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefixlen | [int64](#int64) |  |  |
| netmask | [string](#string) |  |  |
| address | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-CpuInfo"></a>

### CpuInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| model | [string](#string) |  |  |
| cores | [int32](#int32) |  |  |
| architecture | [string](#string) |  |  |
| mhz | [double](#double) |  |  |
| hypervisor | [string](#string) |  |  |
| cpus | [int32](#int32) |  |  |
| virtualization | [string](#string) |  |  |
| cache | [CpuInfo.CacheEntry](#f5-nginx-agent-sdk-CpuInfo-CacheEntry) | repeated |  |






<a name="f5-nginx-agent-sdk-CpuInfo-CacheEntry"></a>

### CpuInfo.CacheEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-DiskPartition"></a>

### DiskPartition



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mount_point | [string](#string) |  |  |
| device | [string](#string) |  |  |
| fs_type | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-HostInfo"></a>

### HostInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent | [string](#string) |  |  |
| boot | [uint64](#uint64) |  |  |
| hostname | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| os_type | [string](#string) |  | note kebab case used for compatibility with legacy |
| uuid | [string](#string) |  |  |
| uname | [string](#string) |  |  |
| partitons | [DiskPartition](#f5-nginx-agent-sdk-DiskPartition) | repeated |  |
| network | [Network](#f5-nginx-agent-sdk-Network) |  |  |
| processor | [CpuInfo](#f5-nginx-agent-sdk-CpuInfo) | repeated |  |
| release | [ReleaseInfo](#f5-nginx-agent-sdk-ReleaseInfo) |  |  |
| tags | [string](#string) | repeated |  |
| agent_accessible_dirs | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-Network"></a>

### Network



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| interfaces | [NetworkInterface](#f5-nginx-agent-sdk-NetworkInterface) | repeated |  |
| default | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-NetworkInterface"></a>

### NetworkInterface



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mac | [string](#string) |  |  |
| ipv6 | [Address](#f5-nginx-agent-sdk-Address) | repeated |  |
| ipv4 | [Address](#f5-nginx-agent-sdk-Address) | repeated |  |
| name | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-ReleaseInfo"></a>

### ReleaseInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| codename | [string](#string) |  |  |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| version_id | [string](#string) |  |  |
| version | [string](#string) |  |  |





 

 

 

 



<a name="metrics-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## metrics.proto



<a name="f5-nginx-agent-sdk-Dimension"></a>

### Dimension
Dimension defines a dimensional attribute used when classifying and categorizing data


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-MetricsReport"></a>

### MetricsReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [Metadata](#f5-nginx-agent-sdk-Metadata) |  |  |
| type | [MetricsReport.Type](#f5-nginx-agent-sdk-MetricsReport-Type) |  |  |
| data | [StatsEntity](#f5-nginx-agent-sdk-StatsEntity) | repeated |  |






<a name="f5-nginx-agent-sdk-SimpleMetric"></a>

### SimpleMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| value | [double](#double) |  |  |






<a name="f5-nginx-agent-sdk-StatsEntity"></a>

### StatsEntity
StatsEntity a timestamped entry for Dimensions and Metrics


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| dimensions | [Dimension](#f5-nginx-agent-sdk-Dimension) | repeated |  |
| simplemetrics | [SimpleMetric](#f5-nginx-agent-sdk-SimpleMetric) | repeated |  |





 


<a name="f5-nginx-agent-sdk-MetricsReport-Type"></a>

### MetricsReport.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| SYSTEM | 0 |  |
| INSTANCE | 1 |  |
| AGENT | 2 |  |


 

 

 



<a name="metrics-svc-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## metrics.svc.proto


 

 

 


<a name="f5-nginx-agent-sdk-MetricsService"></a>

### MetricsService
MetricsService is responsible for ingesting high volume metrics and events

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Stream | [MetricsReport](#f5-nginx-agent-sdk-MetricsReport) stream | [.google.protobuf.Empty](#google-protobuf-Empty) | A client-to-server streaming RPC to deliver high volume metrics reports. |
| StreamEvents | [events.EventReport](#f5-nginx-agent-sdk-events-EventReport) stream | [.google.protobuf.Empty](#google-protobuf-Empty) | A client-to-server streaming RPC to deliver high volume event reports. |

 



<a name="nap-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## nap.proto



<a name="f5-nginx-agent-sdk-AppProtectWAFDetails"></a>

### AppProtectWAFDetails
AppProtectWAFDetails reports the details of Nginx App Protect


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| waf_version | [string](#string) |  |  |
| attack_signatures_version | [string](#string) |  | deprecating |
| threat_campaigns_version | [string](#string) |  | deprecating |
| health | [AppProtectWAFHealth](#f5-nginx-agent-sdk-AppProtectWAFHealth) |  | deprecating |






<a name="f5-nginx-agent-sdk-AppProtectWAFHealth"></a>

### AppProtectWAFHealth
AppProtectWAFHealth reports the health details of Nginx App Protect


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| system_id | [string](#string) |  |  |
| app_protect_waf_status | [AppProtectWAFHealth.AppProtectWAFStatus](#f5-nginx-agent-sdk-AppProtectWAFHealth-AppProtectWAFStatus) |  |  |
| degraded_reason | [string](#string) |  |  |





 


<a name="f5-nginx-agent-sdk-AppProtectWAFHealth-AppProtectWAFStatus"></a>

### AppProtectWAFHealth.AppProtectWAFStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| ACTIVE | 1 |  |
| DEGRADED | 2 |  |


 

 

 



<a name="nginx-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## nginx.proto



<a name="f5-nginx-agent-sdk-AccessLog"></a>

### AccessLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| format | [string](#string) |  |  |
| permissions | [string](#string) |  |  |
| readable | [bool](#bool) |  |  |






<a name="f5-nginx-agent-sdk-AccessLogs"></a>

### AccessLogs



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_log | [AccessLog](#f5-nginx-agent-sdk-AccessLog) | repeated |  |






<a name="f5-nginx-agent-sdk-ErrorLog"></a>

### ErrorLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| log_level | [string](#string) |  |  |
| permissions | [string](#string) |  |  |
| readable | [bool](#bool) |  |  |






<a name="f5-nginx-agent-sdk-ErrorLogs"></a>

### ErrorLogs



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error_log | [ErrorLog](#f5-nginx-agent-sdk-ErrorLog) | repeated |  |






<a name="f5-nginx-agent-sdk-NginxConfig"></a>

### NginxConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [NginxConfigAction](#f5-nginx-agent-sdk-NginxConfigAction) |  |  |
| config_data | [ConfigDescriptor](#f5-nginx-agent-sdk-ConfigDescriptor) |  |  |
| zconfig | [ZippedFile](#f5-nginx-agent-sdk-ZippedFile) |  |  |
| zaux | [ZippedFile](#f5-nginx-agent-sdk-ZippedFile) |  |  |
| access_logs | [AccessLogs](#f5-nginx-agent-sdk-AccessLogs) |  |  |
| error_logs | [ErrorLogs](#f5-nginx-agent-sdk-ErrorLogs) |  |  |
| ssl | [SslCertificates](#f5-nginx-agent-sdk-SslCertificates) |  |  |
| directory_map | [DirectoryMap](#f5-nginx-agent-sdk-DirectoryMap) |  |  |






<a name="f5-nginx-agent-sdk-NginxDetails"></a>

### NginxDetails
Each NGINXDetails is associated with with a single NGINX instance.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_id | [string](#string) |  |  |
| version | [string](#string) |  |  |
| conf_path | [string](#string) |  |  |
| process_id | [string](#string) |  |  |
| process_path | [string](#string) |  |  |
| start_time | [int64](#int64) |  |  |
| built_from_source | [bool](#bool) |  |  |
| loadable_modules | [string](#string) | repeated |  |
| runtime_modules | [string](#string) | repeated |  |
| plus | [NginxPlusMetaData](#f5-nginx-agent-sdk-NginxPlusMetaData) |  |  |
| ssl | [NginxSslMetaData](#f5-nginx-agent-sdk-NginxSslMetaData) |  |  |
| status_url | [string](#string) |  |  |
| configure_args | [string](#string) | repeated |  |






<a name="f5-nginx-agent-sdk-NginxHealth"></a>

### NginxHealth



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nginx_id | [string](#string) |  |  |
| nginx_status | [NginxHealth.NginxStatus](#f5-nginx-agent-sdk-NginxHealth-NginxStatus) |  |  |
| degraded_reason | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-NginxPlusMetaData"></a>

### NginxPlusMetaData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| release | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-NginxSslMetaData"></a>

### NginxSslMetaData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ssl_type | [NginxSslMetaData.NginxSslType](#f5-nginx-agent-sdk-NginxSslMetaData-NginxSslType) |  |  |
| details | [string](#string) | repeated |  |






<a name="f5-nginx-agent-sdk-PathInfo"></a>

### PathInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bin | [string](#string) |  |  |
| conf | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-PlusInfo"></a>

### PlusInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| release | [string](#string) |  |  |






<a name="f5-nginx-agent-sdk-SSLInfo"></a>

### SSLInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| built | [string](#string) | repeated |  |
| run | [string](#string) | repeated |  |





 


<a name="f5-nginx-agent-sdk-NginxConfigAction"></a>

### NginxConfigAction


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| APPLY | 1 |  |
| TEST | 2 |  |
| ROLLBACK | 3 |  |
| RETURN | 4 |  |
| FORCE | 5 |  |



<a name="f5-nginx-agent-sdk-NginxHealth-NginxStatus"></a>

### NginxHealth.NginxStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| ACTIVE | 1 |  |
| DEGRADED | 2 |  |



<a name="f5-nginx-agent-sdk-NginxSslMetaData-NginxSslType"></a>

### NginxSslMetaData.NginxSslType


| Name | Number | Description |
| ---- | ------ | ----------- |
| BUILT | 0 |  |
| RUN | 1 |  |


 

 

 



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

