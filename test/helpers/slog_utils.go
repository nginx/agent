// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"context"
	"log/slog"
	"sync"
)

// LogEntry represents a single log entry.
type LogEntry struct {
	Message string      // string, 16 bytes
	Attrs   []slog.Attr // slice, 24 bytes
	Level   slog.Level  // 8 bytes
}

// TestLogHandler is a custom slog.Handler that captures log entries.
type TestLogHandler struct {
	entries []LogEntry
	mu      sync.Mutex
}

// NewTestLogHandler creates a new TestLogHandler instance.
func NewTestLogHandler() *TestLogHandler {
	return &TestLogHandler{}
}

// Handle captures the log record.
func (h *TestLogHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	h.entries = append(h.entries, LogEntry{
		Level:   record.Level,
		Message: record.Message,
		Attrs:   attrs,
	})

	return nil
}

// Logs returns the captured log entries.
func (h *TestLogHandler) Logs() []LogEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]LogEntry(nil), h.entries...)
}

// Enabled implements the slog.Handler interface (always enabled).
func (h *TestLogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

// WithAttrs implements the slog.Handler interface (no-op for testing).
func (h *TestLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup implements the slog.Handler interface (no-op for testing).
func (h *TestLogHandler) WithGroup(name string) slog.Handler {
	return h
}
