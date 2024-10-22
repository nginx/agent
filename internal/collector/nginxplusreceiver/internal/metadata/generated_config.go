// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/filter"
)

// MetricConfig provides common config for a particular metric.
type MetricConfig struct {
	Enabled bool `mapstructure:"enabled"`

	enabledSetByUser bool
}

func (ms *MetricConfig) Unmarshal(parser *confmap.Conf) error {
	if parser == nil {
		return nil
	}
	err := parser.Unmarshal(ms)
	if err != nil {
		return err
	}
	ms.enabledSetByUser = parser.IsSet("enabled")
	return nil
}

// MetricsConfig provides config for nginxplus metrics.
type MetricsConfig struct {
	NginxCacheBytes                      MetricConfig `mapstructure:"nginx.cache.bytes"`
	NginxCacheMemoryLimit                MetricConfig `mapstructure:"nginx.cache.memory.limit"`
	NginxCacheMemoryUsage                MetricConfig `mapstructure:"nginx.cache.memory.usage"`
	NginxCacheResponses                  MetricConfig `mapstructure:"nginx.cache.responses"`
	NginxConfigReloads                   MetricConfig `mapstructure:"nginx.config.reloads"`
	NginxHTTPConn                        MetricConfig `mapstructure:"nginx.http.conn"`
	NginxHTTPConnCount                   MetricConfig `mapstructure:"nginx.http.conn.count"`
	NginxHTTPLimitConnRequests           MetricConfig `mapstructure:"nginx.http.limit_conn.requests"`
	NginxHTTPLimitReqRequests            MetricConfig `mapstructure:"nginx.http.limit_req.requests"`
	NginxHTTPRequestByteIo               MetricConfig `mapstructure:"nginx.http.request.byte.io"`
	NginxHTTPRequestDiscarded            MetricConfig `mapstructure:"nginx.http.request.discarded"`
	NginxHTTPRequestProcessingCount      MetricConfig `mapstructure:"nginx.http.request.processing.count"`
	NginxHTTPRequests                    MetricConfig `mapstructure:"nginx.http.requests"`
	NginxHTTPRequestsCount               MetricConfig `mapstructure:"nginx.http.requests.count"`
	NginxHTTPResponseStatus              MetricConfig `mapstructure:"nginx.http.response.status"`
	NginxHTTPResponses                   MetricConfig `mapstructure:"nginx.http.responses"`
	NginxHTTPUpstreamKeepaliveCount      MetricConfig `mapstructure:"nginx.http.upstream.keepalive.count"`
	NginxHTTPUpstreamPeerByteIo          MetricConfig `mapstructure:"nginx.http.upstream.peer.byte.io"`
	NginxHTTPUpstreamPeerConnCount       MetricConfig `mapstructure:"nginx.http.upstream.peer.conn.count"`
	NginxHTTPUpstreamPeerCount           MetricConfig `mapstructure:"nginx.http.upstream.peer.count"`
	NginxHTTPUpstreamPeerFails           MetricConfig `mapstructure:"nginx.http.upstream.peer.fails"`
	NginxHTTPUpstreamPeerHeaderTime      MetricConfig `mapstructure:"nginx.http.upstream.peer.header.time"`
	NginxHTTPUpstreamPeerHealthChecks    MetricConfig `mapstructure:"nginx.http.upstream.peer.health_checks"`
	NginxHTTPUpstreamPeerRequests        MetricConfig `mapstructure:"nginx.http.upstream.peer.requests"`
	NginxHTTPUpstreamPeerResponseTime    MetricConfig `mapstructure:"nginx.http.upstream.peer.response.time"`
	NginxHTTPUpstreamPeerResponses       MetricConfig `mapstructure:"nginx.http.upstream.peer.responses"`
	NginxHTTPUpstreamPeerState           MetricConfig `mapstructure:"nginx.http.upstream.peer.state"`
	NginxHTTPUpstreamPeerUnavailables    MetricConfig `mapstructure:"nginx.http.upstream.peer.unavailables"`
	NginxHTTPUpstreamQueueLimit          MetricConfig `mapstructure:"nginx.http.upstream.queue.limit"`
	NginxHTTPUpstreamQueueOverflows      MetricConfig `mapstructure:"nginx.http.upstream.queue.overflows"`
	NginxHTTPUpstreamQueueUsage          MetricConfig `mapstructure:"nginx.http.upstream.queue.usage"`
	NginxHTTPUpstreamZombieCount         MetricConfig `mapstructure:"nginx.http.upstream.zombie.count"`
	NginxSlabPageFree                    MetricConfig `mapstructure:"nginx.slab.page.free"`
	NginxSlabPageLimit                   MetricConfig `mapstructure:"nginx.slab.page.limit"`
	NginxSlabPageUsage                   MetricConfig `mapstructure:"nginx.slab.page.usage"`
	NginxSlabPageUtilization             MetricConfig `mapstructure:"nginx.slab.page.utilization"`
	NginxSlabSlotAllocations             MetricConfig `mapstructure:"nginx.slab.slot.allocations"`
	NginxSlabSlotFree                    MetricConfig `mapstructure:"nginx.slab.slot.free"`
	NginxSlabSlotUsage                   MetricConfig `mapstructure:"nginx.slab.slot.usage"`
	NginxSslCertificateVerifyFailures    MetricConfig `mapstructure:"nginx.ssl.certificate.verify_failures"`
	NginxSslHandshakes                   MetricConfig `mapstructure:"nginx.ssl.handshakes"`
	NginxStreamByteIo                    MetricConfig `mapstructure:"nginx.stream.byte.io"`
	NginxStreamConnectionAccepted        MetricConfig `mapstructure:"nginx.stream.connection.accepted"`
	NginxStreamConnectionDiscarded       MetricConfig `mapstructure:"nginx.stream.connection.discarded"`
	NginxStreamConnectionProcessingCount MetricConfig `mapstructure:"nginx.stream.connection.processing.count"`
	NginxStreamSessionStatus             MetricConfig `mapstructure:"nginx.stream.session.status"`
	NginxStreamUpstreamPeerByteIo        MetricConfig `mapstructure:"nginx.stream.upstream.peer.byte.io"`
	NginxStreamUpstreamPeerConnCount     MetricConfig `mapstructure:"nginx.stream.upstream.peer.conn.count"`
	NginxStreamUpstreamPeerConnTime      MetricConfig `mapstructure:"nginx.stream.upstream.peer.conn.time"`
	NginxStreamUpstreamPeerConns         MetricConfig `mapstructure:"nginx.stream.upstream.peer.conns"`
	NginxStreamUpstreamPeerCount         MetricConfig `mapstructure:"nginx.stream.upstream.peer.count"`
	NginxStreamUpstreamPeerFails         MetricConfig `mapstructure:"nginx.stream.upstream.peer.fails"`
	NginxStreamUpstreamPeerHealthChecks  MetricConfig `mapstructure:"nginx.stream.upstream.peer.health_checks"`
	NginxStreamUpstreamPeerResponseTime  MetricConfig `mapstructure:"nginx.stream.upstream.peer.response.time"`
	NginxStreamUpstreamPeerState         MetricConfig `mapstructure:"nginx.stream.upstream.peer.state"`
	NginxStreamUpstreamPeerTtfbTime      MetricConfig `mapstructure:"nginx.stream.upstream.peer.ttfb.time"`
	NginxStreamUpstreamPeerUnavailable   MetricConfig `mapstructure:"nginx.stream.upstream.peer.unavailable"`
	NginxStreamUpstreamZombieCount       MetricConfig `mapstructure:"nginx.stream.upstream.zombie.count"`
}

