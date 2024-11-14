// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestMetricsBuilderConfig(t *testing.T) {
	tests := []struct {
		name string
		want MetricsBuilderConfig
	}{
		{
			name: "default",
			want: DefaultMetricsBuilderConfig(),
		},
		{
			name: "all_set",
			want: MetricsBuilderConfig{
				Metrics: MetricsConfig{
					NginxCacheBytes:                         MetricConfig{Enabled: true},
					NginxCacheMemoryLimit:                   MetricConfig{Enabled: true},
					NginxCacheMemoryUsage:                   MetricConfig{Enabled: true},
					NginxCacheResponses:                     MetricConfig{Enabled: true},
					NginxConfigReloads:                      MetricConfig{Enabled: true},
					NginxHTTPConnections:                    MetricConfig{Enabled: true},
					NginxHTTPConnectionsCount:               MetricConfig{Enabled: true},
					NginxHTTPLimitConnRequests:              MetricConfig{Enabled: true},
					NginxHTTPLimitReqRequests:               MetricConfig{Enabled: true},
					NginxHTTPRequestByteIo:                  MetricConfig{Enabled: true},
					NginxHTTPRequestDiscarded:               MetricConfig{Enabled: true},
					NginxHTTPRequestProcessingCount:         MetricConfig{Enabled: true},
					NginxHTTPRequests:                       MetricConfig{Enabled: true},
					NginxHTTPRequestsCount:                  MetricConfig{Enabled: true},
					NginxHTTPResponseStatus:                 MetricConfig{Enabled: true},
					NginxHTTPResponses:                      MetricConfig{Enabled: true},
					NginxHTTPUpstreamKeepaliveCount:         MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerByteIo:             MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerConnectionsCount:   MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerCount:              MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerFails:              MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerHeaderTime:         MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerHealthChecks:       MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerRequests:           MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerResponseTime:       MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerResponses:          MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerState:              MetricConfig{Enabled: true},
					NginxHTTPUpstreamPeerUnavailables:       MetricConfig{Enabled: true},
					NginxHTTPUpstreamQueueLimit:             MetricConfig{Enabled: true},
					NginxHTTPUpstreamQueueOverflows:         MetricConfig{Enabled: true},
					NginxHTTPUpstreamQueueUsage:             MetricConfig{Enabled: true},
					NginxHTTPUpstreamZombieCount:            MetricConfig{Enabled: true},
					NginxSlabPageFree:                       MetricConfig{Enabled: true},
					NginxSlabPageLimit:                      MetricConfig{Enabled: true},
					NginxSlabPageUsage:                      MetricConfig{Enabled: true},
					NginxSlabPageUtilization:                MetricConfig{Enabled: true},
					NginxSlabSlotAllocations:                MetricConfig{Enabled: true},
					NginxSlabSlotFree:                       MetricConfig{Enabled: true},
					NginxSlabSlotUsage:                      MetricConfig{Enabled: true},
					NginxSslCertificateVerifyFailures:       MetricConfig{Enabled: true},
					NginxSslHandshakes:                      MetricConfig{Enabled: true},
					NginxStreamByteIo:                       MetricConfig{Enabled: true},
					NginxStreamConnectionsAccepted:          MetricConfig{Enabled: true},
					NginxStreamConnectionsDiscarded:         MetricConfig{Enabled: true},
					NginxStreamConnectionsProcessingCount:   MetricConfig{Enabled: true},
					NginxStreamSessionStatus:                MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerByteIo:           MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerConnections:      MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerConnectionsCount: MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerConnectionsTime:  MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerCount:            MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerFails:            MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerHealthChecks:     MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerResponseTime:     MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerState:            MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerTtfbTime:         MetricConfig{Enabled: true},
					NginxStreamUpstreamPeerUnavailable:      MetricConfig{Enabled: true},
					NginxStreamUpstreamZombieCount:          MetricConfig{Enabled: true},
				},
				ResourceAttributes: ResourceAttributesConfig{
					InstanceID:   ResourceAttributeConfig{Enabled: true},
					InstanceType: ResourceAttributeConfig{Enabled: true},
				},
			},
		},
		{
			name: "none_set",
			want: MetricsBuilderConfig{
				Metrics: MetricsConfig{
					NginxCacheBytes:                         MetricConfig{Enabled: false},
					NginxCacheMemoryLimit:                   MetricConfig{Enabled: false},
					NginxCacheMemoryUsage:                   MetricConfig{Enabled: false},
					NginxCacheResponses:                     MetricConfig{Enabled: false},
					NginxConfigReloads:                      MetricConfig{Enabled: false},
					NginxHTTPConnections:                    MetricConfig{Enabled: false},
					NginxHTTPConnectionsCount:               MetricConfig{Enabled: false},
					NginxHTTPLimitConnRequests:              MetricConfig{Enabled: false},
					NginxHTTPLimitReqRequests:               MetricConfig{Enabled: false},
					NginxHTTPRequestByteIo:                  MetricConfig{Enabled: false},
					NginxHTTPRequestDiscarded:               MetricConfig{Enabled: false},
					NginxHTTPRequestProcessingCount:         MetricConfig{Enabled: false},
					NginxHTTPRequests:                       MetricConfig{Enabled: false},
					NginxHTTPRequestsCount:                  MetricConfig{Enabled: false},
					NginxHTTPResponseStatus:                 MetricConfig{Enabled: false},
					NginxHTTPResponses:                      MetricConfig{Enabled: false},
					NginxHTTPUpstreamKeepaliveCount:         MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerByteIo:             MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerConnectionsCount:   MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerCount:              MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerFails:              MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerHeaderTime:         MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerHealthChecks:       MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerRequests:           MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerResponseTime:       MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerResponses:          MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerState:              MetricConfig{Enabled: false},
					NginxHTTPUpstreamPeerUnavailables:       MetricConfig{Enabled: false},
					NginxHTTPUpstreamQueueLimit:             MetricConfig{Enabled: false},
					NginxHTTPUpstreamQueueOverflows:         MetricConfig{Enabled: false},
					NginxHTTPUpstreamQueueUsage:             MetricConfig{Enabled: false},
					NginxHTTPUpstreamZombieCount:            MetricConfig{Enabled: false},
					NginxSlabPageFree:                       MetricConfig{Enabled: false},
					NginxSlabPageLimit:                      MetricConfig{Enabled: false},
					NginxSlabPageUsage:                      MetricConfig{Enabled: false},
					NginxSlabPageUtilization:                MetricConfig{Enabled: false},
					NginxSlabSlotAllocations:                MetricConfig{Enabled: false},
					NginxSlabSlotFree:                       MetricConfig{Enabled: false},
					NginxSlabSlotUsage:                      MetricConfig{Enabled: false},
					NginxSslCertificateVerifyFailures:       MetricConfig{Enabled: false},
					NginxSslHandshakes:                      MetricConfig{Enabled: false},
					NginxStreamByteIo:                       MetricConfig{Enabled: false},
					NginxStreamConnectionsAccepted:          MetricConfig{Enabled: false},
					NginxStreamConnectionsDiscarded:         MetricConfig{Enabled: false},
					NginxStreamConnectionsProcessingCount:   MetricConfig{Enabled: false},
					NginxStreamSessionStatus:                MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerByteIo:           MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerConnections:      MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerConnectionsCount: MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerConnectionsTime:  MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerCount:            MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerFails:            MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerHealthChecks:     MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerResponseTime:     MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerState:            MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerTtfbTime:         MetricConfig{Enabled: false},
					NginxStreamUpstreamPeerUnavailable:      MetricConfig{Enabled: false},
					NginxStreamUpstreamZombieCount:          MetricConfig{Enabled: false},
				},
				ResourceAttributes: ResourceAttributesConfig{
					InstanceID:   ResourceAttributeConfig{Enabled: false},
					InstanceType: ResourceAttributeConfig{Enabled: false},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadMetricsBuilderConfig(t, tt.name)
			if diff := cmp.Diff(tt.want, cfg, cmpopts.IgnoreUnexported(MetricConfig{}, ResourceAttributeConfig{})); diff != "" {
				t.Errorf("Config mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func loadMetricsBuilderConfig(t *testing.T, name string) MetricsBuilderConfig {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	sub, err := cm.Sub(name)
	require.NoError(t, err)
	cfg := DefaultMetricsBuilderConfig()
	require.NoError(t, sub.Unmarshal(&cfg))
	return cfg
}

func TestResourceAttributesConfig(t *testing.T) {
	tests := []struct {
		name string
		want ResourceAttributesConfig
	}{
		{
			name: "default",
			want: DefaultResourceAttributesConfig(),
		},
		{
			name: "all_set",
			want: ResourceAttributesConfig{
				InstanceID:   ResourceAttributeConfig{Enabled: true},
				InstanceType: ResourceAttributeConfig{Enabled: true},
			},
		},
		{
			name: "none_set",
			want: ResourceAttributesConfig{
				InstanceID:   ResourceAttributeConfig{Enabled: false},
				InstanceType: ResourceAttributeConfig{Enabled: false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadResourceAttributesConfig(t, tt.name)
			if diff := cmp.Diff(tt.want, cfg, cmpopts.IgnoreUnexported(ResourceAttributeConfig{})); diff != "" {
				t.Errorf("Config mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func loadResourceAttributesConfig(t *testing.T, name string) ResourceAttributesConfig {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	sub, err := cm.Sub(name)
	require.NoError(t, err)
	sub, err = sub.Sub("resource_attributes")
	require.NoError(t, err)
	cfg := DefaultResourceAttributesConfig()
	require.NoError(t, sub.Unmarshal(&cfg))
	return cfg
}
