// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package model

type (
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
)
