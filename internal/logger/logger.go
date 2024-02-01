/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package logger

import (
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/nginx/agent/v3/internal/config"
)

const (
	defaultLogFile = "agent.log"
)

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func New(params config.Log) *slog.Logger {
	handler := slog.NewTextHandler(
		getLogWriter(params.Path),
		&slog.HandlerOptions{
			Level: getLogLevel(params.Level),
		},
	)

	return slog.New(handler)
}

func getLogLevel(level string) slog.Level {
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

		logFileHandle, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			slog.Error("Failed to open log file, proceeding to log only to stdout/stderr", "error", err)
			return os.Stderr
		}
		return logFileHandle
	}
	return os.Stderr
}
