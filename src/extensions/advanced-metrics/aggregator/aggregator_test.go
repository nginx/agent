/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package aggregator_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/aggregator"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/aggregator/mocks"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	tablesMocks "github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/mocks"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/priority_table"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAggregatorPublish(t *testing.T) {
	testSample := sample.NewSample(11, 11)
	assert.NoError(t, testSample.SetMetric(0, 12))

	testSample2 := sample.NewSample(12, 12)
	assert.NoError(t, testSample2.SetMetric(0, 13))
	assert.NoError(t, testSample2.Key().AddKeyPart(1, 1))

	ctrl := gomock.NewController(t)
	readTableMock := mocks.NewMockReadTable(ctrl)
	publisherMock := mocks.NewMockPublisher(ctrl)
	samplesView := tablesMocks.NewMockSamplesView(ctrl)

	schema := schema.NewSchema()
	l, err := limits.NewLimits(1000, 100)
	assert.NoError(t, err)
	aggregator := aggregator.New(readTableMock, publisherMock, schema, l)

	ctx, cancel := context.WithCancel(context.Background())
	aggregationTickerChannel := make(chan time.Time)
	publishTickerChannel := make(chan time.Time)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		aggregator.Run(ctx, aggregationTickerChannel, publishTickerChannel)
		wg.Done()
	}()

	readTableMock.EXPECT().ReadSnapshot(false).Return(samplesView, nil)
	samplesView.EXPECT().Range(gomock.Any()).Do(func(callback interface{}) {
		rangeCallback, ok := callback.(func(sample *sample.Sample))
		assert.True(t, ok)
		rangeCallback(&testSample)
	})
	aggregationTickerChannel <- time.Now()

	readTableMock.EXPECT().ReadSnapshot(true).Return(samplesView, nil)
	samplesView.EXPECT().Range(gomock.Any()).Do(func(callback func(sample *sample.Sample)) {
		callback(&testSample2)
	})
	publisherMock.EXPECT().Publish(gomock.Any(), nil, gomock.Any()).Do(
		func(ctx context.Context, lookup tables.LookupSet, table *priority_table.PriorityTable) {
			assert.Equal(t, table.Samples(), map[string]*sample.Sample{
				testSample.Key().AsStringKey():  &testSample,
				testSample2.Key().AsStringKey(): &testSample2,
			})
		})

	publishTickerChannel <- time.Now()
	cancel()
	wg.Wait()
}

func TestAggregatorPublishClearPriorityTable(t *testing.T) {
	testSample := sample.NewSample(11, 11)
	assert.NoError(t, testSample.SetMetric(0, 12))

	testSample2 := sample.NewSample(12, 12)
	assert.NoError(t, testSample2.SetMetric(0, 13))
	assert.NoError(t, testSample2.Key().AddKeyPart(1, 1))

	testSample3 := sample.NewSample(12, 12)
	assert.NoError(t, testSample3.SetMetric(0, 14))
	assert.NoError(t, testSample3.Key().AddKeyPart(2, 2))

	ctrl := gomock.NewController(t)
	readTableMock := mocks.NewMockReadTable(ctrl)
	publisherMock := mocks.NewMockPublisher(ctrl)
	samplesView := tablesMocks.NewMockSamplesView(ctrl)

	schema := schema.NewSchema()
	l, err := limits.NewLimits(1000, 100)
	assert.NoError(t, err)
	aggregator := aggregator.New(readTableMock, publisherMock, schema, l)

	ctx, cancel := context.WithCancel(context.Background())
	aggregationTickerChannel := make(chan time.Time)
	publishTickerChannel := make(chan time.Time)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		aggregator.Run(ctx, aggregationTickerChannel, publishTickerChannel)
		wg.Done()
	}()

	readTableMock.EXPECT().ReadSnapshot(false).Return(samplesView, nil)
	samplesView.EXPECT().Range(gomock.Any()).Do(func(callback interface{}) {
		rangeCallback, ok := callback.(func(sample *sample.Sample))
		assert.True(t, ok)
		rangeCallback(&testSample)
	})
	aggregationTickerChannel <- time.Now()

	readTableMock.EXPECT().ReadSnapshot(true).Return(samplesView, nil)
	samplesView.EXPECT().Range(gomock.Any()).Do(func(callback func(sample *sample.Sample)) {
		callback(&testSample2)
	})
	publisherMock.EXPECT().Publish(gomock.Any(), nil, gomock.Any()).Do(
		func(ctx context.Context, lookup tables.LookupSet, table *priority_table.PriorityTable) {
			assert.Equal(t, table.Samples(), map[string]*sample.Sample{
				testSample.Key().AsStringKey():  &testSample,
				testSample2.Key().AsStringKey(): &testSample2,
			})
		})

	publishTickerChannel <- time.Now()

	readTableMock.EXPECT().ReadSnapshot(true).Return(samplesView, nil)
	samplesView.EXPECT().Range(gomock.Any()).Do(func(callback func(sample *sample.Sample)) {
		callback(&testSample3)
	})
	publisherMock.EXPECT().Publish(gomock.Any(), nil, gomock.Any()).Do(
		func(ctx context.Context, lookup tables.LookupSet, table *priority_table.PriorityTable) {
			assert.Equal(t, table.Samples(), map[string]*sample.Sample{
				testSample3.Key().AsStringKey(): &testSample3,
			})
		})

	publishTickerChannel <- time.Now()

	cancel()
	wg.Wait()
}
