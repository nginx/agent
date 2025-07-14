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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nginx/agent/v3/pkg/id"
)

const (
	defaultLogFile = "agent.log"
	filePermission = 0o600

	CorrelationIDKey = "correlation_id"
	ServerTypeKey    = "server_type"
)

var (
	logLevels = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	CorrelationIDContextKey = contextKey(CorrelationIDKey)
	ServerTypeContextKey    = contextKey(ServerTypeKey)
)

type (
	contextKey string

	contextHandler struct {
		slog.Handler
		keys []any
	}
)

func New(logPath, level string) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{
		Level: LogLevel(level),
	}

	if level == "debug" {
		handlerOptions.AddSource = true
		handlerOptions.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source, ok := a.Value.Any().(*slog.Source)
				if ok {
					directory := filepath.Dir(source.File)
					relativePath := path.Join(filepath.Base(directory), filepath.Base(source.File))
					a.Value = slog.StringValue(relativePath + ":" + strconv.Itoa(source.Line))
				}
			}

			return a
		}
	}

	handler := slog.NewTextHandler(
		logWriter(logPath),
		handlerOptions,
	)

	return slog.New(
		contextHandler{
			handler, []any{
				CorrelationIDContextKey,
				ServerTypeContextKey,
			},
		})
}

func LogLevel(level string) slog.Level {
	if level == "" {
		return slog.LevelInfo
	}

	return logLevels[strings.ToLower(level)]
}

func logWriter(logFile string) io.Writer {
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

		// Use io.MultiWriter to log to both Stdout and the file
		multiWriter := io.MultiWriter(os.Stdout, logFileHandle)

		return multiWriter
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
	return slog.Any(CorrelationIDKey, id.GenerateMessageID())
}

func CorrelationID(ctx context.Context) string {
	return CorrelationIDAttr(ctx).Value.String()
}

func CorrelationIDAttr(ctx context.Context) slog.Attr {
	value, ok := ctx.Value(CorrelationIDContextKey).(slog.Attr)
	if !ok {
		correlationID := GenerateCorrelationID()
		slog.DebugContext(
			ctx,
			"Correlation ID not found in context, generating new correlation ID",
			correlationID)

		return GenerateCorrelationID()
	}

	return value
}

func ServerType(ctx context.Context) string {
	return ServerTypeAttr(ctx).Value.String()
}

func ServerTypeAttr(ctx context.Context) slog.Attr {
	value, ok := ctx.Value(ServerTypeContextKey).(slog.Attr)
	if !ok {
		return slog.Any(ServerTypeKey, "")
	}

	return value
}
