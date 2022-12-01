/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"strings"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
)

const (
	systemIDKey      = "system_id"
	hostnameKey      = "hostname"
	systemTagsKey    = "system.tags"
	displayNameKey   = "display_name"
	nginxIDKey       = "nginx_id"
	instanceGroupKey = "instance_group"
)

// CommonDim is the set of dimensions that apply to all metrics
type CommonDim struct {
	SystemId, Hostname, InstanceTags, InstanceGroup,
	DisplayName, NginxId string

	PublishedAPI, NginxType, NginxBuild, NginxVersion,
	NginxBinPath, NginxConfPath string

	NginxAccessLogPaths []string
}

func NewCommonDim(hostInfo *proto.HostInfo, conf *config.Config, nginxId string) *CommonDim {
	var hostTags string
	if len(conf.Tags) > 0 {
		hostTags = strings.Join(conf.Tags, ",")
	}

	commonDim := &CommonDim{
		SystemId:      hostInfo.Uuid,
		Hostname:      hostInfo.Hostname,
		InstanceTags:  hostTags,
		InstanceGroup: conf.InstanceGroup,
		DisplayName:   conf.DisplayName,
		NginxId:       nginxId,
	}

	log.Debugf("Common Metric Dimensions: %v", commonDim.ToDimensions())

	return commonDim
}

// ToDimensions returns the set of common agent dimensions
// Ensures dimensions are generated in the same order every time, as required by control plane
func (c *CommonDim) ToDimensions() []*proto.Dimension {
	return []*proto.Dimension{
		{
			Name:  systemIDKey,
			Value: c.SystemId,
		},
		{
			Name:  hostnameKey,
			Value: c.Hostname,
		},
		{
			Name:  systemTagsKey,
			Value: c.InstanceTags,
		},
		{
			Name:  instanceGroupKey,
			Value: c.InstanceGroup,
		},
		{
			Name:  displayNameKey,
			Value: c.DisplayName,
		},
		{
			Name:  nginxIDKey,
			Value: c.NginxId,
		},
	}
}
