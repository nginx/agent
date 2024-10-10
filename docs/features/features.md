# NGINX Agent Features

- [Introduction](#introduction)
- [Feature Flags Overview](#feature-flags-overview)
- [Conflicting Combinations](#conflicting-combinations)
- [Cobra CLI Parameters](#cobra-cli-parameters)
- [Environment Variables](#environment-variables)
- [Configuration File](#configuration-file)
- [Dynamic Updates via gRPC](#dynamic-updates-via-grpc)
- [Internal State Management](#internal-state-management)
- [Code Path Management](#code-path-management)
- [Dynamic Feature Toggling](#dynamic-feature-toggling)
- [Security Considerations](#security-considerations)
- [Conclusion](#conclusion)

## Introduction

This document outlines the design of a feature flag management system
for the NGINX Agent. The system enables dynamic toggling of features
based on configurations provided via multiple sources, including CLI
parameters, environment variables, configuration files, and via protobuf
gRPC messages. Feature flags control the execution paths within the
application, allowing for flexible and controlled feature management.

## Feature Flags Overview

The design goal of this document is to capture fine grained features in
NGINX Agent v3.

The ultimate goal of this design is to delegate fine-grained control of
high-level feature sets to the configuration.

The system manages the following feature flags:

| Feature Flag  | Sub Category 1     | Description                                                                                                                                      | Default                                       |
|---------------|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------|
| configuration |                    | Full read/write management of configurations toggled by DataPlaneConfig ConfigMode                                                               | On                                            |
| certificates  |                    | Inclusion of public keys and other certificates in the configurations toggled by DataPlaneConfig CertMode                                        | Off                                           |
| connection    |                    | Sends an initial connection message reporting instance information on presence of Command ServerConfig Host and Port                             | On                                            |
| file-watcher  |                    | Monitoring of file changes in allowed directories; may trigger DataPlane (DP) => ManagementPlane synchronisation of files configured by Watchers | On                                            |
| agent-api     |                    | REST Application Programming Interface (API) for NGINX Agent                                                                                     | Off                                           |
| metrics       |                    | Full metrics reporting                                                                                                                           | On                                            |
|               | metrics-host       | All host-level metrics                                                                                                                           | On (if running in a host, virtualised or not) |
|               | metrics-container  | Container-level metrics read from cgroup information                                                                                             | On (if running in a container)                |
|               | metrics-instance   | All OSS and Plus Metrics, depending on what instance is present                                                                                  | On (if instance present e.g. NGINX)           |

## Conflicting Combinations

If there is a feature flag enabled and conflicting features are
enabled, the more specific feature flag takes precedence. If the
feature flag is set e.g. metrics-container in a non-containerised
execution environment, then no useful metrics will be reported.

**config-certificates** being read-only and reporting ssl under
instances may conflict.

**metrics** will include every specified metrics category (hence the
inherited specification). If fine-grained metrics are required, this
needs to be absent from the list. 

### Cobra CLI Parameters

**Usage**: Users can enable or disable features via CLI flags when
    launching the application.

**Example**:

```bash
./nginx-agent --features=connection,config,metrics,process-watcher,file-watcher,agent-api
```

Specifies a comma-separated list of features enabled for the Agent.

- **Implementation**: Utilises [Cobra](https://github.com/spf13/cobra)
    to define and parse CLI flags for each feature.

### Environment Variables

- **Usage**: Environment variables provide an alternative
    configuration method, suitable for containerised deployments.

- **Naming Convention**: NGINX_AGENT_FEATURES and is added to the list

- **Implementation**: Use a library
    [Viper](https://github.com/spf13/viper) to bind environment
    variables to feature flags.

### Configuration File

- **Usage**: A configuration file (e.g., YAML, JSON, TOML) can list
    enabled features and their parameters.

- **Example (YAML)**:

```yaml
features:
- connection
- configuration
- metrics
- process-watcher
- file-watcher
- agent-api

```

**Implementation**: Parse the configuration file during initialisation using Viper.

### Dynamic Updates via gRPC

- Through the MPI
    send an updated AgentConfig message with a different list of
    features.

## Internal State Management

- **Singleton Pattern**: Implement a singleton FeaturePlugin to
    maintain the state of all feature flags. On configuration change,
    use the message bus to notify throughout the application of any
    changes.

- **Thread-Safety**: Use synchronisation mechanisms to ensure safe
    concurrent access of changes

## Code Path Management

**Conditional Execution**: Use the FeatureManager to check feature
    states before executing specific code paths.

**Example**:

```go
    if featureManager.IsEnabled("metrics") {
        // Execute full metrics reporting code path
    } else {
        // Execute alternative or no-op code path
    }
```

- **Abstraction**: Encapsulate feature checks within helper functions
    or middleware to streamline conditional logic across the codebase.

## Dynamic Feature Toggling

Implement methods within FeaturePlugin to enable, disable, and retrieve
feature states. Watch the nginx-agent.conf file for changes. Listen to
gRPC messages for AgentConfig changes.

Example useful functionality:

```go

func (fm *FeatureManager) IsEnabled(featureName string) bool {

    fm.mutex.RLock()

    defer fm.mutex.RUnlock()

    // Code check to see if the feature is enabled in the AgentConfig

}

func (fm *FeatureManager) UpdateFeature(featureName string, enabled bool, parameters map[string]interface{}) error {

    fm.mutex.Lock()

    defer fm.mutex.Unlock()

    switch featureName {

        case "config":

        // update the AgentConfig with the new feature and it\'s configuration

        }

    return nil

} 
```

## Security Considerations

- **Authentication & Authorisation**: Ensure that only authorised
    entities can send gRPC messages to update feature flags.

- **Validation**: Validate feature names and parameters received via
    gRPC to prevent invalid configurations (e.g. using
    <https://buf.build/bufbuild/protovalidate/docs/main:buf.validate> )

- **Audit Logging**: Log all feature flag changes for auditing and
    rollback purposes.

- **Secret Management**: Securely handle sensitive configuration
    parameters, especially for features like dealing with secrets and
    connection settings.

## Conclusion

This design provides a flexible and dynamic feature flag management
system that integrates multiple configuration sources and allows
real-time updates via gRPC.

By centralising feature state management and ensuring thread safety, the
system enables controlled feature toggling with minimal impact on the
running application.

Proper security measures and validation ensure the integrity and
reliability of the feature management process.


[def]: #conflictingcombinations
