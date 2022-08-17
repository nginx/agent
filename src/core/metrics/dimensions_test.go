package metrics

import (
	"testing"

	"github.com/shirou/gopsutil/host"
	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	displayName1   = "displayName1"
	instanceGroup1 = "group1"
)

func TestNewCommonDim(t *testing.T) {
	h, _ := host.Info()
	hostInfo := &proto.HostInfo{
		Hostname:    h.Hostname,
		DisplayName: h.Hostname,
		Uuid:        h.HostID,
	}

	type args struct {
		agentVersion string
		config       config.Config
		binpath      string
	}
	tests := []struct {
		name     string
		args     args
		expected *CommonDim
	}{
		{
			name: "base new dimension",
			args: args{
				agentVersion: "v1",
				config:       config.Config{},
				binpath:      "/usr/sbin/nginx",
			},
			expected: &CommonDim{
				SystemId:      h.HostID,
				Hostname:      h.Hostname,
				InstanceTags:  "",
				InstanceGroup: "",
				DisplayName:   "",
				NginxId:       "",
			},
		},
		{
			name: "base new dimension",
			args: args{
				agentVersion: "v1",
				config: config.Config{
					DisplayName:   displayName1,
					InstanceGroup: instanceGroup1,
					Tags:          []string{"FooBar"},
				},
				binpath: "/usr/sbin/nginx",
			},
			expected: &CommonDim{
				SystemId:      h.HostID,
				Hostname:      h.Hostname,
				InstanceTags:  "FooBar",
				InstanceGroup: instanceGroup1,
				DisplayName:   displayName1,
				NginxId:       "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostInfo.Agent = tt.args.agentVersion
			got := NewCommonDim(hostInfo, &tt.args.config, "")

			assert.Equal(t, got.SystemId, tt.expected.SystemId)
			assert.Equal(t, got.Hostname, tt.expected.Hostname)
			assert.Equal(t, got.InstanceTags, tt.expected.InstanceTags)
			assert.Equal(t, got.InstanceGroup, tt.expected.InstanceGroup)
			assert.Equal(t, got.DisplayName, tt.expected.DisplayName)

			// TODO
			assert.Empty(t, got.NginxId)
		})
	}
}

func TestCommonDim_ToDimensions(t *testing.T) {
	h, _ := host.Info()
	hostInfo := &proto.HostInfo{
		Hostname:    h.Hostname,
		DisplayName: h.Hostname,
		Uuid:        h.HostID,
	}
	baseDim := NewCommonDim(hostInfo, &config.Config{}, "")

	tests := []struct {
		name     string
		dims     *CommonDim
		expected []*proto.Dimension
	}{
		{
			name: "base case",
			dims: baseDim,
			expected: []*proto.Dimension{
				{
					Name:  systemIDKey,
					Value: baseDim.SystemId,
				},
				{
					Name:  hostnameKey,
					Value: baseDim.Hostname,
				},
				{
					Name:  systemTagsKey,
					Value: baseDim.InstanceTags,
				},
				{
					Name:  instanceGroupKey,
					Value: baseDim.InstanceGroup,
				},
				{
					Name:  displayNameKey,
					Value: baseDim.DisplayName,
				},
				{
					Name:  nginxIDKey,
					Value: baseDim.NginxId,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dims.ToDimensions()
			assert.Equal(t, got, tt.expected)
		})
	}
}
