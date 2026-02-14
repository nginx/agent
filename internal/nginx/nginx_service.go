// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/nginx/agent/v3/pkg/host/exec"
	"github.com/nginx/agent/v3/pkg/nginxprocess"

	"github.com/nginx/agent/v3/pkg/host"

	parser "github.com/nginx/agent/v3/internal/datasource/config"
	datasource "github.com/nginx/agent/v3/internal/datasource/proto"
	"github.com/nginx/agent/v3/internal/model"

	"google.golang.org/protobuf/proto"

	"github.com/nginx/nginx-plus-go-client/v3/client"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nginx/agent/v3/internal/config"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

const (
	apiFormat         = "http://%s%s"
	unixPlusAPIFormat = "http://nginx-plus-api%s"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . nginxServiceInterface

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . logTailerOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . instanceOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . processOperator

type nginxServiceInterface interface {
	UpdateResource(ctx context.Context, resource *mpi.Resource) *mpi.Resource
	ApplyConfig(ctx context.Context, instanceID string) (*model.NginxConfigContext, error)
	Instance(instanceID string) *mpi.Instance
	GetHTTPUpstreamServers(ctx context.Context, instance *mpi.Instance, upstreams string) ([]client.UpstreamServer,
		error)
	UpdateHTTPUpstreamServers(ctx context.Context, instance *mpi.Instance, upstream string,
		upstreams []*structpb.Struct) (added, updated, deleted []client.UpstreamServer, err error)
	GetUpstreams(ctx context.Context, instance *mpi.Instance) (*client.Upstreams, error)
	GetStreamUpstreams(ctx context.Context, instance *mpi.Instance) (*client.StreamUpstreams, error)
	UpdateStreamServers(ctx context.Context, instance *mpi.Instance, upstream string,
		upstreams []*structpb.Struct) (added, updated, deleted []client.StreamUpstreamServer, err error)
}

type (
	instanceOperator interface {
		Validate(ctx context.Context, instance *mpi.Instance) error
		Reload(ctx context.Context, instance *mpi.Instance) error
	}

	logTailerOperator interface {
		Tail(ctx context.Context, errorLogs string, errorChannel chan error)
	}

	processOperator interface {
		FindNginxProcesses(ctx context.Context) ([]*nginxprocess.Process, error)
		NginxWorkerProcesses(ctx context.Context, masterProcessPid int32) []*nginxprocess.Process
		FindParentProcessID(ctx context.Context, instanceID string, nginxProcesses []*nginxprocess.Process,
			executer exec.ExecInterface) (int32, error)
	}
)

type NginxService struct {
	resource          *mpi.Resource
	nginxConfigParser parser.ConfigParser
	agentConfig       *config.Config
	instanceOperator  instanceOperator
	info              host.InfoInterface
	manifestFilePath  string
	resourceMutex     sync.Mutex
	operatorsMutex    sync.Mutex
}

func NewNginxService(ctx context.Context, agentConfig *config.Config) *NginxService {
	resourceService := &NginxService{
		resource:          &mpi.Resource{},
		resourceMutex:     sync.Mutex{},
		info:              host.NewInfo(),
		operatorsMutex:    sync.Mutex{},
		instanceOperator:  NewInstanceOperator(agentConfig),
		nginxConfigParser: parser.NewNginxConfigParser(agentConfig),
		agentConfig:       agentConfig,
		manifestFilePath:  agentConfig.LibDir + "/manifest.json",
	}

	resourceService.updateResourceInfo(ctx)

	return resourceService
}

func (n *NginxService) Instance(instanceID string) *mpi.Instance {
	for _, instance := range n.resource.GetInstances() {
		if instance.GetInstanceMeta().GetInstanceId() == instanceID {
			return instance
		}
	}

	return nil
}

func (n *NginxService) UpdateResource(ctx context.Context, resource *mpi.Resource) *mpi.Resource {
	slog.DebugContext(ctx, "Updating resource", "resource", resource)
	n.resourceMutex.Lock()
	defer n.resourceMutex.Unlock()

	n.resource = resource

	return n.resource
}

func (n *NginxService) ApplyConfig(ctx context.Context, instanceID string) (*model.NginxConfigContext, error) {
	var instance *mpi.Instance

	if n.instanceOperator == nil {
		return nil, errors.New("instance operator is nil")
	}

	for _, resourceInstance := range n.resource.GetInstances() {
		if resourceInstance.GetInstanceMeta().GetInstanceId() == instanceID {
			instance = resourceInstance
		}
	}

	if instance == nil {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	// Need to parse config to determine what error logs to watch if new ones are added as part of the NGINX reload
	nginxConfigContext, parseErr := n.nginxConfigParser.Parse(ctx, instance)
	if parseErr != nil || nginxConfigContext == nil {
		return nil, fmt.Errorf("failed to parse config %w", parseErr)
	}

	nginxConfigContext = n.updateConfigContextFiles(ctx, nginxConfigContext)

	datasource.UpdateNginxInstanceRuntime(instance, nginxConfigContext)

	slog.DebugContext(ctx, "Updated Instance Runtime after parsing config", "instance", instance.GetInstanceRuntime())

	valErr := n.instanceOperator.Validate(ctx, instance)
	if valErr != nil {
		return nil, fmt.Errorf("failed validating config %w", valErr)
	}

	reloadErr := n.instanceOperator.Reload(ctx, instance)
	if reloadErr != nil {
		return nil, fmt.Errorf("failed to reload NGINX %w", reloadErr)
	}

	// Check if APIs have been added/updated/removed
	nginxConfigContext.StubStatus = n.nginxConfigParser.FindStubStatusAPI(ctx, nginxConfigContext)
	nginxConfigContext.PlusAPI = n.nginxConfigParser.FindPlusAPI(ctx, nginxConfigContext)

	datasource.UpdateNginxInstanceRuntime(instance, nginxConfigContext)
	n.updateInstances(ctx, []*mpi.Instance{instance})

	slog.DebugContext(ctx, "Updated Instance Runtime after reloading NGINX", "instance", instance.GetInstanceRuntime())

	return nginxConfigContext, nil
}

func (n *NginxService) GetHTTPUpstreamServers(ctx context.Context, instance *mpi.Instance,
	upstream string,
) ([]client.UpstreamServer, error) {
	plusClient, err := n.createPlusClient(ctx, instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, err
	}

	servers, getServersErr := plusClient.GetHTTPServers(ctx, upstream)
	if getServersErr != nil {
		slog.WarnContext(ctx, "Error returned from NGINX Plus client, GetHTTPUpstreamServers", "error", getServersErr)
	}

	return servers, createPlusAPIError(getServersErr)
}

func (n *NginxService) GetUpstreams(ctx context.Context, instance *mpi.Instance,
) (*client.Upstreams, error) {
	plusClient, err := n.createPlusClient(ctx, instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, err
	}

	servers, getUpstreamsErr := plusClient.GetUpstreams(ctx)

	if getUpstreamsErr != nil {
		slog.WarnContext(ctx, "Error returned from NGINX Plus client, GetUpstreams", "error", getUpstreamsErr)
	}

	return servers, createPlusAPIError(getUpstreamsErr)
}

func (n *NginxService) GetStreamUpstreams(ctx context.Context, instance *mpi.Instance,
) (*client.StreamUpstreams, error) {
	plusClient, err := n.createPlusClient(ctx, instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, err
	}

	streamUpstreams, getServersErr := plusClient.GetStreamUpstreams(ctx)

	if getServersErr != nil {
		slog.WarnContext(ctx, "Error returned from NGINX Plus client, GetStreamUpstreams", "error", getServersErr)
	}

	return streamUpstreams, createPlusAPIError(getServersErr)
}

// max number of returns from function is 3
//
//nolint:revive // maximum return allowed is 3
func (n *NginxService) UpdateStreamServers(ctx context.Context, instance *mpi.Instance, upstream string,
	upstreams []*structpb.Struct,
) (added, updated, deleted []client.StreamUpstreamServer, err error) {
	plusClient, err := n.createPlusClient(ctx, instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, nil, nil, err
	}

	servers := convertToStreamUpstreamServer(upstreams)

	added, updated, deleted, updateError := plusClient.UpdateStreamServers(ctx, upstream, servers)

	if updateError != nil {
		slog.WarnContext(ctx, "Error returned from NGINX Plus client, UpdateStreamServers", "error", updateError)
	}

	return added, updated, deleted, createPlusAPIError(updateError)
}

// max number of returns from function is 3
//
//nolint:revive // maximum return allowed is 3
func (n *NginxService) UpdateHTTPUpstreamServers(ctx context.Context, instance *mpi.Instance, upstream string,
	upstreams []*structpb.Struct,
) (added, updated, deleted []client.UpstreamServer, err error) {
	plusClient, err := n.createPlusClient(ctx, instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, nil, nil, err
	}

	servers := convertToUpstreamServer(upstreams)

	added, updated, deleted, updateError := plusClient.UpdateHTTPServers(ctx, upstream, servers)

	if updateError != nil {
		slog.WarnContext(ctx, "Error returned from NGINX Plus client, UpdateHTTPUpstreamServers", "error", updateError)
	}

	return added, updated, deleted, createPlusAPIError(updateError)
}

func (n *NginxService) updateInstances(ctx context.Context, instanceList []*mpi.Instance) {
	n.resourceMutex.Lock()
	defer n.resourceMutex.Unlock()

	for _, updatedInstance := range instanceList {
		resourceCopy, ok := proto.Clone(n.resource).(*mpi.Resource)
		if ok {
			for _, instance := range resourceCopy.GetInstances() {
				if updatedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
					instance.InstanceMeta = updatedInstance.GetInstanceMeta()
					instance.InstanceRuntime = updatedInstance.GetInstanceRuntime()
					instance.InstanceConfig = updatedInstance.GetInstanceConfig()
				}
			}
			n.resource = resourceCopy
		} else {
			slog.WarnContext(ctx, "Unable to clone resource while updating instances", "resource",
				n.resource, "instances", instanceList)
		}
	}
}

func (n *NginxService) updateConfigContextFiles(ctx context.Context,
	nginxConfigContext *model.NginxConfigContext,
) *model.NginxConfigContext {
	manifestFiles, manifestErr := n.manifestFile()
	if manifestErr != nil {
		slog.ErrorContext(ctx, "Error getting manifest files", "error", manifestErr)
	}

	for _, manifestFile := range manifestFiles {
		if manifestFile.ManifestFileMeta.Unmanaged {
			for _, configFile := range nginxConfigContext.Files {
				if configFile.GetFileMeta().GetName() == manifestFile.ManifestFileMeta.Name {
					configFile.Unmanaged = true
				}
			}
		}
	}

	return nginxConfigContext
}

func (n *NginxService) manifestFile() (map[string]*model.ManifestFile, error) {
	if _, err := os.Stat(n.manifestFilePath); err != nil {
		return nil, err
	}

	file, err := os.ReadFile(n.manifestFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifestFiles map[string]*model.ManifestFile

	err = json.Unmarshal(file, &manifestFiles)
	if err != nil {
		if len(file) == 0 {
			return nil, fmt.Errorf("manifest file is empty: %w", err)
		}

		return nil, fmt.Errorf("failed to parse manifest file: %w", err)
	}

	return manifestFiles, nil
}

func convertToUpstreamServer(upstreams []*structpb.Struct) []client.UpstreamServer {
	var servers []client.UpstreamServer
	res, err := json.Marshal(upstreams)
	if err != nil {
		slog.Error("Failed to marshal upstreams", "error", err, "upstreams", upstreams)
	}
	err = json.Unmarshal(res, &servers)
	if err != nil {
		slog.Error("Failed to unmarshal upstreams", "error", err, "servers", servers)
	}

	return servers
}

func convertToStreamUpstreamServer(streamUpstreams []*structpb.Struct) []client.StreamUpstreamServer {
	var servers []client.StreamUpstreamServer
	res, err := json.Marshal(streamUpstreams)
	if err != nil {
		slog.Error("Failed to marshal stream upstream server", "error", err, "stream_upstreams", streamUpstreams)
	}
	err = json.Unmarshal(res, &servers)
	if err != nil {
		slog.Error("Failed to unmarshal stream upstream server", "error", err, "stream_upstreams", streamUpstreams)
	}

	return servers
}

func (n *NginxService) createPlusClient(ctx context.Context, instance *mpi.Instance) (*client.NginxClient, error) {
	plusAPI := instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo().GetPlusApi()
	var endpoint string

	if plusAPI.GetLocation() == "" || plusAPI.GetListen() == "" {
		return nil, errors.New("failed to preform API action, NGINX Plus API is not configured")
	}

	if strings.HasPrefix(plusAPI.GetListen(), "unix:") {
		endpoint = fmt.Sprintf(unixPlusAPIFormat, plusAPI.GetLocation())
	} else {
		endpoint = fmt.Sprintf(apiFormat, plusAPI.GetListen(), plusAPI.GetLocation())
	}

	httpClient := http.DefaultClient
	caCertLocation := plusAPI.GetCa()
	if caCertLocation != "" {
		slog.DebugContext(ctx, "Reading CA certificate", "file_path", caCertLocation)
		caCert, err := os.ReadFile(caCertLocation)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    caCertPool,
					MinVersion: tls.VersionTLS13,
				},
			},
		}
	}
	if strings.HasPrefix(plusAPI.GetListen(), "unix:") {
		httpClient = socketClient(ctx, strings.TrimPrefix(plusAPI.GetListen(), "unix:"))
	}

	return client.NewNginxClient(endpoint,
		client.WithMaxAPIVersion(), client.WithHTTPClient(httpClient),
	)
}

