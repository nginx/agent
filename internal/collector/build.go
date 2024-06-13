// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"github.com/nginx/agent/v3/internal/config"
	"go.opentelemetry.io/collector/component"
)

// BuildInfo returns all the OTel collector BuildInfo supported
// based on https://github.com/DataDog/datadog-agent/blob/main/comp/otelcol/collector-contrib/impl/collectorcontrib.go
func BuildInfo(cfg *config.Config) component.BuildInfo {
	return component.BuildInfo{
		Version:     cfg.Version,
		Command:     "otel-nginx-agent",
		Description: "NGINX Agent OpenTelemetry Collector",
	}
}
