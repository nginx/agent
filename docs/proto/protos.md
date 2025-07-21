# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [mpi/v1/common.proto](#mpi_v1_common-proto)
    - [AuthSettings](#mpi-v1-AuthSettings)
    - [CommandResponse](#mpi-v1-CommandResponse)
    - [MessageMeta](#mpi-v1-MessageMeta)
    - [ServerSettings](#mpi-v1-ServerSettings)
    - [TLSSettings](#mpi-v1-TLSSettings)
  
    - [CommandResponse.CommandStatus](#mpi-v1-CommandResponse-CommandStatus)
    - [ServerSettings.ServerType](#mpi-v1-ServerSettings-ServerType)
  
- [mpi/v1/files.proto](#mpi_v1_files-proto)
    - [AttributeTypeAndValue](#mpi-v1-AttributeTypeAndValue)
    - [CertificateDates](#mpi-v1-CertificateDates)
    - [CertificateMeta](#mpi-v1-CertificateMeta)
    - [ConfigVersion](#mpi-v1-ConfigVersion)
    - [File](#mpi-v1-File)
    - [FileContents](#mpi-v1-FileContents)
    - [FileDataChunk](#mpi-v1-FileDataChunk)
    - [FileDataChunkContent](#mpi-v1-FileDataChunkContent)
    - [FileDataChunkHeader](#mpi-v1-FileDataChunkHeader)
    - [FileMeta](#mpi-v1-FileMeta)
    - [FileOverview](#mpi-v1-FileOverview)
    - [GetFileRequest](#mpi-v1-GetFileRequest)
    - [GetFileResponse](#mpi-v1-GetFileResponse)
    - [GetOverviewRequest](#mpi-v1-GetOverviewRequest)
    - [GetOverviewResponse](#mpi-v1-GetOverviewResponse)
    - [SubjectAlternativeNames](#mpi-v1-SubjectAlternativeNames)
    - [UpdateFileRequest](#mpi-v1-UpdateFileRequest)
    - [UpdateFileResponse](#mpi-v1-UpdateFileResponse)
    - [UpdateOverviewRequest](#mpi-v1-UpdateOverviewRequest)
    - [UpdateOverviewResponse](#mpi-v1-UpdateOverviewResponse)
    - [X509Name](#mpi-v1-X509Name)
  
    - [SignatureAlgorithm](#mpi-v1-SignatureAlgorithm)
  
    - [FileService](#mpi-v1-FileService)
  
- [mpi/v1/command.proto](#mpi_v1_command-proto)
    - [APIActionRequest](#mpi-v1-APIActionRequest)
    - [APIDetails](#mpi-v1-APIDetails)
    - [AgentConfig](#mpi-v1-AgentConfig)
    - [AuxiliaryCommandServer](#mpi-v1-AuxiliaryCommandServer)
    - [CommandServer](#mpi-v1-CommandServer)
    - [CommandStatusRequest](#mpi-v1-CommandStatusRequest)
    - [ConfigApplyRequest](#mpi-v1-ConfigApplyRequest)
    - [ConfigUploadRequest](#mpi-v1-ConfigUploadRequest)
    - [ContainerInfo](#mpi-v1-ContainerInfo)
    - [CreateConnectionRequest](#mpi-v1-CreateConnectionRequest)
    - [CreateConnectionResponse](#mpi-v1-CreateConnectionResponse)
    - [DataPlaneResponse](#mpi-v1-DataPlaneResponse)
    - [FileServer](#mpi-v1-FileServer)
    - [GetHTTPUpstreamServers](#mpi-v1-GetHTTPUpstreamServers)
    - [GetStreamUpstreams](#mpi-v1-GetStreamUpstreams)
    - [GetUpstreams](#mpi-v1-GetUpstreams)
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
    - [NGINXAppProtectRuntimeInfo](#mpi-v1-NGINXAppProtectRuntimeInfo)
    - [NGINXPlusAction](#mpi-v1-NGINXPlusAction)
    - [NGINXPlusRuntimeInfo](#mpi-v1-NGINXPlusRuntimeInfo)
    - [NGINXRuntimeInfo](#mpi-v1-NGINXRuntimeInfo)
    - [ReleaseInfo](#mpi-v1-ReleaseInfo)
    - [Resource](#mpi-v1-Resource)
    - [StatusRequest](#mpi-v1-StatusRequest)
    - [UpdateDataPlaneHealthRequest](#mpi-v1-UpdateDataPlaneHealthRequest)
    - [UpdateDataPlaneHealthResponse](#mpi-v1-UpdateDataPlaneHealthResponse)
    - [UpdateDataPlaneStatusRequest](#mpi-v1-UpdateDataPlaneStatusRequest)
    - [UpdateDataPlaneStatusResponse](#mpi-v1-UpdateDataPlaneStatusResponse)
    - [UpdateHTTPUpstreamServers](#mpi-v1-UpdateHTTPUpstreamServers)
    - [UpdateStreamServers](#mpi-v1-UpdateStreamServers)
  
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


<a name="mpi-v1-AuthSettings"></a>

### AuthSettings
Defines the authentication configuration






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






<a name="mpi-v1-ServerSettings"></a>

### ServerSettings
The top-level configuration for the command server


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  | Command server host |
| port | [int32](#int32) |  | Command server port |
| type | [ServerSettings.ServerType](#mpi-v1-ServerSettings-ServerType) |  | Server type (enum for gRPC, HTTP, etc.) |






<a name="mpi-v1-TLSSettings"></a>

### TLSSettings



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cert | [string](#string) |  | TLS certificate for the command server (e.g., &#34;/path/to/cert.pem&#34;) |
| key | [string](#string) |  | TLS key for the command server (e.g., &#34;/path/to/key.pem&#34;) |
| ca | [string](#string) |  | CA certificate for the command server (e.g., &#34;/path/to/ca.pem&#34;) |
| skip_verify | [bool](#bool) |  | Controls whether a client verifies the server&#39;s certificate chain and host name. If skip_verify is true, accepts any certificate presented by the server and any host name in that certificate. |
| server_name | [string](#string) |  | Server name for TLS |





 


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



<a name="mpi-v1-ServerSettings-ServerType"></a>

### ServerSettings.ServerType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SERVER_SETTINGS_TYPE_UNDEFINED | 0 | Undefined server type |
| SERVER_SETTINGS_TYPE_GRPC | 1 | gRPC server type |
| SERVER_SETTINGS_TYPE_HTTP | 2 | HTTP server type |


 

 

 



<a name="mpi_v1_files-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mpi/v1/files.proto



<a name="mpi-v1-AttributeTypeAndValue"></a>

### AttributeTypeAndValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | The type (or identifier) of the attribute (e.g., OID). |
| value | [string](#string) |  | The value associated with the attribute. |






<a name="mpi-v1-CertificateDates"></a>

### CertificateDates
Represents the dates for which a certificate is valid


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| not_before | [int64](#int64) |  | The start date that for when the certificate is valid |
| not_after | [int64](#int64) |  | The end date that for when the certificate is valid |






<a name="mpi-v1-CertificateMeta"></a>

### CertificateMeta
Define the certificate message based on https://pkg.go.dev/crypto/x509#Certificate 
and https://github.com/googleapis/googleapis/blob/005df4681b89bd204a90b76168a6dc9d9e7bf4fe/google/cloud/iot/v1/resources.proto#L341


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| serial_number | [string](#string) |  | Serial number of the certificate, usually a unique identifier, the max length is the length of an interger |
| issuer | [X509Name](#mpi-v1-X509Name) |  | Issuer details (who issued the certificate) |
| subject | [X509Name](#mpi-v1-X509Name) |  | Subject details (to whom the certificate is issued) |
| sans | [SubjectAlternativeNames](#mpi-v1-SubjectAlternativeNames) |  | Subject Alternative Names (SAN) including DNS names and IP addresses |
| dates | [CertificateDates](#mpi-v1-CertificateDates) |  | Timestamps representing the start of certificate validity (Not Before, Not After) |
| signature_algorithm | [SignatureAlgorithm](#mpi-v1-SignatureAlgorithm) |  | The algorithm used to sign the certificate (e.g., SHA256-RSA) |
| public_key_algorithm | [string](#string) |  | The type of public key in the certificate. |






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
| unmanaged | [bool](#bool) |  | Unmanaged files will not be modified |






<a name="mpi-v1-FileContents"></a>

### FileContents
Represents the bytes contents of the file https://protobuf.dev/programming-guides/api/#dont-encode-data-in-a-string


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contents | [bytes](#bytes) |  | Byte representation of a file without encoding |






<a name="mpi-v1-FileDataChunk"></a>

### FileDataChunk
Represents a data chunk for streaming file transfer.
For any Stream file transfer, following assumptions should be asserted (by implementation):
- invalid to contain more or less than one FileDataChunkHeaders
- invalid to have FileDataChunkContents before FileDataChunkHeaders
- invalid to have more/fewer FileDataChunkContents than FileDataChunkHeader.chunks
- invalid to have two FileDataChunkContents with same chunk_id
- invalid to have FileDataChunkContent with zero-length data
- invalid to have FileDataChunk message without either header or content
- hash of the combined contents should match FileDataChunkHeader.file_meta.hash
- total size of the combined contents should match FileDataChunkHeader.file_meta.size
- chunk_size should be less than the gRPC max message size


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| meta | [MessageMeta](#mpi-v1-MessageMeta) |  | meta regarding the transfer request |
| header | [FileDataChunkHeader](#mpi-v1-FileDataChunkHeader) |  | Chunk header |
| content | [FileDataChunkContent](#mpi-v1-FileDataChunkContent) |  | Chunk data |






<a name="mpi-v1-FileDataChunkContent"></a>

### FileDataChunkContent
Represents a chunked resource chunk


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| chunk_id | [uint32](#uint32) |  | chunk id, i.e. x of y, zero-indexed |
| data | [bytes](#bytes) |  | chunk data, should be at most chunk_size |






<a name="mpi-v1-FileDataChunkHeader"></a>

### FileDataChunkHeader
Represents a chunked resource Header


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_meta | [FileMeta](#mpi-v1-FileMeta) |  | meta regarding the file, help identity the file name, size, hash, perm receiver should validate the hash against the combined contents |
| chunks | [uint32](#uint32) |  | total number of chunks expected in the transfer |
| chunk_size | [uint32](#uint32) |  | max size of individual chunks, can be undersized if EOF |






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
| certificate_meta | [CertificateMeta](#mpi-v1-CertificateMeta) |  |  |






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
Represents the response payload to a GetOverviewRequest, requesting a list of logically grouped files e.g. configuration payload


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| overview | [FileOverview](#mpi-v1-FileOverview) |  | The file overview of an instance |






<a name="mpi-v1-SubjectAlternativeNames"></a>

### SubjectAlternativeNames
Represents the Subject Alternative Names for a certificate


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dns_names | [string](#string) | repeated | List of DNS names in the Subject Alternative Name (SAN) extension |
| ip_addresses | [string](#string) | repeated | List of ip addresses in the SAN extension |






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
Represents a list of logically grouped files that have changed e.g. configuration payload


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message_meta | [MessageMeta](#mpi-v1-MessageMeta) |  | Meta-information associated with a message |
| overview | [FileOverview](#mpi-v1-FileOverview) |  | The file overview of an instance |






<a name="mpi-v1-UpdateOverviewResponse"></a>

### UpdateOverviewResponse
Represents a the response from an UpdateOverviewRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| overview | [FileOverview](#mpi-v1-FileOverview) |  | The file overview with the list of files that were uploaded |






<a name="mpi-v1-X509Name"></a>

### X509Name
Represents the dates for which a certificate is valid as seen at https://pkg.go.dev/crypto/x509/pkix#Name


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| country | [string](#string) | repeated | Country name (C): Two-letter country code as per ISO 3166. Must be exactly 2 characters. |
| organization | [string](#string) | repeated | Organization name (O): Name of the organization. |
| organizational_unit | [string](#string) | repeated | Organizational Unit name (OU): Name of a subdivision or unit within the organization. |
| locality | [string](#string) | repeated | Locality name (L): Name of the city or locality. Must be non-empty and a reasonable length (e.g., max 100 characters). |
| province | [string](#string) | repeated | State or Province name (ST): Name of the state or province. |
| street_address | [string](#string) | repeated | Street Address (STREET): Physical street address. |
| postal_code | [string](#string) | repeated | Postal Code (PC): Postal or ZIP code for the address. |
| serial_number | [string](#string) |  | Serial Number (SN): Unique identifier or serial number. |
| common_name | [string](#string) |  | Common Name (CN): Typically the person’s or entity&#39;s full name. |
| names | [AttributeTypeAndValue](#mpi-v1-AttributeTypeAndValue) | repeated | Parsed attributes including any non-standard attributes, as specified in RFC 2253. These attributes are parsed but not marshaled by this package. |
| extra_names | [AttributeTypeAndValue](#mpi-v1-AttributeTypeAndValue) | repeated | Additional attributes to be included in the marshaled distinguished names. These override any attributes with the same OID in `names`. |





 


<a name="mpi-v1-SignatureAlgorithm"></a>

### SignatureAlgorithm
Enum to represent the possible signature algorithms used for certificates

| Name | Number | Description |
| ---- | ------ | ----------- |
| SIGNATURE_ALGORITHM_UNKNOWN | 0 | Default, unknown or unsupported algorithm |
| MD2_WITH_RSA | 1 | MD2 with RSA (Unsupported) |
| MD5_WITH_RSA | 2 | MD5 with RSA (Only supported for signing, not verification) |
| SHA1_WITH_RSA | 3 | SHA-1 with RSA (Only supported for signing and for verification of CRLs, CSRs, and OCSP responses) |
| SHA256_WITH_RSA | 4 | SHA-256 with RSA |
| SHA384_WITH_RSA | 5 | SHA-384 with RSA |
| SHA512_WITH_RSA | 6 | SHA-512 with RSA |
| DSA_WITH_SHA1 | 7 | DSA with SHA-1 (Unsupported) |
| DSA_WITH_SHA256 | 8 | DSA with SHA-256 (Unsupported) |
| ECDSA_WITH_SHA1 | 9 | ECDSA with SHA-1 (Only supported for signing and for verification of CRLs, CSRs, and OCSP responses) |
| ECDSA_WITH_SHA256 | 10 | ECDSA with SHA-256 |
| ECDSA_WITH_SHA384 | 11 | ECDSA with SHA-384 |
| ECDSA_WITH_SHA512 | 12 | ECDSA with SHA-512 |
| SHA256_WITH_RSA_PSS | 13 | SHA-256 with RSA-PSS |
| SHA384_WITH_RSA_PSS | 14 | SHA-384 with RSA-PSS |
| SHA512_WITH_RSA_PSS | 15 | SHA-512 with RSA-PSS |
| PURE_ED25519 | 16 | Pure Ed25519 |


 

 


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
| GetFileStream | [GetFileRequest](#mpi-v1-GetFileRequest) | [FileDataChunk](#mpi-v1-FileDataChunk) stream | GetFileStream requests the file content in chunks. MP and agent should agree on size to use stream vs non-stream. For smaller files, it may be more efficient to not-stream. |
| UpdateFileStream | [FileDataChunk](#mpi-v1-FileDataChunk) stream | [UpdateFileResponse](#mpi-v1-UpdateFileResponse) | UpdateFileStream uploads the file content in streams. MP and agent should agree on size to use stream vs non-stream. For smaller files, it may be more efficient to not-stream. |

 



<a name="mpi_v1_command-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mpi/v1/command.proto
These proto definitions follow https://protobuf.dev/programming-guides/style/
and recommendations outlined in https://static.sched.com/hosted_files/kccncna17/ad/2017%20CloudNativeCon%20-%20Mod%20gRPC%20Services.pdf


<a name="mpi-v1-APIActionRequest"></a>

### APIActionRequest
Perform an associated API action on an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_id | [string](#string) |  | the identifier associated with the instance |
| nginx_plus_action | [NGINXPlusAction](#mpi-v1-NGINXPlusAction) |  |  |






<a name="mpi-v1-APIDetails"></a>

### APIDetails



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| location | [string](#string) |  | the API location directive |
| listen | [string](#string) |  | the API listen directive |
| Ca | [string](#string) |  | the API CA file path |






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
| auxiliary_command | [AuxiliaryCommandServer](#mpi-v1-AuxiliaryCommandServer) |  | Auxiliary Command server settings |






<a name="mpi-v1-AuxiliaryCommandServer"></a>

### AuxiliaryCommandServer
The auxiliary server settings, associated with messaging from an external source


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [ServerSettings](#mpi-v1-ServerSettings) |  | Server configuration (e.g., host, port, type) |
| auth | [AuthSettings](#mpi-v1-AuthSettings) |  | Authentication configuration (e.g., token) |
| tls | [TLSSettings](#mpi-v1-TLSSettings) |  | TLS configuration for secure communication |






<a name="mpi-v1-CommandServer"></a>

### CommandServer
The command server settings, associated with messaging from an external source


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [ServerSettings](#mpi-v1-ServerSettings) |  | Server configuration (e.g., host, port, type) |
| auth | [AuthSettings](#mpi-v1-AuthSettings) |  | Authentication configuration (e.g., token) |
| tls | [TLSSettings](#mpi-v1-TLSSettings) |  | TLS configuration for secure communication |






<a name="mpi-v1-CommandStatusRequest"></a>

### CommandStatusRequest
Request an update on a particular command






<a name="mpi-v1-ConfigApplyRequest"></a>

### ConfigApplyRequest
Additional information associated with a ConfigApplyRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| overview | [FileOverview](#mpi-v1-FileOverview) |  | set of files related to the request |






<a name="mpi-v1-ConfigUploadRequest"></a>

### ConfigUploadRequest
Additional information associated with a ConfigUploadRequest


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| overview | [FileOverview](#mpi-v1-FileOverview) |  | set of files related to the request |






<a name="mpi-v1-ContainerInfo"></a>

### ContainerInfo
Container information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| container_id | [string](#string) |  | The identifier of the container |
| hostname | [string](#string) |  | The name of the host |
| release_info | [ReleaseInfo](#mpi-v1-ReleaseInfo) |  | Release information of the container |






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






<a name="mpi-v1-GetHTTPUpstreamServers"></a>

### GetHTTPUpstreamServers
Get HTTP Upstream Servers for an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_upstream_name | [string](#string) |  | the name of the upstream |






<a name="mpi-v1-GetStreamUpstreams"></a>

### GetStreamUpstreams
Get Stream Upstream Servers for an instance






<a name="mpi-v1-GetUpstreams"></a>

### GetUpstreams
Get Upstreams for an instance






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
| nginx_app_protect_runtime_info | [NGINXAppProtectRuntimeInfo](#mpi-v1-NGINXAppProtectRuntimeInfo) |  | NGINX App Protect runtime information |
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






<a name="mpi-v1-NGINXAppProtectRuntimeInfo"></a>

### NGINXAppProtectRuntimeInfo
A set of runtime NGINX App Protect settings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| release | [string](#string) |  | NGINX App Protect Release |
| attack_signature_version | [string](#string) |  | Attack signature version |
| threat_campaign_version | [string](#string) |  | Threat campaign version |
| enforcer_engine_version | [string](#string) |  | Enforcer engine version |






<a name="mpi-v1-NGINXPlusAction"></a>

### NGINXPlusAction
Perform an action using the NGINX Plus API on an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| update_http_upstream_servers | [UpdateHTTPUpstreamServers](#mpi-v1-UpdateHTTPUpstreamServers) |  |  |
| get_http_upstream_servers | [GetHTTPUpstreamServers](#mpi-v1-GetHTTPUpstreamServers) |  |  |
| update_stream_servers | [UpdateStreamServers](#mpi-v1-UpdateStreamServers) |  |  |
| get_upstreams | [GetUpstreams](#mpi-v1-GetUpstreams) |  |  |
| get_stream_upstreams | [GetStreamUpstreams](#mpi-v1-GetStreamUpstreams) |  |  |






<a name="mpi-v1-NGINXPlusRuntimeInfo"></a>

### NGINXPlusRuntimeInfo
A set of runtime NGINX Plus settings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stub_status | [APIDetails](#mpi-v1-APIDetails) |  | the stub status API details |
| access_logs | [string](#string) | repeated | a list of access_logs |
| error_logs | [string](#string) | repeated | a list of error_logs |
| loadable_modules | [string](#string) | repeated | List of NGINX potentially loadable modules (installed but not loaded). |
| dynamic_modules | [string](#string) | repeated | List of NGINX dynamic modules. |
| plus_api | [APIDetails](#mpi-v1-APIDetails) |  | the plus API details |






<a name="mpi-v1-NGINXRuntimeInfo"></a>

### NGINXRuntimeInfo
A set of runtime NGINX OSS settings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stub_status | [APIDetails](#mpi-v1-APIDetails) |  | the stub status API details |
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






<a name="mpi-v1-UpdateHTTPUpstreamServers"></a>

### UpdateHTTPUpstreamServers
Update HTTP Upstream Servers for an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_upstream_name | [string](#string) |  | the name of the upstream to update |
| servers | [google.protobuf.Struct](#google-protobuf-Struct) | repeated | a list of upstream servers |






<a name="mpi-v1-UpdateStreamServers"></a>

### UpdateStreamServers
Update Upstream Stream Servers for an instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| upstream_stream_name | [string](#string) |  | the name of the upstream stream |
| servers | [google.protobuf.Struct](#google-protobuf-Struct) | repeated | a list of upstream stream servers |





 


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
| INSTANCE_TYPE_NGINX_APP_PROTECT | 5 | NGINX App Protect |


 

 


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

