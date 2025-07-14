// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package logsgzipprocessor

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
)

// Helper to generate logs with variable size and content
func generateLogs(numRecords, recordSize int) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	for range numRecords {
		lr := sl.LogRecords().AppendEmpty()
		content, _ := randomString(recordSize)
		lr.Body().SetStr(content)
	}

	return logs
}

func randomString(n int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	lettersSize := big.NewInt(int64(len(letters)))
	for i := range b {
		num, err := rand.Int(rand.Reader, lettersSize)
		if err != nil {
			return "", err
		}
		b[i] = letters[num.Int64()]
	}

	return string(b), nil
}

func BenchmarkGzipProcessor(b *testing.B) {
	benchmarks := []struct {
		name       string
		numRecords int
		recordSize int
	}{
		{"SmallRecords", 100, 50},
		{"MediumRecords", 100, 500},
		{"LargeRecords", 100, 5000},
		{"ManySmallRecords", 10000, 50},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			consumer := &consumertest.LogsSink{}
			p := newLogsGzipProcessor(consumer, processor.Settings{})
			logs := generateLogs(bm.numRecords, bm.recordSize)

			b.ResetTimer()
			for range b.N {
				_ = p.ConsumeLogs(context.Background(), logs)
			}
		})
	}
}

// Optional: Benchmark with concurrency to simulate real pipeline load
func BenchmarkGzipProcessor_Concurrent(b *testing.B) {
	// nolint:unused // concurrent runs require total parallel workers to be specified
	const workers = 8
	logs := generateLogs(1000, 1000)
	consumer := &consumertest.LogsSink{}
	p := newLogsGzipProcessor(consumer, processor.Settings{})

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = p.ConsumeLogs(context.Background(), logs)
		}
	})
}
