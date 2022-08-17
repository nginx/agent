package priority_table

import (
	"container/heap"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/lookup"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	log "github.com/sirupsen/logrus"
)

// PriorityTable represents set of samples with limited size.
// Samples added to the table are ordered by the hitcount of the sample.
// Priority determines which samples dimensions will be aggregated.
type PriorityTable struct {
	samples map[string]*sample.Sample

	schema *schema.Schema
	limits limits.Limits
}

func NewPriorityTable(schema *schema.Schema, limits limits.Limits) *PriorityTable {
	return &PriorityTable{
		samples: map[string]*sample.Sample{},
		schema:  schema,
		limits:  limits,
	}
}

func (p *PriorityTable) Add(s *sample.Sample) error {
	return addSampleToTable(s, p.samples)
}

func addSampleToTable(s *sample.Sample, table map[string]*sample.Sample) error {
	sampleKey := s.Key().AsStringKey()
	existingSample, ok := table[sampleKey]
	if ok {
		return existingSample.AddSample(s)
	}
	table[sampleKey] = s
	return nil
}

func (p *PriorityTable) CollapseSamples() error {
	if !p.shouldCollapseSamples() {
		return nil
	}

	log.Debugf("Collapsing priority table. Size of table before collapsing: %d", len(p.samples))

	collapseLevel := p.limits.GetCurrentCollapsingLevel(len(p.samples))
	newSamples := make(map[string]*sample.Sample, len(p.samples))
	priorityQueue := sampleQueue{}
	for _, sample := range p.samples {
		switch {
		case priorityQueue.Len() < p.limits.Threshold():
			heap.Push(&priorityQueue, sample)
			err := addSampleToTable(sample, newSamples)
			if err != nil {
				return err
			}
		case sample.HitCount() < priorityQueue.Peek().HitCount():
			p.collapseSample(sample, collapseLevel)
			err := addSampleToTable(sample, newSamples)
			if err != nil {
				return err
			}
		default:
			sampleToCollapse := priorityQueue.Peek()
			delete(newSamples, sampleToCollapse.Key().AsStringKey())
			p.collapseSample(sampleToCollapse, collapseLevel)
			err := addSampleToTable(sampleToCollapse, newSamples)
			if err != nil {
				return err
			}

			err = addSampleToTable(sample, newSamples)
			if err != nil {
				return err
			}
			priorityQueue.ReplaceTop(sample)
			heap.Fix(&priorityQueue, 0)
		}

	}
	p.samples = newSamples
	log.Debugf("Collapsing priority table. Size of table after collapsing: %d", len(p.samples))

	return nil
}

func (p *PriorityTable) shouldCollapseSamples() bool {
	return len(p.samples) > p.limits.Threshold()
}

func (p *PriorityTable) collapseSample(sample *sample.Sample, currentCollapseLevel limits.CollapsingLevel) {
	for _, dim := range p.schema.Dimensions() {
		if dim.ShouldCollapse(currentCollapseLevel) {
			sample.Key().SetKeyPart(lookup.LookupAggrCode, dim.KeyBitSize, dim.KeyBitPositionInCompoudKey)
		}
	}
}

func (p *PriorityTable) Samples() map[string]*sample.Sample {
	return p.samples
}
