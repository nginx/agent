package publisher

import (
	"context"
	"errors"
	"time"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/aggregator"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/lookup"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	"github.com/sirupsen/logrus"
)

// Publisher is responsible for translation of internal tables to public structures and publishing process of metrics.
type Publisher struct {
	schema         *schema.Schema
	metricsChannel chan<- []*MetricSet
}

func New(metricsChannel chan []*MetricSet, schema *schema.Schema) *Publisher {
	return &Publisher{
		schema:         schema,
		metricsChannel: metricsChannel,
	}
}

func (p *Publisher) Publish(ctx context.Context, lookups tables.LookupSet, priorityTable aggregator.PriorityTable) error {
	dimensionKeyPartSizes := p.schema.DimensionKeyPartSizes()
	metrics := make([]*MetricSet, 0, len(priorityTable.Samples()))

	for _, s := range priorityTable.Samples() {
		metric := &MetricSet{
			Dimensions: p.buildDimensions(s, lookups, dimensionKeyPartSizes),
			Metrics:    p.buildMetrics(s),
		}

		metrics = append(metrics, metric)
	}

	select {
	case <-time.After(time.Second):
		return errors.New("timed out while publishing metrics report")
	case <-ctx.Done():
		return ctx.Err()
	case p.metricsChannel <- metrics:

	}
	return nil
}

func (p *Publisher) buildMetrics(s *sample.Sample) []Metric {
	metrics := make([]Metric, 0, len(s.Metrics()))
	for i, metric := range s.Metrics() {
		if metric.Count == 0 {
			continue
		}
		metrics = append(metrics, Metric{
			Name:   p.schema.Metric(i).Name,
			Values: metric,
		})
	}

	return metrics
}

func (p *Publisher) buildDimensions(s *sample.Sample, lookups tables.LookupSet, dimensionKeyPartSizes []int) []Dimension {
	dimensionLookupCodes := s.Key().GetKeyParts(dimensionKeyPartSizes)
	dimensions := make([]Dimension, 0, len(dimensionLookupCodes))
	for _, dimensionSchema := range p.schema.Dimensions() {
		var dimensionValue string
		var err error
		lookupCode := dimensionLookupCodes[dimensionSchema.Index()]
		if lookupCode == lookup.LookupNACode {
			continue
		}
		if dimensionSchema.Transform != nil {
			dimensionValue, err = dimensionSchema.Transform.FromLookupCodeToValue(lookupCode)
			if err != nil {
				logrus.Warnf("Code transform for dimension named '%s' failed with error '%v", dimensionSchema.Name, err)
				continue
			}
		} else {
			dimensionValue, err = lookups.LookupCode(dimensionSchema.Index(), lookupCode)
			if err != nil {
				logrus.Warnf("Code lookup for dimension named '%s' failed with error '%v", dimensionSchema.Name, err)
				continue
			}
		}

		dimensions = append(dimensions, Dimension{
			Name:  dimensionSchema.Name,
			Value: dimensionValue,
		})
	}
	return dimensions
}