func (n *NginxService) updateResourceInfo(ctx context.Context) {
	n.resourceMutex.Lock()
	defer n.resourceMutex.Unlock()

	isContainer, err := n.info.IsContainer()
	if err != nil {
		slog.WarnContext(ctx, "Failed to check if resource is container", "error", err)
	}

	if isContainer {
		n.resource.Info, err = n.info.ContainerInfo(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get container info", "error", err)
			return
		}
		n.resource.ResourceId = n.resource.GetContainerInfo().GetContainerId()
		n.resource.Instances = []*mpi.Instance{}
	} else {
		n.resource.Info, err = n.info.HostInfo(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get host info", "error", err)
			return
		}
		n.resource.ResourceId = n.resource.GetHostInfo().GetHostId()
		n.resource.Instances = []*mpi.Instance{}
	}
}

func socketClient(ctx context.Context, socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// createPlusAPIError converts the error returned by the plus go client into the json format used by the NGINX Plus API
func createPlusAPIError(apiErr error) error {
	if apiErr == nil {
		return nil
	}
	_, after, _ := strings.Cut(apiErr.Error(), "error.status")
	errorSlice := strings.Split(after, ";")

	for i, errStr := range errorSlice {
		_, value, _ := strings.Cut(errStr, "=")
		errorSlice[i] = value
	}

	plusErr := plusAPIErr{
		Error: errResponse{
			Status: errorSlice[0],
			Text:   errorSlice[1],
			Code:   errorSlice[2],
		},
		RequestID: errorSlice[3],
		Href:      errorSlice[4],
	}

	r, err := json.Marshal(plusErr)
	if err != nil {
		slog.Error("Unable to marshal NGINX Plus API error", "error", err)
		return apiErr
	}

	return errors.New(string(r))
}
