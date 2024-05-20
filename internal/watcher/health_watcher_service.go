package watcher

import (
	"context"
	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type (
	healthWatcherOperator interface {
		Watch(ctx context.Context) Health
	}

	Health struct {
		healthStatus *v1.InstanceHealth_InstanceHealthStatus
		description  string
	}

	HealthWatcherService struct {
		cache    map[string]Health // key is instanceID
		watchers map[string]healthWatcherOperator
	}
)

func NewHealthWatcherService() *HealthWatcherService {
	return &HealthWatcherService{
		cache:    make(map[string]Health),
		watchers: make(map[string]healthWatcherOperator),
	}
}

func (hw *HealthWatcherService) AddHealthWatcher(instanceID string) {

}