func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		NginxCacheBytes: MetricConfig{
			Enabled: true,
		},
		NginxCacheMemoryLimit: MetricConfig{
			Enabled: true,
		},
		NginxCacheMemoryUsage: MetricConfig{
			Enabled: true,
		},
		NginxCacheResponses: MetricConfig{
			Enabled: true,
		},
		NginxConfigReloads: MetricConfig{
			Enabled: true,
		},
		NginxHTTPConn: MetricConfig{
			Enabled: true,
		},
		NginxHTTPConnCount: MetricConfig{
			Enabled: true,
		},
		NginxHTTPLimitConnRequests: MetricConfig{
			Enabled: true,
		},
		NginxHTTPLimitReqRequests: MetricConfig{
			Enabled: true,
		},
		NginxHTTPRequestByteIo: MetricConfig{
			Enabled: true,
		},
		NginxHTTPRequestDiscarded: MetricConfig{
			Enabled: true,
		},
		NginxHTTPRequestProcessingCount: MetricConfig{
			Enabled: true,
		},
		NginxHTTPRequests: MetricConfig{
			Enabled: true,
		},
		NginxHTTPRequestsCount: MetricConfig{
			Enabled: true,
		},
		NginxHTTPResponseStatus: MetricConfig{
			Enabled: true,
		},
		NginxHTTPResponses: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamKeepaliveCount: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerByteIo: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerConnCount: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerCount: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerFails: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerHeaderTime: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerHealthChecks: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerRequests: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerResponseTime: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerResponses: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerState: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamPeerUnavailables: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamQueueLimit: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamQueueOverflows: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamQueueUsage: MetricConfig{
			Enabled: true,
		},
		NginxHTTPUpstreamZombieCount: MetricConfig{
			Enabled: true,
		},
		NginxSlabPageFree: MetricConfig{
			Enabled: true,
		},
		NginxSlabPageLimit: MetricConfig{
			Enabled: true,
		},
		NginxSlabPageUsage: MetricConfig{
			Enabled: true,
		},
		NginxSlabPageUtilization: MetricConfig{
			Enabled: true,
		},
		NginxSlabSlotAllocations: MetricConfig{
			Enabled: true,
		},
		NginxSlabSlotFree: MetricConfig{
			Enabled: true,
		},
		NginxSlabSlotUsage: MetricConfig{
			Enabled: true,
		},
		NginxSslCertificateVerifyFailures: MetricConfig{
			Enabled: true,
		},
		NginxSslHandshakes: MetricConfig{
			Enabled: true,
		},
		NginxStreamByteIo: MetricConfig{
			Enabled: true,
		},
		NginxStreamConnectionAccepted: MetricConfig{
			Enabled: true,
		},
		NginxStreamConnectionDiscarded: MetricConfig{
			Enabled: true,
		},
		NginxStreamConnectionProcessingCount: MetricConfig{
			Enabled: true,
		},
		NginxStreamSessionStatus: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerByteIo: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerConnCount: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerConnTime: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerConns: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerCount: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerFails: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerHealthChecks: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerResponseTime: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerState: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerTtfbTime: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamPeerUnavailable: MetricConfig{
			Enabled: true,
		},
		NginxStreamUpstreamZombieCount: MetricConfig{
			Enabled: true,
		},
	}
}

