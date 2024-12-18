// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

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
	UpdateInstances(instanceList []*mpi.Instance) *mpi.Resource
	DeleteInstances(instanceList []*mpi.Instance) *mpi.Resource
	ApplyConfig(ctx context.Context, instanceID string) error
	Instance(instanceID string) *mpi.Instance
	GetUpstreams(ctx context.Context, instance *mpi.Instance, upstreams string) ([]client.UpstreamServer, error)
	UpdateHTTPUpstreams(ctx context.Context, instance *mpi.Instance, upstream string,
		upstreams []*structpb.Struct) (added, updated, deleted []client.UpstreamServer, err error)
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

func (r *ResourceService) UpdateInstances(instanceList []*mpi.Instance) *mpi.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	for _, updatedInstance := range instanceList {
		for _, instance := range r.resource.GetInstances() {
			if updatedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
				instance.InstanceMeta = updatedInstance.GetInstanceMeta()
				instance.InstanceRuntime = updatedInstance.GetInstanceRuntime()
				instance.InstanceConfig = updatedInstance.GetInstanceConfig()
			}
		}
	}

	return r.resource
}

func (r *ResourceService) DeleteInstances(instanceList []*mpi.Instance) *mpi.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	for _, deletedInstance := range instanceList {
		for index, instance := range r.resource.GetInstances() {
			if deletedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
				r.resource.Instances = append(r.resource.Instances[:index], r.resource.GetInstances()[index+1:]...)
			}
		}
	}
	r.RemoveOperator(instanceList)

	return r.resource
}

func (r *ResourceService) ApplyConfig(ctx context.Context, instanceID string) error {
	var instance *mpi.Instance
	operator := r.instanceOperators[instanceID]

	for _, resourceInstance := range r.resource.GetInstances() {
		if resourceInstance.GetInstanceMeta().GetInstanceId() == instanceID {
			instance = resourceInstance
		}
	}

	valErr := operator.Validate(ctx, instance)
	if valErr != nil {
		return fmt.Errorf("failed validating config %w", valErr)
	}

	reloadErr := operator.Reload(ctx, instance)
	if reloadErr != nil {
		return fmt.Errorf("failed to reload NGINX %w", reloadErr)
	}

	return nil
}

func (r *ResourceService) GetUpstreams(ctx context.Context, instance *mpi.Instance,
	upstream string,
) ([]client.UpstreamServer, error) {
	plusClient, err := r.createPlusClient(instance)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create plus client ", "err", err)
		return nil, err
	}

	servers, getServersErr := plusClient.GetHTTPServers(ctx, upstream)

	return servers, createPlusAPIError(getServersErr)
}

// max number of returns from function is 3
// nolint: revive
func (r *ResourceService) UpdateHTTPUpstreams(ctx context.Context, instance *mpi.Instance, upstream string,
	upstreams []*structpb.Struct,
) (added, updated, deleted []client.UpstreamServer, err error) {
	plusClient, err := r.createPlusClient(instance)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create plus client ", "err", err)
		return nil, nil, nil, err
	}

	servers := convertToUpstreamServer(upstreams)

	return plusClient.UpdateHTTPServers(ctx, upstream, servers)
}

func convertToUpstreamServer(upstreams []*structpb.Struct) []client.UpstreamServer {
	var servers []client.UpstreamServer
	res, err := json.Marshal(upstreams)
	if err != nil {
		slog.Error("", "", err)
	}
	err = json.Unmarshal(res, &servers)
	if err != nil {
		slog.Error("", "", err)
	}

	return servers
}

func (r *ResourceService) createPlusClient(instance *mpi.Instance) (*client.NginxClient, error) {
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
			Test:   errorSlice[1],
			Code:   errorSlice[2],
		},
		RequestID: errorSlice[3],
		Href:      errorSlice[4],
	}

	r, err := json.Marshal(plusErr)
	if err != nil {
		slog.Error("error marshaling error", "err", err)
		return apiErr
	}

	return errors.New(string(r))
}
