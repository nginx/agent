/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/nap"
	tutils "github.com/nginx/agent/v2/test/utils"
)

const (
	testSystemID      = "12345678"
	testSigDate1      = "2022.02.14"
	testCampaignDate1 = "2022.02.07"
	testWAFVersion    = "3.780.1"
)

var (
	testNAPDetailsActive = &proto.DataplaneSoftwareDetails_AppProtectWafDetails{
		AppProtectWafDetails: &proto.AppProtectWAFDetails{
			WafLocation:             nap.APP_PROTECT_METADATA_FILE_PATH,
			WafVersion:              testWAFVersion,
			AttackSignaturesVersion: testSigDate1,
			ThreatCampaignsVersion:  testCampaignDate1,
			Health: &proto.AppProtectWAFHealth{
				SystemId:            testSystemID,
				AppProtectWafStatus: proto.AppProtectWAFHealth_ACTIVE,
			},
		},
	}

	testNAPDetailsUnknown = &proto.DataplaneSoftwareDetails_AppProtectWafDetails{
		AppProtectWafDetails: &proto.AppProtectWAFDetails{
			WafLocation: nap.APP_PROTECT_METADATA_FILE_PATH,
			Health: &proto.AppProtectWAFHealth{
				SystemId:            testSystemID,
				AppProtectWafStatus: proto.AppProtectWAFHealth_UNKNOWN,
			},
		},
	}

	testNAPDetailsDegraded = &proto.DataplaneSoftwareDetails_AppProtectWafDetails{
		AppProtectWafDetails: &proto.AppProtectWAFDetails{
			WafLocation:             nap.APP_PROTECT_METADATA_FILE_PATH,
			WafVersion:              testWAFVersion,
			AttackSignaturesVersion: testSigDate1,
			ThreatCampaignsVersion:  testCampaignDate1,
			Health: &proto.AppProtectWAFHealth{
				SystemId:            testSystemID,
				AppProtectWafStatus: proto.AppProtectWAFHealth_DEGRADED,
				DegradedReason:      napDegradedMessage,
			},
		},
	}
)

func TestNginxAppProtect(t *testing.T) {
	env := tutils.GetMockEnvWithProcess()

	config := &config.Config{}

	napPlugin, err := NewNginxAppProtect(config, env, NginxAppProtectConfig{
		ReportInterval: time.Duration(15) * time.Second,
	})
	assert.NoError(t, err)
	defer napPlugin.Close()

	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{}, []core.ExtensionPlugin{napPlugin})
	messagePipe.Run()

	t.Run("returns get response", func(t *testing.T) {
		currentNAPPluginDetails := napPlugin.generateNAPDetailsProtoCommand()
		assert.Equal(t, testNAPDetailsUnknown, currentNAPPluginDetails)

		// Update the NAP information to active/running
		napPlugin.nap = nap.NginxAppProtect{
			Status:                  nap.RUNNING.String(),
			AttackSignaturesVersion: testSigDate1,
			ThreatCampaignsVersion:  testCampaignDate1,
			Release:                 nap.ReleaseUnmappedBuild("3.780.1"),
		}
		currentNAPPluginDetails = napPlugin.generateNAPDetailsProtoCommand()
		assert.Equal(t, testNAPDetailsActive, currentNAPPluginDetails)

		// Update the NAP information to degraded/installed
		napPlugin.nap = nap.NginxAppProtect{
			Status:                  nap.INSTALLED.String(),
			AttackSignaturesVersion: testSigDate1,
			ThreatCampaignsVersion:  testCampaignDate1,
			Release:                 nap.ReleaseUnmappedBuild("3.780.1"),
		}
		currentNAPPluginDetails = napPlugin.generateNAPDetailsProtoCommand()
		assert.Equal(t, testNAPDetailsDegraded, currentNAPPluginDetails)
	})
}
