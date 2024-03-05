// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package model

import (
	"testing"

	"github.com/nginx/agent/v3/internal/bus"

	"github.com/stretchr/testify/assert"
)

func TestEntryBuilder(t *testing.T) {
	expected := DataEntry{
		Name:        "test-name",
		Description: "test-description",
		Type:        Gauge,
		SourceType:  Prometheus,
		Values: []DataPoint{
			{
				Name: "test-point",
				Labels: map[string]string{
					"test-label-name": "test-label-value",
				},
				Value: 32.0,
			},
		},
	}

	eb := NewEntryBuilder(WithSourceType(UnknownSourceType))
	assert.NotNil(t, eb)
	assert.False(t, eb.CanBuild())
	assert.Equal(t, UnknownSourceType, eb.entry.SourceType)

	eb.WithName("test-name")
	assert.Equal(t, expected.Name, eb.entry.Name)
	assert.False(t, eb.CanBuild())

	eb.WithDescription("test-description")
	assert.Equal(t, expected.Description, eb.entry.Description)
	assert.False(t, eb.CanBuild())

	eb.WithType(Gauge)
	assert.Equal(t, expected.Type, eb.entry.Type)
	assert.False(t, eb.CanBuild())

	eb.WithSourceType(Prometheus)
	assert.Equal(t, expected.SourceType, eb.entry.SourceType)
	assert.True(t, eb.CanBuild())

	eb.WithValues(DataPoint{
		Name: "test-point",
		Labels: map[string]string{
			"test-label-name": "test-label-value",
		},
		Value: 32.0,
	})
	assert.Equal(t, expected.Values, eb.entry.Values)
	assert.True(t, eb.CanBuild())

	actual := eb.Build()
	assert.NotNil(t, actual)
	assert.Equal(t, expected, actual)

	busMsg := actual.ToBusMessage()
	assert.NotNil(t, busMsg)
	assert.Equal(t, bus.MetricsTopic, busMsg.Topic)
	assert.Equal(t, expected, busMsg.Data)
}

func TestMetricsSourceType(t *testing.T) {
	mst := Prometheus
	assert.Equal(t, "Prometheus", mst.String())

	mst = 90
	assert.Equal(t, "Unknown", mst.String())

	mst = UnknownSourceType
	assert.Equal(t, "Unknown", mst.String())

	mst = ToMetricsSourceType("ProMeTHeUs")
	assert.Equal(t, Prometheus, mst)

	mst = ToMetricsSourceType("not-a-valid-type")
	assert.Equal(t, UnknownSourceType, mst)
}

func TestInstrumentType(t *testing.T) {
	it := Counter
	assert.Equal(t, "counter", it.String())

	it = Gauge
	assert.Equal(t, "gauge", it.String())

	it = Histogram
	assert.Equal(t, "histogram", it.String())

	it = Summary
	assert.Equal(t, "summary", it.String())

	it = UnknownInstrument
	assert.Equal(t, "unknown", it.String())

	it = 90
	assert.Equal(t, "unknown", it.String())

	it = ToInstrumentType("coUNter")
	assert.Equal(t, Counter, it)

	it = ToInstrumentType("GAuge")
	assert.Equal(t, Gauge, it)

	it = ToInstrumentType("HISToGRAM")
	assert.Equal(t, Histogram, it)

	it = ToInstrumentType("SUMmary")
	assert.Equal(t, Summary, it)

	it = ToInstrumentType("not-a-valid-type")
	assert.Equal(t, UnknownInstrument, it)
}

func TestExporterType(t *testing.T) {
	et := OTel
	assert.Equal(t, "otel", et.String())

	et = UnknownExporter
	assert.Equal(t, "unknown", et.String())

	et = 90
	assert.Equal(t, "unknown", et.String())

	et = ToExporterType("oTEl")
	assert.Equal(t, OTel, et)

	et = ToExporterType("unknOWn")
	assert.Equal(t, UnknownExporter, et)

	et = ToExporterType("not-a-valid-exporter-type")
	assert.Equal(t, UnknownExporter, et)
}
