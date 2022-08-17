package sources

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/stretchr/testify/assert"
)

func TestNewLoadSource(t *testing.T) {
	namespace := "test"
	actual := NewLoadSource(namespace)

	assert.Equal(t, "load", actual.group)
	assert.Equal(t, namespace, actual.namespace)
}

func TestLoadCollect(t *testing.T) {
	namespace := "test"

	loadSource := NewLoadSource(namespace)
	loadSource.avgStatsFunc = func() (*load.AvgStat, error) {
		return &load.AvgStat{}, nil
	}

	ctx := context.TODO()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	channel := make(chan *proto.StatsEntity, 100)
	loadSource.Collect(ctx, wg, channel)
	wg.Wait()

	actual := <-channel

	actualMetricNames := []string{}
	for _, simpleMetric := range actual.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
	}
	sort.Strings(actualMetricNames)
	expected := []string{"test.load.1", "test.load.15", "test.load.5"}

	assert.Equal(t, expected, actualMetricNames)
}
