// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/nxadm/tail"
	"github.com/trivago/grok"
)

const numberOfTimesToSplitLogLine = 2

var tailConfig = tail.Config{
	Follow:    true,
	ReOpen:    true,
	MustExist: true,
	Poll:      true,
	Location: &tail.SeekInfo{
		Whence: io.SeekEnd,
	},
}

type (
	// NginxAccessItem represents the decoded access log data
	NginxAccessItem struct {
		BodyBytesSent          string `mapstructure:"body_bytes_sent"`
		Status                 string `mapstructure:"status"`
		RemoteAddress          string `mapstructure:"remote_addr"`
		HTTPUserAgent          string `mapstructure:"http_user_agent"`
		Request                string `mapstructure:"request"`
		BytesSent              string `mapstructure:"bytes_sent"`
		RequestLength          string `mapstructure:"request_length"`
		RequestTime            string `mapstructure:"request_time"`
		GzipRatio              string `mapstructure:"gzip_ratio"`
		ServerProtocol         string `mapstructure:"server_protocol"`
		UpstreamConnectTime    string `mapstructure:"upstream_connect_time"`
		UpstreamHeaderTime     string `mapstructure:"upstream_header_time"`
		UpstreamResponseTime   string `mapstructure:"upstream_response_time"`
		UpstreamResponseLength string `mapstructure:"upstream_response_length"`
		UpstreamStatus         string `mapstructure:"upstream_status"`
		UpstreamCacheStatus    string `mapstructure:"upstream_cache_status"`
	}

	Tailer struct {
		handle *tail.Tail
	}

	PatternTailer struct {
		handle *tail.Tail
		gc     *grok.CompiledGrok
	}

	// LTSV (Labeled Tab-separated Values) Tailer
	LTSVTailer struct {
		handle *tail.Tail
	}
)

func NewNginxAccessItem(v map[string]string) (*NginxAccessItem, error) {
	res := &NginxAccessItem{}
	if err := mapstructure.Decode(v, res); err != nil {
		return nil, err
	}

	return res, nil
}

func NewTailer(file string) (*Tailer, error) {
	t, err := tail.TailFile(file, tailConfig)
	if err != nil {
		return nil, err
	}

	return &Tailer{t}, nil
}

func NewPatternTailer(file string, patterns map[string]string) (*PatternTailer, error) {
	g, err := grok.New(grok.Config{
		NamedCapturesOnly: false,
		Patterns:          patterns,
	})
	if err != nil {
		return nil, err
	}
	gc, err := g.Compile("%{DEFAULT}")
	if err != nil {
		return nil, err
	}
	t, err := tail.TailFile(file, tailConfig)
	if err != nil {
		return nil, err
	}

	return &PatternTailer{t, gc}, nil
}

func NewLTSVTailer(file string) (*LTSVTailer, error) {
	t, err := tail.TailFile(file, tailConfig)
	if err != nil {
		return nil, err
	}

	return &LTSVTailer{t}, nil
}

func (t *Tailer) Tail(ctx context.Context, data chan<- string) {
	for {
		select {
		case line := <-t.handle.Lines:
			lineContent := t.parseLine(line)
			if lineContent != "" {
				data <- lineContent
			}
		case <-ctx.Done():
			handleContextDone(ctx)

			return
		}
	}
}

func (t *Tailer) parseLine(line *tail.Line) string {
	if line == nil {
		return ""
	}

	if line.Err != nil {
		return ""
	}

	return line.Text
}

func (t *PatternTailer) Tail(ctx context.Context, data chan<- map[string]string) {
	for {
		select {
		case line := <-t.handle.Lines:
			lineContent := t.parseLine(line)
			if lineContent != nil {
				data <- lineContent
			}
		case <-ctx.Done():
			handleContextDone(ctx)

			return
		}
	}
}

func (t *PatternTailer) parseLine(line *tail.Line) map[string]string {
	if line == nil {
		return nil
	}

	if line.Err != nil {
		return nil
	}

	return t.gc.ParseString(line.Text)
}

func (t *LTSVTailer) Tail(ctx context.Context, data chan<- map[string]string) {
	for {
		select {
		case line := <-t.handle.Lines:
			lineText := t.parseLine(line)
			if lineText != nil {
				data <- lineText
			}
		case <-ctx.Done():
			handleContextDone(ctx)

			return
		}
	}
}

func (t *LTSVTailer) parse(line string) map[string]string {
	columns := strings.Split(line, "\t")
	lineMap := make(map[string]string)

	for _, column := range columns {
		labelValue := strings.SplitN(column, ":", numberOfTimesToSplitLogLine)
		if len(labelValue) < numberOfTimesToSplitLogLine {
			continue
		}
		label, value := strings.TrimSpace(labelValue[0]), strings.TrimSpace(labelValue[1])
		lineMap[label] = value
	}

	return lineMap
}

func (t *LTSVTailer) parseLine(line *tail.Line) map[string]string {
	if line == nil {
		return nil
	}

	if line.Err != nil {
		return nil
	}

	return t.parse(line.Text)
}

func handleContextDone(ctx context.Context) {
	ctxErr := ctx.Err()
	switch ctxErr {
	case context.DeadlineExceeded:
		slog.DebugContext(ctx, "Tailer canceled because deadline was exceeded", "error", ctxErr)
	case context.Canceled:
		slog.DebugContext(ctx, "Tailer forcibly canceled", "error", ctxErr)
	}
	slog.DebugContext(ctx, "Tailer is done")
}
