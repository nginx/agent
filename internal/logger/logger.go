// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/internal/config"
)

const (
	defaultLogFile = "agent.log"
	filePermission = 0o600

	CorrelationIDKey = "correlation_id"
)

var (
	logLevels = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	CorrelationIDContextKey = contextKey(CorrelationIDKey)
)

type (
	contextKey string

	contextHandler struct {
		slog.Handler
		keys []any
	}
)

func New(params config.Log) *slog.Logger {
	handler := slog.NewTextHandler(
		getLogWriter(params.Path),
		&slog.HandlerOptions{
			Level: GetLogLevel(params.Level),
		},
	)

	return slog.New(
		contextHandler{
			handler, []any{
				CorrelationIDContextKey,
			},
		})
}

func GetLogLevel(level string) slog.Level {
	if level == "" {
		return slog.LevelInfo
	}

	return logLevels[strings.ToLower(level)]
}

func getLogWriter(logFile string) io.Writer {
	logPath := logFile
	if logFile != "" {
		fileInfo, err := os.Stat(logPath)
		if err != nil {
			slog.Error("Error reading log directory, proceeding to log only to stdout/stderr", "error", err)

			return os.Stderr
		}

		if fileInfo.IsDir() {
			logPath = path.Join(logPath, defaultLogFile)
		}

		logFileHandle, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, filePermission)
		if err != nil {
			slog.Error("Failed to open log file, proceeding to log only to stdout/stderr", "error", err)

			return os.Stderr
		}

		return logFileHandle
	}

	return os.Stderr
}

func (c contextKey) String() string {
	return string(c)
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(h.observe(ctx)...)
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) observe(ctx context.Context) (as []slog.Attr) {
	for _, k := range h.keys {
		a, ok := ctx.Value(k).(slog.Attr)
		if !ok {
			continue
		}
		a.Value = a.Value.Resolve()
		as = append(as, a)
	}

	return as
}

func GenerateCorrelationID() slog.Attr {
	return slog.Any(CorrelationIDKey, uuid.NewString())
}

func GetCorrelationID(ctx context.Context) string {
	value, ok := ctx.Value(CorrelationIDContextKey).(slog.Attr)
	if !ok {
		slog.Debug("Correlation ID not found in context")
		return ""
	}

	return value.Value.String()
}
