// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

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

	parser "github.com/nginx/agent/v3/internal/datasource/config"
	datasource "github.com/nginx/agent/v3/internal/datasource/proto"
	"github.com/nginx/agent/v3/internal/model"

	"google.golang.org/protobuf/proto"

	"github.com/nginxinc/nginx-plus-go-client/v2/client"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/internal/datasource/host"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

const (
	apiFormat         = "http://%s%s"
	unixPlusAPIFormat = "http://nginx-plus-api%s"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . resourceServiceInterface

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . logTailerOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . instanceOperator

type resourceServiceInterface interface {
	AddInstances(instanceList []*mpi.Instance) *mpi.Resource
	UpdateInstances(ctx context.Context, instanceList []*mpi.Instance) *mpi.Resource
	DeleteInstances(ctx context.Context, instanceList []*mpi.Instance) *mpi.Resource
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
)

type ResourceService struct {
	resource          *mpi.Resource
	nginxConfigParser parser.ConfigParser
	agentConfig       *config.Config
	instanceOperators map[string]instanceOperator // key is instance ID
	info              host.InfoInterface
	resourceMutex     sync.Mutex
	operatorsMutex    sync.Mutex
}

func NewResourceService(ctx context.Context, agentConfig *config.Config) *ResourceService {
	resourceService := &ResourceService{
		resource:          &mpi.Resource{},
		resourceMutex:     sync.Mutex{},
		info:              host.NewInfo(),
		operatorsMutex:    sync.Mutex{},
		instanceOperators: make(map[string]instanceOperator),
		nginxConfigParser: parser.NewNginxConfigParser(agentConfig),
		agentConfig:       agentConfig,
	}

	resourceService.updateResourceInfo(ctx)

	return resourceService
}

func (r *ResourceService) AddInstances(instanceList []*mpi.Instance) *mpi.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()
	r.resource.Instances = append(r.resource.GetInstances(), instanceList...)
	r.AddOperator(instanceList)

	return r.resource
}

func (r *ResourceService) Instance(instanceID string) *mpi.Instance {
	for _, instance := range r.resource.GetInstances() {
		if instance.GetInstanceMeta().GetInstanceId() == instanceID {
			return instance
		}
	}

	return nil
}

func (r *ResourceService) AddOperator(instanceList []*mpi.Instance) {
	r.operatorsMutex.Lock()
	defer r.operatorsMutex.Unlock()
	for _, instance := range instanceList {
		r.instanceOperators[instance.GetInstanceMeta().GetInstanceId()] = NewInstanceOperator(r.agentConfig)
	}
}

func (r *ResourceService) RemoveOperator(instanceList []*mpi.Instance) {
	r.operatorsMutex.Lock()
	defer r.operatorsMutex.Unlock()
	for _, instance := range instanceList {
		delete(r.instanceOperators, instance.GetInstanceMeta().GetInstanceId())
	}
}

func (r *ResourceService) UpdateInstances(ctx context.Context, instanceList []*mpi.Instance) *mpi.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	for _, updatedInstance := range instanceList {
		resourceCopy, ok := proto.Clone(r.resource).(*mpi.Resource)
		if ok {
			for _, instance := range resourceCopy.GetInstances() {
				if updatedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
					instance.InstanceMeta = updatedInstance.GetInstanceMeta()
					instance.InstanceRuntime = updatedInstance.GetInstanceRuntime()
					instance.InstanceConfig = updatedInstance.GetInstanceConfig()
				}
			}
			r.resource = resourceCopy
		} else {
			slog.WarnContext(ctx, "Unable to clone resource while updating instances", "resource",
				r.resource, "instances", instanceList)
		}
	}

	return r.resource
}

func (r *ResourceService) DeleteInstances(ctx context.Context, instanceList []*mpi.Instance) *mpi.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	for _, deletedInstance := range instanceList {
		resourceCopy, ok := proto.Clone(r.resource).(*mpi.Resource)
		if ok {
			for index, instance := range resourceCopy.GetInstances() {
				if deletedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
					r.resource.Instances = append(r.resource.Instances[:index], r.resource.GetInstances()[index+1:]...)
				}
			}
		} else {
			slog.WarnContext(ctx, "Unable to clone resource while deleting instances", "resource",
				r.resource, "instances", instanceList)
		}
	}
	r.RemoveOperator(instanceList)

	return r.resource
}

