/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package aggregator

import (
	"context"
	"time"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/priority_table"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	log "github.com/sirupsen/logrus"
)

//go:generate mockgen -source aggregator.go -destination mocks/aggregator_mock.go -package mocks

type ReadTable interface {
	ReadSnapshot(resetLookups bool) (tables.SamplesView, tables.LookupSet)
}

type PriorityTable interface {
	Samples() map[string]*sample.Sample
}

type Publisher interface {
	Publish(context.Context, tables.LookupSet, PriorityTable) error
}

// Aggregator is responsible for collection and aggregation in timed manner samples
// gathered from StagingTable.
// Aggregator is driven by two timers which determine aggregation and publication periods.
type Aggregator struct {
	table         ReadTable
	priorityTable *priority_table.PriorityTable
	publisher     Publisher

	schema *schema.Schema
	limits limits.Limits
}

func New(t ReadTable, publisher Publisher, schema *schema.Schema, limit limits.Limits) *Aggregator {
	return &Aggregator{
		table:         t,
		priorityTable: priority_table.NewPriorityTable(schema, limit),
		publisher:     publisher,
		schema:        schema,
		limits:        limit,
	}
}

func (a *Aggregator) Run(ctx context.Context, aggregationTicker <-chan time.Time, publishTicker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-aggregationTicker:
			a.aggregate(ctx, false)
		case <-publishTicker:
			a.aggregate(ctx, true)

		}
	}
}

func (a *Aggregator) aggregate(ctx context.Context, publish bool) {
	samplesView, publishLookupSet := a.table.ReadSnapshot(publish)

	samplesView.Range(func(sample *sample.Sample) {
		err := a.priorityTable.Add(sample)
		if err != nil {
			log.Warningf("Failed to aggregate metric %v: %s", sample, err.Error())
			return
		}
	})

	err := a.priorityTable.CollapseSamples()
	if err != nil {
		log.Errorf("fail to collapse samples: %s", err.Error())
		return
	}

	if publish {
		pt := a.priorityTable
		a.priorityTable = priority_table.NewPriorityTable(a.schema, a.limits)
		err := a.publisher.Publish(ctx, publishLookupSet, pt)
		if err != nil {
			log.Warningf("Failed to publish metrics: %s", err.Error())
		}
	}
}
