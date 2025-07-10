// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package logsgzipprocessor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// nolint: ireturn
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("logsgzip"),
		func() component.Config {
			return &struct{}{}
		},
		processor.WithLogs(createLogsGzipProcessor, component.StabilityLevelBeta),
	)
}

// nolint: ireturn
func createLogsGzipProcessor(_ context.Context,
	settings processor.Settings,
	cfg component.Config,
	logs consumer.Logs,
) (processor.Logs, error) {
	logger := settings.Logger
	logger.Info("Creating logs gzip processor")

	return newLogsGzipProcessor(logs, settings), nil
}

// logsGzipProcessor is a custom-processor implementation for compressing individual log records into
// gzip format. This can be used to reduce the size of log records and improve performance when processing
// large log volumes. This processor will be used by default for agent interacting with NGINX One
// console (https://docs.nginx.com/nginx-one/about/).
type logsGzipProcessor struct {
	nextConsumer consumer.Logs
	// We use sync.Pool to efficiently manage and reuse gzip.Writer instances within this processor.
	// Otherwise, creating a new compressor for every log record would result in frequent memory allocations
	// and increased garbage collection overhead, especially under high-throughput workload like this one.
	// By pooling these objects, we minimize allocation churn, reduce GC pressure, and improve overall performance.
	pool     *sync.Pool
	settings processor.Settings
}

type GzipWriter interface {
	Write(p []byte) (int, error)
	Close() error
	Reset(w io.Writer)
}

func newLogsGzipProcessor(logs consumer.Logs, settings processor.Settings) *logsGzipProcessor {
	return &logsGzipProcessor{
		nextConsumer: logs,
		pool: &sync.Pool{
			New: func() any {
				return gzip.NewWriter(nil)
			},
		},
		settings: settings,
	}
}

func (p *logsGzipProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	var errs error
	resourceLogs := ld.ResourceLogs()
	for i := range resourceLogs.Len() {
		scopeLogs := resourceLogs.At(i).ScopeLogs()
		for j := range scopeLogs.Len() {
			err := p.processLogRecords(scopeLogs.At(j).LogRecords())
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}
	if errs != nil {
		return fmt.Errorf("failed processing log records: %w", errs)
	}

	return p.nextConsumer.ConsumeLogs(ctx, ld)
}

func (p *logsGzipProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{
		MutatesData: true,
	}
}

func (p *logsGzipProcessor) Start(ctx context.Context, _ component.Host) error {
	p.settings.Logger.Info("Starting logs gzip processor")
	return nil
}

func (p *logsGzipProcessor) Shutdown(ctx context.Context) error {
	p.settings.Logger.Info("Shutting down logs gzip processor")
	return nil
}

func (p *logsGzipProcessor) processLogRecords(logRecords plog.LogRecordSlice) error {
	var errs error
	// Filter out unsupported data types in the log before processing
	logRecords.RemoveIf(func(lr plog.LogRecord) bool {
		body := lr.Body()
		// Keep only STRING or BYTES types
		if body.Type() != pcommon.ValueTypeStr &&
			body.Type() != pcommon.ValueTypeBytes {
			p.settings.Logger.Debug("Skipping log record with unsupported body type", zap.Any("type", body.Type()))
			return true
		}

		return false
	})
	// Process remaining valid records
	for k := range logRecords.Len() {
		record := logRecords.At(k)
		body := record.Body()
		var data []byte
		//nolint:exhaustive // Already filtered out other types with RemoveIf
		switch body.Type() {
		case pcommon.ValueTypeStr:
			data = []byte(body.Str())
		case pcommon.ValueTypeBytes:
			data = body.Bytes().AsRaw()
		}
		gzipped, err := p.gzipCompress(data)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to compress log record: %w", err))

			continue
		}
		err = record.Body().FromRaw(gzipped)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to set gzipped data to log record body: %w", err))

			continue
		}
	}

	return errs
}

func (p *logsGzipProcessor) gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	wIface := p.pool.Get()
	w, ok := wIface.(GzipWriter)
	if !ok {
		return nil, fmt.Errorf("writer of type %T not supported", wIface)
	}
	w.Reset(&buf)
	defer func() {
		if err = w.Close(); err != nil {
			p.settings.Logger.Error("Failed to close gzip writer", zap.Error(err))
		}
		p.pool.Put(w)
	}()

	_, err = w.Write(data)
	if err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
