/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNginxProcessUpdate(t *testing.T) {
	nginxProcess := NewNginxProcess(&metrics.CommonDim{}, "test", utils.NewMockNginxBinary())

	assert.Equal(t, "", nginxProcess.baseDimensions.InstanceTags)

	nginxProcess.Update(
		&metrics.CommonDim{
			InstanceTags: "new-tag",
		},
		&metrics.NginxCollectorConfig{},
	)

	assert.Equal(t, "new-tag", nginxProcess.baseDimensions.InstanceTags)
}

func TestNginxProcessCollector_Collect_Process(t *testing.T) {
	dimensions := &metrics.CommonDim{
		SystemId:      systemId,
		Hostname:      host,
		InstanceGroup: instanceGroup,
		DisplayName:   displayName,
		NginxId:       nginxId,
	}

	mockBinary := &utils.MockNginxBinary{}
	mockBinary.On("GetNginxDetailsByID", mock.Anything).Return(&proto.NginxDetails{
		NginxId: nginxId,
		Plus: &proto.NginxPlusMetaData{
			Enabled: true,
			Release: "1.25.0",
		},
	})

	n := NewNginxProcess(dimensions, OSSNamespace, mockBinary)

	// tell the mock nginx binary to return something
	ctx := context.TODO()

	wg := sync.WaitGroup{}
	wg.Add(1)
	m := make(chan *metrics.StatsEntityWrapper)
	go n.Collect(ctx, &wg, m)

	time.Sleep(100 * time.Millisecond)
	mockBinary.AssertNumberOfCalls(t, "GetNginxDetailsByID", 1)

	// prev stats will initially all be 0 because sync.Once will set
	// the prev stats as equal to the initial stats collected
	// that's ok, but we should test the counter gauge computations again later
	metricReport := <-m
	for _, metric := range metricReport.Data.Simplemetrics {
		switch metric.Name {
		case "plus.instance.count":
			assert.Equal(t, float64(1), metric.Value)
		default:
			// if there is an unknown metric, we should fail because
			// we should't have anything but the above
			assert.Failf(t, "saw an unknown metric in test", "saw an unknown metric in test %s", metric.Name)
		}
	}
}

func TestNginxProcessCollector_Collect_NoProcess(t *testing.T) {
	dimensions := &metrics.CommonDim{
		SystemId:      systemId,
		Hostname:      host,
		InstanceGroup: instanceGroup,
		DisplayName:   displayName,
		NginxId:       nginxId,
	}

	mockBinary := &utils.MockNginxBinary{}
	mockBinary.On("GetNginxDetailsByID", mock.Anything).Return(&proto.NginxDetails{
		NginxId: "",
		Plus:    nil,
	})

	n := NewNginxProcess(dimensions, OSSNamespace, mockBinary)

	// tell the mock nginx binary to return something
	ctx := context.TODO()

	wg := sync.WaitGroup{}
	wg.Add(1)
	m := make(chan *metrics.StatsEntityWrapper)
	go n.Collect(ctx, &wg, m)

	time.Sleep(100 * time.Millisecond)
	mockBinary.AssertNumberOfCalls(t, "GetNginxDetailsByID", 1)

	// prev stats will initially all be 0 because sync.Once will set
	// the prev stats as equal to the initial stats collected
	// that's ok, but we should test the counter gauge computations again later
	metricReport := <-m
	for _, metric := range metricReport.Data.Simplemetrics {
		switch metric.Name {
		case "plus.instance.count":
			assert.Equal(t, float64(0), metric.Value)
		default:
			// if there is an unknown metric, we should fail because
			// we should't have anything but the above
			assert.Failf(t, "saw an unknown metric in test", "saw an unknown metric in test %s", metric.Name)
		}
	}
}

func TestNginxProcessCollector_Collect_NotPlus(t *testing.T) {
	dimensions := &metrics.CommonDim{
		SystemId:      systemId,
		Hostname:      host,
		InstanceGroup: instanceGroup,
		DisplayName:   displayName,
		NginxId:       nginxId,
	}

	mockBinary := &utils.MockNginxBinary{}
	mockBinary.On("GetNginxDetailsByID", mock.Anything).Return(&proto.NginxDetails{
		NginxId: nginxId,
		Plus: &proto.NginxPlusMetaData{
			Enabled: false,
			Release: "1.1.1",
		},
	})

	n := NewNginxProcess(dimensions, OSSNamespace, mockBinary)

	// tell the mock nginx binary to return something
	ctx := context.TODO()

	wg := sync.WaitGroup{}
	wg.Add(1)
	m := make(chan *metrics.StatsEntityWrapper)
	go n.Collect(ctx, &wg, m)

	time.Sleep(100 * time.Millisecond)
	mockBinary.AssertNumberOfCalls(t, "GetNginxDetailsByID", 1)

	// prev stats will initially all be 0 because sync.Once will set
	// the prev stats as equal to the initial stats collected
	// that's ok, but we should test the counter gauge computations again later
	metricReport := <-m
	for _, metric := range metricReport.Data.Simplemetrics {
		switch metric.Name {
		case "plus.instance.count":
			assert.Equal(t, float64(0), metric.Value)
		default:
			// if there is an unknown metric, we should fail because
			// we should't have anything but the above
			assert.Failf(t, "saw an unknown metric in test", "saw an unknown metric in test %s", metric.Name)
		}
	}
}