// ResourceAttributeConfig provides common config for a particular resource attribute.
type ResourceAttributeConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Experimental: MetricsInclude defines a list of filters for attribute values.
	// If the list is not empty, only metrics with matching resource attribute values will be emitted.
	MetricsInclude []filter.Config `mapstructure:"metrics_include"`
	// Experimental: MetricsExclude defines a list of filters for attribute values.
	// If the list is not empty, metrics with matching resource attribute values will not be emitted.
	// MetricsInclude has higher priority than MetricsExclude.
	MetricsExclude []filter.Config `mapstructure:"metrics_exclude"`

	enabledSetByUser bool
}

func (rac *ResourceAttributeConfig) Unmarshal(parser *confmap.Conf) error {
	if parser == nil {
		return nil
	}
	err := parser.Unmarshal(rac)
	if err != nil {
		return err
	}
	rac.enabledSetByUser = parser.IsSet("enabled")
	return nil
}

// ResourceAttributesConfig provides config for nginxplus resource attributes.
type ResourceAttributesConfig struct {
	NginxInstanceID   ResourceAttributeConfig `mapstructure:"nginx.instance.id"`
	NginxInstanceType ResourceAttributeConfig `mapstructure:"nginx.instance.type"`
}

func DefaultResourceAttributesConfig() ResourceAttributesConfig {
	return ResourceAttributesConfig{
		NginxInstanceID: ResourceAttributeConfig{
			Enabled: true,
		},
		NginxInstanceType: ResourceAttributeConfig{
			Enabled: true,
		},
	}
}

// MetricsBuilderConfig is a configuration for nginxplus metrics builder.
type MetricsBuilderConfig struct {
	Metrics            MetricsConfig            `mapstructure:"metrics"`
	ResourceAttributes ResourceAttributesConfig `mapstructure:"resource_attributes"`
}

func DefaultMetricsBuilderConfig() MetricsBuilderConfig {
	return MetricsBuilderConfig{
		Metrics:            DefaultMetricsConfig(),
		ResourceAttributes: DefaultResourceAttributesConfig(),
	}
}