func (r *ResourceService) ApplyConfig(ctx context.Context, instanceID string) (*model.NginxConfigContext, error) {
	var instance *mpi.Instance
	operator := r.instanceOperators[instanceID]

	if operator == nil {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	for _, resourceInstance := range r.resource.GetInstances() {
		if resourceInstance.GetInstanceMeta().GetInstanceId() == instanceID {
			instance = resourceInstance
		}
	}

	nginxConfigContext, parseErr := r.nginxConfigParser.Parse(ctx, instance)
	if parseErr != nil || nginxConfigContext == nil {
		return nil, fmt.Errorf("failed to parse config %w", parseErr)
	}

	datasource.UpdateNginxInstanceRuntime(instance, nginxConfigContext)

	slog.DebugContext(ctx, "Updated Instance Runtime after parsing config", "instance", instance.GetInstanceRuntime())

	valErr := operator.Validate(ctx, instance)
	if valErr != nil {
		return nil, fmt.Errorf("failed validating config %w", valErr)
	}

	reloadErr := operator.Reload(ctx, instance)
	if reloadErr != nil {
		return nil, fmt.Errorf("failed to reload NGINX %w", reloadErr)
	}

	return nginxConfigContext, nil
}

func (r *ResourceService) GetHTTPUpstreamServers(ctx context.Context, instance *mpi.Instance,
	upstream string,
) ([]client.UpstreamServer, error) {
	plusClient, err := r.createPlusClient(instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, err
	}

	servers, getServersErr := plusClient.GetHTTPServers(ctx, upstream)

	slog.WarnContext(ctx, "Error returned from NGINX Plus client, GetHTTPUpstreamServers", "err", getServersErr)

	return servers, createPlusAPIError(getServersErr)
}

func (r *ResourceService) GetUpstreams(ctx context.Context, instance *mpi.Instance,
) (*client.Upstreams, error) {
	plusClient, err := r.createPlusClient(instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, err
	}

	servers, getUpstreamsErr := plusClient.GetUpstreams(ctx)

	slog.WarnContext(ctx, "Error returned from NGINX Plus client, GetUpstreams", "err", getUpstreamsErr)

	return servers, createPlusAPIError(getUpstreamsErr)
}

func (r *ResourceService) GetStreamUpstreams(ctx context.Context, instance *mpi.Instance,
) (*client.StreamUpstreams, error) {
	plusClient, err := r.createPlusClient(instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, err
	}

	streamUpstreams, getServersErr := plusClient.GetStreamUpstreams(ctx)

	slog.WarnContext(ctx, "Error returned from NGINX Plus client, GetStreamUpstreams", "err", getServersErr)

	return streamUpstreams, createPlusAPIError(getServersErr)
}

// max number of returns from function is 3
// nolint: revive
func (r *ResourceService) UpdateStreamServers(ctx context.Context, instance *mpi.Instance, upstream string,
	upstreams []*structpb.Struct,
) (added, updated, deleted []client.StreamUpstreamServer, err error) {
	plusClient, err := r.createPlusClient(instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, nil, nil, err
	}

	servers := convertToStreamUpstreamServer(upstreams)

	added, updated, deleted, updateError := plusClient.UpdateStreamServers(ctx, upstream, servers)

	slog.WarnContext(ctx, "Error returned from NGINX Plus client, UpdateStreamServers", "err", updateError)

	return added, updated, deleted, createPlusAPIError(updateError)
}

// max number of returns from function is 3
// nolint: revive
func (r *ResourceService) UpdateHTTPUpstreamServers(ctx context.Context, instance *mpi.Instance, upstream string,
	upstreams []*structpb.Struct,
) (added, updated, deleted []client.UpstreamServer, err error) {
	plusClient, err := r.createPlusClient(instance)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create plus client ", "error", err)
		return nil, nil, nil, err
	}

	servers := convertToUpstreamServer(upstreams)

	added, updated, deleted, updateError := plusClient.UpdateHTTPServers(ctx, upstream, servers)

	if updateError != nil {
		slog.WarnContext(ctx, "Error returned from NGINX Plus client, UpdateHTTPUpstreamServers", "err", updateError)
	}

	return added, updated, deleted, createPlusAPIError(updateError)
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

func (r *ResourceService) createPlusClient(instance *mpi.Instance) (*client.NginxClient, error) {
	plusAPI := instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo().GetPlusApi()
	var endpoint string

	if plusAPI.GetLocation() == "" || plusAPI.GetListen() == "" {
		return nil, errors.New("failed to preform API action, NGINX Plus API is not configured")
	}

	slog.Info("location", "", plusAPI.GetListen())
	if strings.HasPrefix(plusAPI.GetListen(), "unix:") {
		endpoint = fmt.Sprintf(unixPlusAPIFormat, plusAPI.GetLocation())
	} else {
		endpoint = fmt.Sprintf(apiFormat, plusAPI.GetListen(), plusAPI.GetLocation())
	}

	httpClient := http.DefaultClient
	caCertLocation := plusAPI.GetCa()
	if caCertLocation != "" {
		slog.Debug("Reading CA certificate", "file_path", caCertLocation)
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
		httpClient = socketClient(strings.TrimPrefix(plusAPI.GetListen(), "unix:"))
	}

	return client.NewNginxClient(endpoint,
		client.WithMaxAPIVersion(), client.WithHTTPClient(httpClient),
	)
}

func (r *ResourceService) updateResourceInfo(ctx context.Context) {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	if r.info.IsContainer() {
		r.resource.Info = r.info.ContainerInfo(ctx)
		r.resource.ResourceId = r.resource.GetContainerInfo().GetContainerId()
		r.resource.Instances = []*mpi.Instance{}
	} else {
		r.resource.Info = r.info.HostInfo(ctx)
		r.resource.ResourceId = r.resource.GetHostInfo().GetHostId()
		r.resource.Instances = []*mpi.Instance{}
	}
}

func socketClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
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
