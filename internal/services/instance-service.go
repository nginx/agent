package services

import (
	"github.com/nginx/agent/v3/internal/data-sources/nginx"
	"github.com/nginx/agent/v3/internal/models"
	"github.com/shirou/gopsutil/v3/process"
)

type InstanceService struct{}

func NewInstanceService() *InstanceService {
	return &InstanceService{}
}

func (is *InstanceService) GetInstances() ([]*instances.Instance, error) {
	processes, err := process.Processes()
	if err != nil {
		return []*instances.Instance{}, err
	}
	return nginx.GetInstances(processes)
}
