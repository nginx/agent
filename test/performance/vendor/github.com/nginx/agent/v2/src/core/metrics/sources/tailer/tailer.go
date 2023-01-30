/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package tailer

import (
	"context"
	"io"

	"github.com/mitchellh/mapstructure"
	"github.com/nxadm/tail"
	log "github.com/sirupsen/logrus"
	"github.com/trivago/grok"
)

var (
	tailConfig = tail.Config{
		Follow: true,
		ReOpen: true,
		Poll:   true,
		Location: &tail.SeekInfo{
			Whence: io.SeekEnd,
		},
	}
)

// NginxAccessItem represents the decoded access log data
type NginxAccessItem struct {
	BodyBytesSent       string `mapstructure:"body_bytes_sent"`
	Status              string `mapstructure:"status"`
	RemoteAddress       string `mapstructure:"remote_addr"`
	HTTPUserAgent       string `mapstructure:"http_user_agent"`
	Request             string `mapstructure:"request"`
	BytesSent           string `mapstructure:"bytes_sent"`
	RequestLength       string `mapstructure:"request_length"`
	RequestTime         string `mapstructure:"request_time"`
	GzipRatio           string `mapstructure:"gzip_ratio"`
	ServerProtocol      string `mapstructure:"server_protocol"`
	UpstreamConnectTime string `mapstructure:"upstream_connect_time"`
	UpstreamHeaderTime  string `mapstructure:"upstream_header_time"`
}

func NewNginxAccessItem(v map[string]string) (*NginxAccessItem, error) {
	res := &NginxAccessItem{}
	if err := mapstructure.Decode(v, res); err != nil {
		return nil, err
	}
	return res, nil
}

type Tailer struct {
	handle *tail.Tail
}

type PatternTailer struct {
	handle *tail.Tail
	gc     *grok.CompiledGrok
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

func (t *Tailer) Tail(ctx context.Context, data chan<- string) {
	for {
		select {
		case line := <-t.handle.Lines:
			if line == nil {
				return
			}
			if line.Err != nil {
				continue
			}

			data <- line.Text

		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				log.Errorf("error in done context Tail %v", err)
			}
			log.Info("tailer is done")
			return
		}
	}
}

func (t *PatternTailer) Tail(ctx context.Context, data chan<- map[string]string) {
	for {
		select {
		case line := <-t.handle.Lines:
			if line == nil {
				return
			}
			if line.Err != nil {
				continue
			}

			l := t.gc.ParseString(line.Text)
			if l != nil {
				data <- l
			}
		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				log.Errorf("error in done context Tail %v", err)
			}
			log.Info("tailer is done")
			return
		}
	}
}
