/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collectors

import (
	"context"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/stretchr/testify/mock"
)

func GetNginxSourceMock() *NginxSourceMock {
	mockSource := new(NginxSourceMock)
	mockSource.On("Collect", mock.Anything, mock.Anything, mock.Anything).Once()
	return mockSource
}

type NginxSourceMock struct {
	mock.Mock
}

func (m *NginxSourceMock) Collect(ctx context.Context, wg *sync.WaitGroup, statsChannel chan<- *proto.StatsEntity) {
	m.Called(ctx, wg, statsChannel)
	wg.Done()
}

func (m *NginxSourceMock) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	m.Called(dimensions, collectorConf)
}

func (m *NginxSourceMock) Stop() {
	m.Called()
}

type SourceMock struct {
	mock.Mock
}

func (m *SourceMock) Collect(ctx context.Context, wg *sync.WaitGroup, statsChannel chan<- *proto.StatsEntity) {
	m.Called(ctx, wg, statsChannel)
	wg.Done()
}
