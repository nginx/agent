package services

import (
	"github.com/nginx/agent/v3/internal/data-sources/nginx"
	"github.com/nginx/agent/v3/internal/data-sources/os"
	"github.com/nginx/agent/v3/internal/models/instances"
)

type InstanceService struct {
	instances []*instances.Instance
}

func NewInstanceService() *InstanceService {
	return &InstanceService{}
}

func (is *InstanceService) GetInstances() ([]*instances.Instance, error) {
	processes, err := os.GetProcesses()
	if err != nil {
		is.instances = []*instances.Instance{}
	} else {
		is.instances, err = nginx.GetInstances(processes)
	}

	return is.instances, err
}
